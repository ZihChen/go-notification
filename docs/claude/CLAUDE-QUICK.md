# CLAUDE-QUICK.md

## 快速開發指南

### 當前狀態
- **v1.3**: 六角架構重構 ✅ 已完成 (2025-09-02)
- **v1.2**: 路由架構重構 ✅ 已完成 (2025-09-01)
- **v1.1**: 會員訊息排程發送系統 ✅ 已完成 (2025-08-28)
- **當前階段**: 六角架構驗證與測試 🔄 進行中
- **下一里程碑**: 系統穩定性驗證與性能優化

### 快速命令

#### 開發環境
```bash
# 啟動所有服務
docker-compose up -d

# 啟動單一服務 (本地開發)
go run main.go web       # API服務 :8080
go run main.go worker    # 背景處理
go run main.go scheduler # 排程服務
go run main.go consumer  # KDS消費者

# 檢視Swagger API文檔
# http://localhost:8080/swagger/index.html
# 
# 檢視性能分析（pprof）
# http://localhost:8080/debug/pprof/
```

#### 測試命令
```bash
# 運行所有測試
go test ./...

# 運行覆蓋率測試
go test -cover ./...

# 運行特定模組測試
go test ./internal/application/usecase/message/...
go test ./internal/adapter/outbound/repository/message/...
```

#### 資料庫操作
```bash
# 應用遷移
./migrate.sh apply

# 檢查遷移狀態
./migrate.sh status

# 生成新遷移
./migrate.sh gen <migration_name>
```

#### 代碼生成
```bash
# 更新Swagger文檔
swag init

# 重新生成依賴注入
wire ./internal/di
```

### 重要檔案位置

#### 六角架構組織 ✨ **NEW v1.3**

##### Domain Layer (核心領域)
- `internal/domain/entity/` - 領域實體
- `internal/domain/ports/inbound/` - 入站介面 (Use Case 介面)
- `internal/domain/ports/outbound/` - 出站介面 (Repository, Service 介面)
- `internal/domain/consts/` - 領域常數
- `internal/domain/errmsg/` - 錯誤定義

##### Application Layer (應用層)
- `internal/application/dto/` - 資料傳輸物件
- `internal/application/service/` - 應用服務 (如 Event Service)
- `internal/application/usecase/` - 業務用例實作
  - `message/` - 訊息相關用例
  - `player/` - 玩家相關用例
  - `merchant/` - 商戶相關用例

##### Adapter Layer (適配器層)
**Inbound Adapters (入站適配器)**
- `internal/adapter/inbound/handler/api/` - HTTP API 處理器
- `internal/adapter/inbound/handler/worker/` - Worker 處理器
- `internal/adapter/inbound/handler/scheduler/` - Scheduler 處理器
- `internal/adapter/inbound/router/` - 路由管理系統
  - `router_manager.go` - 路由管理器
  - `api_router.go` - API路由組件
  - `swagger_router.go` - Swagger路由組件
  - `health_router.go` - 健康檢查路由
  - `pprof_router.go` - 性能分析路由
- `internal/adapter/inbound/middleware/` - HTTP 中間件
- `internal/adapter/inbound/job/` - 排程任務實作

**Outbound Adapters (出站適配器)**
- `internal/adapter/outbound/repository/` - 資料庫操作 (按業務分組)
  - `merchant/` - 商戶相關 Repository
  - `player/` - 玩家相關 Repository
  - `manager/` - 管理員相關 Repository
  - `message/` - 訊息相關 Repository

##### Infrastructure Layer (基礎設施層)
- `internal/infrastructure/database/` - 資料庫連接
- `internal/infrastructure/cache/` - 快取管理
- `internal/infrastructure/utils/httpresponse/` - HTTP 響應工具
- `internal/infrastructure/tracing/` - 分散式追蹤
- `internal/infrastructure/config/` - 系統配置

#### 配置檔案
- `internal/di/wire.go` - 依賴注入配置
- `docker-compose.yml` - 本地開發環境
- `CLAUDE.md` - 專案指引文件

