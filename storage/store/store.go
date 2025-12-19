package store

import (
	"context"
	"errors"
	"time"
)

// Store-level errors
var (
	ErrLogNotFound = errors.New("log not found")
)

// Status defines the task status enum type
type Status string

const (
	StatusReceived   Status = "RECEIVED"
	StatusProcessing Status = "PROCESSING"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

// CompletionRecord represents a completed log record for batch updates
type CompletionRecord struct {
	RequestID      string
	TxHash         string
	LogHashOnChain string
	BlockHeight    uint64
}

// FailureRecord represents a failed log record for batch updates
type FailureRecord struct {
	RequestID    string
	ErrorMessage string
}

// LogStatus is the Go struct corresponding to the database table Tbl_Log_Status
type LogStatus struct {
	RequestID            string     `db:"request_id"`
	LogHash              string     `db:"log_hash"`
	SourceOrgID          string     `db:"source_org_id"`
	ReceivedTimestamp    time.Time  `db:"received_timestamp"`
	Status               Status     `db:"status"`
	ReceivedAtDB         time.Time  `db:"received_at_db"`
	ProcessingStartedAt  *time.Time `db:"processing_started_at"`
	ProcessingFinishedAt *time.Time `db:"processing_finished_at"`
	TxHash               *string    `db:"tx_hash"`
	BlockHeight          *int64     `db:"block_height"`
	LogHashOnChain       *string    `db:"log_hash_on_chain"`
	ErrorMessage         *string    `db:"error_message"`
	RetryCount           int        `db:"retry_count"`
}

// Store is the data storage interface
type Store interface {

	// GetAndMarkBatchAsProcessing attempts to batch lock tasks with RECEIVED status
	GetAndMarkBatchAsProcessing(ctx context.Context, requestIDs []string, maxRetries int) (map[string]*LogStatus, error)

	// MarkBatchAsCompleted marks multiple tasks as successfully completed in a single transaction
	MarkBatchAsCompleted(ctx context.Context, completions []CompletionRecord) error

	// MarkBatchAsFailed marks multiple tasks as failed in a single transaction
	MarkBatchAsFailed(ctx context.Context, failures []FailureRecord) error

	// MarkBatchForRetry restores a batch of tasks to Received and increments retry count
	MarkBatchForRetry(ctx context.Context, requestIDs []string, lastError string) error

	// InsertLogStatusBatch performs bulk insertion of log statuses
	InsertLogStatusBatch(ctx context.Context, statuses []*LogStatus) error

	// GetLogStatusByRequestID queries log status by request_id
	GetLogStatusByRequestID(ctx context.Context, requestID string) (*LogStatus, error)

	// GetLogStatusByHash queries log status by log_hash
	GetLogStatusByHash(ctx context.Context, logHash string) (*LogStatus, error)

	// Close closes the database connection
	Close()
}
