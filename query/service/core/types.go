package core

import "time"

// LogStatusResponse represents the response for log status queries
type LogStatusResponse struct {
	RequestID            string     `json:"request_id"`
	LogHash              string     `json:"log_hash"`
	SourceOrgID          string     `json:"source_org_id"`
	Status               string     `json:"status"`
	ReceivedTimestamp    time.Time  `json:"received_timestamp"`
	ProcessingStartedAt  *time.Time `json:"processing_started_at,omitempty"`
	ProcessingFinishedAt *time.Time `json:"processing_finished_at,omitempty"`
	TxHash               string     `json:"tx_hash,omitempty"`
	BlockHeight          int64      `json:"block_height,omitempty"`
	ErrorMessage         string     `json:"error_message,omitempty"`
}

// OnChainLogResponse represents the response for blockchain audit queries
type OnChainLogResponse struct {
	Source      string `json:"source"`
	LogHash     string `json:"log_hash"`
	LogContent  string `json:"log_content"`
	SenderOrgID string `json:"sender_org_id"`
	Timestamp   string `json:"timestamp"`
}
