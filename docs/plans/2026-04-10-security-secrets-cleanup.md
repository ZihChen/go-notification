# Security Secrets Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除所有 git repo 中的明文敏感資訊，包含目前程式碼與 git 歷史記錄。

**Architecture:** 使用 Bitnami SealedSecrets 加密存儲敏感值到 `encryptedEnv`，以 `git-filter-repo` 清除歷史記錄，對已洩漏的憑證進行輪換。

**Tech Stack:** Helm, Bitnami SealedSecrets (`kubeseal`), `git-filter-repo`

---

## 問題清單

| 優先級 | 問題 | 狀態 |
|--------|------|------|
| P0 | `METRICS_OTLP_HEADERS` 含明文 Basic Auth (`winston.l@jvd.tw:REDACTED_OO_PASSWORD`) | ❌ 待處理 |
| P0 | 須輪換已洩漏的 OpenObserve 密碼 | ⏳ 人工作業 |
| P1 | git 歷史中有明文 `SSE_JWT_SECRET_KEY` | ❌ 待清理 |
| P1 | git 歷史中有明文 `AUTH_API_KEYS` (多組 UUID) | ❌ 待清理 |
| P1 | 確認 `SSE_JWT_SECRET_KEY` 舊值已輪換 | ⏳ 確認中 |
| P1 | 確認 `AUTH_API_KEYS` 舊 UUID 已輪換 | ⏳ 確認中 |

---

### Task 1: 移除 values 檔案中的明文 METRICS_OTLP_HEADERS（P0）

**Files:**
- Modify: `helm/values-pro.yaml`
- Modify: `helm/values-demo.yaml`

**目標：** 從 `env` 區段移除 `METRICS_OTLP_HEADERS`，並在 `encryptedEnv` 加入 TODO 佔位符。

**Step 1: 修改 helm/values-pro.yaml**

從 `env` 區段刪除：
```yaml
METRICS_OTLP_HEADERS: '{"Authorization":"Basic REDACTED_OO_AUTH_B64","stream-name":"fat-notification-cat"}'
```

在 `encryptedEnv` 區段加入（需 kubeseal 後替換）：
```yaml
  # METRICS_OTLP_HEADERS: <待使用 kubeseal 加密，見 Task 3>
```

**Step 2: 修改 helm/values-demo.yaml（同上）**

**Step 3: Commit**
```bash
git add helm/values-pro.yaml helm/values-demo.yaml
git commit -m "security(helm): remove plaintext METRICS_OTLP_HEADERS from env"
```

---

### Task 2: 建立 kubeseal 輔助腳本（P0）

**Files:**
- Create: `scripts/seal-secret.sh`

**目標：** 提供一個腳本，讓使用者在輪換密碼後可以快速產生 SealedSecret 值。

**Step 1: 建立腳本**

```bash
#!/bin/bash
# scripts/seal-secret.sh
# 用法: ./scripts/seal-secret.sh <namespace> <key> <value>
# 例: ./scripts/seal-secret.sh fatcat-extension METRICS_OTLP_HEADERS '{"Authorization":"Basic <base64>","stream-name":"fat-notification-cat"}'

set -euo pipefail
NAMESPACE="${1:?需要提供 namespace}"
KEY="${2:?需要提供 key 名稱}"
VALUE="${3:?需要提供要加密的值}"
RELEASE_NAME="fatnotificationcat"

echo "Sealing $KEY for namespace $NAMESPACE..."
echo -n "$VALUE" | \
  kubectl create secret generic "$RELEASE_NAME" \
    --namespace="$NAMESPACE" \
    --dry-run=client \
    --from-literal="$KEY=$VALUE" \
    -o yaml | \
  kubeseal \
    --format yaml \
    --namespace="$NAMESPACE" | \
  grep -A1 "encryptedData:" | tail -1 | \
  sed "s/^    $KEY: //"
```

**Step 2: 設定執行權限並 commit**
```bash
chmod +x scripts/seal-secret.sh
git add scripts/seal-secret.sh
git commit -m "chore: add seal-secret helper script"
```

---

### Task 3: 人工作業——輪換 OpenObserve 密碼（P0，需使用者執行）

