# LogChain - Trusted Log Attestation System

## Overview

LogChain is an enterprise-grade log attestation platform that leverages blockchain technology to provide **tamper-proof**, **transparent**, and **verifiable** log storage. By combining high-performance log ingestion with consortium blockchain attestation, LogChain ensures the integrity and authenticity of critical log data for compliance, auditing, and dispute resolution.

## Why LogChain?

Traditional log management systems face several challenges:

| Challenge | Traditional Approach | LogChain Solution |
|-----------|---------------------|-------------------|
| **Tampering Risk** | Logs can be modified or deleted | Blockchain-backed immutability |
| **Trust Issues** | Single point of control | Multi-party consensus verification |
| **Audit Complexity** | Manual verification processes | Cryptographic proof with hash chains |
| **Integration Burden** | Custom adapters for each source | Universal protocol adapters |
| **Scalability** | Bottlenecks under high volume | Async processing with message queues |

## Core Capabilities

### 🔐 Immutable Attestation
Every log entry is hashed (SHA-256) and recorded on ChainMaker consortium blockchain. Once attested, the record cannot be altered or removed, providing irrefutable proof of log existence and content.

### 🔄 Universal Ingestion
Support multiple ingestion methods to fit your existing infrastructure:
- **API Integration**: RESTful HTTP and gRPC interfaces for direct submission
- **Syslog**: Industry-standard UDP/TCP syslog protocol
- **Kafka**: Stream processing from existing Kafka clusters
- **S3**: Batch processing from object storage

### 🔍 Multi-Dimensional Query
Different stakeholders access data through appropriate channels:
- **Status Query**: Track attestation progress by request ID
- **Content Query**: Verify log attestation using original content
- **On-Chain Audit**: Consortium members directly query blockchain records

### 🛡️ Enterprise Security
- **API Key Authentication**: For application integration
- **mTLS + IP Whitelist**: Dual authentication for consortium audit access
- **Rate Limiting**: Protection against abuse and DDoS
- **Audit Logging**: Complete authentication and operation trails

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Log Sources                              │
│         (Applications, Servers, IoT Devices, Cloud Services)    │
└─────────────────┬───────────────────────────┬───────────────────┘
                  │                           │
        ┌─────────▼─────────┐       ┌─────────▼─────────┐
        │    API Gateway    │       │  Protocol Adapters│
        │  (HTTPS / gRPC)   │       │ (Syslog/Kafka/S3) │
        └─────────┬─────────┘       └─────────┬─────────┘
                  │                           │
                  └───────────┬───────────────┘
                              ▼
                  ┌───────────────────────┐
                  │   Ingestion Service   │
                  │  (Hash + Queue + DB)  │
                  └───────────┬───────────┘
                              │
                              ▼
                  ┌───────────────────────┐
                  │   Processing Engine   │
                  │  (Batch Attestation)  │
                  └───────────┬───────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
    ┌─────────────────┐             ┌─────────────────┐
    │   State DB      │             │   Blockchain    │
    │  (PostgreSQL)   │             │  (ChainMaker)   │
    └─────────────────┘             └─────────────────┘
```

**Key Design Principles:**
- **Decoupling**: Message queue separates high-speed ingestion from blockchain consensus
- **Separation of Concerns**: Off-chain DB for status tracking, on-chain for content attestation
- **Horizontal Scalability**: Stateless services enable easy scaling

## Use Cases

### Regulatory Compliance
Financial institutions, healthcare providers, and government agencies can use LogChain to maintain audit trails that satisfy regulatory requirements (SOX, HIPAA, GDPR).

### Security Incident Investigation
When security incidents occur, LogChain provides tamper-proof evidence that logs have not been altered since the time of collection.

### Multi-Party Business Processes
In supply chain, cross-border trade, or consortium business scenarios, LogChain enables all parties to trust shared log records without relying on a single authority.

### Legal Evidence Preservation
Logs attested through LogChain can serve as admissible evidence, with blockchain records proving authenticity and timestamp.

## Integration Options

### Direct API Integration
```bash
# Submit log via HTTPS
curl -X POST https://logchain.example.com/v1/logs \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"log_content": "User login from 192.168.1.100"}'

# Response
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "server_log_hash": "a1b2c3d4..."
}
```

### Syslog Integration
```bash
# Configure existing syslog to forward to LogChain
# UDP port 5514, TCP port 6514
logger -n logchain.example.com -P 5514 "Application event occurred"
```

### Kafka Integration
Connect LogChain to consume from your existing Kafka topics with TLS/mTLS support.

### S3 Integration
Configure LogChain to process log files from S3-compatible storage (AWS S3, MinIO).

## Query & Verification

### For Log Submitters
Query attestation status using the returned `request_id`:
```bash
curl -H "X-API-Key: your-api-key" \
  https://logchain.example.com/v1/query/status/{request_id}
```

### For Non-API Users (Syslog/Kafka)
Verify attestation using original log content:
```bash
curl -X POST https://logchain.example.com/v1/query_by_content \
  -H "X-API-Key: your-api-key" \
  -d '{"log_content": "original log message"}'
```

### For Consortium Auditors
Direct blockchain query with mTLS authentication:
```bash
curl https://logchain.example.com/v1/audit/log/{log_hash} \
  --cert client.pem --key client-key.pem
```

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| API Gateway | Nginx + OpenResty | TLS termination, authentication, routing |
| Protocol Adapters | Redpanda Connect | Syslog, Kafka, S3 protocol conversion |
| Backend Services | Go | High-performance log processing |
| Message Queue | Kafka | Async processing buffer |
| State Database | PostgreSQL | Status tracking and indexing |
| Blockchain | ChainMaker | Consortium blockchain attestation |

## Getting Started

1. **Contact Us**: Reach out to discuss your requirements and deployment options
2. **Pilot Deployment**: We'll help you set up a proof-of-concept environment
3. **Integration**: Connect your log sources using our adapters or APIs
4. **Go Live**: Scale to production with our support

## Learn More

- [Technical Architecture](design.md) - Detailed system design and specifications
- [Deployment Guide](chainmaker_deployment.md) - Self-hosted deployment instructions
- [GitHub Repository](https://github.com/your-org/logchain) - Source code and documentation

---

*LogChain - Making every log trustworthy.*
