# Configuration Directory

Configuration files and templates for LogChain services.

## Quick Start

### 1. Setup Environment Variables

```bash
# In project root
cp .env.example .env
```

Edit `.env` with your values:
```env
CHAINMAKER_PATH=/path/to/chainmaker-go
CHAINMAKER_NODE_HOST=30.177.108.73
CHAINMAKER_NODE_PORT_1=12301
# ... other ports
```

### 2. Generate Blockchain Client Config

```bash
bash scripts/generate-chainmaker-config.sh
```

Generates `clients/chainmaker.yml` from template.

### 3. Start Services

```bash
docker compose up -d
```

## Configuration Files

### Service Configurations

- **`ingestion.defaults.yml`**: Ingestion service (HTTP/gRPC ports, database, Kafka, batch processing)
- **`engine.defaults.yml`**: Engine service (Kafka consumer, batch processing, blockchain client)
- **`query.defaults.yml`**: Query service (HTTP port, database, blockchain client)
- **`blockchain.defaults.yml`**: Blockchain client settings (type, connection parameters)

### Blockchain Client

- **`clients/chainmaker.yml.template`**: Template with environment variable placeholders
- **`clients/chainmaker.yml`**: Generated config (git ignored, regenerate after `.env` changes)

## Environment Variables

- **`CHAINMAKER_PATH`**: Absolute path to ChainMaker installation (mounted read-only)
- **`CHAINMAKER_NODE_HOST`**: Node hostname or IP
- **`CHAINMAKER_NODE_PORT_*`**: Ports for each consensus node

## Notes

- Always regenerate `clients/chainmaker.yml` after updating `.env`
- `.env` and `clients/chainmaker.yml` are not committed to git
- All `*.defaults.yml` files are mounted into Docker containers
