# Trusted Log Attestation System (可信日志存证系统)

A system for transparently attesting logs on a blockchain with multi-dimensional verifiability.

## 📚 Documentation Guide

| Document | Purpose | Audience |
|----------|---------|----------|
| **[design.md](design.md)** | 🏗️ **Complete system architecture, design decisions, and specifications** | Architects, Senior Developers |
| **[README.md](#)** | 🚀 **Project overview, quick start, and setup instructions** | New Contributors, Users |
| **[CLAUDE.md](CLAUDE.md)** | 💻 **Development guidelines, coding standards, and implementation details** | Developers, AI Assistants |

📖 **This README focuses on getting started. For architecture details, see [design.md](design.md). For development specifics, see [CLAUDE.md](CLAUDE.md).**

## Architecture Overview

This project implements the architecture described in the [System Design Document](design.md). The Trusted Log Attestation System follows a layered microservices architecture with:

- **Decoupling**: Message queue separates high-speed log ingestion from slow blockchain consensus
- **Encapsulation**: Core business APIs with Benthos for heterogeneous data source adaptation
- **State vs Content Separation**: Off-chain state DB tracks task status, on-chain blockchain stores content
- **Multi-Key Verifiability**: Multiple query methods for different user types

📖 **For detailed architecture specifications, component responsibilities, and design decisions, see [design.md](design.md)**

## Directory Structure

```
tlng/
├── cmd/                     # Service entry points
│   ├── ingestion/          # ✅ Log Ingestion Service
│   └── engine/             # ✅ Blockchain Processing Service
├── ingestion/              # ✅ Ingestion Layer implementation
├── processing/             # ✅ Processing Layer implementation
├── blockchain/             # ✅ Blockchain Layer
├── storage/                # ✅ Storage Layer
├── internal/               # Shared utilities
├── config/                 # Configuration files
├── proto/                  # gRPC definitions
├── scripts/               # Utility scripts
├── docker-compose.yml     # Infrastructure
├── design.md              # 📖 System design document
├── CLAUDE.md              # 📖 Development guide
├── benchmark/             # Performance testing
└── ❌ TODO directories:
    ├── query/               # Query Layer
    └── ingress/             # API Gateway
```

📖 **For detailed directory descriptions, see [CLAUDE.md](CLAUDE.md)**

## Implementation Status

### ✅ Implemented Services

📖 **For detailed implementation guidance, see [CLAUDE.md](CLAUDE.md)**

- **Log Ingestion Service** - HTTP/gRPC endpoints with SHA256 hashing and Kafka integration
- **Blockchain Processing Service** - Multi-worker Kafka consumer with ChainMaker integration
- **Supporting Infrastructure** - PostgreSQL state database, Kafka message queue, ChainMaker blockchain client

📖 **For detailed architecture specifications, component responsibilities, and design decisions, see [design.md](design.md)**

### ❌ TODO Components
📖 **For detailed component specifications and implementation priorities, see [design.md](design.md)**

Key components to be implemented:
- **API Gateway** - TLS termination, unified authentication, and protocol routing
- **Benthos Adapters** - Direct protocol reception (S3, Syslog, Kafka) with security controls
- **Query Layer** - Multi-dimensional query APIs for different user types

## Development

### Prerequisites
```bash
# Start infrastructure dependencies
docker-compose up -d
```

### Building
```bash
# Build Log Ingestion Service
go build -o bin/ingestion ./cmd/ingestion

# Build Blockchain Processing Service
go build -o bin/engine ./cmd/engine
```

### Running Services
```bash
# Run Log Ingestion Service
./bin/ingestion

# Run Blockchain Processing Service
./bin/engine
```

### Testing
Waiting for future implementation.

## Message Flow

📖 **For detailed message flow specifications and data transformation, see [design.md](design.md)**

The system supports multiple ingestion paths:

1. **Standard HTTP/gRPC Clients** - Direct API submission through TLS-protected endpoints
2. **Heterogeneous Protocol Sources** - Benthos adapters for S3, Syslog, and Kafka
3. **Query Requests** - Multi-dimensional log status and content verification

## Configuration

Each service loads configuration from YAML files:
- Log Ingestion Service: `config/ingestion.defaults.yml`
- Blockchain Processing Service: `config/engine.defaults.yml`
- Blockchain client: `config/blockchain.defaults.yml`

## Documentation

- [System Design Document](design.md) - Complete architecture and specifications
- [CLAUDE.md](CLAUDE.md) - Development guidelines for AI assistants
- API documentation - TODO (to be implemented with query layer)