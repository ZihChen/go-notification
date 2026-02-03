# Phase 3: Adapter Layer 實作

**時程**: 第4-6天  
**返回**: [總覽文檔](overview.md) | [Phase 2](phase-2.md) | [Phase 4](phase-4.md)

---

## 📋 Phase 目標

實作 SSE 通知系統的 Adapter Layer，包含 HTTP Handler、SSE Writer、SSE Manager (含 Redis Pub/Sub) 等核心組件。

**⭐ 本 Phase 為整個系統的核心，包含 Redis Pub/Sub 跨 Pod 訊息路由實作**

---

## ✅ 任務清單

### 任務 3.1: 建立 HTTP Handler (第4天)
- [ ] 建立 `internal/adapter/inbound/handler/api/sse_notification_handler.go`
  - 後端推播 API handlers (API Key 認證)
  - 前端 SSE 串流 handler (JWT 認證)
  - Swagger 註解

**API 端點**:
- `POST /api/v1/admin/notifications/broadcast` - 廣播推送
- `POST /api/v1/admin/notifications/send` - 個別推送
- `GET /api/v1/notifications/stream` - SSE 長連接

### 任務 3.2: 建立 SSE Writer (第4天)
- [ ] 建立 `internal/adapter/inbound/handler/api/gin_sse_writer.go`
  - 實作 `inbound.SSEWriter` 介面
  - Write/Flush/Close 方法
  - SSE 格式輸出: `event:` + `data:` + `\n\n`

### 任務 3.3: ~~建立 Repository 實作~~ (已移除)
**⚠️ 重大變更**: 完全移除 Repository，不使用 MySQL 持久化

### 任務 3.4: 建立 SSE Manager 實作 (第5天) ⭐ **核心組件**
- [ ] 建立 `internal/adapter/outbound/service/sse_manager.go`
  
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

### 任務 3.5: Pub/Sub 監聽器實作 (第5天)
- [ ] 實作 `startPubSubListener()` 方法
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

### 任務 3.6: JWT 認證 Middleware (第6天)
- [ ] 建立 `internal/adapter/inbound/middleware/jwt_auth.go`
  - JWT Token 驗證
  - Player ID 提取與注入
  - 錯誤處理

### 任務 3.7: 優雅關閉機制 (第6天)
- [ ] 實作 `Shutdown()` 方法
  - 發送關閉通知給所有連接
  - 清理 Redis 路由表與統計
  - 關閉 Pub/Sub 訂閱
  - 等待所有 Goroutine 結束

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

- [ ] HTTP Handler 完成，支援 API Key 和 JWT 認證
- [ ] SSE Writer 實作正確，SSE 格式符合標準
- [ ] SSE Manager 實作完成，包含 Redis Pub/Sub 功能
- [ ] Pub/Sub 監聽器正常運作
- [ ] 玩家路由表管理正確
- [ ] 跨 Pod 訊息轉發測試通過
- [ ] 優雅關閉機制運作正常
- [ ] 單元測試覆蓋率 > 80%

---

## 🔗 下一步

完成 Phase 3 後，前往 [Phase 4: Infrastructure Layer 實作](phase-4.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
