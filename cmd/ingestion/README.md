# Log Ingestion Service

The Log Ingestion Service provides HTTP REST and gRPC endpoints for log submission to the Trusted Log Attestation System.

## Quick Start

Start all services using Docker Compose:

```bash
docker compose up -d
```

This starts:
- **ingestion-service-tlng**: HTTP (port 8091), gRPC (port 50051)
- **postgres-tlng**: PostgreSQL database (port 5433)
- **kafka-tlng**: Message queue (port 9093)
- **zookeeper-tlng**: Kafka coordination

## Usage Examples

### Submit Log via HTTP

```bash
curl -X POST http://localhost:8091/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "log_content": "This is a test log from curl",
    "client_source_org_id": "test-org"
  }'
```

**Response:**
```json
{
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "RECEIVED"
}
```

### Submit Log via gRPC

```bash
grpcurl -plaintext -d '{
  "log_content": "Test log via gRPC",
  "client_source_org_id": "grpc-test-org"
}' localhost:50051 logingestion.LogIngestion/SubmitLog
```

## Verify Submission

### Check Database

```bash
# View recent logs
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, log_hash, source_org_id, status, received_timestamp
FROM tbl_log_status
ORDER BY received_at_db DESC
LIMIT 5;
"
```

### Check Kafka Messages

```bash
# Using kcat
kcat -C -b localhost:9093 -t log_submissions -o beginning -e

# Or using Docker
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server kafka:29092 \
  --topic log_submissions \
  --from-beginning \
  --max-messages 5
```

## Troubleshooting

### Service Not Responding

```bash
# Check container status
docker compose ps

# View ingestion service logs
docker compose logs -f ingestion

# Restart ingestion service
docker compose restart ingestion
```

### Port Conflicts

If ports 8091 or 50051 are already in use, modify `docker-compose.yml`:

```yaml
services:
  ingestion:
    ports:
      - "8092:8091"  # Change external port
      - "50052:50051"
```

### Database Connection Issues

```bash
# Verify PostgreSQL is healthy
docker compose exec postgres psql -U testuser -d testdb -c "SELECT 1;"

# Check database logs
docker compose logs postgres
```

### Kafka Connection Issues

```bash
# List topics
docker compose exec kafka kafka-topics --list --bootstrap-server kafka:29092

# Recreate topic if needed
docker compose exec kafka kafka-topics --delete --bootstrap-server kafka:29092 --topic log_submissions
docker compose restart kafka-init
```

## Clean Up Test Data

```bash
docker compose exec postgres psql -U testuser -d testdb -c "
DELETE FROM tbl_log_status WHERE source_org_id LIKE '%-test-%';
"
```

## Notes

- Request IDs are auto-generated UUIDs
- Log hashes are computed as SHA-256 of content
- All timestamps use RFC3339 format with timezone
- Database schema is auto-initialized on first startup