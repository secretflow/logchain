# Benthos Adapters Test Utilities

This directory contains testing utilities and mock services for validating Benthos adapter configurations before deploying to production environments.

## Contents

- **`mock_ingestion_server.py`** - A lightweight mock log ingestion service that simulates the backend ingestion API
- **`mock_minio/`** - Docker Compose setup for a local MinIO (S3-compatible) storage instance

## Mock Ingestion Server

### Overview

The `mock_ingestion_server.py` is a Python Flask application that mimics the behavior of the Log Ingestion Service. It receives POST requests at `/v1/logs`, validates API keys, and writes ingested logs to a JSONL file for inspection.

### Features

- **JSON Payload Handling**: Accepts both single log objects and batched arrays
- **File Output**: Writes received logs to a JSONL file with timestamps
- **Error Handling**: Returns appropriate HTTP status codes for validation failures

### Prerequisites

```bash
pip install flask werkzeug
```

### Usage

#### Basic Usage

```bash
# Run with default settings (listens on 0.0.0.0:8093)
python3 mock_ingestion_server.py
```

#### Custom Configuration

```bash
# Set custom host, port, and output file
export INGESTION_HOST=127.0.0.1
export INGESTION_PORT=8093
export LOG_OUTPUT_FILE=./test_logs.jsonl

python3 mock_ingestion_server.py
```

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `INGESTION_HOST` | `0.0.0.0` | Host address to bind the server |
| `INGESTION_PORT` | `8093` | Port number to listen on |
| `LOG_OUTPUT_FILE` | `ingested_logs.jsonl` | Path to output file for received logs |

### Testing Benthos Adapters

1. **Start the mock server**:
   ```bash
   export INGESTION_PORT=8093
   export LOG_OUTPUT_FILE=./test_output.jsonl
   python3 mock_ingestion_server.py
   ```

2. **Configure Benthos adapter** to point to the mock server:
   ```bash
   export SYSLOG_UDP_ADDR=127.0.0.1:5514
   export SYSLOG_TCP_ADDR=127.0.0.1:6514
   export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
   export DEFAULT_ORG_ID=test-org
   
   # Run your Benthos adapter (e.g., syslog, kafka, s3)
   redpanda-connect run ingestion/adapters/syslog.yml
   ```

3. **Verify results**:
   ```bash
   # Check the output file
   cat test_output.jsonl
   # or
   tail -f test_output.jsonl
   ```

### Output Format

The mock server writes logs in JSONL format (one JSON object per line):

```json
{"received_at": "2025-12-22T10:30:45.123456", "data": {"log_content": "...", "client_source_org_id": "...", "client_timestamp": "..."}}
```

For batched requests, each item in the batch is written as a separate line with the same `received_at` timestamp.

## Mock MinIO (S3-Compatible Storage)

### Overview

The `mock_minio/` directory contains a Docker Compose configuration for running a local MinIO instance, which provides S3-compatible object storage for testing the S3 adapter.

### Features

- **S3-Compatible API**: Full compatibility with AWS S3 API
- **Automatic Bucket Creation**: Creates a default bucket (`my-bucket`) on startup
- **Web Console**: MinIO web UI available on port 9001 (if exposed)
- **Persistent Storage**: Data stored in `./minio-data` directory

### Usage

#### Start MinIO

```bash
cd mock_minio
docker-compose up -d
```

#### Access MinIO

- **API Endpoint**: `http://localhost:9000` (if ports are exposed)
- **Web Console**: `http://localhost:9001` (if ports are exposed)
- **Default Credentials**:
  - Access Key: `minio`
  - Secret Key: `redpandaTieredStorage7`

#### Configure Ports (Optional)

If you need to access MinIO from outside Docker, add port mappings to `docker-compose.yml`:

```yaml
services:
  minio:
    ports:
      - "9000:9000"   # API endpoint
      - "9001:9001"   # Web console
```

#### Testing S3 Adapter

1. **Start MinIO**:
   ```bash
   cd mock_minio
   docker-compose up -d
   ```

2. **Upload test files to MinIO**:
   ```bash
   # Using MinIO client (mc)
   docker exec -it minio-client mc cp /path/to/test.log myminio/my-bucket/
   
   # Or use AWS CLI (if configured)
   aws --endpoint-url http://localhost:9000 s3 cp test.log s3://my-bucket/
   ```

3. **Configure and run S3 adapter**:
   ```bash
   export S3_BUCKET_NAME=my-bucket
   export AWS_REGION=local
   export COMPATIBLE_END_POINT=http://localhost:9000
   export AWS_ACCESS_KEY_ID=minio
   export AWS_SECRET_ACCESS_KEY=redpandaTieredStorage7
   export DEFAULT_ORG_ID=dkbmtb
   export INGESTION_ENDPOINT=http://127.0.0.1:8093/v1/logs
   
   redpanda-connect run ../s3-processor.yml
   ```

4. **Stop MinIO**:
   ```bash
   docker-compose down
   ```

### Cleanup

To remove MinIO data:

```bash
docker-compose down -v
rm -rf minio-data
```

## Troubleshooting

### Mock Server Issues

- **Port already in use**: Change `INGESTION_PORT` to an available port
- **Permission denied**: Ensure the output directory is writable
- **Module not found**: Install Flask: `pip install flask werkzeug`

### MinIO Issues

- **Container won't start**: Check Docker logs: `docker-compose logs minio`
- **Can't connect from host**: Add port mappings to `docker-compose.yml`
- **Bucket not created**: Check `minio-client` container logs: `docker-compose logs mc`
