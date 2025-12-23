# Engine Service

Background worker that consumes logs from Kafka and submits them to blockchain in batches.

## Architecture

```
Kafka (log_submissions) → Engine Worker → Blockchain → Database (status update)
```

Core logic: [`processing/worker.go`](../../processing/worker.go)

## Development Setup

### 1. Start Dependencies

```bash
# Start Kafka, Postgres, ChainMaker
docker compose up -d kafka postgres
```

### 2. Run Engine Locally

```bash
cd cmd/engine
go run main.go
```

### 3. Submit Test Logs

Via Ingestion Service:
```bash
curl -X POST http://localhost:8091/v1/logs \
  -H "Content-Type: application/json" \
  -d '{"log_content": "test log", "client_source_org_id": "test-org"}'
```

Or via Kafka directly:
```bash
docker compose exec kafka kafka-console-producer \
  --bootstrap-server kafka:29092 \
  --topic log_submissions
# Paste JSON message and press Enter
```

## Monitoring

### Check Processing Status

```bash
docker compose exec postgres psql -U testuser -d testdb -c \
"SELECT status, COUNT(*) FROM tbl_log_status GROUP BY status;"
```

### View Engine Logs

```bash
docker compose logs -f engine
```

## Configuration

`config/engine.defaults.yml`:
- Kafka consumer settings
- Worker batch size and timeout
- Database connection pool
- Blockchain client config path

## Notes

- Processes logs in batches for efficiency
- Auto-retry failed submissions
- Idempotent using log hash as deduplication key