# Query Service

HTTP API for log status queries and blockchain audits.

## Architecture

```
HTTP Request → Auth Middleware → Core Service → Database/Blockchain
```

Core logic: [`query/service/`](../../query/service/)

## APIs

1. `GET /v1/query/status/{request_id}` - Status by request ID (API Key)
2. `POST /v1/query_by_content` - Query by log content (API Key)
3. `GET /v1/audit/log/{log_hash}` - Audit from blockchain (mTLS)

## Development Setup

### 1. Start Dependencies

```bash
# Start Postgres and ChainMaker
docker compose up -d postgres
```

### 2. Run Query Service Locally

```bash
cd cmd/query
go run main.go
```

### 3. Test APIs

**API 1 - Status by Request ID:**
```bash
curl "http://localhost:8083/v1/query/status/{request_id}" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -H "X-Client-Org-ID: test-org"
```

**API 2 - Query by Content:**
```bash
curl -X POST "http://localhost:8083/v1/query_by_content" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Method: api-key" \
  -H "X-API-Client-ID: client-001" \
  -d '{"log_content": "test log"}'
```

**API 3 - Blockchain Audit:**
```bash
curl "http://localhost:8083/v1/audit/log/{log_hash}" \
  -H "X-Auth-Method: mtls" \
  -H "X-Member-ID: member-001"
```

### 4. Verify in Database

```bash
docker compose exec postgres psql -U testuser -d testdb -c \
"SELECT request_id, log_hash, status, tx_hash FROM tbl_log_status LIMIT 5;"
```

## Authentication Headers

**For local testing without Nginx, include these headers manually:**

API 1 & 2 (API Key):
- `X-Auth-Method: api-key`
- `X-API-Client-ID: client-001`
- `X-Client-Org-ID: test-org`

API 3 (mTLS):
- `X-Auth-Method: mtls`
- `X-Member-ID: member-001`

**Note:** In production, Nginx gateway sets these headers automatically.

## Configuration

`config/query.defaults.yml`:
- HTTP server port
- Database connection
- Blockchain client config path

## Notes

- API 1 & 2 query from database (fast)
- API 3 queries from blockchain (authoritative)
- Log hash: SHA-256 of log content