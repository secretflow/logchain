package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"

	blockchain "tlng/blockchain/client"
	"tlng/storage/store"
)

// Service provides core query business logic
type Service struct {
	store      store.Store
	blockchain blockchain.BlockchainClient
	logger     *log.Logger
}

// NewService creates a new query service instance
func NewService(storeDB store.Store, bc blockchain.BlockchainClient, logger *log.Logger) *Service {
	return &Service{
		store:      storeDB,
		blockchain: bc,
		logger:     logger,
	}
}

// GetStatusByRequestID queries log status by request_id
// Only allows querying logs from the caller's organization
func (s *Service) GetStatusByRequestID(ctx context.Context, requestID, callerOrgID string) (*LogStatusResponse, error) {
	if requestID == "" {
		return nil, ErrInvalidRequest
	}

	// Query from State DB
	status, err := s.store.GetLogStatusByRequestID(ctx, requestID)
	if err != nil {
		if errors.Is(err, store.ErrLogNotFound) {
			return nil, ErrLogNotFound
		}
		s.logger.Printf("Failed to query log status by request_id=%s: %v", requestID, err)
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	// Permission check: only allow querying own organization's logs
	if status.SourceOrgID != callerOrgID {
		s.logger.Printf("Permission denied: caller_org=%s tried to access log from org=%s", callerOrgID, status.SourceOrgID)
		return nil, ErrPermissionDenied
	}

	// Convert to response format
	return convertToResponse(status), nil
}

// QueryByContent queries log status by calculating hash from content
// Only allows querying logs from the caller's organization
func (s *Service) QueryByContent(ctx context.Context, logContent, callerOrgID string) (*LogStatusResponse, error) {
	if logContent == "" {
		return nil, ErrInvalidRequest
	}

	// Calculate log_hash from content
	logHash := calculateSHA256(logContent)

	// Query from State DB
	status, err := s.store.GetLogStatusByHash(ctx, logHash)
	if err != nil {
		if errors.Is(err, store.ErrLogNotFound) {
			return nil, ErrLogNotFound
		}
		s.logger.Printf("Failed to query log status by log_hash=%s: %v", logHash, err)
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	// Permission check: only allow querying own organization's logs
	if status.SourceOrgID != callerOrgID {
		s.logger.Printf("Permission denied: caller_org=%s tried to access log from org=%s", callerOrgID, status.SourceOrgID)
		return nil, ErrPermissionDenied
	}

	// Convert to response format
	return convertToResponse(status), nil
}

// AuditLogByHash performs on-chain audit query by log_hash
// No permission restrictions - consortium members can audit all logs
func (s *Service) AuditLogByHash(ctx context.Context, logHash string) (*OnChainLogResponse, error) {
	if logHash == "" {
		return nil, ErrInvalidRequest
	}

	if s.blockchain == nil {
		return nil, fmt.Errorf("blockchain client not available")
	}

	// Query blockchain
	rawData, err := s.blockchain.FindLogByHash(ctx, logHash)
	if err != nil {
		s.logger.Printf("Failed to query blockchain for log_hash=%s: %v", logHash, err)
		return nil, ErrBlockchainError
	}

	if rawData == "" {
		return nil, ErrLogNotFound
	}

	// Parse on-chain data
	logData, err := parseOnChainData(rawData)
	if err != nil {
		s.logger.Printf("Failed to parse on-chain data for log_hash=%s: %v", logHash, err)
		return nil, fmt.Errorf("failed to parse on-chain data: %w", err)
	}

	// Return structured response
	return &OnChainLogResponse{
		Source:      "blockchain",
		LogHash:     logHash,
		LogContent:  logData.Content,
		SenderOrgID: logData.OrgID,
		Timestamp:   logData.Timestamp,
	}, nil
}

// OnChainLogData represents parsed on-chain log data
type OnChainLogData struct {
	OrgID     string
	Timestamp string
	Content   string
}

// parseOnChainData parses blockchain response data in key=value&key=value format
// ChainMaker SDK GetContractInfo returns the result field as a plain string
func parseOnChainData(raw string) (*OnChainLogData, error) {
	// Parse key=value pairs (reusing url.ParseQuery for parsing convenience)
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query string from '%s': %w", raw, err)
	}

	data := &OnChainLogData{
		OrgID:     values.Get("org_id"),
		Timestamp: values.Get("ts"),
		Content:   values.Get("content"),
	}

	// Validate required fields
	if data.OrgID == "" || data.Timestamp == "" || data.Content == "" {
		return nil, fmt.Errorf("incomplete on-chain data: org_id=%s, ts=%s, content_len=%d",
			data.OrgID, data.Timestamp, len(data.Content))
	}

	return data, nil
}

// calculateSHA256 computes the SHA256 hash of the input string
func calculateSHA256(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// convertToResponse converts store.LogStatus to LogStatusResponse
func convertToResponse(status *store.LogStatus) *LogStatusResponse {
	resp := &LogStatusResponse{
		RequestID:         status.RequestID,
		LogHash:           status.LogHash,
		SourceOrgID:       status.SourceOrgID,
		Status:            string(status.Status),
		ReceivedTimestamp: status.ReceivedTimestamp,
	}

	// Add optional fields if present
	if status.ProcessingStartedAt != nil {
		resp.ProcessingStartedAt = status.ProcessingStartedAt
	}
	if status.ProcessingFinishedAt != nil {
		resp.ProcessingFinishedAt = status.ProcessingFinishedAt
	}
	if status.TxHash != nil {
		resp.TxHash = *status.TxHash
	}
	if status.BlockHeight != nil {
		resp.BlockHeight = *status.BlockHeight
	}
	if status.ErrorMessage != nil {
		resp.ErrorMessage = *status.ErrorMessage
	}

	return resp
}
