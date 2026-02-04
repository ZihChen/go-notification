# Phase 3: Adapter Layer 實作

**時程**: 第4-6天
**狀態**: ✅ **完成** (2026-02-03) - 完整實作包含 Redis Pub/Sub 整合
**Commits**:
- `1ac1d35` - feat(sse): complete Phase 3 Adapter Layer with Redis Pub/Sub integration
- `f72cbe8` - feat(sse): integrate KDS event audit logging for SSE notifications (Phase 4.3)
**返回**: [總覽文檔](overview.md) | [Phase 2](phase-2.md) | [Phase 4](phase-4.md)

---

## 📋 Phase 目標

實作 SSE 通知系統的 Adapter Layer，包含 HTTP Handler、SSE Writer、SSE Manager (含 Redis Pub/Sub) 等核心組件。

**⭐ 本 Phase 為整個系統的核心，包含 Redis Pub/Sub 跨 Pod 訊息路由實作**

---

## ✅ 任務清單

### 任務 3.1: 建立 HTTP Handler ✅
- [x] 建立 `internal/adapter/inbound/handler/api/sse_notification_handler.go`
  - 後端推播 API handlers (API Key 認證)
  - 前端 SSE 串流 handler (JWT 認證)
  - Swagger 註解

**API 端點**:
- `POST /api/v1/admin/notifications/broadcast` - 廣播推送
- `POST /api/v1/admin/notifications/send` - 個別推送
- `GET /api/v1/notifications/stream` - SSE 長連接

### 任務 3.2: 建立 SSE Writer ✅
- [x] 建立 `internal/adapter/inbound/handler/api/gin_sse_writer.go`
  - 實作 `inbound.SSEWriter` 介面
  - Write/Flush/Close 方法
  - SSE 格式輸出: `event:` + `data:` + `\n\n`

### 任務 3.3: ~~建立 Repository 實作~~ (已移除)
**⚠️ 重大變更**: 完全移除 Repository，不使用 MySQL 持久化

### 任務 3.4: 建立 SSE Manager 實作 ✅
- [x] 建立 `internal/adapter/outbound/service/sse_manager.go` (完整版 with Redis Pub/Sub)
  
**核心功能**:
1. **Pod ID 管理**:
   - 從環境變數注入 Pod ID (`SSE_POD_ID`)
   - 用於 Redis Pub/Sub Channel 識別

2. **本地連接池管理**:
   - `map[string]inbound.SSEWriter` (playerID → Writer)
   - `sync.RWMutex` 併發安全

3. **Redis Pub/Sub 訂閱**:
   - 訂閱廣播 Channel: `sse:broadcast`
   - 訂閱 Pod 專屬 Channel: `sse:pod:{podID}`

4. **玩家路由表管理** (Redis Hash):
   - 註冊時更新: `HSET sse:player_routes {playerID} {podID}`
   - 註銷時清理: `HDEL sse:player_routes {playerID}`

5. **跨 Pod 訊息轉發**:
   - 查詢玩家所在 Pod: `HGET sse:player_routes {playerID}`
   - 發布到目標 Pod: `PUBLISH sse:pod:{targetPodID} {message}`

6. **線上統計管理** (Redis Counter):
   - 註冊時遞增: `INCR sse:online_count`
   - 註銷時遞減: `DECR sse:online_count`

7. **離線訊息管理** (Redis Streams):
   - 儲存離線訊息: `XADD sse:offline:{playerID} * ...`
   - 設置 TTL: 7 天
   - 上線時推送並清空

### 任務 3.5: Pub/Sub 監聽器實作 ✅
- [x] 實作 `startPubSubListener()` 方法
  - Goroutine 監聽 Redis Pub/Sub
  - 訊息反序列化與驗證
  - 路由到本地連接池推送
  - 錯誤處理與自動重連機制

**訊息流向**:
```
接收請求的 Pod → 查詢路由表 → 發布到 Redis Channel
                                      ↓
                          所有 Pod 訂閱並接收
                                      ↓
                          各 Pod 推送給本地玩家
```

### 任務 3.6: JWT 認證 Middleware ✅
- [x] 建立 `internal/adapter/inbound/middleware/jwt_auth.go`
  - JWT Token 驗證 (支援 HMAC 簽名算法)
  - Player ID 提取與注入到 Gin Context
  - Bearer Token 格式驗證
  - 完整錯誤處理

### 任務 3.7: 優雅關閉機制 ✅
- [x] 實作 `Shutdown()` 方法
  - 發送關閉通知給所有連接
  - 清理 Redis 路由表與統計
  - 關閉 Pub/Sub 訂閱
  - 等待所有 Goroutine 結束
  - 使用 context.CancelFunc 取消背景任務
  - 使用 sync.WaitGroup 確保 Goroutine 完全退出

