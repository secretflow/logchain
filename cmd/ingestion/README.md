# Ingestion Service

HTTP and gRPC service for log submission.

## Architecture

```
HTTP/gRPC API → Core Service → Kafka + Database
```

Core logic: [`ingestion/service/`](../../ingestion/service/)

## Development Setup

### 1. Start Dependencies

```bash
# Start Kafka and Postgres
docker compose up -d kafka postgres
```

### 2. Run Ingestion Locally

```bash
cd cmd/ingestion
go run main.go
```

### 3. Test Submission

**HTTP:**
```bash
curl -X POST http://localhost:8091/v1/logs \
  -H "Content-Type: application/json" \
  -d '{"log_content": "test log", "client_source_org_id": "test-org"}'
```

**gRPC:**
```bash
grpcurl -plaintext -d '{"log_content": "test log", "client_source_org_id": "test-org"}' \
  localhost:50051 logingestion.LogIngestion/SubmitLog
```

### 4. Verify

**Database:**
```bash
docker compose exec postgres psql -U testuser -d testdb -c \
"SELECT request_id, log_hash, status FROM tbl_log_status ORDER BY received_timestamp DESC LIMIT 5;"
```

**Kafka:**
```bash
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server kafka:29092 --topic log_submissions --from-beginning --max-messages 5
```

## Configuration

`config/ingestion.defaults.yml`:
- HTTP/gRPC server ports
- Kafka producer settings
- Database connection
- Batch processing parameters

## Notes

- Returns `request_id` immediately (async processing)
- Log hash computed as SHA-256
- Messages batched to Kafka for efficiency