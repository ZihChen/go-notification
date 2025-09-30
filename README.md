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
   - 使用 Cron 表達式定時執行 Job
   - 支援訊息活動觸發和系統維護任務

### 領域實體

- **Merchant** - 商戶實體
- **Player** - 終端用戶/客戶  
- **Manager** - 管理員用戶
- **Message Campaign** - 通知活動和訊息推送
- **Tags/Levels** - 用戶分類和層級管理

###  Clean Architecture 分層

專案遵循六角架構，具有明確的關注點分離：

```
.
├── cmd/                 # 應用程式入口點
│   ├── root.go         # Cobra CLI 根命令
│   ├── web/            # Web API 服務
│   ├── consumer/       # Kinesis 事件消費者
│   ├── worker/         # 背景任務處理器
│   └── scheduler/      # 排程任務管理
├── docs/               # API 文件
│   ├── swagger.json    # Swagger JSON 規格
│   └── swagger.yaml    # Swagger YAML 規格
├── internal/
│   ├── domain/         # 核心業務邏輯、介面定義（Ports）
│   │   ├── entity/     # 領域實體
│   │   ├── event/      # 事件定義
│   │   ├── consts/     # 常數定義
│   │   ├── errmsg/     # 自定義錯誤類型
│   │   └── ports/      # 接口定義 (Ports)
│   │       ├── inbound/  # 入站接口（Use Case 接口）
│   │       └── outbound/ # 出站接口（Repository、Service 等）
│   │           ├── infrastructure/ # 基礎設施接口
│   │           ├── job/           # 排程任務接口
│   │           ├── repository/    # Repository 接口
│   │           └── service/       # 外部服務接口
│   ├── application/    # 應用層（Use Cases、Services）
│   │   ├── dto/        # 資料傳輸物件
│   │   ├── service/    # 應用服務
│   │   └── usecase/    # 業務用例實作
│   │       ├── level/      # 等級管理用例
│   │       ├── manager/    # 管理員管理用例
│   │       ├── merchant/   # 商戶管理用例
│   │       ├── message/    # 訊息活動用例
│   │       ├── player/     # 玩家管理用例
│   │       └── testutil/   # 測試工具
│   ├── adapter/        # 適配器層（Adapters）
│   │   ├── inbound/    # 入站適配器
│   │   │   ├── handler/    # HTTP/Worker/Scheduler 處理器
│   │   │   │   ├── api/        # HTTP API 處理器
│   │   │   │   ├── scheduler/  # 排程處理器
│   │   │   │   └── worker/     # 工作者處理器
│   │   │   ├── job/        # 排程任務實作
│   │   │   ├── middleware/ # HTTP 中間件
│   │   │   └── router/     # 路由管理器
│   │   └── outbound/   # 出站適配器
│   │       └── repository/ # 資料庫操作實作
│   │           ├── manager/    # 管理員資料庫
│   │           ├── merchant/   # 商戶資料庫
│   │           ├── message/    # 訊息資料庫
│   │           └── player/     # 玩家資料庫
│   ├── infrastructure/ # 基礎設施層
│   │   ├── cache/      # 快取管理
│   │   │   └── redis/  # Redis 實作
│   │   ├── config/     # 配置管理
│   │   ├── database/   # 資料庫連線管理
│   │   │   └── mysql/  # MySQL 實作
│   │   ├── kds/        # AWS Kinesis 整合
│   │   ├── logger/     # 日誌服務
│   │   ├── models/     # 資料庫模型
│   │   ├── queue/      # Asynq 任務佇列
│   │   ├── tracing/    # OpenTelemetry 追蹤
│   │   └── utils/      # 工具類
│   │       └── response/   # HTTP 響應工具
│   └── di/             # 依賴注入（Wire）
│       ├── wire.go     # Wire 配置
│       └── wire_gen.go # Wire 產生的程式碼
├── migrations/         # 資料庫遷移檔案
├── test/              # 測試基礎設施
│   ├── mocks/         # 統一Mock框架
│   │   ├── base_mock.go       # BaseMock模式基礎
│   │   ├── repository_mocks.go # Repository層Mock
│   │   └── service_mocks.go    # Service層Mock
│   ├── factories/     # 測試數據工廠
│   │   ├── test_data_factory.go # 主要數據工廠
│   │   └── edge_case_factory.go # 邊界條件數據工廠
│   └── helper/        # 測試輔助工具
│       ├── logger_mock.go      # Logger Mock
│       └── test_utils.go       # 測試工具函數
└── helm/              # Kubernetes Helm Charts
    └── templates/     # K8s 資源模板
```

