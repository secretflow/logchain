# Query Service

The Query Service provides three APIs for querying log status and performing blockchain audits.

## Quick Start

Start all services using Docker Compose:

```bash
docker compose up -d
```

This starts:
- **query-service-tlng**: HTTP API (port 8083)
- **postgres-tlng**: Database for log status
- **ChainMaker**: Blockchain for audit queries (via mounted certificates)

## API Overview

### API 1: Query Status by Request ID
**Endpoint:** `GET /v1/status/{request_id}`

Returns the current processing status of a log submission.

### API 2: Query by Content
**Endpoint:** `POST /v1/query`

Computes log hash from content and returns status if found.

### API 3: Audit Log from Blockchain
**Endpoint:** `GET /v1/audit/log/{log_hash}`

Retrieves log data directly from blockchain for verification.

## Usage Examples

### API 1: Query Status by Request ID

```bash
# Get status by request_id (returned from ingestion)
curl -X GET "http://localhost:8083/v1/status/a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -H "X-Client-Org-ID: test-org"
```

**Response (COMPLETED):**
```json
{
  "source": "database",
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "log_hash": "93d9aa176a7a608df6534572c44cc39dcb07b55d189450b9ff74c353669c8e59",
  "source_org_id": "test-org",
  "status": "COMPLETED",
  "received_timestamp": "2025-12-18T19:01:56.496326175+08:00",
  "processing_started_at": "2025-12-18T19:02:01.123456789+08:00",
  "processing_finished_at": "2025-12-18T19:02:03.987654321+08:00",
  "tx_hash": "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890",
  "block_height": 12345
}
```

### API 2: Query by Content

```bash
# Query using log content (must match exact content submitted)
curl -X POST "http://localhost:8083/v1/query" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -H "X-Client-Org-ID: test-org" \
  -d '{
    "log_content": "This is a test log from curl"
  }'
```

**Response:**
```json
{
  "source": "database",
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "log_hash": "93d9aa176a7a608df6534572c44cc39dcb07b55d189450b9ff74c353669c8e59",
  "source_org_id": "test-org",
  "status": "COMPLETED",
  "received_timestamp": "2025-12-18T19:01:56.496326175+08:00",
  "processing_finished_at": "2025-12-18T19:02:03.987654321+08:00",
  "tx_hash": "a1b2c3d4e5f67890...",
  "block_height": 12345
}
```

### API 3: Audit Log from Blockchain

```bash
# Audit using log_hash (from API 1 or API 2)
curl -X GET "http://localhost:8083/v1/audit/log/93d9aa176a7a608df6534572c44cc39dcb07b55d189450b9ff74c353669c8e59" \
  -H "X-Cert-Subject: CN=member1,O=consortium" \
  -H "X-Member-ID: member-001" \
  -H "X-Auth-Method: mtls"
```

**Response:**
```json
{
  "source": "blockchain",
  "log_hash": "93d9aa176a7a608df6534572c44cc39dcb07b55d189450b9ff74c353669c8e59",
  "log_content": "This is a test log from curl",
  "sender_org_id": "test-org",
  "timestamp": "2025-12-18T19:01:56.496326175+08:00"
}
```

## Complete Workflow Example

```bash
# 1. Submit a log
REQUEST_ID=$(curl -s -X POST http://localhost:8091/v1/logs \
  -H "Content-Type: application/json" \
  -d '{"log_content": "My test log", "client_source_org_id": "test"}' \
  | jq -r '.request_id')

echo "Request ID: $REQUEST_ID"

# 2. Wait a few seconds for processing
sleep 5

# 3. Query status by request_id (API Key auth)
curl -X GET "http://localhost:8083/v1/status/$REQUEST_ID" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -H "X-Client-Org-ID: test-org"

# 4. Query by content (API Key auth)
curl -X POST "http://localhost:8083/v1/query" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -H "X-Client-Org-ID: test-org" \
  -d '{"log_content": "My test log"}'

# 5. Get log_hash from response and audit from blockchain (mTLS auth)
LOG_HASH="93d9aa176a7a608df6534572c44cc39dcb07b55d189450b9ff74c353669c8e59"
curl -X GET "http://localhost:8083/v1/audit/log/$LOG_HASH" \
  -H "X-Auth-Method: mtls" \
  -H "X-Cert-Subject: CN=member1,O=consortium" \
  -H "X-Member-ID: member-001"
```

## Authentication Headers

Query APIs use different authentication methods:

**API 1 & 2 (API Key Authentication):**
- `X-Auth-Method: api-key`
- `X-API-Client-ID`: Client identifier (e.g., `client-001`)
- `X-Client-Org-ID`: Organization identifier (e.g., `test-org`)

**API 3 (mTLS Authentication):**
- `X-Auth-Method: mtls`
- `X-Cert-Subject`: Client certificate subject (e.g., `CN=member1,O=consortium`)
- `X-Member-ID`: Member identifier (e.g., `member-001`)

Headers are normally set by Nginx gateway. For direct testing without Nginx, include these headers manually.

## Troubleshooting

### Service Not Responding

```bash
# Check query service status
docker compose ps query

# View query service logs
docker compose logs -f query

# Restart query service
docker compose restart query
```

### Database Connection Issues

```bash
# Test database connectivity
docker compose exec postgres psql -U testuser -d testdb -c "SELECT COUNT(*) FROM tbl_log_status;"

# Check query service database config
docker compose exec query cat /app/config/query.defaults.yml
```

### Blockchain Connection Failures

```bash
# Check ChainMaker certificate mount
docker compose exec query ls -la /app/chainmaker-go/build/crypto-config/

# Verify blockchain config
docker compose exec query cat /app/config/blockchain.defaults.yml
docker compose exec query cat /app/config/clients/chainmaker.yml

# Check connectivity to ChainMaker nodes
docker compose exec query ping -c 3 30.177.108.73
```

### API Returns 404 Not Found

```bash
# For API 1 & 2: Check if log exists in database
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, log_hash, status FROM tbl_log_status WHERE request_id = 'your-request-id';
"

# For API 3: Check if log submitted to blockchain
# Verify status is COMPLETED
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, status, tx_hash, block_height FROM tbl_log_status WHERE log_hash = 'your-log-hash';
"
```

### Invalid Authentication Headers

If testing directly (without Nginx), ensure correct headers for each API:

```bash
# API 1 & 2: API Key authentication
-H "X-Auth-Method: api-key"
-H "X-API-Client-ID: client-001"
-H "X-Client-Org-ID: test-org"

# API 3: mTLS authentication
-H "X-Auth-Method: mtls"
-H "X-Cert-Subject: CN=member1,O=consortium"
-H "X-Member-ID: member-001"
```

## Configuration

Query service configuration is in `config/query.defaults.yml`:

- **HTTP Port**: Service listening port (default: 8083)
- **Database**: PostgreSQL connection settings
- **Blockchain**: ChainMaker client configuration

## Notes

- API 1 & 2 query from PostgreSQL database (fast)
- API 3 queries from blockchain (authoritative, slower)
- Use API 3 for audit/verification purposes
- Log hash is SHA-256 of log content
- All timestamps use RFC3339 format with timezone
