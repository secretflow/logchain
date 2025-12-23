# Scripts Directory

Utility scripts for LogChain system management, testing, and configuration.

## Scripts Overview

### API Testing Scripts

#### `test-ingestion-query-api.sh`
Tests log ingestion and query APIs with API Key authentication.

**Usage:**
```bash
./test-ingestion-query-api.sh [api-key]
```

**Default API Key:** `example-api-key-12345`

**Tests performed:**
1. Submit log via `POST /v1/logs`
2. Query status by request_id via `GET /v1/query/status/{id}`
3. Query by content via `POST /v1/query_by_content`
4. Permission isolation test (should fail for non-existent logs)

#### `test-consortium-audit-api.sh`
Tests consortium audit APIs with mTLS authentication.

**Usage:**
```bash
./test-consortium-audit-api.sh <log-hash>
```

**Example:**
```bash
./test-consortium-audit-api.sh 40dc7a0be4aaab2b8cd7982104bb5f029da283766451f1a8de41f1458da8a80c
```

**Prerequisites:** Generate client certificates first:
```bash
cd ../ingress
bash scripts/generate-client-cert.sh member-001 "Regulatory Authority A"
```

### Configuration Scripts

#### `generate-chainmaker-config.sh`
Generates ChainMaker client configuration from template by substituting environment variables.

**Usage:**
```bash
./generate-chainmaker-config.sh
```

**Prerequisites:**
- `.env` file configured (copy from `.env.example`)
- Set `CHAINMAKER_NODE_HOST` and optional port variables
- Template at `config/clients/chainmaker.yml.template`

**Output:** `config/clients/chainmaker.yml`

## Database Scripts

### `db/init-db.sql`
PostgreSQL initialization script for log status tracking.

**Creates:**
- `tbl_log_status` table with indexes
- Optimized indexes for query performance

**Usage:** Automatically mounted and executed by PostgreSQL container on first startup via docker-compose.

### `db/test-data.sql`
Sample test data for development and testing.