### 依賴注入

使用 Google Wire 進行編譯時期依賴注入 (`internal/di/wire.go`)

## 快速開始

### 環境需求

- Go 1.23+
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

專案採用統一的測試架構，包含Mock框架和測試數據工廠：

```bash
# 執行所有測試
go test ./...

# 執行測試並產生覆蓋率報告
go test -cover ./...

# 執行特定層級測試
go test -v ./internal/application/usecase/...  # Use Case 層測試
go test -v ./internal/adapter/repository/...   # Repository 層測試

# 執行測試並產生 HTML 覆蓋率報告
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# 檢查程式碼品質（linter）
go vet ./...
```

### 程式碼品質

```bash
# 執行 linter
golangci-lint run --fix
```

### 資料庫遷移

專案支援兩種配置方式：

#### 本地開發
本地開發使用 `.env` 文件配置，migrate.sh 會自動偵測並載入：

```bash
# 產生新的遷移檔案
./migrate.sh gen <migration_name> 

# 執行遷移 (自動使用 .env 配置)
./migrate.sh apply

# 查看遷移狀態
./migrate.sh status
```

#### Kubernetes 部署
部署環境使用 ConfigMap 和 Secrets 提供環境變數：

```bash
# 在 Kubernetes Pod 中執行遷移
kubectl exec -it <pod-name> -- ./migrate.sh apply

# 或在部署時通過 Init Container 執行
```

#### 配置管理

- **本地開發**：`.env` 文件
- **部署環境**：`helm/templates/configmap.yaml` + Kubernetes Secrets

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

- **資料庫連線池配置**（MySQL）
  - `DB_MAX_IDLE`
  - `DB_MAX_OPEN`
  - `DB_MAX_LIFETIME`
  - `DB_MAX_IDLE_TIME`

- **Redis 連線**
  - `REDIS_DOMAIN`
  - `REDIS_PORT`
  - `REDIS_PWD`
  - `REDIS_DB`
  - 
- **Redis Pool 設置**
  - `REDIS_POOL_SIZE`
  - `REDIS_MIN_IDLE_CONNS`
  - `REDIS_MAX_RETRIES`
  - `REDIS_DIAL_TIMEOUT`
  - `REDIS_READ_TIMEOUT`
  - `REDIS_WRITE_TIMEOUT`
  - `REDIS_POOL_TIMEOUT`
  - `REDIS_IDLE_TIMEOUT`
  - `REDIS_MAX_CONN_AGE`

- **AWS 設定**
  - `AWS_REGION`
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `KINESIS_STREAM_NAME`
  - `KINESIS_STREAM_ARN`
  - `DYNAMODB_TABLE`
  - `DYNAMODB_PARTITION_KEY`

- **OpenTelemetry 追蹤**
  - `OTEL_EXPORTER_OTLP_ENDPOINT`
  - `OTEL_SERVICE_NAME`

- **服務端口**
  - `WEB_PORT`（預設：8080）
  - `WORKER_CONCURRENCY`
  - `SCHEDULER_INTERVAL`

- **Middleware 中間件驗證**
  - `AUTH_ENABLED`
  - `AUTH_API_KEYS`
  - `AUTH_HEADER_KEY`
  
## 設計模式

### Repository Pattern
所有資料庫操作都通過定義在 `internal/domain/ports/outbound/repository/` 的 repository 介面進行

### Use Case Pattern
業務邏輯封裝在 use case 中，協調 repositories 和 services

### Job Pattern
所有排程任務實作 `ScheduledJob` 介面，統一在 `internal/adapter/inbound/job/` 管理
- Job Registry 模式進行集中管理
- 使用 Wire 進行依賴注入
- 編譯時檢查確保介面實作

