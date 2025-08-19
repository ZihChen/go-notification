# Fat Notification Cat

基於 Go 語言開發的微服務通知管理系統，專為商戶、玩家和管理員提供訊息推送與通知管理服務。採用六角架構（Hexagonal Architecture）設計，整合 AWS Kinesis Data Streams 進行事件處理，並使用 Redis 進行快取與任務佇列管理。

## 專案架構

### 核心服務

系統包含四個可獨立運行的核心服務：

1. **Web Service** (`cmd/web`) 
   - HTTP API 服務器，使用 Gin 框架
   - 提供 RESTful API 端點進行 CRUD 操作
   - 預設端口：8080

2. **Consumer Service** (`cmd/consumer`)
   - 處理來自 AWS Kinesis Data Streams 的事件
   - 實時消費並處理串流資料

3. **Worker Service** (`cmd/worker`)
   - 背景任務處理器
   - 使用 Asynq 進行非同步任務處理

4. **Scheduler Service** (`cmd/scheduler`)
   - 管理排程任務和訊息活動
   - 定時執行批次作業

### 領域實體

- **Merchant** - 商戶實體
- **Player** - 終端用戶/客戶
- **Manager** - 管理員用戶
- **Message Campaign** - 通知活動和訊息推送
- **Tags/Levels** - 用戶分類和層級管理

### 清潔架構分層

專案遵循六角架構，具有明確的關注點分離：

```
internal/
├── domain/              # 核心業務邏輯、介面定義（Ports）
│   ├── entity/         # 領域實體
│   ├── repositoryport/ # Repository 介面
│   └── errmsg/        # 自定義錯誤類型
├── adapter/            # 領域介面實作（Adapters）
│   ├── handler/       # HTTP/Worker/Scheduler 處理器
│   ├── repository/    # 資料庫操作實作
│   ├── usecase/      # 業務用例
│   └── service/      # 外部服務整合
└── infrastructure/     # 外部依賴
    ├── database/      # MySQL with GORM
    ├── cache/redis/   # Redis 快取
    ├── kds/          # AWS Kinesis 整合
    ├── queue/        # Asynq 任務佇列
    └── tracing/      # OpenTelemetry 整合
```

### 依賴注入

使用 Google Wire 進行編譯時期依賴注入 (`internal/di/wire.go`)

## 快速開始

### 環境需求

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 7.0+
- AWS 帳號（用於 Kinesis Data Streams）

### 本地開發

1. 複製專案並安裝依賴
```bash
git clone https://gitlab.jvdtech.dev/fatcat/fat_notification_cat.git
cd fat_notification_cat
go mod download
```

2. 設定環境變數
```bash
cp .env.example .env
# 編輯 .env 檔案，設定必要的環境變數
```

3. 啟動依賴服務
```bash
docker-compose up -d mysql redis
```

4. 執行資料庫遷移
```bash
atlas migrate apply --env local
```

5. 啟動服務
```bash
# 啟動 Web 服務
go run main.go web

# 啟動 Consumer 服務
go run main.go consumer

# 啟動 Worker 服務
go run main.go worker

# 啟動 Scheduler 服務
go run main.go scheduler
```

## 開發指令

### 建置與執行

```bash
# 使用 Docker Compose 啟動所有服務
docker-compose up -d --build
```

### 測試

```bash
# 執行所有測試
go test ./...

# 執行測試並產生覆蓋率報告
go test -cover ./...

# 執行組件測試
go test -v ./internal/adapter/...

# 執行測試並產生 HTML 覆蓋率報告
go test -coverprofile=coverage.out ./internal/adapter/...
go tool cover -func=coverage.out 
```

### 程式碼品質

```bash
# 執行 linter
golangci-lint run --fix
```

### 資料庫遷移

```bash
# 產生新的遷移檔案
./migrate.sh gen <migration_name> 

# 執行遷移
./migrate.sh apply

# 查看遷移狀態
./migrate.sh status
```

### Wire 依賴注入

```bash
# 安裝 Wire
go install github.com/google/wire/cmd/wire@latest

# 重新產生 wire_gen.go
wire ./internal/di
```

## API 文件

專案使用 Swagger 產生 API 文件：