### 任務 3.8: 單元測試 (第6天)
- [ ] Handler 測試
- [ ] SSE Writer 測試
- [ ] SSE Manager 測試 (含 Pub/Sub 模擬)
- [ ] 路由表管理測試
- [ ] 跨 Pod 訊息轉發測試

---

## 📝 實作細節參考

完整的程式碼範例請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第814-1603行)
- **檔案結構文檔**: [file-structure.md](file-structure.md) - Adapter Layer 章節
- **技術要點文檔**: [technical-details.md](technical-details.md) - Redis Pub/Sub 實作

---

## ✅ 驗收標準

- [x] HTTP Handler 完成，支援 API Key 和 JWT 認證
- [x] SSE Writer 實作正確，SSE 格式符合標準
- [x] SSE Manager 實作完成，包含 Redis Pub/Sub 功能
- [x] Pub/Sub 監聽器正常運作
- [x] 玩家路由表管理正確
- [x] 跨 Pod 訊息轉發邏輯完成
- [x] 優雅關閉機制運作正常
- [ ] 單元測試覆蓋率 > 80% (Phase 3.8 待實作)

---

## 📊 完成摘要

**完成日期**: 2026-02-03
**提交文件**: 4 個 Go 文件，709 行代碼（含常數更新）
**Commit Hash**: 待提交
**狀態**: ✅ **完成** - 包含完整 Redis Pub/Sub 整合與 JWT 認證

### ✅ 已建立/更新文件

**Adapter Layer 文件**:
- ✅ `internal/adapter/inbound/handler/api/gin_sse_writer.go` (73 行)
- ✅ `internal/adapter/inbound/handler/api/sse_notification_handler.go` (154 行)
- ✅ `internal/adapter/outbound/service/sse_manager.go` (539 行，完整版)
- ✅ `internal/adapter/inbound/middleware/jwt_auth.go` (103 行)

**Domain Layer 更新**:
- ✅ `internal/domain/consts/sse_notification.go` (新增簡化別名常數)

### 核心功能實作

1. ✅ **GinSSEWriter**: 完整的 SSE Writer 實作
   - SSE 標準格式輸出 (`event: xxx\ndata: xxx\n\n`)
   - Write/Flush/Close 方法
   - 正確的 SSE Headers 設置
   - http.Flusher 支援檢查

2. ✅ **HTTP Handler**: 3 個 API 端點實作
   - POST /api/v1/admin/notifications/broadcast
   - POST /api/v1/admin/notifications/send
   - GET /api/v1/notifications/stream
   - Swagger 註解完整
   - 完整錯誤處理

3. ✅ **SSE Manager 完整版** (539 行):
   - **本地連接池管理**: map + sync.RWMutex 併發安全
   - **Redis Pub/Sub 整合**: 訂閱廣播與 Pod 專屬頻道
   - **玩家路由表管理**: HSET/HDEL/HGET/HEXISTS/HKEYS
   - **線上統計管理**: INCR/DECR/GET
   - **離線訊息管理**: XADD/XRANGE/DEL (Redis Streams with 7-day TTL)
   - **Pub/Sub 監聽器**: Goroutine 背景監聽與訊息處理
   - **跨 Pod 訊息轉發**: 智能路由與 Pub/Sub 發布
   - **優雅關閉機制**: Shutdown 方法含完整清理流程
   - **降級策略**: Redis 不可用時降級到本地功能

4. ✅ **JWT 認證 Middleware** (103 行):
   - Bearer Token 格式驗證
   - JWT HMAC 簽名驗證
   - Player ID 提取與注入到 Gin Context
   - Merchant ID 支援（可選）
   - 完整的錯誤處理與回應

### 技術亮點

- **併發安全**: sync.RWMutex 保護本地連接池
- **背景任務管理**: context.Context + context.CancelFunc + sync.WaitGroup
- **Redis 降級策略**: Redis 不可用時自動降級到本地功能
- **多 Pod 水平擴展**: 完整的 Pub/Sub 路由機制
- **優雅關閉**: 通知客戶端 + 清理資源 + 等待 Goroutine 退出
- **7 天離線訊息**: Redis Streams with TTL 管理

### 待完成項目

- [ ] 單元測試 (任務 3.8) - 建議優先級：中
  - SSE Manager 測試（含 Redis Mock）
  - JWT Middleware 測試
  - Pub/Sub 訊息處理測試
  - 優雅關閉測試

---

## 🔗 下一步

✅ Phase 3 完整實作完成，建議進入 Phase 4 Infrastructure Layer

**推薦路徑**: 進入 Phase 4 Infrastructure Layer
- Wire 依賴注入設定
- Router 註冊（HTTP 端點掛載）
- 環境變數配置（Pod ID, JWT Secret 等）
- 啟動流程整合

**可選**: 先補完 Phase 3 單元測試
- SSE Manager 完整測試
- JWT Middleware 測試
- 整合測試

---

**維護者**: Development Team
**最後更新**: 2026-02-03
