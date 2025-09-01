# CLAUDE-QUICK.md

## 快速開發指南

### 當前狀態
- **v1.2**: 路由架構重構 ✅ 已完成 (2025-09-01)
- **v1.1**: 會員訊息排程發送系統 ✅ 已完成 (2025-08-28)
- **當前階段**: 架構優化與測試驗證 🔄 進行中
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
go test ./internal/adapter/usecase/message_campaign/...
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

#### 路由系統 ✨ **NEW**
- `internal/adapter/router/router_manager.go` - 路由管理器
- `internal/adapter/router/api_router.go` - API路由組件
- `internal/adapter/router/swagger_router.go` - Swagger路由組件
- `internal/adapter/router/health_router.go` - 健康檢查路由
- `internal/adapter/router/pprof_router.go` - 性能分析路由

#### 核心業務邏輯
- `internal/adapter/usecase/message_campaign/` - 訊息活動用例
- `internal/adapter/repository/message_campaign_repository.go` - 資料存取
- `internal/infrastructure/models/message_campaign.go` - 資料模型

#### API處理
- `internal/adapter/handler/api/http_handler.go` - HTTP端點處理器
- `internal/infrastructure/http/middleware/auth.go` - 認證中間件

#### 排程系統
- `internal/adapter/job/message_campaign_trigger_job.go` - 排程任務
- `internal/adapter/usecase/message_campaign/scheduler.go` - 排程邏輯

#### 配置檔案
- `internal/infrastructure/config/config.go` - 系統配置
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

### 路由架構新功能 ✨

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
**更新日期**: 2025-09-01  
**版本**: v1.2 (路由架構重構)  
**用途**: 日常開發快速參考