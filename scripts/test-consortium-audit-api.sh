#!/bin/bash

# LogChain 审计接口测试脚本（使用 mTLS）
# 使用方法: ./test-audit-api.sh <log-hash>

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <log-hash>"
    echo "Example: $0 40dc7a0be4aaab2b8cd7982104bb5f029da283766451f1a8de41f1458da8a80c"
    exit 1
fi

LOG_HASH="$1"
BASE_URL="https://localhost"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_CERT="${SCRIPT_DIR}/../ingress/scripts/clients/member-001/client-cert.pem"
CLIENT_KEY="${SCRIPT_DIR}/../ingress/scripts/clients/member-001/client-key.pem"
CA_CERT="${SCRIPT_DIR}/../ingress/nginx/ssl/ca-cert.pem"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "==================================="
echo "LogChain 审计接口测试 (mTLS)"
echo "==================================="
echo "Log Hash: $LOG_HASH"
echo "Base URL: $BASE_URL"
echo ""

# 检查证书文件
if [ ! -f "$CLIENT_CERT" ] || [ ! -f "$CLIENT_KEY" ]; then
    echo -e "${RED}✗ 客户端证书不存在${NC}"
    echo "请先运行: cd ingress && bash scripts/generate-client-cert.sh member-001 \"Regulatory Authority A\""
    exit 1
fi

echo -e "${YELLOW}测试审计接口 (GET /v1/audit/log/{hash})${NC}"
echo "使用 mTLS 认证..."
echo ""

AUDIT_RESPONSE=$(curl -sk \
  --cert "$CLIENT_CERT" \
  --key "$CLIENT_KEY" \
  --cacert "$CA_CERT" \
  "$BASE_URL/v1/audit/log/$LOG_HASH")

echo "$AUDIT_RESPONSE" | jq .

if echo "$AUDIT_RESPONSE" | jq -e '.log_hash' >/dev/null 2>&1; then
    SOURCE=$(echo "$AUDIT_RESPONSE" | jq -r '.source')
    ORG_ID=$(echo "$AUDIT_RESPONSE" | jq -r '.sender_org_id')
    echo ""
    echo -e "${GREEN}✓ 审计查询成功${NC}"
    echo "  数据来源: $SOURCE"
    echo "  组织 ID: $ORG_ID"
else
    echo ""
    ERROR_MSG=$(echo "$AUDIT_RESPONSE" | jq -r '.message // .error')
    echo -e "${RED}✗ 审计查询失败: $ERROR_MSG${NC}"
    exit 1
fi

echo ""
echo "==================================="
echo -e "${GREEN}测试完成${NC}"
echo "==================================="
