-- PostgreSQL initialization script for TLNG project
-- Creates tbl_log_status table and indexes if they don't exist

-- Create the main table
CREATE TABLE IF NOT EXISTS tbl_log_status (
    request_id TEXT PRIMARY KEY,
    log_hash TEXT NOT NULL,
    source_org_id TEXT,
    received_timestamp TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    received_at_db TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_started_at TIMESTAMPTZ,
    processing_finished_at TIMESTAMPTZ,
    tx_hash TEXT,
    block_height BIGINT,
    log_hash_on_chain TEXT,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0
);

-- Indexes for query APIs
-- API 1: GET /v1/status/{request_id} - uses request_id (already PRIMARY KEY, no extra index needed)
-- API 2: POST /v1/query - uses log_hash for content-based lookup
CREATE INDEX IF NOT EXISTS idx_log_status_log_hash ON tbl_log_status (log_hash);
-- API 3: GET /v1/audit/log/{log_hash} - uses log_hash (covered by above index)