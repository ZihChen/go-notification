#!/bin/bash
# seal-secret.sh — 使用 kubeseal 加密單一環境變數，輸出可貼入 encryptedEnv 的值
#
# 用法:
#   ./scripts/seal-secret.sh <namespace> <key> <value>
#
# 範例 (pro):
#   ./scripts/seal-secret.sh fatcat-extension METRICS_OTLP_HEADERS \
#     '{"Authorization":"Basic <base64>","stream-name":"fat-notification-cat"}'
#
# 範例 (demo):
#   ./scripts/seal-secret.sh bonuscat METRICS_OTLP_HEADERS \
#     '{"Authorization":"Basic <base64>","stream-name":"fat-notification-cat"}'
#
# 前置條件:
#   - kubectl 已連線到目標 cluster (aws sso login)
#   - kubeseal 已安裝 (brew install kubeseal)

set -euo pipefail

NAMESPACE="${1:?用法: $0 <namespace> <key> <value>}"
KEY="${2:?用法: $0 <namespace> <key> <value>}"
VALUE="${3:?用法: $0 <namespace> <key> <value>}"
RELEASE_NAME="fatnotificationcat"

echo "=== 加密 $KEY (namespace: $NAMESPACE) ===" >&2

SEALED=$(echo -n "" | \
  kubectl create secret generic "$RELEASE_NAME" \
    --namespace="$NAMESPACE" \
    --dry-run=client \
    --from-literal="${KEY}=${VALUE}" \
    -o yaml | \
  kubeseal \
    --format yaml \
    --namespace="$NAMESPACE" | \
  python3 -c "
import sys, yaml
data = yaml.safe_load(sys.stdin)
key = '${KEY}'
print(data['spec']['encryptedData'][key])
")

echo ""
echo "請將以下值加入對應 values 檔案的 encryptedEnv 區段:"
echo ""
echo "  ${KEY}: ${SEALED}"
echo ""
