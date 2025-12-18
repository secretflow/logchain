package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresStore implements the Store interface
// This is a shared store used by both API Gateway and Engine
// It should be moved to a common location like internal/store/
type PostgresStore struct {
	db     *pgxpool.Pool
	logger *log.Logger
}

// NewPostgresStore creates a new PostgresStore instance
// Uses shared configuration for both API Gateway and Engine
func NewPostgresStore(ctx context.Context, dsn string, maxConns, minConns int, logger *log.Logger) (*PostgresStore, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database connection string: %w", err)
	}

	// Configure connection pool for shared usage
	if maxConns > 0 {
		poolConfig.MaxConns = int32(maxConns)
	} else {
		poolConfig.MaxConns = 50 // Default
	}

	if minConns > 0 {
		poolConfig.MinConns = int32(minConns)
	} else {
		poolConfig.MinConns = 10 // Default
	}

	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	dbpool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbpool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Println("Successfully connected to PostgreSQL database")
	return &PostgresStore{db: dbpool, logger: logger}, nil
}

// Close closes the database connection pool
func (s *PostgresStore) Close() {
	s.db.Close()
	s.logger.Println("PostgreSQL database connection closed")
}

// GetAndMarkBatchAsProcessing uses a single atomic CTE query to lock, filter,
// update, and return tasks ready for processing.
func (s *PostgresStore) GetAndMarkBatchAsProcessing(ctx context.Context, requestIDs []string, maxRetries int) (map[string]*LogStatus, error) {
	if len(requestIDs) == 0 {
		return make(map[string]*LogStatus), nil
	}

	processingTasks := make(map[string]*LogStatus)
	now := time.Now()
	failedReason := fmt.Sprintf("reached maximum retry count (%d)", maxRetries)

	atomicQuery := `
        WITH locked_rows AS (
            -- 1. Lock and select only the tasks we care about
            SELECT request_id, log_hash, source_org_id, received_timestamp, status, retry_count
            FROM tbl_log_status
            WHERE request_id = ANY($1) AND status = $2 -- $1=requestIDs, $2=StatusReceived
            FOR UPDATE SKIP LOCKED
        ),
        failed_tasks AS (
            -- 2. Update tasks that have exceeded the retry limit
            UPDATE tbl_log_status
            SET status = $3, -- StatusFailed
                error_message = $4, -- failedReason
                processing_finished_at = $5 -- now
            FROM locked_rows
            WHERE tbl_log_status.request_id = locked_rows.request_id
              AND locked_rows.retry_count >= $6 -- maxRetries
        )
        -- 3. Update tasks that are ready for processing
        UPDATE tbl_log_status
        SET status = $7, -- StatusProcessing
            processing_started_at = $5 -- now
        FROM locked_rows
        WHERE tbl_log_status.request_id = locked_rows.request_id
          AND locked_rows.retry_count < $6 -- maxRetries
        -- 4. Return *only* the tasks we just marked for processing
        RETURNING
            tbl_log_status.request_id,
            tbl_log_status.log_hash,
            tbl_log_status.source_org_id,
            tbl_log_status.received_timestamp,
            tbl_log_status.status, -- Will be 'PROCESSING'
            tbl_log_status.retry_count,
            tbl_log_status.processing_started_at;
    `

	// We keep your original BeginFunc pattern for transactional safety
	err := s.db.BeginFunc(ctx, func(tx pgx.Tx) error {

		// **REMOVED pgtype.TextArray**
		// pgx v5 handles Go slices (like requestIDs) directly.

		// Execute the single atomic query
		// We pass requestIDs ([]string) directly as the $1 argument
		rows, err := tx.Query(ctx, atomicQuery,
			requestIDs,       // $1
			StatusReceived,   // $2
			StatusFailed,     // $3
			failedReason,     // $4
			now,              // $5
			maxRetries,       // $6
			StatusProcessing, // $7
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // No rows were returned (none to process), not an error
			}
			return fmt.Errorf("failed to execute atomic update/select: %w", err)
		}
		defer rows.Close()

		// Scan the rows that were returned by the RETURNING clause
		for rows.Next() {
			var task LogStatus
			var processingStartedAt time.Time // Use non-pointer for scanning

			if err := rows.Scan(
				&task.RequestID,
				&task.LogHash,
				&task.SourceOrgID,
				&task.ReceivedTimestamp,
				&task.Status,
				&task.RetryCount,
				&processingStartedAt, // Scan into the local variable
			); err != nil {
				return fmt.Errorf("failed to scan processed task row: %w", err)
			}

			task.ProcessingStartedAt = &processingStartedAt // Assign pointer
			processingTasks[task.RequestID] = &task
		}
		if rows.Err() != nil {
			return fmt.Errorf("error iterating query results: %w", rows.Err())
		}

		// s.logger.Printf("Atomically marked %d tasks as PROCESSING", len(processingTasks))

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err
	}

	return processingTasks, nil
}

