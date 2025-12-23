#!/bin/bash
# Setup configuration files for Nginx API Gateway
# This script copies example files and creates initial configuration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONF_DIR="${SCRIPT_DIR}/../nginx/conf.d"

echo "Setting up Nginx API Gateway configuration..."

# Copy example files if they don't exist
if [ ! -f "${CONF_DIR}/api-keys.json" ]; then
    echo "Creating api-keys.json from example..."
    cp "${CONF_DIR}/api-keys.json.example" "${CONF_DIR}/api-keys.json"
    echo "Please update ${CONF_DIR}/api-keys.json with your actual API keys"
else
    echo "api-keys.json already exists"
fi

if [ ! -f "${CONF_DIR}/consortium-ip-whitelist.json" ]; then
    echo "Creating consortium-ip-whitelist.json from example..."
    cp "${CONF_DIR}/consortium-ip-whitelist.json.example" "${CONF_DIR}/consortium-ip-whitelist.json"
    echo "Please update ${CONF_DIR}/consortium-ip-whitelist.json with your IP whitelist"
else
    echo "consortium-ip-whitelist.json already exists"
fi

echo ""
echo "Configuration setup complete!"
echo ""
echo "Next steps:"
echo "1. Update ${CONF_DIR}/api-keys.json with your API keys"
echo "2. Update ${CONF_DIR}/consortium-ip-whitelist.json with your IP whitelist"
echo "3. Generate SSL certificates: ./scripts/generate-ssl-certs.sh"
echo "4. Generate client certificates: ./scripts/generate-client-cert.sh <member-id> <member-name>"

