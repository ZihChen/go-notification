# SSE 通知系統 - 檔案結構規劃

**返回**: [總覽文檔](overview.md)

---

## 📋 文檔說明

本文檔整理了 SSE 實時推播通知系統的完整檔案結構規劃，涵蓋四個主要架構層：

1. **Domain Layer** (內部核心層) - 實體、常數、介面定義
2. **Application Layer** (應用服務層) - DTO、UseCase 實作
3. **Adapter Layer** (適配器層) - Handler、Repository、SSE Manager
4. **Infrastructure Layer** (基礎設施層) - Database Migration、Router、Wire

---

## 1. Domain Layer (內部核心層)

### 核心文件清單

| 檔案路徑 | 職責 | 狀態 |
|---------|------|------|
| `internal/domain/entity/sse_notification.go` | SSE 實時推播通知實體定義 (簡化版) | ✅ 已完成 |
| `internal/domain/consts/sse_notification.go` | SSE 通知相關常數定義 | ✅ 已完成 |
| `internal/domain/ports/inbound/sse_notification.go` | SSE 通知使用案例介面定義 | ✅ 已完成 |
| `internal/domain/ports/outbound/service/sse_manager.go` | SSE 連接管理器服務介面定義 | ✅ 已完成 |

**Phase 1 完成日期**: 2026-02-03
**Commit**: `708eaa8` - feat(sse): implement Phase 1 Domain Layer for SSE notification system

### 關鍵設計決策

**架構簡化 (Plan A)**:
- ❌ **移除 SSENotificationRepository**: 不使用 MySQL 持久化
- ❌ **移除 NotificationStats Entity**: 統計改用 Prometheus Metrics
- ✅ **保留 SSEManager Port**: 處理連接管理與訊息推送
- ✅ **保留核心 UseCase 介面**: BroadcastNotification、SendNotification、StreamNotifications

**核心 Entity: SSENotification**
```go
type SSENotification struct {
    ID        string    `json:"id"`         // UUID v4
    Title     string    `json:"title"`      // 標題 (最多 100 字元)
    Message   string    `json:"message"`    // 訊息內容 (最多 500 字元)
    Type      string    `json:"type"`       // activity|promotion|system|announcement
    Priority  string    `json:"priority"`   // low|medium|high|critical
    IconURL   *string   `json:"icon_url,omitempty"`
    ActionURL *string   `json:"action_url,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

**核心常數定義**:
- 通知類型: activity, promotion, system, announcement
- 優先級: low, medium, high, critical
- TTL: 離線訊息保留 7 天 (Redis Streams)
- 限制: 單次推送最多 1000 位玩家
- Redis Keys: 玩家路由表、線上計數器、離線訊息前綴
- Redis Channels: 廣播頻道、Pod 專屬頻道

---

## 2. Application Layer (應用服務層)

### 核心文件清單

| 檔案路徑 | 職責 | 狀態 |
|---------|------|------|
| `internal/application/dto/sse_notification.go` | SSE 通知相關 DTO (簡化版) | 📋 待建立 |
| `internal/application/usecase/sse_notification/sse_notification_usecase.go` | SSE 通知使用案例實作 | 📋 待建立 |

### 核心 DTO

**後端推播 API**:
- `BroadcastRequest` - 廣播推送請求
- `BroadcastResponse` - 廣播推送回應
- `SendNotificationRequest` - 個別推送請求
- `SendNotificationResponse` - 個別推送回應

**架構簡化**:
- ❌ 移除 Visibility、Metadata 欄位
- ❌ 移除 NotificationStatsResponse (統計改用 Prometheus)
- ❌ 移除 NotificationHistoryResponse (不提供歷史查詢)
- ✅ 保留核心推送 DTO，簡化欄位

### UseCase 實作

**依賴注入**:
```go
type sseNotificationUseCase struct {
    sseManager   service.SSEManager      // SSE 連接管理與訊息推送
    eventService service.EventService    // KDS 審計日誌
    logger       infrastructure.Logger
}
```

