# 環境配置指南

## 概述

Fat Notification Cat 採用簡化的環境配置管理方式，支援兩種主要的部署模式：本地開發和 Kubernetes 部署。

## 支援的環境

| 環境 | 配置方式 | Atlas 環境 | 用途 |
|------|----------|-----------|------|
| **local** | `.env` 文件 | `local` | 本地開發環境 |
| **deployment** | ConfigMap + Secrets | `deployment` | Kubernetes 部署環境 |

## 快速設置

### 1. 本地開發設置

```bash
# 複製環境配置範本
cp .env.example .env

# 編輯 .env 文件，設定本地開發所需的配置
vim .env
```

### 2. 部署環境設置

部署環境的配置通過 Helm Chart 管理：

```bash
# 編輯 ConfigMap 配置
vim helm/templates/configmap.yaml

# 編輯 Values 文件
vim helm/values.yaml
```

## 資料庫遷移使用方式

### 本地開發

```bash
# migrate.sh 會自動偵測 .env 文件並載入配置
./migrate.sh status         # 查看遷移狀態
./migrate.sh apply          # 應用遷移
./migrate.sh gen <name>     # 生成新遷移檔案
./migrate.sh inspect        # 檢查資料庫結構
./migrate.sh diff          # 比較 schema 差異
./migrate.sh rollback      # 回滾最新遷移
./migrate.sh hash          # 重新計算遷移檔案哈希
```

### 部署環境

```bash
# 在 Kubernetes Pod 中執行
kubectl exec -it <pod-name> -- ./migrate.sh apply

# 使用 Atlas 變數覆蓋 (如果需要)
atlas migrate apply \
  --env deployment \
  --var db_host=your-host \
  --var db_user=your-user \
  --var db_password=your-password \
  --var db_name=your-database
```

## 環境配置詳解

### 本地開發環境 (local)

**配置來源**：`.env` 文件  
**特色**：
- 自動載入 `.env` 文件配置
- 適合本地開發和測試
- 較短的遷移鎖定超時時間 (10s)
- 支援 Atlas dev 資料庫用於 schema 比較和 migrate gen 功能
- 使用變數管理 dev 資料庫連接 (ATLAS_DEV_USER, ATLAS_DEV_PASSWORD)

### 部署環境 (deployment)

**配置來源**：Kubernetes ConfigMap + Secrets  
**特色**：
- 從系統環境變數讀取配置
- 較長的遷移鎖定超時時間 (60s)
- 設定基準版本 (baseline) 確保部署安全
- 適合 Kubernetes 生產環境

## Atlas 配置特色

### 環境隔離
- **local**：適合本地開發，較短超時時間 (10s)
- **deployment**：適合生產部署，較長超時時間 (60s)，包含基準版本設定

### 向後兼容
保留原有的 `gorm` 環境配置，映射到 `local` 環境。

## 環境變數優先順序

1. **系統環境變數** (最高優先級)
2. **`.env` 文件** (本地開發)
3. **Atlas 變數默認值** (最低優先級)

## 安全建議

### 敏感資訊管理
- **本地開發**：直接在 `.env` 文件中配置
- **部署環境**：使用 Kubernetes Secrets 管理敏感資訊

## 故障排除

### 常見問題

1. **環境文件未載入**
   ```bash
   # 檢查 .env 文件是否存在
   ls -la .env
   
   # 檢查文件權限
   chmod 644 .env
   
   # 檢查 JSON 格式是否正確 (AUTH_API_KEYS)
   cat .env | grep AUTH_API_KEYS
   ```

2. **Atlas 環境不存在**
   ```bash
   # 檢查可用環境
   atlas env list
   
   # 檢查 atlas.hcl 配置
   cat atlas.hcl
   ```

3. **資料庫連接失敗**
   ```bash
   # 檢查環境變數
   echo $DB_HOST $DB_USER $DB_NAME
   
   # 檢查 Atlas dev 資料庫配置
   echo $ATLAS_DEV_USER $ATLAS_DEV_PASSWORD
   
   # 測試連接
   atlas schema inspect --env local
   ```

4. **遷移回滾失敗 (Assertion Check)**
   ```bash
   # 問題：Atlas 安全檢查阻止回滾
   # 原因：要刪除的欄位包含非 NULL 資料
   # 解決：刪除測試遷移檔案或先清理資料
   
   # 檢查遷移檔案
   ls -la migrations/
   
   # 刪除測試遷移檔案 (如果確認安全)
   rm migrations/<test_migration_file>.sql
   
   # 重新計算哈希
   ./migrate.sh hash
   ```

5. **複雜環境變數解析錯誤**
   ```bash
   # 問題：包含特殊字符的環境變數 (如 JSON)
   # 解決：使用 source 命令替代 export
   
   # migrate.sh 已自動處理此問題
   # 使用 set -a; source .env; set +a 方式載入
   ```

### 調試模式

```bash
# 顯示配置資訊
./migrate.sh status

# Atlas 詳細模式
atlas migrate apply --env local --verbose
```

## 重要配置項說明

### Atlas Dev 資料庫

用於 `./migrate.sh gen` 功能的測試資料庫配置：

```bash
# .env 文件中配置
ATLAS_DEV_USER=your_dev_db_user
ATLAS_DEV_PASSWORD=your_dev_db_password

# ConfigMap 中配置 (部署環境)
ATLAS_DEV_USER: "your_dev_db_user"
# ATLAS_DEV_PASSWORD: <secret>
```

### API 認證配置

```bash
# JSON 格式的 API Keys (注意引號使用)
AUTH_API_KEYS='{"api-key-1":"MERCHANT-1", "api-key-2":"MERCHANT-2"}'
```

### 環境變數載入順序

1. **系統環境變數** (最高優先級)
2. **`.env` 文件** (本地開發，使用 `source .env`)
3. **Atlas 變數默認值** (最低優先級)

## 相關文件

- `migrate.sh` - 遷移腳本 (支援環境自動檢測)
- `atlas.hcl` - Atlas 配置文件 (變數化DSN配置)
- `.env.example` - 環境配置範本
- `.env` - 本地開發環境配置
- `helm/templates/configmap.yaml` - Kubernetes 部署配置
- `helm/templates/secrets.yaml` - Kubernetes 敏感資訊配置