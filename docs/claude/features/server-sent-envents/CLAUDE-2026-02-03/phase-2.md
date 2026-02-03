# Phase 2: Application Layer 實作

**時程**: 第3天  
**返回**: [總覽文檔](overview.md) | [Phase 1](phase-1.md) | [Phase 3](phase-3.md)

---

## 📋 Phase 目標

實作 SSE 通知系統的 Application Layer，包含 DTO 定義和 UseCase 實作 (簡化版，移除 MySQL 依賴)。

---

## ✅ 任務清單

### 任務 2.1: 建立 DTO
- [ ] 建立 `internal/application/dto/sse_notification.go`
  - BroadcastRequest/Response (廣播推送)
  - SendNotificationRequest/Response (個別推送)

**簡化說明**:
- ❌ 移除 Visibility/Metadata 欄位
- ❌ 移除 NotificationStatsResponse (統計改用 Prometheus)
- ❌ 移除 NotificationHistoryResponse/NotificationItem (無歷史查詢)
- ✅ 保留核心推送 DTO，簡化欄位

### 任務 2.2: 實作 UseCase
- [ ] 建立 `internal/application/usecase/sse_notification/sse_notification_usecase.go`
  - BroadcastNotification 實作 (透過 SSEManager)
  - SendNotification 實作 (透過 SSEManager)
  - StreamNotifications 實作 (SSE 長連接 + 離線訊息)

**依賴注入**:
```go
type sseNotificationUseCase struct {
    sseManager   service.SSEManager      // SSE 連接管理與訊息推送
    eventService service.EventService    // KDS 審計日誌
    logger       infrastructure.Logger
}
```

**核心邏輯** (簡化版):
1. `BroadcastNotification()`:
   - 創建通知實體 (純記憶體對象，不持久化)
   - 透過 SSEManager 廣播到所有 Pod
   - 記錄審計日誌到 KDS
   - 返回推送統計

2. `SendNotification()`:
   - 驗證玩家 ID 數量 (最多 1000 位)
   - 批次推送給目標玩家
   - 記錄審計日誌到 KDS
   - 返回推送統計

3. `StreamNotifications()`:
   - 註冊玩家連接
   - 推送離線訊息 (Redis Streams)
   - 保持連接直到 context 取消

### 任務 2.3: UseCase 單元測試
- [ ] 建立 `internal/application/usecase/sse_notification/sse_notification_usecase_test.go`
  - 廣播推送測試
  - 個別推送測試
  - SSE 串流測試
  - 離線訊息測試
  - Mock SSEManager、EventService

---

## 📝 實作細節參考

完整的程式碼範例請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第548-813行)
- **檔案結構文檔**: [file-structure.md](file-structure.md) - Application Layer 章節

---

## ✅ 驗收標準

- [ ] 所有 DTO 定義完成，符合簡化設計
- [ ] UseCase 實作完成，不依賴 MySQL Repository
- [ ] UseCase 單元測試覆蓋率 100%
- [ ] Mock 對象使用正確
- [ ] 代碼編譯通過，測試全部通過

---

## 🔗 下一步

完成 Phase 2 後，前往 [Phase 3: Adapter Layer 實作](phase-3.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