### Event-Driven Architecture
系統使用事件進行服務間通訊，通過 KDS 和 Redis 佇列實現

### Testing Architecture Pattern ✨ **NEW**
統一的測試基礎設施，提升測試品質與維護性：
- **統一Mock框架** - 所有Repository Mock使用一致的BaseMock模式
- **測試數據工廠** - 使用Builder模式創建測試實體
- **邊界條件測試** - 全面的邊界條件場景和錯誤模擬
- **Context洩漏防護** - 適當的context管理預防資源洩漏
- **集中化Mock管理** - 統一的Mock定義與管理

### Error Handling
在 `internal/domain/errmsg/` 定義自定義錯誤類型，確保整個應用程式的錯誤處理一致性

### Hexagonal Architecture Pattern
遵循六角架構原則，將適配器分為：
- **Inbound Adapters** (`internal/adapter/inbound/`) - 處理外部請求（HTTP、Jobs、Events）
- **Outbound Adapters** (`internal/adapter/outbound/`) - 處理外部依賴（資料庫、緩存、第三方服務）

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

## 最新功能更新

### v1.9 玩家訊息API系統 ✨ **NEW** (2025-09-30)

- **前台玩家訊息管理**
  - 實現玩家訊息列表查詢API，支援分頁（預設20筆/頁）
  - 玩家訊息已讀標記功能，提供即時狀態更新
  - 訊息統計功能：未讀數量、已讀數量、總數量統計
  - 訊息摘要自動生成，支援HTML轉義防護（300字符限制）

- **DDD架構完善**
  - PlayerMessageAggregate 移至 domain/aggregate 目錄
  - 跨Aggregate查詢結果的正確domain層實現
  - Repository模式JOIN查詢優化，提升查詢效率
  - Clean Architecture嚴格遵循，達到enterprise-grade標準

- **性能優化實現**
  - 消除N+1查詢問題，使用LEFT JOIN優化數據庫操作
  - 直接ID查詢取代低效映射查詢，性能提升顯著
  - 查詢邏輯從多次分離查詢改為單次JOIN查詢
  - 測試架構統一，UseCase和Handler層測試100%通過

### v1.8 App推播功能實作 (2025-09-22)

- **多渠道通知系統**
  - 新增App推播功能，支援位元遮罩管理（1=站內信, 2=App推播, 4=其他）
  - 整合第三方推播服務，實現統一通知介面
  - merchant_push_api_keys表管理商戶API金鑰
  - message_campaigns表新增app_content和notification_types欄位

### v1.5+ 資料庫遷移系統強化 (2025-09-16)

- **遷移系統強化**
  - 實現 DSN 驗證和確認機制，提升遷移程序的可靠性
  - 遷移系統程式碼品質改進，全面優化錯誤處理機制
  - 增強資料庫遷移參數處理和資料查詢邏輯
  - 建立穩健的遷移系統架構，提升系統穩定性

- **資料搬遷系統規劃**
  - 開始設計 v1.6 DB資料搬遷系統
  - 大量歷史資料搬遷的性能優化方案
  - UpdateOrCreate 模式的可靠資料同步機制
  - 全面的欄位映射和資料轉換邏輯

### v1.5 系統優化與資料架構統一 (2025-09-15)

- **訊息活動物件類型統一**
  - 實現訊息活動物件類型統一架構
  - 優化資料庫欄位類型，提升系統穩定性
  - 消除硬編碼數值，改用常數值提升維護性
  - 標準化物件類型處理流程

- **測試架構持續優化**
  - 完善測試資料工廠功能
  - 提升測試可維護性和擴展性
  - 標準化測試數據創建流程
  - 完善錯誤場景模擬能力

### v1.4 測試架構統一完成 ✅ (2025-09-11)

- **統一Mock架構實現**
  - 採用BaseMock模式的一致性Mock框架
  - 集中化Repository Mock管理 (`test/mocks/repository_mocks.go`)
  - Service層Mock標準化
  - Logger Mock統一化 (`helper.NewMockLogger()`)

