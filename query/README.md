# Query Service

Multi-dimensional query APIs for different user types.

## Structure

```
query/
├── service/          # Query service implementation
│   ├── core/        # Business logic
│   └── http/        # HTTP handlers
└── auth/            # Authentication middleware
    └── middleware.go
```

## APIs

### API 1: Status Query by Request ID
- **Endpoint:** `GET /v1/query/status/{request_id}`
- **Auth:** API Key
- **Purpose:** Check attestation status using `request_id` from submission
- **Data Source:** Database (fast)

### API 2: Query by Log Content
- **Endpoint:** `POST /v1/query_by_content`
- **Auth:** API Key
- **Purpose:** Find credentials using original log content (for Syslog/Kafka users)
- **Data Source:** Database (computes hash, then queries)

### API 3: Blockchain Audit
- **Endpoint:** `GET /v1/audit/log/{log_hash}`
- **Auth:** mTLS + IP Whitelist
- **Purpose:** Verify log data from blockchain (consortium members)
- **Data Source:** Blockchain (authoritative)

## Architecture

### Query Flow

```
HTTP Request → Auth Middleware → Handler → Core Service
                                               ↓
                                    Database or Blockchain
```

### Authentication

**API Key (API 1 & 2):**
- Middleware: `auth.RequireAPIKey()`
- Headers: `X-Auth-Method`, `X-API-Client-ID`, `X-Client-Org-ID`
- Scope: Can only query own organization's logs

**mTLS (API 3):**
- Middleware: `auth.RequireMTLS()`
- Headers: `X-Auth-Method`, `X-Member-ID`
- Scope: Can audit all on-chain data

### Response Structure

**Database Query (API 1 & 2):**
```json
{
  "source": "database",
  "request_id": "uuid",
  "log_hash": "sha256",
  "source_org_id": "org-id",
  "status": "COMPLETED",
  "tx_hash": "blockchain-tx-hash",
  "block_height": 12345
}
```

**Blockchain Audit (API 3):**
```json
{
  "source": "blockchain",
  "log_hash": "sha256",
  "log_content": "original log",
  "sender_org_id": "org-id",
  "timestamp": "2025-12-23T10:00:00Z"
}
```

## Key Features

- **Multi-source Queries**: Database for speed, blockchain for verification
- **Role-based Access**: Different APIs for different user types
- **Org Isolation**: API Key users can only see their own logs
- **Audit Trail**: All queries logged for compliance
- **Error Handling**: Clear error messages for debugging

## Status Values

- `RECEIVED` - Log accepted, pending processing
- `PROCESSING` - Being submitted to blockchain
- `COMPLETED` - Successfully on blockchain
- `FAILED` - Processing failed (with error details)

## Development

See [`cmd/query/README.md`](../cmd/query/README.md) for running the service locally.