### 文檔結構
```
docs/claude/
├── CLAUDE-QUICK.md      # 此文件 - 快速參考
├── CLAUDE-CURRENT.md    # 當前任務狀態
├── features/
│   └── message-campaign/
│       └── spec.md      # 功能規格文檔
└── archive/
    └── 2025-08/
        └── message-campaign-v1.1/
            └── CLAUDE-2025-08-28-v1.1-COMPLETED.md # 已完成功能歸檔
```

### API快速測試

#### 健康檢查
```bash
curl -X GET http://localhost:8080/health
```

#### 創建訊息活動
```bash
curl -X POST http://localhost:8080/api/v1/message-campaigns \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "測試活動",
    "content": "這是一個測試訊息",
    "scheduled_at": "2025-09-01T10:00:00Z"
  }'
```

#### 查詢活動列表
```bash
curl -X GET http://localhost:8080/api/v1/message-campaigns \
  -H "API-Key: YOUR_BASE64_ENCODED_API_KEY"
```

#### Swagger API測試
- 訪問: http://localhost:8080/swagger/index.html
- 點擊 "Authorize" 按鈕
- 輸入 base64 編碼的 API Key

### 除錯技巧

#### 日誌檢視
```bash
# 查看容器日誌
docker-compose logs -f web
docker-compose logs -f worker
docker-compose logs -f scheduler

# 查看Redis佇列
docker exec -it redis redis-cli
127.0.0.1:6379> KEYS *
127.0.0.1:6379> LLEN asynq:default
```

#### 性能分析 ✨ **NEW**
```bash
# CPU 性能分析
go tool pprof http://localhost:8080/debug/pprof/profile

# 記憶體使用分析
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

#### 資料庫檢查
```bash
# 連接MySQL
docker exec -it mysql mysql -u root -p

# 常用查詢
SELECT * FROM message_campaigns ORDER BY created_at DESC LIMIT 10;
SELECT * FROM player_messages WHERE message_campaign_id = 1;
```

### 常見問題

#### 編譯問題
```bash
# 清理模組快取
go clean -modcache
go mod tidy

# 重新生成wire
wire ./internal/di
```

#### 測試失敗
```bash
# 重置測試環境
docker-compose down
docker-compose up -d
sleep 10  # 等待服務啟動

# 重新運行測試
go test ./...
```

### 六角架構新功能 ✨ v1.3

#### Clean Architecture 實現
```go
// 使用 Ports & Adapters 模式
// Domain -> Application -> Adapters -> Infrastructure

// Use Case 介面定義在 domain/ports/inbound/
type MessageUseCase interface {
    CreateMessageCampaign(ctx context.Context, req *dto.CreateMessageCampaignRequest) error
    // ...
}

// Repository 介面定義在 domain/ports/outbound/
type MessageCampaignRepository interface {
    Create(ctx context.Context, campaign *entity.MessageCampaign) error
    // ...
}
```

#### Repository 組織優化
- **按業務邏輯分組**: merchant/, player/, manager/, message/
- **清晰的依賴方向**: Domain <- Application <- Adapters <- Infrastructure
- **Wire 依賴注入**: 使用 import aliases 解決命名衝突

#### 應用層重構
- **DTO 統一管理**: application/dto/ 包含所有資料傳輸物件
- **Use Cases 實作**: application/usecase/ 實現 domain ports
- **應用服務**: application/service/ 協調不同用例

### 路由架構功能 ✨ v1.2

#### 模組化路由管理
```go
// 使用新的路由管理器
routerManager := router.NewRouterManager(httpHandler)
routerManager.SetupRoutersWithMiddleware(ginEngine, config)
```

#### 獨立路由組件
- **API Router**: 處理業務邏輯路由，使用認證中間件
- **Swagger Router**: 處理API文檔，不使用認證中間件  
- **Health Router**: 處理健康檢查，不使用認證中間件
- **Pprof Router**: 處理性能分析，不使用認證中間件

#### CORS 配置優化
- 開發環境使用寬鬆的CORS設定
- 解決Swagger UI調用API的CORS問題
- 支援所有必要的HTTP方法和headers

---
**更新日期**: 2025-09-02  
**版本**: v1.3 (六角架構重構)  
**用途**: 日常開發快速參考