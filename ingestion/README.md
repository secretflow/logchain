# Ingestion Layer

Log ingestion and protocol adaptation components.

## Structure

```
ingestion/
├── service/          # Log Ingestion Service (HTTP/gRPC)
│   ├── core/        # Business logic & batch processing
│   ├── http/        # HTTP REST handlers
│   └── grpc/        # gRPC service implementation
└── adapters/        # Benthos protocol adapters (Syslog/Kafka/S3)
```

## Components

### 1. Log Ingestion Service ([`service/`](service/))

Unified entry point for log submissions:
- **HTTP API**: `POST /v1/logs`
- **gRPC API**: `LogIngestion.SubmitLog`
- **Processing**: Hash generation, batch write to DB, Kafka publishing

### 2. Benthos Adapters ([`adapters/`](adapters/))

Protocol conversion for heterogeneous sources:
- **Syslog**: UDP/TCP 514 → HTTP
- **Kafka**: Topic consumption → HTTP
- **S3**: File processing → HTTP

## Data Flow

### Direct Submission
```
Client → Nginx → Ingestion Service → Kafka → Engine
                       ↓
                   Database
```

### Adapter Flow
```
Syslog/Kafka/S3 → Benthos → Ingestion Service → Kafka → Engine
                                   ↓
                               Database
```

## Key Features

- **Async Processing**: Immediate response with `request_id`
- **Batch Optimization**: Batched DB writes and Kafka publishes
- **Hash Deduplication**: SHA-256 log hash as unique key
- **Protocol Agnostic**: HTTP, gRPC, Syslog, Kafka, S3 supported

## Configuration

- Service: `config/ingestion.defaults.yml`
- Adapters: `docker-compose.yml` environment variables

## Development

See:
- [`cmd/ingestion/README.md`](../cmd/ingestion/README.md) - Running service locally
- [`service/README.md`](service/README.md) - Service architecture
- [`adapters/README.md`](adapters/README.md) - Adapter configuration