**核心方法**:
1. `BroadcastNotification()` - 廣播推送給所有線上玩家
   - 創建通知實體 (純記憶體對象，不持久化)
   - 透過 SSEManager 廣播到所有 Pod
   - 記錄審計日誌到 KDS
   - 返回推送統計

2. `SendNotification()` - 個別推送給特定玩家清單
   - 驗證玩家 ID 數量 (最多 1000 位)
   - 批次推送給目標玩家
   - 記錄審計日誌到 KDS
   - 返回推送統計

3. `StreamNotifications()` - SSE 長連接
   - 註冊玩家連接
   - 推送離線訊息
   - 保持連接直到 context 取消

---

## 3. Adapter Layer (適配器層)

### 核心文件清單

| 檔案路徑 | 職責 | 狀態 |
|---------|------|------|
| `internal/adapter/inbound/handler/api/sse_notification_handler.go` | SSE 通知 HTTP 處理器 | 📋 待建立 |
| `internal/adapter/inbound/handler/api/gin_sse_writer.go` | Gin SSE Writer 實作 | 📋 待建立 |
| `internal/adapter/outbound/service/sse_manager.go` | SSE 連接管理器實作 ⭐ | 📋 待建立 |
| `internal/adapter/outbound/service/sse_pubsub_listener.go` | Redis Pub/Sub 監聽器 ⭐ | 📋 待建立 |

### 關鍵組件

#### SSENotificationHandler
**後端推播 API** (API Key 認證):
- `POST /api/v1/admin/notifications/broadcast` - 廣播推送
- `POST /api/v1/admin/notifications/send` - 個別推送

**前端接收 API** (JWT 認證):
- `GET /api/v1/notifications/stream` - SSE 長連接串流

#### GinSSEWriter
實作 `inbound.SSEWriter` 介面:
```go
type GinSSEWriter struct {
    writer  gin.ResponseWriter
    flusher http.Flusher
    ctx     *gin.Context
}
```

#### SSEManager ⭐ 核心組件
**職責**:
- 管理本地 SSE 連接池 (`map[string]inbound.SSEWriter`)
- 處理 Redis Pub/Sub 訊息路由
- 管理玩家路由表 (Redis Hash)
- 處理離線訊息 (Redis Streams)

**連接管理**:
- `RegisterConnection()` - 註冊玩家連接，更新路由表
- `UnregisterConnection()` - 取消註冊，移除路由
- `IsPlayerOnline()` - 檢查玩家是否在線
- `GetOnlinePlayerCount()` - 獲取線上玩家數量

**訊息推送**:
- `BroadcastToAll()` - 廣播到所有本地連接
- `SendToPlayer()` - 推送給單一玩家
- `SendToPlayers()` - 批次推送給多位玩家

**離線訊息**:
- `EnqueueOfflineMessage()` - 加入離線佇列 (Redis Streams, 7天 TTL)
- `GetOfflineMessages()` - 獲取玩家離線訊息
- `ClearOfflineMessages()` - 清除已推送的離線訊息

#### PubSubListener ⭐ 訊息路由核心
**職責**:
- 訂閱 Redis Pub/Sub 頻道
  - `sse:broadcast` - 廣播頻道 (所有 Pod 訂閱)
  - `sse:pod:{podID}` - Pod 專屬頻道
- 接收訊息並路由到本地連接池
- 處理跨 Pod 訊息轉發

**訊息流向**:
```
接收請求的 Pod (Pod-1)
    ↓
查詢路由表 (Redis Hash: sse:player_routes)
    ↓
發布到對應頻道:
  - 廣播: sse:broadcast → 所有 Pod
  - 個別: sse:pod:2 → Pod-2
    ↓
目標 Pod 接收訊息
    ↓
推送到本地連接池的玩家
```

---

## 4. Infrastructure Layer (基礎設施層)

### 核心文件清單

