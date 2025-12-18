-- Migration: Add index for query layer performance
-- Description: Add request_id index for query by request_id API
-- Note: idx_log_status_log_hash already exists in init-db.sql

-- Add index for request_id queries (API 1: GET /v1/status/{request_id})
CREATE INDEX IF NOT EXISTS idx_log_status_request_id ON tbl_log_status(request_id);

-- Add comment for documentation
COMMENT ON INDEX idx_log_status_request_id IS 'Index for query by request_id (Query Layer API 1)';
