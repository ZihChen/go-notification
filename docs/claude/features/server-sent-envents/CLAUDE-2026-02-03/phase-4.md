# Phase 4: Infrastructure Layer 實作

**時程**: 第7天
**狀態**: 🚧 **進行中** (2026-02-04) - KDS 整合與 Wire 配置已完成
**Commit**: `f72cbe8` - feat(sse): integrate KDS event audit logging for SSE notifications
**返回**: [總覽文檔](overview.md) | [Phase 3](phase-3.md) | [Phase 5](phase-5.md)

---

## 📋 Phase 目標

完成基礎設施層配置，包含 Redis 配置、KDS 審計日誌整合、Wire 依賴注入 (含 Pod ID 注入)。

**⚠️ 簡化設計**: 移除 MySQL Migration，改用 Redis Streams

---

## ✅ 任務清單

### 任務 4.1: ~~Database Migration~~ (已移除)
**⚠️ 重大變更**: 完全移除 MySQL 訊息持久化

**原因**: 改用 Redis Streams 儲存離線訊息 (7天 TTL)

### 任務 4.2: Redis 配置與測試
- [ ] 確認 Redis Streams 支援 (Redis 5.0+)
- [ ] 確認 Redis Pub/Sub 支援
- [ ] 設定 Redis 連接池參數
- [ ] 測試 Pub/Sub 訂閱與發布
- [ ] 測試離線訊息 Streams (XADD/XRANGE/XDEL)
- [ ] 測試 Hash 路由表操作 (HSET/HGET/HDEL)
- [ ] 測試 Counter 操作 (INCR/DECR/GET)

**Redis Keys 設計**:
```
sse:player_routes     (Hash)    - 玩家路由表
sse:online_count      (String)  - 線上計數器
sse:offline:{playerID} (Stream) - 離線訊息
sse:broadcast         (Channel) - 廣播頻道
sse:pod:{podID}       (Channel) - Pod 專屬頻道
```

### 任務 4.3: KDS 審計日誌整合 ✅
- [x] 確認現有 EventService 可用
- [x] 定義 SSE 相關事件類型（新增至 `internal/domain/event/event.go`）:
  - `EventTypeSSENotificationBroadcast` - 廣播推送事件
  - `EventTypeSSENotificationSend` - 個別推送事件
  - `EventTypeSSEPlayerConnect` - 玩家連接事件
  - `EventTypeSSEPlayerDisconnect` - 玩家斷線事件
- [x] UseCase 整合事件發送邏輯
- [x] 新增 EventService Port 方法
- [x] 單元測試更新（含 EventService Mock）

**完成日期**: 2026-02-04
**變更文件**:
- `internal/domain/event/event.go` (新增 57 行)
- `internal/domain/ports/outbound/service/event.go` (新增 3 個方法)
- `internal/application/service/event_service.go` (新增 10 行)
- `internal/application/usecase/sse_notification/sse_notification_usecase.go` (整合事件發送)
- `internal/application/usecase/sse_notification/sse_notification_usecase_test.go` (測試更新)
- `internal/di/wire.go` (依賴注入更新)
- `internal/di/wire_gen.go` (自動生成)
- `internal/infrastructure/config/config.go` (配置支援)
- `test/mocks/service_mocks.go` (Mock 更新)

### 任務 4.4: Wire 依賴注入 ⭐ **含 EventService 注入** ✅
- [x] 更新 `internal/di/wire.go`
- [x] 新增 SSE Notification UseCase Provider（含 EventService 注入）
- [x] 執行 `wire ./internal/di` 生成 wire_gen.go
- [x] 驗證依賴注入編譯通過

**完成日期**: 2026-02-04

**已實作 Provider**:
```go
// SSE Notification UseCase Provider (含 EventService)
func provideSSENotificationUseCase(
    sseManager service.SSEManager,
    eventService service.EventService,  // ⭐ EventService 注入
    logger infrastructure.Logger,
) inbound.SSENotificationUseCase {
    return sse_notification.NewSSENotificationUseCase(sseManager, eventService, logger)
}
```

