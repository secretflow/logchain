# Configuration Guide

## Quick Start

### 1. Create Environment Configuration

Copy the example file and update with your values:

```bash
cp .env.example .env
```

Edit `.env` and set:
- `CHAINMAKER_PATH`: Absolute path to your ChainMaker installation
- `CHAINMAKER_NODE_HOST`: ChainMaker node hostname or IP
- `CHAINMAKER_NODE_PORT_*`: Ports for each node (default: 12301-12304)

Example:
```env
CHAINMAKER_PATH=/home/user/chainmaker/chainmaker-go
CHAINMAKER_NODE_HOST=30.177.108.73
CHAINMAKER_NODE_PORT_1=12301
CHAINMAKER_NODE_PORT_2=12302
CHAINMAKER_NODE_PORT_3=12303
CHAINMAKER_NODE_PORT_4=12304
```

### 2. Generate ChainMaker Configuration

Run the script to generate `config/clients/chainmaker.yml` from template:

```bash
bash scripts/generate-chainmaker-config.sh
```

This will substitute environment variables into the configuration template.

### 3. Start Services

```bash
docker compose up -d
```

## Configuration Files

- `.env.example` - Template for environment variables
- `.env` - Your actual environment configuration (not committed to git)
- `config/clients/chainmaker.yml.template` - ChainMaker config template
- `config/clients/chainmaker.yml` - Generated config (not committed to git)

## Notes

- The `.env` file is ignored by git to keep your local paths private
- Always run `generate-chainmaker-config.sh` after updating `.env`
- The ChainMaker installation path is mounted read-only to Docker containers
