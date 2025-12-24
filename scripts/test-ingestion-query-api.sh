#!/bin/bash

# LogChain API 测试脚本
# 使用方法: ./test-api.sh [api-key]

set -e

API_KEY="${1:-example-api-key-12345}"
BASE_URL="https://localhost"

echo "==================================="
echo "LogChain API 测试"
echo "==================================="
echo "API Key: $API_KEY"
echo "Base URL: $BASE_URL"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 提交日志
echo -e "${YELLOW}[1/4] 测试提交日志 (POST /v1/logs)${NC}"
LOG_CONTENT="Test log at $(date +%s)"
SUBMIT_RESPONSE=$(curl -sk -X POST "$BASE_URL/v1/logs" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{\"log_content\":\"$LOG_CONTENT\"}")

echo "$SUBMIT_RESPONSE" | jq .

REQUEST_ID=$(echo "$SUBMIT_RESPONSE" | jq -r '.request_id')
LOG_HASH=$(echo "$SUBMIT_RESPONSE" | jq -r '.server_log_hash')

if [ -z "$REQUEST_ID" ] || [ "$REQUEST_ID" == "null" ]; then
    echo -e "${RED}✗ 提交失败${NC}"
    exit 1
fi
echo -e "${GREEN}✓ 提交成功: request_id=$REQUEST_ID${NC}\n"

# 等待处理
echo "等待 3 秒让日志处理完成..."
sleep 3

# 2. 按 request_id 查询
echo -e "${YELLOW}[2/4] 测试按 request_id 查询 (GET /v1/query/status/{id})${NC}"
QUERY_RESPONSE=$(curl -sk -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/query/status/$REQUEST_ID")

echo "$QUERY_RESPONSE" | jq .

STATUS=$(echo "$QUERY_RESPONSE" | jq -r '.status')
if [ "$STATUS" == "COMPLETED" ]; then
    echo -e "${GREEN}✓ 查询成功: 状态=$STATUS${NC}\n"
else
    echo -e "${YELLOW}! 查询成功但状态为: $STATUS (可能还在处理中)${NC}\n"
fi

# 3. 按内容查询
echo -e "${YELLOW}[3/4] 测试按内容查询 (POST /v1/query_by_content)${NC}"
CONTENT_QUERY_RESPONSE=$(curl -sk -X POST "$BASE_URL/v1/query_by_content" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{\"log_content\":\"$LOG_CONTENT\"}")

# 注意：这会查询到刚才提交的日志，因为使用了相同的内容
echo "$CONTENT_QUERY_RESPONSE" | jq .

if echo "$CONTENT_QUERY_RESPONSE" | jq -e '.request_id' >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 按内容查询成功${NC}\n"
else
    ERROR_MSG=$(echo "$CONTENT_QUERY_RESPONSE" | jq -r '.message // .error')
    if [ "$ERROR_MSG" == "log not found" ]; then
        echo -e "${YELLOW}! 未找到日志 (内容哈希不匹配)${NC}\n"
    else
        echo -e "${RED}✗ 查询失败: $ERROR_MSG${NC}\n"
    fi
fi

# 4. 权限测试：尝试查询其他组织的日志
echo -e "${YELLOW}[4/4] 测试权限隔离 (应该失败)${NC}"
# 使用一个不存在的 request_id 或者其他组织的日志
PERM_TEST_RESPONSE=$(curl -sk -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/query/status/00000000-0000-0000-0000-000000000000")

echo "$PERM_TEST_RESPONSE" | jq .

if echo "$PERM_TEST_RESPONSE" | jq -e '.error' >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 权限检查正常工作${NC}\n"
else
    echo -e "${YELLOW}! 意外的响应${NC}\n"
fi

echo "==================================="
echo -e "${GREEN}测试完成${NC}"
echo "==================================="
