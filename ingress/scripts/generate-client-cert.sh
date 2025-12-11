#!/bin/bash
# Generate client certificate for mTLS authentication
# This script generates a client certificate for consortium members

set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <member-id> <member-name>"
    echo "Example: $0 member-001 'Regulatory Authority A'"
    exit 1
fi

MEMBER_ID="$1"
MEMBER_NAME="$2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSL_DIR="${SCRIPT_DIR}/../ssl"
CA_DIR="${SSL_DIR}/ca"
CLIENT_DIR="${SSL_DIR}/clients/${MEMBER_ID}"

# Check if CA exists
if [ ! -f "${CA_DIR}/ca-cert.pem" ] || [ ! -f "${CA_DIR}/ca-key.pem" ]; then
    echo "Error: CA certificate not found. Please run generate-ssl-certs.sh first."
    exit 1
fi

# Create client directory
mkdir -p "${CLIENT_DIR}"

echo "Generating client certificate for member: ${MEMBER_NAME} (${MEMBER_ID})..."

# Generate client private key
echo "1. Generating client private key..."
openssl genrsa -out "${CLIENT_DIR}/client-key.pem" 2048

# Generate client certificate signing request
echo "2. Generating client certificate signing request..."
openssl req -new -key "${CLIENT_DIR}/client-key.pem" \
    -out "${CLIENT_DIR}/client.csr" \
    -subj "/C=US/ST=State/L=City/O=Consortium/CN=${MEMBER_NAME}"

# Generate client certificate signed by CA
echo "3. Generating client certificate..."
openssl x509 -req -days 365 -in "${CLIENT_DIR}/client.csr" \
    -CA "${CA_DIR}/ca-cert.pem" \
    -CAkey "${CA_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${CLIENT_DIR}/client-cert.pem" \
    -extensions v3_client \
    -extfile <(cat <<EOF
[v3_client]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
)


# Set proper permissions
chmod 600 "${CLIENT_DIR}/client-key.pem"
chmod 644 "${CLIENT_DIR}/client-cert.pem"

# Clean up
rm -f "${CLIENT_DIR}/client.csr"

echo ""
echo "Client certificate generated successfully!"
echo "Certificate files:"
echo "  - Client cert: ${CLIENT_DIR}/client-cert.pem"
echo "  - Client key: ${CLIENT_DIR}/client-key.pem"



