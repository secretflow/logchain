# Ingress Layer - API Gateway

## Overview

The Ingress Layer is the API Gateway for the log attestation system, built with **Nginx + OpenResty**. It handles TLS termination, authentication, routing, and load balancing for all incoming traffic.

## Core Features

### 1. TLS Termination
- Processes all HTTPS requests
- Decrypts traffic so internal services don't need to handle SSL/TLS certificates
- Ensures secure communication channels
- Supports HTTP/2

### 2. Authentication
- **API Key**: For log submission and query operations (file/Redis/external service)
- **mTLS + IP Whitelist**: Dual authentication for consortium audit access

### 3. Protocol Routing
- **Log Submission** (API Key):
  - `POST /v1/logs` → Ingestion Service (HTTP)
  - `gRPC SubmitLog` → Ingestion Service (gRPC on :50052)
- **Query** (API Key):
  - `GET /v1/query/status/{request_id}` → Query Service
  - `POST /v1/query_by_content` → Query Service
- **Audit** (mTLS + IP Whitelist):
  - `GET /v1/audit/log/{log_hash}` → Query Service
  - `GET /log/by_tx/{tx_hash}` → Query Service
  - `GET /log/{on_chain_log_id}` → Query Service

### 4. Load Balancing
- Least-connection algorithm for backend services
- Health checks and automatic failover
- Horizontal scaling support

### 5. Rate Limiting
- Log submission: 100 req/s (burst: 20)
- Query: 50 req/s (burst: 10)
- Audit: 20 req/s (burst: 5)

### 6. Audit Logging
- Authentication events (success/failure)
- Critical operations with context
- Security compliance audit trail

## Quick Start

**Note**: For production deployment, use the root `docker-compose.yml` which includes all services. This guide focuses on standalone ingress setup for development/testing.

### Prerequisites

- Docker and Docker Compose
- OpenSSL (for certificate generation)

### 1. Generate SSL Certificates

```bash
cd ingress
bash scripts/generate-ssl-certs.sh
```

This creates:
- **Server certificates**: `nginx/ssl/cert.pem`, `nginx/ssl/key.pem`, `nginx/ssl/ca-cert.pem`
- **CA files**: `scripts/ca/ca-cert.pem`, `scripts/ca/ca-key.pem` (for signing client certificates)

### 2. Setup Configuration Files

```bash
bash scripts/setup-config.sh
```

This will create configuration files from examples:
- `nginx/conf.d/api-keys.json`
- `nginx/conf.d/consortium-ip-whitelist.json`

**Important**: Update these files with your actual API keys and IP whitelists.

### 3. Configure API-Key Authentication

API keys are stored in `nginx/conf.d/api-keys.json`:

```json
{
  "your-api-key-here": {
    "client_id": "client-001",
    "org_id": "org-abc",
    "status": "active",
    "permissions": ["submit_log", "query_status", "query_by_content"],
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z"
  }
}
```

**Authentication Methods** (configured via `API_KEY_AUTH_METHOD` environment variable):
- `file`: File-based (default, for development)
- `redis`: Redis-based (for production)
- `service`: External auth service (for centralized management)

### 4. Configure mTLS + IP Whitelist Authentication

For consortium members, configure:

**IP Whitelist** (`nginx/conf.d/consortium-ip-whitelist.json`):
```json
{
  "members": {
    "member-001": {
      "name": "Regulatory Authority A",
      "ip_whitelist": [
        "192.168.1.0",
        "10.0.0.100"
      ]
    },
    "member-002": {
      "name": "Regulatory Authority B",
      "ip_whitelist": [
        "203.0.113.0",
        "198.51.100.50"
      ]
    }
  }
}
```

**Important** `Currently, the memberId (like member-001) and name fields can be configured freely. These two fields are reserved as redundancies for future expansion`

### 5. Generate Client Certificates

For consortium members to access audit endpoints:

```bash
bash scripts/generate-client-cert.sh member-001 "Regulatory Authority A"
```

This creates:
- Client certificate: `scripts/clients/member-001/client-cert.pem`
- Client private key: `scripts/clients/member-001/client-key.pem`

**Note**: Certificates are signed by the CA generated in step 1. For production, use your own CA by updating `ssl_client_certificate` in `nginx.conf`.

### 6. Configure Backend Services

**For testing API Gateway only**: You can skip this and use mock services.

**For integration with actual services**: Update upstream addresses in `nginx/nginx.conf`:

```nginx
upstream ingestion_http {
    least_conn;
    server ingestion-service-tlng:8091 max_fails=3 fail_timeout=30s;
    # Add more instances:
    # server ingestion-service-tlng-2:8091 max_fails=3 fail_timeout=30s;
    keepalive 32;
}
```

### 7. Start the API Gateway

```bash
cd ..
docker-compose up -d nginx
```

The API Gateway will be available at:
- HTTP: `http://localhost:80` (redirects to HTTPS)
- HTTPS: `https://localhost:443`
- gRPC: `localhost:50052`

**Note**: Log directory permissions are handled automatically by the Dockerfile.

### 8. Verify Health

```bash
curl http://localhost/health
# Expected: healthy
```

## API Endpoints

### Log Submission (API Key Authentication)

**HTTP:**
```bash
curl -k -XPOST https://localhost/v1/logs \
  -H "X-API-Key: example-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "your log content here"}'
```
```bash
curl -k -XPOST https://localhost/v1/logs \
  -H "X-API-Key: example-api-key-67890" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "your log content here"}'
```

**gRPC:**
```bash
grpcurl -proto ../proto/logingestion.proto -insecure \
  -H "X-API-Key: example-api-key-12345" \
  -d '{"log_content": "your log content here"}' \
  localhost:50052 \
  logingestion.LogIngestion/SubmitLog
```

### Query Operations (API Key Authentication)

**Status Query:**
```bash
curl -k -H "X-API-Key: example-api-key-12345" https://localhost/v1/query/status/{request_id}
```

**Content-based Query:**
```bash
curl -k -X POST https://localhost/v1/query_by_content \
  -H "X-API-Key: example-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "your log content here"}'
```

### Audit Operations (mTLS + IP Whitelist)

**Query by Transaction Hash:**
```bash
curl https://localhost/log/by_tx/{tx_hash} \
  --cert scripts/clients/member-001/client-cert.pem \
  --key scripts/clients/member-001/client-key.pem \
  --cacert scripts/ca/ca-cert.pem
```

**Query by Log Hash:**
```bash
curl https://localhost/v1/audit/log/{log_hash} \
  --cert scripts/clients/member-001/client-cert.pem \
  --key scripts/clients/member-001/client-key.pem \
  --cacert scripts/ca/ca-cert.pem
```
## Production Deployment

### 1. Replace Certificates

Use certificates from a trusted CA:

```bash
cp /path/to/your/cert.pem nginx/ssl/cert.pem
cp /path/to/your/key.pem nginx/ssl/key.pem
cp /path/to/your/ca-cert.pem nginx/ssl/ca-cert.pem
chmod 600 nginx/ssl/key.pem
chmod 644 nginx/ssl/cert.pem nginx/ssl/ca-cert.pem
```

### 2. Configure API Keys

Update `nginx/conf.d/api-keys.json` with production API keys. For centralized management, use Redis or external auth service.

### 3. Configure IP Whitelists

Update `nginx/conf.d/consortium-ip-whitelist.json` with actual consortium member IPs.

### 4. Update Upstream Services

In `nginx/nginx.conf`, configure actual backend service addresses for production.

### 5. Monitoring

Monitor:
- Access logs: `logs/access.log`
- Error logs: `logs/error.log`
- Audit logs: `logs/audit.log`

### 6. Environment Variables

Configure in `docker-compose.yml`:

```yaml
API_KEY_AUTH_METHOD: redis  # file | redis | service
REDIS_HOST: redis
REDIS_PORT: 6379
AUTH_SERVICE_URL: http://auth-service:8080/validate
```

## Troubleshooting

### Check Nginx Configuration

```bash
docker exec nginx-gateway-tlng nginx -t
```

### View Logs

```bash
# Access logs
docker exec nginx-gateway-tlng tail -f /var/log/nginx/access.log

# Error logs
docker exec nginx-gateway-tlng tail -f /var/log/nginx/error.log

# Audit logs
docker exec nginx-gateway-tlng tail -f /var/log/nginx/audit.log
```

### Reload Configuration

```bash
docker exec nginx-gateway-tlng openresty -c /etc/nginx/nginx.conf -s reload
```

## Integration

The API Gateway routes to:
- **Ingestion Service**: Log submission (HTTP :8091, gRPC :50051)
- **Query Service**: Status/content queries and audit operations (:8083)
- **Redis** (optional): API key storage
- **Auth Service** (optional): External authentication

When using the root `docker-compose.yml`, all services are on the same Docker network automatically.
