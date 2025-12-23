# Trusted Log Attestation System

A blockchain-based log attestation system providing transparent, immutable, and multi-dimensional verifiable log storage.

## Architecture

The system follows a layered microservices architecture:

```
┌─────────────────────────────────────────────────────────────────┐
│                     External Clients                             │
└───────────────┬─────────────────────────────┬───────────────────┘
                │                             │
                ▼                             ▼
┌───────────────────────┐     ┌───────────────────────────────────┐
│   Nginx API Gateway   │     │    Benthos Adapters               │
│   (mTLS, API Key)     │     │    (Syslog, Kafka, S3)            │
└───────────┬───────────┘     └───────────────┬───────────────────┘
            │                                 │
            ▼                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Log Ingestion Service                          │
│                   (HTTP/gRPC, SHA256, Kafka)                     │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Blockchain Engine Service                      │
│                   (Kafka Consumer, ChainMaker)                   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
            ┌───────────────────┴───────────────────┐
            ▼                                       ▼
┌───────────────────────┐             ┌─────────────────────────┐
│   PostgreSQL          │             │   ChainMaker            │
│   (Task Status)       │             │   (On-Chain Storage)    │
└───────────────────────┘             └─────────────────────────┘
```

**For detailed architecture**, see [docs/design.md](docs/design.md).

## Quick Start

### Prerequisites

1. Docker and Docker Compose
2. ChainMaker deployment (see [docs/chainmaker_deployment.md](docs/chainmaker_deployment.md))

### Setup

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env with your ChainMaker path and node addresses

# 2. Generate ChainMaker client config
bash scripts/generate-chainmaker-config.sh

# 3. Setup Nginx authentication
cd ingress
bash scripts/generate-ssl-certs.sh                              # Server certs + CA
bash scripts/setup-config.sh                                    # API keys + IP whitelist
bash scripts/generate-client-cert.sh member-001 "Member One"    # Client cert for mTLS (optional)
cd ..

# 4. Start all services
docker compose up -d
```

### Verify

```bash
# Check service health
curl http://localhost/health

# Test log submission via API Gateway (HTTPS)
curl -k -X POST https://localhost/v1/logs \
  -H "X-API-Key: example-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "test log message"}'

# Test log submission via Benthos Syslog adapter (UDP)
echo "<14>1 $(date -u +%Y-%m-%dT%H:%M:%SZ) localhost test - - - Test syslog message" | nc -u localhost 5514
```

## Directory Structure

```
├── cmd/                    # Service entry points
│   ├── ingestion/         # Log Ingestion Service
│   ├── engine/            # Blockchain Processing Service
│   └── query/             # Query Service
├── ingestion/             # Ingestion layer (service + Benthos adapters)
├── ingress/               # API Gateway (Nginx + OpenResty)
├── processing/            # Batch processing worker
├── query/                 # Query service implementation
├── blockchain/            # Blockchain client abstraction
├── storage/               # Database store interface
├── config/                # Configuration files
├── scripts/               # Utility and test scripts
└── docs/                  # Design documents
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| **Nginx Gateway** | 80, 443, 50052 | API Gateway with TLS, API Key, mTLS authentication |
| **Ingestion** | 8091, 50051 | Log submission (HTTP/gRPC) |
| **Engine** | - | Kafka consumer, blockchain attestation |
| **Query** | 8083 | Status and audit queries |
| **Benthos** | 5514, 6514 | Syslog adapter (UDP/TCP) |
| **Kafka** | 9092 | Internal message queue |
| **PostgreSQL** | 5432 | State database |

## API Endpoints

### Log Submission (API Key Authentication)
- `POST /v1/logs` - Submit log for attestation

### Query (API Key Authentication)
- `GET /v1/query/status/{request_id}` - Query attestation status
- `POST /v1/query_by_content` - Query by log content

### Audit (mTLS + IP Whitelist)
- `GET /v1/audit/log/{log_hash}` - On-chain audit for consortium members

## Configuration

| File | Purpose |
|------|---------|
| `.env` | Environment variables (ChainMaker path, node addresses) |
| `config/*.defaults.yml` | Service configurations |
| `ingress/nginx/conf.d/api-keys.json` | API Key definitions |
| `ingress/nginx/conf.d/consortium-ip-whitelist.json` | mTLS IP whitelist |

**For configuration details**, see [config/README.md](config/README.md).

## Testing

```bash
# Test API endpoints
bash scripts/test-ingestion-query-api.sh

# Test consortium audit API (requires mTLS cert)
bash scripts/test-consortium-audit-api.sh <log_hash>
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/design.md](docs/design.md) | System architecture and design |
| [docs/chainmaker_deployment.md](docs/chainmaker_deployment.md) | Blockchain setup guide |
| [config/README.md](config/README.md) | Configuration guide |
| [ingress/README.md](ingress/README.md) | API Gateway documentation |
| [ingestion/README.md](ingestion/README.md) | Ingestion layer overview |

## License

See [LICENSE](LICENSE) and [LEGAL.md](LEGAL.md).