**待實作 Provider** (Phase 5):
- [ ] Pod ID Provider（從環境變數讀取）
- [ ] SSE Manager Provider（注入 Pod ID）
- [ ] PubSub Listener Provider
- [ ] SSE Notification Handler Provider

**新增 Provider 函數**:
```go
// Pod ID Provider (從環境變數讀取)
func providePodID() string {
    podID := os.Getenv("SSE_POD_ID")
    if podID == "" {
        podID = fmt.Sprintf("pod-%s", uuid.New().String()[:8])
    }
    return podID
}

// SSE Manager Provider (注入 Pod ID)
func provideSSEManager(
    podID string,                    // ⭐ Pod ID 注入
    redisClient *redis.Client,
    logger infrastructure.Logger,
) service.SSEManager {
    return service_impl.NewSSEManager(podID, redisClient, logger)
}

// PubSub Listener Provider
func providePubSubListener(
    sseManager service.SSEManager,
    redisClient *redis.Client,
    logger infrastructure.Logger,
) *service.PubSubListener {
    return service_impl.NewPubSubListener(sseManager, redisClient, logger)
}

// SSE Notification UseCase Provider
func provideSSENotificationUseCase(
    sseManager service.SSEManager,
    eventService service.EventService,
    logger infrastructure.Logger,
) inbound.SSENotificationUseCase {
    return sse_notification.NewSSENotificationUseCase(sseManager, eventService, logger)
}

// SSE Notification Handler Provider
func provideSSENotificationHandler(
    useCase inbound.SSENotificationUseCase,
    logger infrastructure.Logger,
) *api.SSENotificationHandler {
    return api.NewSSENotificationHandler(useCase, logger)
}
```

- [ ] 在 `wire.Build` 中新增依賴
- [ ] 執行 `wire ./internal/di`
- [ ] 驗證 `wire_gen.go` 生成正確

### 環境變數配置
- [ ] 新增 `.env` 配置:
```env
SSE_POD_ID=${POD_NAME}  # Kubernetes 自動注入
REDIS_ADDR=redis-service:6379
REDIS_PASSWORD=
REDIS_DB=0
```

- [ ] 驗證環境變數讀取正確

---

## 📝 實作細節參考

完整的程式碼範例請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第1604-1746行)
- **檔案結構文檔**: [file-structure.md](file-structure.md) - Infrastructure Layer 章節

---

## ✅ 驗收標準

- [ ] Redis 連接正常，支援 Streams 和 Pub/Sub（待測試）
- [x] KDS 審計日誌整合完成 ✅
- [x] Wire 依賴注入配置正確（EventService 注入完成）✅
- [ ] Pod ID 注入實作（待 Phase 5 整合）
- [ ] 環境變數配置完整（待 Phase 5 整合）
- [x] `wire_gen.go` 編譯通過 ✅

**Phase 4 完成度**: 50% (2/4 核心任務完成)

---

## 🔗 下一步

**當前狀態**: Phase 4 部分完成 (KDS 整合 + Wire EventService 注入)

**待完成任務**:
1. ✅ 任務 4.3: KDS 審計日誌整合（已完成）
2. 📋 任務 4.2: Redis 配置與測試（待開始）
3. ✅ 任務 4.4: Wire 依賴注入（EventService 部分已完成，Pod ID 注入待 Phase 5）

**推薦路徑**:
- **選項 A**: 完成 Phase 4.2 Redis 配置與測試，驗證 Pub/Sub 和 Streams 功能
- **選項 B**: 進入 Phase 5 Router 註冊與整合，完成 Pod ID 注入與路由掛載

**建議**: 優先進入 Phase 5，在整合時一併驗證 Redis 功能

---

**維護者**: Development Team
**最後更新**: 2026-02-04