func (s *PostgresStore) MarkBatchAsCompleted(ctx context.Context, completions []CompletionRecord) error {
	if len(completions) == 0 {
		return nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	err := s.db.BeginFunc(queryCtx, func(tx pgx.Tx) error {
		now := time.Now()

		requestIDs := make([]string, len(completions))
		txHashes := make([]string, len(completions))
		logHashes := make([]string, len(completions))
		blockHeights := make([]int64, len(completions))

		for i, c := range completions {
			requestIDs[i] = c.RequestID
			txHashes[i] = c.TxHash
			logHashes[i] = c.LogHashOnChain
			blockHeights[i] = int64(c.BlockHeight)
		}

		updateQuery := `
            UPDATE tbl_log_status
            SET status = 'COMPLETED',
                tx_hash = data.tx_hash,
                log_hash_on_chain = data.log_hash,
                block_height = data.block_height,
                processing_finished_at = $1,
                error_message = NULL
            FROM (
                SELECT 
                    request_id,
                    ($3::text[])[idx] AS tx_hash,
                    ($4::text[])[idx] AS log_hash,
                    ($5::bigint[])[idx] AS block_height
                FROM
                    UNNEST($2::text[]) WITH ORDINALITY AS t(request_id, idx)
            ) AS data
            WHERE tbl_log_status.request_id = data.request_id 
              AND tbl_log_status.status = 'PROCESSING'
        `

		cmdTag, err := tx.Exec(queryCtx, updateQuery,
			now,
			requestIDs,
			txHashes,
			logHashes,
			blockHeights,
		)
		if err != nil {
			return fmt.Errorf("batch update failed: %w", err)
		}

		rowsAffected := cmdTag.RowsAffected()
		if rowsAffected != int64(len(completions)) {
			s.logger.Printf("Warning: expected to update %d rows, but updated %d rows",
				len(completions), rowsAffected)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("batch completion update failed: %w", err)
	}

	return nil
}

// MarkBatchAsFailed efficiently updates a batch of records to 'FAILED' status
// using a single database query with UNNEST.
func (s *PostgresStore) MarkBatchAsFailed(ctx context.Context, failures []FailureRecord) error {
	if len(failures) == 0 {
		return nil // Nothing to do
	}

	// Use a slightly longer timeout for batch operations
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	err := s.db.BeginFunc(queryCtx, func(tx pgx.Tx) error {
		now := time.Now()

		// 1. Prepare data slices for batch parameters
		requestIDs := make([]string, len(failures))
		errorMessages := make([]string, len(failures))

		for i, f := range failures {
			requestIDs[i] = f.RequestID
			errorMessages[i] = f.ErrorMessage
		}

		// 2. Construct a single UPDATE query using UNNEST WITH ORDINALITY.
		// This avoids the N+1 query problem.
		updateQuery := `
            UPDATE tbl_log_status
            SET
                status = 'FAILED',
                error_message = data.error_msg,
                processing_finished_at = $1 -- now
            FROM (
                SELECT
                    request_id,
                    ($3::text[])[idx] AS error_msg -- Get error_msg by index
                FROM
                    -- Unnest only the primary array (requestIDs) to get the index (idx)
                    UNNEST($2::text[]) WITH ORDINALITY AS t(request_id, idx)
            ) AS data
            WHERE tbl_log_status.request_id = data.request_id
              AND tbl_log_status.status != 'FAILED' -- Maintain original logic
        `

		// 3. Execute the single batch query
		cmdTag, err := tx.Exec(queryCtx, updateQuery,
			now,           // $1
			requestIDs,    // $2
			errorMessages, // $3
		)
		if err != nil {
			return fmt.Errorf("batch failure update failed: %w", err)
		}

		// 4. (Optional) Check the number of rows affected
		rowsAffected := cmdTag.RowsAffected()
		if rowsAffected != int64(len(failures)) {
			// This is just a warning. Some rows might have already been 'FAILED'
			// or the request_id might not match, so they were skipped.
			s.logger.Printf("Warning: batch failure update expected to affect %d rows, but affected %d rows",
				len(failures), rowsAffected)
		}

		return nil // Commit the transaction
	})

	if err != nil {
		return fmt.Errorf("batch failure update failed: %w", err)
	}

	return nil
}

// MarkBatchForRetry restores a batch of tasks to Received and increments retry count
func (s *PostgresStore) MarkBatchForRetry(ctx context.Context, requestIDs []string, lastError string) error {
	if len(requestIDs) == 0 {
		return nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := s.db.BeginFunc(queryCtx, func(tx pgx.Tx) error {
		query := `
            UPDATE tbl_log_status
            SET status = $1, retry_count = retry_count + 1, error_message = $2, processing_started_at = NULL
            WHERE request_id = ANY($3) AND status = $4
        `

		cmdTag, err := tx.Exec(queryCtx, query, StatusReceived, lastError, requestIDs, StatusProcessing)
		if err != nil {
			return fmt.Errorf("failed to batch mark tasks as RETRY: %w", err)
		}
		s.logger.Printf("Attempted to mark %d tasks as RETRY, actually updated %d rows", len(requestIDs), cmdTag.RowsAffected())
		return nil
	})
	return err
}

// InsertLogStatusBatch performs a high-performance bulk insertion using UNNEST
func (s *PostgresStore) InsertLogStatusBatch(ctx context.Context, statuses []*LogStatus) error {
	if len(statuses) == 0 {
		return nil
	}

	// 10-15s might be safer for a single large query
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 1. Prepare parallel slices for all columns
	requestIDs := make([]string, len(statuses))
	logHashes := make([]string, len(statuses))
	sourceOrgIDs := make([]string, len(statuses))
	receivedTimestamps := make([]time.Time, len(statuses))
	statusStrings := make([]string, len(statuses))
	// retry_count is static (0), so we don't need a slice for it

	for i, status := range statuses {
		requestIDs[i] = status.RequestID
		logHashes[i] = status.LogHash
		sourceOrgIDs[i] = status.SourceOrgID
		receivedTimestamps[i] = status.ReceivedTimestamp
		statusStrings[i] = string(status.Status)
	}

	// 2. Construct a single query using UNNEST WITH ORDINALITY
	query := `
        INSERT INTO tbl_log_status (
            request_id, 
            log_hash, 
            source_org_id, 
            received_timestamp, 
            status, 
            retry_count
        )
        SELECT
            request_id,                             -- From the UNNEST
            ($2::text[])[idx] AS log_hash,          -- Indexed from param $2
            ($3::text[])[idx] AS source_org_id,     -- Indexed from param $3
            ($4::timestamptz[])[idx] AS received_timestamp, -- Indexed from param $4
            ($5::text[])[idx] AS status,            -- Indexed from param $5
            0 AS retry_count                        -- Static value
        FROM
            -- Unnest the primary key array to drive the loop
            UNNEST($1::text[]) WITH ORDINALITY AS t(request_id, idx)
        ON CONFLICT (request_id) DO NOTHING
    `

	// 3. Execute the single query
	_, err := s.db.Exec(queryCtx, query,
		requestIDs,         // $1
		logHashes,          // $2
		sourceOrgIDs,       // $3
		receivedTimestamps, // $4
		statusStrings,      // $5
	)

	if err != nil {
		return fmt.Errorf("failed to batch insert log statuses with unnest: %w", err)
	}

	return nil
}

// GetLogStatusByRequestID queries log status by request_id
func (s *PostgresStore) GetLogStatusByRequestID(ctx context.Context, requestID string) (*LogStatus, error) {
	query := `
		SELECT request_id, log_hash, source_org_id, received_timestamp,
		       status, received_at_db, processing_started_at, processing_finished_at,
		       tx_hash, block_height, log_hash_on_chain, error_message, retry_count
		FROM tbl_log_status
		WHERE request_id = $1
	`

	var status LogStatus
	err := s.db.QueryRow(ctx, query, requestID).Scan(
		&status.RequestID,
		&status.LogHash,
		&status.SourceOrgID,
		&status.ReceivedTimestamp,
		&status.Status,
		&status.ReceivedAtDB,
		&status.ProcessingStartedAt,
		&status.ProcessingFinishedAt,
		&status.TxHash,
		&status.BlockHeight,
		&status.LogHashOnChain,
		&status.ErrorMessage,
		&status.RetryCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("log not found with request_id=%s: %w", requestID, err)
		}
		return nil, fmt.Errorf("failed to query log status by request_id: %w", err)
	}

	return &status, nil
}

// GetLogStatusByHash queries log status by log_hash
func (s *PostgresStore) GetLogStatusByHash(ctx context.Context, logHash string) (*LogStatus, error) {
	query := `
		SELECT request_id, log_hash, source_org_id, received_timestamp,
		       status, received_at_db, processing_started_at, processing_finished_at,
		       tx_hash, block_height, log_hash_on_chain, error_message, retry_count
		FROM tbl_log_status
		WHERE log_hash = $1
	`

	var status LogStatus
	err := s.db.QueryRow(ctx, query, logHash).Scan(
		&status.RequestID,
		&status.LogHash,
		&status.SourceOrgID,
		&status.ReceivedTimestamp,
		&status.Status,
		&status.ReceivedAtDB,
		&status.ProcessingStartedAt,
		&status.ProcessingFinishedAt,
		&status.TxHash,
		&status.BlockHeight,
		&status.LogHashOnChain,
		&status.ErrorMessage,
		&status.RetryCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("log not found with log_hash=%s: %w", logHash, err)
		}
		return nil, fmt.Errorf("failed to query log status by log_hash: %w", err)
	}

	return &status, nil
}