```bash
# 安裝 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 產生/更新 Swagger 文件
swag init

# 存取 Swagger UI（Web 服務啟動後）
# http://localhost:8080/swagger/index.html
```

## 事件流程

1. **事件接收**：事件通過 AWS Kinesis Data Streams 抵達
2. **事件消費**：Consumer 服務處理 KDS 事件並將任務加入 Redis 佇列
3. **非同步處理**：Worker 服務非同步處理 Redis 佇列中的任務
4. **API 互動**：Web 服務提供 HTTP API 進行直接互動

## 配置管理

應用程式使用 Viper 進行配置管理，配置從環境變數和 `.env` 檔案載入。

### 主要配置項目

- **資料庫連線**（MySQL）
  - `DB_HOST`
  - `DB_PORT`
  - `DB_NAME`
  - `DB_USER`
  - `DB_PASSWORD`

- **Redis 連線**
  - `REDIS_HOST`
  - `REDIS_PORT`
  - `REDIS_PASSWORD`

- **AWS 設定**
  - `AWS_REGION`
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `KINESIS_STREAM_NAME`

- **OpenTelemetry 追蹤**
  - `OTEL_EXPORTER_OTLP_ENDPOINT`
  - `OTEL_SERVICE_NAME`

- **服務端口**
  - `WEB_PORT`（預設：8080）
  - `WORKER_CONCURRENCY`
  - `SCHEDULER_INTERVAL`

## 設計模式

### Repository Pattern
所有資料庫操作都通過定義在 `internal/domain/repositoryport/` 的 repository 介面進行

### Use Case Pattern
業務邏輯封裝在 use case 中，協調 repositories 和 services

### Event-Driven Architecture
系統使用事件進行服務間通訊，通過 KDS 和 Redis 佇列實現

### Error Handling
在 `internal/domain/errmsg/` 定義自定義錯誤類型，確保整個應用程式的錯誤處理一致性

## 監控與追蹤

專案整合 OpenTelemetry 進行分散式追蹤：

- 自動追蹤 HTTP 請求
- 追蹤資料庫查詢
- 追蹤 Redis 操作
- 追蹤 AWS Kinesis 操作
- 自定義 span 用於業務邏輯追蹤

## 部署

### Docker 部署

```bash
# 建置映像
docker build -t fat-notification-cat:latest .

# 執行容器
docker run -p 8080:8080 --env-file .env fat-notification-cat:latest web
```

### Kubernetes 部署

專案包含 Helm charts 用於 Kubernetes 部署：

```bash
# 安裝 Helm chart
helm install fat-notification-cat ./helm/fat-notification-cat

# 升級部署
helm upgrade fat-notification-cat ./helm/fat-notification-cat

# 查看部署狀態
kubectl get pods -l app=fat-notification-cat
```

## 故障排除

### 常見問題

1. **資料庫連線失敗**
   - 確認 MySQL 服務正在運行
   - 檢查資料庫連線配置
   - 確認防火牆規則

2. **Redis 連線問題**
   - 確認 Redis 服務正在運行
   - 檢查 Redis 密碼配置
   - 確認網路連通性

3. **AWS Kinesis 錯誤**
   - 驗證 AWS 憑證
   - 確認 Kinesis stream 存在
   - 檢查 IAM 權限

4. **Wire 產生錯誤**
   - 確保已安裝最新版 Wire
   - 檢查 provider 函數簽名
   - 確認所有依賴都已正確註冊

## 貢獻指南

1. Fork 專案
2. 建立功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交變更 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 開啟 Merge Request

### 程式碼規範

- 遵循 [Effective Go](https://golang.org/doc/effective_go.html) 指南
- 使用 `gofmt` 格式化程式碼
- 撰寫單元測試覆蓋核心邏輯
- 保持函數簡潔，單一職責
- 使用有意義的變數和函數名稱

## 授權

本專案採用專有授權。詳情請聯繫專案維護者。

## 專案狀態

專案目前處於積極開發階段。歡迎提出問題和建議。

## 聯絡資訊

- 專案維護者：FatCat Team
- GitLab：https://gitlab.jvdtech.dev/fatcat/fat_notification_cat
- 問題追蹤：https://gitlab.jvdtech.dev/fatcat/fat_notification_cat/-/issues