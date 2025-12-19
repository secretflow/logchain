# Attestation Engine Service

The Attestation Engine consumes log submissions from Kafka and submits them to the blockchain for attestation. It processes logs in batches with automatic retry handling.

## Quick Start

Start all services using Docker Compose:

```bash
docker compose up -d
```

This starts:
- **engine-service-tlng**: Background worker processing Kafka messages
- **kafka-tlng**: Message queue (port 9093)
- **postgres-tlng**: Database for state tracking

The engine automatically starts consuming from the `log_submissions` topic.

## Usage

### 1. Submit Logs via Ingestion Service

```bash
# Submit a test log
curl -X POST http://localhost:8091/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "log_content": "Test log for blockchain attestation",
    "client_source_org_id": "test-org"
  }'
```

### 2. Monitor Processing Status

```bash
# Check status transitions (RECEIVED → PROCESSING → COMPLETED)
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, status, processing_started_at, processing_finished_at, tx_hash, block_height
FROM tbl_log_status
ORDER BY received_at_db DESC
LIMIT 10;
"
```

### 3. View Engine Logs

```bash
# Real-time logs
docker compose logs -f engine

# Recent logs
docker compose logs --tail=50 engine
```

## Monitoring Examples

### Check Processing Statistics

```bash
# Count logs by status
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT status, COUNT(*) as count
FROM tbl_log_status
GROUP BY status;
"
```

### View Failed Logs

```bash
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, error_message, retry_count
FROM tbl_log_status
WHERE status = 'FAILED'
ORDER BY received_at_db DESC
LIMIT 10;
"
```

### Check Blockchain Submissions

```bash
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT request_id, tx_hash, block_height, processing_finished_at
FROM tbl_log_status
WHERE status = 'COMPLETED'
ORDER BY processing_finished_at DESC
LIMIT 10;
"
```

## Troubleshooting

### Engine Not Processing Messages

```bash
# Check engine container status
docker compose ps engine

# View engine logs for errors
docker compose logs engine | grep -i error

# Restart engine
docker compose restart engine
```

### Kafka Connection Issues

```bash
# Check Kafka consumer group lag
docker compose exec kafka kafka-consumer-groups \
  --bootstrap-server kafka:29092 \
  --describe --group engine-consumer-group

# Verify topic exists and has messages
docker compose exec kafka kafka-topics --list --bootstrap-server kafka:29092
```

### Blockchain Connection Failures

```bash
# Check engine logs for blockchain errors
docker compose logs engine | grep -i "blockchain\|chainmaker"

# Verify ChainMaker certificate mount
docker compose exec engine ls -la /app/chainmaker-go/build/crypto-config/

# Check blockchain config
docker compose exec engine cat /app/config/blockchain.defaults.yml
```

### Database Connection Issues

```bash
# Test database connectivity
docker compose exec postgres psql -U testuser -d testdb -c "SELECT NOW();"

# Check database pool connections
docker compose exec postgres psql -U testuser -d testdb -c "
SELECT count(*) FROM pg_stat_activity WHERE datname = 'testdb';
"
```

## Configuration

Engine configuration is in `config/engine.defaults.yml`:

- **Kafka**: Bootstrap servers, topic, consumer group
- **Database**: Connection pool settings
- **Workers**: Concurrent processing count
- **Blockchain**: ChainMaker connection and contract settings
- **Retry**: Max attempts and backoff intervals

## Notes

- Engine processes logs in batches for better blockchain performance
- Failed logs are automatically retried with exponential backoff
- Max retry count is configurable (default: 3)
- All blockchain submissions are idempotent using log hash as key