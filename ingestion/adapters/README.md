# Benthos Adapters

Protocol conversion adapters for heterogeneous data sources.

## Purpose

Convert non-HTTP/gRPC protocols into standardized HTTP requests to Ingestion Service:
- **Syslog** - UDP/TCP syslog → HTTP
- **Kafka** - Topic consumption → HTTP  
- **S3** - File processing → HTTP

## Available Adapters

### 1. Syslog Adapter (`syslog.yml`)

**Listens on:**
- UDP: 5514
- TCP: 6514

**Format:** RFC5424 syslog messages

**Output:** HTTP POST to ingestion service

### 2. Kafka Adapter (`kafka-consumer.yml`)

**Consumes from:** Kafka topics

**Features:**
- TLS support
- Consumer group management
- Offset tracking

### 3. S3 Adapter (`s3-processor.yml`)

**Processes:** Line-delimited log files from S3 buckets

**Features:**
- Automatic file discovery
- Line-by-line processing

## Docker Deployment

Adapters run as separate containers in `docker-compose.yml`:

```yaml
benthos-syslog:
  image: redpandadata/connect:latest
  ports:
    - "5514:5514/udp"
    - "6514:6514/tcp"
  environment:
    INGESTION_ENDPOINT: "http://ingestion:8091/v1/logs"
    DEFAULT_ORG_ID: "org-abc"
```

## Configuration

### Common Environment Variables

- `INGESTION_ENDPOINT` - Target service URL (default: `http://ingestion:8091/v1/logs`)
- `DEFAULT_ORG_ID` - Default organization ID
- `HTTP_BATCH_COUNT` - Batch size for HTTP requests (default: 200)
- `HTTP_BATCH_PERIOD` - Max batch wait time (default: 1s)

### Rate Limiting (Optional)

- `RATE_LIMIT_ENABLED` - Enable rate limiting (default: false)
- `RATE_LIMIT_COUNT` - Max requests per period (default: 500)
- `RATE_LIMIT_PERIOD` - Rate limit window (default: 1s)

## Testing

### Test Syslog Adapter

**UDP:**
```bash
echo '<14>1 2025-12-23T10:00:00Z myhost app 1234 - - Test UDP' | nc -u -w1 localhost 5514
```

**TCP:**
```bash
echo '<14>1 2025-12-23T10:00:00Z myhost app 1234 - - Test TCP' | nc localhost 6514
```

### Test S3 Adapter

Example using MinIO:
```bash
# Put test file into S3 bucket
docker exec -it minio sh
echo "test log message" > /tmp/test.txt
mc cp /tmp/test.txt myminio/my-container-bucket
```

### Monitor Logs

```bash
docker compose logs -f benthos-syslog
```

## Advanced Configuration

### Kafka Adapter Environment Variables

```bash
export KAFKA_BROKERS=localhost:9093
export KAFKA_TOPIC=log-topic
export KAFKA_CONSUMER_GROUP=benthos-consumer
export KAFKA_TLS_ENABLED=true
export KAFKA_CA_FILE=/path/to/ca-cert.pem
export KAFKA_CLIENT_CERT_FILE=/path/to/client.crt
export KAFKA_CLIENT_KEY_FILE=/path/to/client.key
export DEFAULT_ORG_ID=org-abc
export INGESTION_ENDPOINT=http://ingestion:8091/v1/logs
```

### S3 Adapter Environment Variables

```bash
export S3_BUCKET_NAME=my-log-bucket
export S3_PREFIX=logs/
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
# For MinIO or S3-compatible storage:
export COMPATIBLE_END_POINT=http://minio:9000
export FORCE_PATH_STYLE_URLS=true
export DEFAULT_ORG_ID=org-abc
export INGESTION_ENDPOINT=http://ingestion:8091/v1/logs
```

## Data Flow

```
External Source → Benthos Adapter → Parse/Transform → HTTP POST → Ingestion Service
                                                                         ↓
                                                                   Kafka + DB
```

## Notes

- All adapters forward to Ingestion Service HTTP endpoint
- Logs output to container stdout (viewable via `docker logs`)
- No local file storage required
- Benthos automatically handles retries and backpressure

## Security Best Practices

### General Principles

- Implement access control at the network infrastructure level (security groups, firewalls, network ACLs)
- Limit access to Benthos adapter services to authorized network traffic only
- Use Layer 4/7 proxy (Nginx/Ingress) for TLS termination and rate limiting in private networks
- Utilize Benthos built-in authentication mechanisms (client certificates, access keys)

### Syslog Adapter (TCP/UDP)

**Infrastructure Level (User Configuration Required):**
- Configure **IP whitelist + port control** in security groups/firewalls
- Allow only trusted IPs to access Syslog listening ports
- UDP protocol does not support TLS - network-level security is critical

**Benthos Built-in Protection:**
- ✅ **Rate Limiting**: When `RATE_LIMIT_ENABLED=true`, applies backpressure instead of dropping messages

### Kafka Adapter

**Infrastructure Level (User Configuration Required):**
- Secure Kafka broker nodes with proper ACLs
- Configure security groups/firewalls to allow only necessary access

**Benthos Built-in Protection:**
- ✅ **Client Certificate Authentication**: Supports TLS client certificates for broker authentication
- ✅ **Server Certificate Verification**: Optional verification of Kafka broker certificates
- ✅ **Rate Limiting**: When `RATE_LIMIT_ENABLED=true`, applies backpressure

### S3 Adapter

**Infrastructure Level (User Configuration Required):**
- Configure S3 bucket ACLs and IAM policies correctly
- Allow only necessary access sources

**Benthos Built-in Protection:**
- ✅ **Access Key Authentication**: Supports AWS Access Key, Secret Key, Session Token
- ✅ **Rate Limiting**: When `RATE_LIMIT_ENABLED=true`, applies backpressure