| 類型 | 檔案路徑 | 職責 | 狀態 |
|------|---------|------|------|
| **Migration** | `migrations/XXXXXX_create_redis_config.sql` | Redis 配置 (僅限配置) | 📋 待建立 |
| **Service Entry** | `cmd/sse/sse.go` | ✅ **SSE 獨立服務入口** (新增) | ✅ 已完成 |
| **Router** | `internal/adapter/inbound/router/sse_router.go` | ✅ **SSE 專用路由管理器** (新增) | ✅ 已完成 |
| **Router** | `internal/adapter/inbound/router/api_router.go` | 業務 API 路由 (移除 SSE) | ✅ 已修改 |
| **Wire** | `internal/di/wire.go` | 依賴注入配置 (分離 Web/SSE Components) | ✅ 已修改 |
| **Config** | `config/config.go` | 新增 SSE Pod ID 配置 | 📋 待修改 |
| **Main** | `main.go` | 新增 SSE service 匯入 | ✅ 已修改 |

### 獨立服務架構 ⭐ **重要變更**

**完成日期**: 2026-02-04
**Commit**: `ee34e77` - feat(sse): refactor to independent SSE Service Pod architecture

#### cmd/sse/sse.go - SSE 獨立服務入口 (273 行)

完全獨立的 SSE Service 進程，與 Web Service 分離部署。

**核心特性**:
```go
// 獨立的服務配置
const (
    defaultPort     = 8081  // 與 Web Service (8080) 不同端口
    shutdownTimeout = 5 * time.Second
)

// 長連接優化配置
server := &http.Server{
    Addr:         fmt.Sprintf(":%d", serverPort),
    Handler:      router,
    ReadTimeout:  5 * time.Minute,   // 延長讀取超時
    WriteTimeout: 5 * time.Minute,   // 延長寫入超時
    IdleTimeout:  10 * time.Minute,  // 延長空閒超時
}
```

**依賴注入**:
- 使用 `di.InitializeSSEComponents()` 而非 `InitializeWebComponents()`
- 包含 SSEHandler 和 Metrics
- 不包含業務 API 相關組件

**啟動方式**:
```bash
go run main.go sse --port 8081
```

#### internal/adapter/inbound/router/sse_router.go - SSE 專用路由管理器 (104 行)

專門管理 SSE 相關路由的獨立路由器。

**結構定義**:
```go
type SSERouterManager struct {
    sseHandler *api.SSENotificationHandler
    metrics    *metrics.Metrics
}
```

**註冊的路由**:
```go
// Admin API (API Key 認證)
POST /api/v1/admin/notifications/broadcast
POST /api/v1/admin/notifications/send

// Player API (JWT 認證)
GET /api/v1/notifications/stream

// 健康檢查 (無認證)
GET /health
GET /readyz
```

**中間件配置**:
- 獨立的 CORS 配置
- 獨立的 Metrics 收集
- 獨立的錯誤處理
- 獨立的 API Key 認證
- 獨立的 JWT 認證

### Database Migration (簡化版)

**⚠️ 架構變更**: 完全移除 MySQL 訊息持久化

原計劃的兩個資料表：
- ❌ `sse_notifications` - 訊息表 (已移除)
- ❌ `sse_notification_stats` - 統計表 (已移除)

**替代方案**:
- 訊息儲存: Redis Streams (7天 TTL)
- 統計資訊: Prometheus Metrics
- 路由管理: Redis Hash (sse:player_routes)
- 線上計數: Redis Counter (sse:online_count)

### Router 註冊

在 `api_router.go` 中新增:
```go
// 後端推播 API (使用 API Key 認證)
admin := rm.router.Group("/api/v1/admin/notifications")
admin.Use(apiKeyAuth)
{
    admin.POST("/broadcast", handler.BroadcastNotification)
    admin.POST("/send", handler.SendNotification)
}

// 前端接收 API (使用 JWT 認證)
player := rm.router.Group("/api/v1/notifications")
player.Use(jwtAuth)
{
    player.GET("/stream", handler.StreamNotifications)
}
```

### Wire 依賴注入 ⭐ **架構分離**

#### 分離的組件結構

