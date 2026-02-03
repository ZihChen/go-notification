# Phase 1: Domain Layer 實作

**時程**: 第1-2天  
**返回**: [總覽文檔](overview.md) | [檔案結構](file-structure.md)

---

## 📋 Phase 目標

建立 SSE 通知系統的 Domain Layer，包含實體定義、常數定義、Inbound Ports (UseCase 介面) 和 Outbound Ports (Repository/Service 介面)。

**核心原則**: Domain Layer 不依賴任何外層，保持純淨的業務邏輯。

---

## ✅ 任務清單

### 任務 1.1: 建立 Domain Entities
- [ ] 建立 `internal/domain/entity/sse_notification.go`
  - SSENotification 實體定義 (簡化版，無 MySQL Tag)
  - 業務方法實現: `IsExpired()`

**簡化說明**:
- ❌ 移除 PlayerID: 不再儲存玩家 ID，僅在推送時動態路由
- ❌ 移除 Visibility/IsRead: 不提供歷史查詢，無需這些欄位
- ❌ 移除 Metadata: 簡化設計
- ❌ 移除 GORM Tags: 不使用 MySQL 持久化
- ✅ 保留核心欄位: ID, Title, Message, Type, Priority, URLs, 時間戳

### 任務 1.2: 建立 Domain Constants
- [ ] 建立 `internal/domain/consts/sse_notification.go`
  - 通知類型常數: activity, promotion, system, announcement
  - 優先級常數: low, medium, high, critical
  - 推送狀態常數: queued, delivered, failed
  - TTL 和限制常數
  - Redis Keys 常數 (含 Pub/Sub Channel)

**新增 Redis Pub/Sub 常數**:
```go
const (
    RedisKeySSEPlayerRoutes  = "sse:player_routes"  // 玩家路由表
    RedisKeySSEOnlineCount   = "sse:online_count"   // 線上計數器
    RedisKeySSEOfflinePrefix = "sse:offline:"       // 離線訊息
    RedisChannelSSEBroadcast = "sse:broadcast"      // 廣播頻道
    RedisChannelSSEPodPrefix = "sse:pod:"           // Pod 頻道
)
```

### 任務 1.3: 定義 Inbound Ports
- [ ] 建立 `internal/domain/ports/inbound/sse_notification.go`
  - SSENotificationUseCase 介面 (簡化版)
  - SSEWriter 介面 (SSE 串流抽象)

**簡化 UseCase 介面**:
```go
type SSENotificationUseCase interface {
    // 後端推播 API
    BroadcastNotification(ctx context.Context, req *dto.BroadcastRequest) (*dto.BroadcastResponse, error)
    SendNotification(ctx context.Context, req *dto.SendNotificationRequest) (*dto.SendNotificationResponse, error)
    
    // 前端 SSE 串流
    StreamNotifications(ctx context.Context, playerID string, writer SSEWriter) error
}
```

**移除的方法**:
- ❌ GetNotificationStats: 統計改用 Prometheus
- ❌ GetNotificationHistory: 不提供歷史查詢
- ❌ MarkNotificationAsRead: 無歷史查詢功能
- ❌ DeleteNotification: 無歷史查詢功能

### 任務 1.4: 定義 Outbound Ports
- [ ] ~~建立 `internal/domain/ports/outbound/repository/sse_notification.go`~~ **已移除**
- [ ] 建立 `internal/domain/ports/outbound/service/sse_manager.go`
  - SSEManager 介面定義

**⚠️ 重大變更**: 完全移除 SSENotificationRepository

**原因**: 不使用 MySQL 持久化，改用 Redis Streams + SSEManager

**SSEManager 介面** (核心組件):
```go
type SSEManager interface {
    // 連接管理
    RegisterConnection(ctx context.Context, playerID string, writer inbound.SSEWriter) error
    UnregisterConnection(ctx context.Context, playerID string) error
    IsPlayerOnline(ctx context.Context, playerID string) (bool, error)
    GetOnlinePlayerCount(ctx context.Context) (int, error)
    GetOnlinePlayerIDs(ctx context.Context) ([]string, error)
    
    // 訊息推送
    BroadcastToAll(ctx context.Context, notification *entity.SSENotification) (int, error)
    SendToPlayer(ctx context.Context, playerID string, notification *entity.SSENotification) error
    SendToPlayers(ctx context.Context, playerIDs []string, notification *entity.SSENotification) (int, error)
    
    // 離線訊息 (Redis Streams)
    EnqueueOfflineMessage(ctx context.Context, playerID string, notification *entity.SSENotification) error
    GetOfflineMessages(ctx context.Context, playerID string) ([]*entity.SSENotification, error)
    ClearOfflineMessages(ctx context.Context, playerID string) error
}
```

---

## 📝 實作細節參考

完整的程式碼範例請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第319-547行)
- **檔案結構文檔**: [file-structure.md](file-structure.md) - Domain Layer 章節

---

## ✅ 驗收標準

- [ ] 所有 Domain Entity 定義完成，無外部依賴
- [ ] 所有常數定義完成，包含 Redis Pub/Sub Channel
- [ ] Inbound Ports 定義清晰，符合簡化設計
- [ ] Outbound Ports 定義完整，SSEManager 介面涵蓋所有核心功能
- [ ] 代碼編譯通過
- [ ] 遵循 Clean Architecture 分層原則

---

## 🔗 下一步

完成 Phase 1 後，前往 [Phase 2: Application Layer 實作](phase-2.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
