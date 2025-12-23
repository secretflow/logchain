#!/bin/bash

# Generate ChainMaker configuration from template
# This script substitutes environment variables into the config template

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_FILE="${PROJECT_ROOT}/config/clients/chainmaker.yml.template"
OUTPUT_FILE="${PROJECT_ROOT}/config/clients/chainmaker.yml"
ENV_FILE="${PROJECT_ROOT}/.env"

# Check if .env file exists
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found at $ENV_FILE"
    echo "Please copy .env.example to .env and configure your settings:"
    echo "  cp .env.example .env"
    exit 1
fi

# Load environment variables
set -a  # automatically export all variables
source "$ENV_FILE"
set +a

# Check required variables
if [ -z "$CHAINMAKER_NODE_HOST" ]; then
    echo "Error: CHAINMAKER_NODE_HOST not set in .env"
    exit 1
fi

# Set defaults for ports if not specified
CHAINMAKER_NODE_PORT_1=${CHAINMAKER_NODE_PORT_1:-12301}
CHAINMAKER_NODE_PORT_2=${CHAINMAKER_NODE_PORT_2:-12302}
CHAINMAKER_NODE_PORT_3=${CHAINMAKER_NODE_PORT_3:-12303}
CHAINMAKER_NODE_PORT_4=${CHAINMAKER_NODE_PORT_4:-12304}

echo "Generating ChainMaker configuration..."
echo "  Template: $TEMPLATE_FILE"
echo "  Output: $OUTPUT_FILE"
echo "  Node Host: $CHAINMAKER_NODE_HOST"

# Substitute environment variables
envsubst < "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "Configuration generated successfully!"
echo ""
echo "Summary:"
echo "  Node addresses:"
echo "    - ${CHAINMAKER_NODE_HOST}:${CHAINMAKER_NODE_PORT_1}"
echo "    - ${CHAINMAKER_NODE_HOST}:${CHAINMAKER_NODE_PORT_2}"
echo "    - ${CHAINMAKER_NODE_HOST}:${CHAINMAKER_NODE_PORT_3}"
echo "    - ${CHAINMAKER_NODE_HOST}:${CHAINMAKER_NODE_PORT_4}"