**WebComponents** - Web Service 專用:
```go
type WebComponents struct {
    HTTPHandler  *api.HTTPHandler   // 業務 API Handler
    AgentHandler *api.AgentHandler  // 代理 API Handler
    Metrics      *metrics.Metrics   // Metrics 服務
}

// 初始化函數
func InitializeWebComponents(
    cfg *config.Config,
    logger infrastructure.Logger,
    redisManager *redis.Manager,
    db *gorm.DB,
) (*WebComponents, error)
```

**SSEComponents** - SSE Service 專用:
```go
type SSEComponents struct {
    SSEHandler *api.SSENotificationHandler  // SSE Handler
    Metrics    *metrics.Metrics              // Metrics 服務
}

// 初始化函數
func InitializeSSEComponents(
    cfg *config.Config,
    logger infrastructure.Logger,
    redisManager *redis.Manager,
    db *gorm.DB,
) (*SSEComponents, error)
```

#### SSE 相關 Provider 函數

```go
// SSE Manager (需注入 Pod ID)
func provideSSEManager(
    podID string,                    // ⭐ 從環境變數注入
    cacheManager cache.CacheManager,
    logger infrastructure.Logger,
) service.SSEManager

// SSE PubSub Listener
func providePubSubListener(
    sseManager service.SSEManager,
    cacheManager cache.CacheManager,
    logger infrastructure.Logger,
) *service.PubSubListener

// SSE Notification UseCase
func provideSSENotificationUseCase(
    sseManager service.SSEManager,
    eventService service.EventService,
    logger infrastructure.Logger,
) inbound.SSENotificationUseCase

// SSE Notification Handler
func provideSSENotificationHandler(
    useCase inbound.SSENotificationUseCase,
    logger infrastructure.Logger,
) *api.SSENotificationHandler
```

#### 依賴鏈對比

**Web Service 依賴鏈**:
```
Config → Logger → RedisManager → DB
  ↓
TracingService, LockManager, MetricsService, ...
  ↓
Repositories, UseCases
  ↓
HTTPHandler, AgentHandler
  ↓
WebComponents
```

**SSE Service 依賴鏈**:
```
Config → Logger → RedisManager → DB
  ↓
TracingService, LockManager, MetricsService, KDSService, ...
  ↓
QueueService, EventService
  ↓
SSEManager, PubSubListener
  ↓
SSENotificationUseCase
  ↓
SSENotificationHandler
  ↓
SSEComponents
```

### 環境變數配置

新增 SSE Pod 配置:
```yaml
SSE_POD_ID: ${POD_NAME}  # Kubernetes 自動注入 Pod 名稱
REDIS_ADDR: redis-service:6379
REDIS_PASSWORD: ""
REDIS_DB: 0
```

---

## 📊 實作順序建議

基於 Clean Architecture 依賴方向，建議實作順序：

### Phase 1: Domain Layer (第1-2天)
1. ✅ Entity: `sse_notification.go`
2. ✅ Constants: `sse_notification.go`
3. ✅ Inbound Ports: `sse_notification.go`
4. ✅ Outbound Ports: `sse_manager.go`

### Phase 2: Application Layer (第3天)
1. ✅ DTOs: `sse_notification.go`
2. ✅ UseCase: `sse_notification_usecase.go`
3. ✅ UseCase 單元測試

### Phase 3: Adapter Layer (第4-6天)
1. ✅ HTTP Handler: `sse_notification_handler.go`
2. ✅ SSE Writer: `gin_sse_writer.go`
3. ✅ SSE Manager: `sse_manager.go` ⭐ 核心
4. ✅ PubSub Listener: `sse_pubsub_listener.go` ⭐ 核心
5. ✅ JWT Middleware
6. ✅ 優雅關閉機制
7. ✅ 單元測試

### Phase 4: Infrastructure Layer (第7天)
1. ✅ Redis 配置與測試
2. ✅ Router 註冊
3. ✅ Wire 依賴注入
4. ✅ Config 環境變數
5. ✅ KDS 審計日誌整合

---

## 🔗 相關文檔

- [總覽文檔](overview.md) - 回到主文檔
- [Phase 1: Domain Layer 實作](phase-1.md)
- [Phase 2: Application Layer 實作](phase-2.md)
- [Phase 3: Adapter Layer 實作](phase-3.md)
- [Phase 4: Infrastructure Layer 實作](phase-4.md)
- [技術要點](technical-details.md)
- [最佳實踐](best-practices.md)