> **此 Task 無法由 Claude 自動執行，需使用者手動完成。**

**Step 1: 登入 OpenObserve 並輪換密碼**
- 前往 OpenObserve Web UI (`https://o2.jvdev.cc`)
- 將 `winston.l@jvd.tw` 帳號的密碼從 `REDACTED_OO_PASSWORD` 改為新的強密碼

**Step 2: 產生新的 Base64**
```bash
echo -n "winston.l@jvd.tw:<新密碼>" | base64
```

**Step 3: 確認 AWS SSO 已登入（pro cluster）**
```bash
aws sso login --profile jvd-pro
```

**Step 4: 執行 kubeseal 腳本，為 pro 環境生成加密值**

demo namespace（bonuscat）：
```bash
./scripts/seal-secret.sh bonuscat METRICS_OTLP_HEADERS \
  '{"Authorization":"Basic <新的Base64>","stream-name":"fat-notification-cat"}'
```

pro namespace（fatcat-extension）：
```bash
./scripts/seal-secret.sh fatcat-extension METRICS_OTLP_HEADERS \
  '{"Authorization":"Basic <新的Base64>","stream-name":"fat-notification-cat"}'
```

**Step 5: 將輸出的 sealed 值填入 values 檔案的 encryptedEnv 區段：**
- `helm/values-demo.yaml` → bonuscat 的 sealed 值
- `helm/values-pro.yaml` → fatcat-extension 的 sealed 值

```yaml
encryptedEnv:
  METRICS_OTLP_HEADERS: <sealed 值>
  # ...其餘現有的 sealed values
```

**Step 6: Commit**
```bash
git add helm/values-pro.yaml helm/values-demo.yaml
git commit -m "security(helm): add sealed METRICS_OTLP_HEADERS to encryptedEnv"
```

---

### Task 4: git-filter-repo 清除歷史記錄（P1）

**目標：** 從整個 git 歷史中替換所有明文敏感值。

**⚠️ 警告：** 此操作會改寫所有 commit SHA。執行後需 force push，所有協作者需重新 clone。

**Step 1: 建立替換規則檔案**（已執行，此步驟僅供參考）

```bash
# 規則包含：OO Base64 token、OO 明文密碼、SSE JWT key、AUTH_API_KEYS JSON
git filter-repo --replace-text /tmp/git-replacements.txt --force
```

**Step 2: 執行 git-filter-repo**（已完成）

**Step 3: 重新加入 remote 並 force push**
```bash
git remote add personal <原本的 remote URL>
git push personal --force --all
git push personal --force --tags
```

**Step 4: 驗證清除成功**
```bash
# 確認歷史中已無明文
git log --all -p | grep -E "(REDACTED_OO_PASSWORD|REDACTED_SSE_JWT_KEY)" | wc -l
# 預期輸出: 0
```

---

### Task 5: 確認舊密鑰是否已輪換（P1，確認作業）

**SSE_JWT_SECRET_KEY 狀態確認：**

目前 `helm/values-pro.yaml` 的 `encryptedEnv` 中有：
```
SSE_JWT_SECRET_KEY: AgAZ52K/...（SealedSecret 加密值）
```

依據 commit 歷史：
- `f32a87f chore(helm):sse secret key` — 首次加入明文值
- `8359794 security(helm): migrate SSE JWT secret to encrypted environment variables` — 移至加密

**確認方式：** 向負責人確認 JWT secret 已在加密之後改過值（若 `AgAZ52K/...` 解密後仍是 `REDACTED_SSE_JWT_KEY` 則需重新生成）。

**AUTH_API_KEYS UUID 狀態確認：**

依據 commit 歷史：
- `7d9145f chore(helm): rotate AUTH_API_KEYS sealed secret in pro values` — 明確標示已輪換

**確認方式：** 向 API 管理員確認舊版 UUID 是否已在下游系統中作廢。

---

## 執行順序

```
Task 1 ──→ Task 2 ──→ Task 4 ──→ Task 5（確認）
                ↓
          Task 3（等待人工輪換密碼後）──→ Task 3 Step 4-6
```

**Task 1, 2, 4, 5 可由 Claude 立即執行。**
**Task 3 需使用者先登入 OpenObserve 輪換密碼。**
