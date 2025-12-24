# Log Ingestion Service

Core service handling log submissions via HTTP and gRPC.

## Structure

```
service/
├── core/             # Business logic
│   ├── service.go   # Service orchestration
│   └── batch_processor.go  # Batch DB/Kafka operations
├── http/            # HTTP REST handlers
│   └── handler.go
└── grpc/            # gRPC implementation
    └── server.go
```

## Architecture

### Request Flow

```
HTTP/gRPC Request → Handler → Core Service → Batch Processor
                                    ↓
                            Immediate Response (request_id)
                                    ↓
                          [Async] DB + Kafka Publish
```

### Core Components

**1. Handler Layer** (`http/`, `grpc/`)
- API endpoint implementation
- Request validation
- Response formatting

**2. Core Service** (`core/service.go`)
- UUID generation
- SHA-256 hash computation
- Orchestration logic

**3. Batch Processor** (`core/batch_processor.go`)
- Accumulates logs in memory
- Batched database writes
- Batched Kafka publishes
- Periodic flush on timeout

## Batch Processing

### Strategy

- **Size trigger**: Flush when batch reaches configured size
- **Time trigger**: Flush after timeout even if batch incomplete
- **Graceful shutdown**: Flush remaining items on shutdown

### Configuration

From `config/ingestion.defaults.yml`:
```yaml
http:
  batch_size: 100          # DB batch size
  batch_flush_interval: 1s # Max wait time

kafka:
  batch_size: 100          # Kafka batch size
  batch_timeout: 1s        # Max wait time
```

## API

### HTTP: `POST /v1/logs`

Request:
```json
{
  "log_content": "raw log text",
  "client_source_org_id": "org-id"
}
```

Response:
```json
{
  "request_id": "uuid",
  "status": "RECEIVED"
}
```

### gRPC: `LogIngestion.SubmitLog`

Proto definition: [`proto/logingestion.proto`](../../proto/logingestion.proto)

## Error Handling

- Invalid input → 400 Bad Request
- Service errors → 500 Internal Server Error
- All errors logged with context
- Failed batch items tracked separately

## Performance

- **Async processing**: Non-blocking log acceptance
- **Batching**: Reduces DB and Kafka overhead
- **Connection pooling**: Reused connections
- **Graceful degradation**: Continues on partial failures