- **測試數據工廠建立**
  - Builder模式的測試數據創建 (`test/factories/`)
  - 邊界條件測試基礎設施
  - 高覆蓋率的Use Case測試
  - Context洩漏修復與預防

- **測試品質提升**
  - 所有Use Case層測試遷移至統一框架
  - 全面的邊界條件和錯誤場景測試
  - 一致的測試結構和斷言模式
  - Linter問題修復（govet context leak）

### v1.3 架構重構完成 ✅

- **六角架構實現**
  - 完整的 Ports & Adapters 模式
  - Inbound/Outbound Adapters 分離
  - 清晰的關注點分離與依賴方向

- **應用層重組**
  - DTO 搬遷至 Application 層
  - 業務服務與 Use Cases 的清晰分界
  - 測試工具的統一管理

- **基礎設施優化**
  - HTTP Response 工具模組化
  - Repository 按業務邏輯分組
  - Event Service 架構優化

### v1.2 路由架構重構 ✅

- **模組化路由管理系統**
  - 全新的路由管理器（Router Manager）架構
  - 分離的路由組件：API、Swagger、健康檢查、pprof 性能分析
  - 各組件獨立的中間件配置，避免相互影響

- **改進的中間件管理**
  - 優化 CORS 配置，解決 Swagger API 呼叫問題
  - 統一的中間件配置管理
  - 支援開發和生產環境的不同配置策略

### v1.1 會員訊息排程發送系統 ✅

- **新增會員訊息活動管理 API**
  - 支援創建、修改、查詢會員訊息活動
  - 整合商戶(Merchant)關聯功能
  - 軟刪除功能與狀態管理

- **排程發送機制**
  - 使用 Scheduler Service 進行定時觸發
  - 支援高併發處理，目標 3 秒完成 10 萬筆資料發送
  - 自動記錄實際發送數量至 `real_sent_count`

- **API 認證系統**
  - 新增 API 認證中間件
  - 支援商戶映射與權限控制
  - 統一錯誤響應格式

### 技術改進

- **架構重組（v1.3）**
  - 完整的六角架構實現
  - Inbound/Outbound Adapters 分離
  - Application 層與 Domain 層的清晰分界
  - Repository 按業務邏輯分組管理

- **工具與公用組件**
  - HTTP Response 工具模組化至 utils 目錄
  - DTO 搬遷至 Application 層
  - Event Service 架構優化

- **路由架構重構（v1.2）**
  - 模組化路由管理器設計
  - 獨立的路由組件與中間件配置
  - 支援性能分析和調試工具

- **品質與可維護性**
  - 統一 HTTP 響應格式
  - 改善錯誤處理機制
  - 優化 CORS 和 API 調用體驗

## 專案狀態

專案目前處於高度成熟的開發階段，最新完成了**v1.9 玩家訊息API系統**（2025-09-30），實現了前台玩家訊息管理的完整功能，包括訊息列表查詢、已讀標記和統計功能，達到production-ready標準。同時完善了DDD架構設計，將PlayerMessageAggregate正確分離至domain層，並實現了顯著的性能優化，消除了N+1查詢問題。

此前完成了App推播功能實作（v1.8）、資料庫遷移系統強化（v1.5+）、系統優化與資料架構統一（v1.5）、測試架構統一（v1.4）、六角架構重構（v1.3）、路由架構重構（v1.2）和會員訊息排程發送系統（v1.1）。

系統現已具備完整的訊息管理能力，包括：
- ✅ 後台訊息活動管理（管理員使用）
- ✅ 多渠道推送系統（站內信+App推播）
- ✅ 前台訊息查詢系統（玩家使用）
- ✅ 企業級架構品質與性能優化
- ✅ 完整的測試覆蓋與CI/CD支援

目前專注於生產環境部署準備、系統監控優化，並開始規劃v2.0的進階功能開發。

## 聯絡資訊

- 專案維護者：FatCat Team
- GitLab：https://gitlab.jvdtech.dev/fatcat/fat_notification_cat
- 問題追蹤：https://gitlab.jvdtech.dev/fatcat/fat_notification_cat/-/issues