---

## 📝 詳細程式碼參考

本文檔提供架構總覽與設計決策。完整的程式碼範例請參考原始任務文檔：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第319-1746行)

**包含內容**:
- ✅ 完整的程式碼範例 (Domain/Application/Adapter/Infrastructure)
- ✅ GORM 資料庫 Schema (如需啟用 MySQL)
- ✅ Redis Streams 操作範例
- ✅ SSE Manager 完整實作
- ✅ PubSub Listener 完整實作
- ✅ Wire 依賴注入完整配置

---

## 5. 完整文件結構總覽

### 服務入口

```
cmd/
├── web/
│   └── web.go              ✅ Web Service 入口 (8080)
│       - 業務 API
│       - 代理 API
│       - 使用 WebComponents
│
└── sse/
    └── sse.go              ✅ SSE Service 入口 (8081) ⭐ 新增
        - SSE 推播 API
        - SSE 串流 API
        - 使用 SSEComponents
```

### Domain Layer

```
internal/domain/
├── entity/
│   └── sse_notification.go         ✅ SSE 通知實體
├── consts/
│   └── sse_notification.go         ✅ SSE 常數定義
└── ports/
    ├── inbound/
    │   └── sse_notification.go     ✅ SSE UseCase 介面
    └── outbound/
        └── service/
            └── sse_manager.go      ✅ SSE Manager 介面
```

### Application Layer

```
internal/application/
├── dto/
│   └── sse_notification.go         ✅ SSE DTO 定義
└── usecase/
    └── sse_notification/
        ├── sse_notification_usecase.go       ✅ UseCase 實作
        └── sse_notification_usecase_test.go  ✅ 單元測試
```

### Adapter Layer

```
internal/adapter/
├── inbound/
│   ├── handler/
│   │   └── api/
│   │       ├── sse_notification_handler.go   ✅ SSE HTTP Handler
│   │       └── gin_sse_writer.go             ✅ Gin SSE Writer
│   └── router/
│       ├── router_manager.go        ✅ Web Service 路由管理器 (修改)
│       ├── api_router.go            ✅ 業務 API 路由 (移除 SSE)
│       └── sse_router.go            ✅ SSE Service 路由管理器 ⭐ 新增
│
└── outbound/
    └── service/
        ├── sse_manager.go                    ✅ SSE Manager 實作
        └── sse_pubsub_listener.go            ✅ Pub/Sub 監聽器
```

### Infrastructure Layer

```
internal/
├── di/
│   ├── wire.go               ✅ Wire 配置 (分離 Web/SSE Components)
│   └── wire_gen.go           ✅ 自動生成
│
└── infrastructure/
    ├── config/
    │   └── config.go         📋 新增 SSE Pod ID 配置
    └── ...

main.go                       ✅ 主入口 (新增 SSE service 匯入)
```

### 架構變更摘要

| 變更類型 | 檔案 | 狀態 | 說明 |
|---------|------|------|------|
| **新增** | `cmd/sse/sse.go` | ✅ 完成 | SSE 獨立服務入口 |
| **新增** | `internal/adapter/inbound/router/sse_router.go` | ✅ 完成 | SSE 專用路由管理器 |
| **修改** | `internal/di/wire.go` | ✅ 完成 | 分離 Web/SSE Components |
| **修改** | `internal/di/wire_gen.go` | ✅ 完成 | Wire 自動生成更新 |
| **修改** | `internal/adapter/inbound/router/router_manager.go` | ✅ 完成 | 移除 SSE Handler |
| **修改** | `internal/adapter/inbound/router/api_router.go` | ✅ 完成 | 移除 SSE 路由 |
| **修改** | `cmd/web/web.go` | ✅ 完成 | 移除 SSE 相關初始化 |
| **修改** | `main.go` | ✅ 完成 | 新增 SSE service 匯入 |

---

**維護者**: Development Team
**最後更新**: 2026-02-04 (架構重構完成)
