#!/bin/bash
# 将 gateway.yaml 发布到配置中心（Admin API :8081）
# 用法：bash publish-config.sh [admin-addr]

set -e

ADMIN="${1:-http://127.0.0.1:8081}"
NAMESPACE="nexus-gateway"
KEY="gateway.yaml"
CONFIG_FILE="$(dirname "$0")/gateway.yaml"

echo "==> 配置中心地址: $ADMIN"
echo "==> 命名空间:     $NAMESPACE"
echo "==> 配置键:       $KEY"
echo ""

# ── 1. 创建命名空间（幂等：已存在时忽略报错）──
echo "[1/3] 创建命名空间..."
curl -s -X POST "$ADMIN/api/v1/namespaces/" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$NAMESPACE\",\"name\":\"Nexus Gateway\",\"description\":\"网关动态配置\"}" \
  | python3 -m json.tool 2>/dev/null || true
echo ""

# ── 2. 保存草稿 ──
echo "[2/3] 保存草稿..."
VALUE=$(cat "$CONFIG_FILE")
BODY=$(python3 -c "
import json, sys
value = open('$CONFIG_FILE').read()
print(json.dumps({
    'namespace': '$NAMESPACE',
    'key':       '$KEY',
    'value':     value,
    'format':    'yaml'
}))
")
curl -s -X POST "$ADMIN/api/v1/configs/draft" \
  -H "Content-Type: application/json" \
  -d "$BODY" \
  | python3 -m json.tool
echo ""

# ── 3. 发布 ──
echo "[3/3] 发布配置..."
curl -s -X POST "$ADMIN/api/v1/configs/publish" \
  -H "Content-Type: application/json" \
  -d "{\"namespace\":\"$NAMESPACE\",\"key\":\"$KEY\"}" \
  | python3 -m json.tool
echo ""

echo "==> 完成！网关会在 poll_timeout 秒内自动拉取新配置。"
