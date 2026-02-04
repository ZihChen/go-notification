# Phase 6: 測試與優化

**時程**: 第9-10天  
**返回**: [總覽文檔](overview.md) | [Phase 5](phase-5.md) | [Phase 7](phase-7.md)

---

## 📋 Phase 目標

進行整合測試、效能測試、壓力測試，並進行效能優化。

**⭐ 包含多 Pod 測試，驗證 Redis Pub/Sub 跨 Pod 訊息路由**

---

## ✅ 任務清單

### 任務 6.1: 整合測試 ✅ **已完成** (2026-02-04)

**單 Pod 測試** (6/6 通過):
- [x] 廣播推送整合測試 (`TestSSE_BroadcastToAll_SinglePod`)
- [x] 個別推送整合測試 (`TestSSE_SendToPlayer_SamePod`)
- [x] SSE 長連接測試 (`TestSSE_RegisterConnection`, `TestSSE_UnregisterConnection`)
- [x] 離線訊息測試 (`TestSSE_OfflineMessages`)
- [x] 線上玩家統計測試 (`TestSSE_GetOnlinePlayerCount`)

**多 Pod 測試** ⭐ **關鍵驗證** (5/5 通過 ✅):
- [x] 啟動 3 個 SSE Pod 實例 (不同 Pod ID)
- [x] 跨 Pod 廣播測試 (驗證所有 Pod 都收到) - `TestMultiPod_BroadcastToAll`
- [x] 跨 Pod 個別推送測試 (玩家路由正確) - `TestMultiPod_SendToPlayer_CrossPod` ✅ **已修復**
- [x] 玩家路由表一致性測試 - `TestMultiPod_PlayerRoutingConsistency`
- [x] Pod 間訊息轉發延遲測試 - `TestMultiPod_MessageLatency` (平均 101ms)
- [x] 高併發跨 Pod 推送測試 - `TestMultiPod_ConcurrentBroadcast` (20 併發 × 15 玩家)

**測試結果**:
- 總計：18 個測試
- 通過：18 個 (100%) ✅
- 跳過：0 個
- 測試檔案：`test/sse_integration_test.go`, `test/sse_multi_pod_test.go`

### 任務 6.2: Redis 功能驗證 ✅ **已完成** (2026-02-04)

**Redis 操作測試** (7/7 通過):
- [x] 基本連接測試 - `TestRedis_BasicConnection`
- [x] Pub/Sub 功能測試 - `TestRedis_PubSubFunctionality` (單頻道/多頻道)
- [x] Streams 操作測試 - `TestRedis_StreamsOperations` (XADD/XRANGE/XLEN/XDEL/MAXLEN)
- [x] Hash 操作測試 - `TestRedis_HashOperations` (HSET/HGET/HEXISTS/HGETALL/HLEN/HDEL)
- [x] Counter 操作測試 - `TestRedis_CounterOperations` (INCR/GET/INCRBY/DECR/DECRBY/DEL)
- [x] 連接池配置測試 - `TestRedis_ConnectionPoolConfiguration`
- [x] 版本兼容性測試 - `TestRedis_VersionCompatibility`

**測試檔案**: `test/sse_redis_test.go`

### 任務 6.3: 效能測試 (第9-10天) ⏳ **待執行**

**單 Pod 效能**:
- [ ] 廣播吞吐量測試 (目標: > 10,000 QPS)
- [ ] 訊息入隊延遲測試 (目標: < 50ms P99)
- [ ] 並發連接測試 (目標: 10,000+ 連接/Pod)

**多 Pod 效能**:
- [ ] 3 Pods 總吞吐量測試 (目標: > 30,000 QPS)
- [ ] 跨 Pod 路由延遲測試 (目標: < 100ms P99)
- [ ] 30,000 並發連接測試
- [ ] Redis Pub/Sub 延遲測試

**遞送率測試**:
- [ ] 單 Pod 遞送率 (目標: 99.5%)
- [ ] 多 Pod 遞送率 (目標: 99%)

**單 Pod 效能**:
- [ ] 廣播吞吐量測試 (目標: > 10,000 QPS)
- [ ] 訊息入隊延遲測試 (目標: < 50ms P99)
- [ ] 並發連接測試 (目標: 10,000+ 連接/Pod)

**多 Pod 效能**:
- [ ] 3 Pods 總吞吐量測試 (目標: > 30,000 QPS)
- [ ] 跨 Pod 路由延遲測試 (目標: < 100ms P99)
- [ ] 30,000 並發連接測試
- [ ] Redis Pub/Sub 延遲測試

**遞送率測試**:
- [ ] 單 Pod 遞送率 (目標: 99.5%)
- [ ] 多 Pod 遞送率 (目標: 99%)

### 任務 6.4: 壓力測試與優化 (第10天) ⏳ **待執行**
- [ ] 使用 wrk/ab 進行壓力測試
- [ ] Goroutine 洩漏檢測
- [ ] 記憶體使用分析 (pprof)
- [ ] CPU 使用分析 (pprof)
- [ ] 效能優化:
  - JSON 序列化優化
  - Redis 連接池調優
  - Pub/Sub 訊息批次處理

### 任務 6.5: 故障恢復測試 (第10天) ⏳ **待執行**
- [ ] Pod 重啟測試 (連接自動遷移)
- [ ] Pod 突然終止測試
- [ ] Redis 短暫斷線測試
- [ ] 網路分區測試

### 任務 6.6: 清理過期訊息 (第10天) ⏳ **待執行**
- [ ] 建立定時任務 (Scheduler)
- [ ] 實作過期訊息清理邏輯 (Redis Streams)
- [ ] 測試清理機制

---

## ✅ 驗收標準

- [x] 所有整合測試通過 ✅ (18/18 通過，100% 通過率)
- [x] 多 Pod 測試通過，訊息路由正確 ✅ (5/5 通過，包含跨 Pod 個別推送)
- [ ] 效能指標達標 ⏳ (Phase 6.3 延後執行)
- [ ] 故障恢復機制運作正常 ⏳ (待執行)
- [ ] 無 Goroutine 洩漏或記憶體洩漏 ⏳ (待執行)

---

## ✅ 已修復問題

### 1. 跨 Pod 個別推送功能 ✅ **已完成** (2026-02-04)

**修復內容**:
- 位置：`internal/adapter/outbound/service/sse_manager.go`, `internal/domain/entity/sse_notification.go`
- 在 `entity.SSENotification` 中新增 `TargetPlayerID` 欄位
- 在 `SendToPlayer()` 發布訊息時設置目標玩家 ID
- 在 `handleTargetedMessage()` 中實現完整發送邏輯：
  - 驗證 TargetPlayerID 存在
  - 查找本地連接
  - 發送訊息到對應 Writer
  - 完整的錯誤處理和日誌記錄

**修復後行為**:
```go
func (m *sseManager) handleTargetedMessage(payload string) {
    var notification entity.SSENotification
    if err := json.Unmarshal([]byte(payload), &notification); err != nil {
        m.logger.WarnLog("Failed to unmarshal targeted notification", ...)
        return
    }

    // 檢查是否有目標玩家 ID
    if notification.TargetPlayerID == "" {
        m.logger.WarnLog("Targeted message missing target player ID", ...)
        return
    }

    // 查找本地連接並發送訊息
    m.mu.RLock()
    writer, exists := m.connections[notification.TargetPlayerID]
    m.mu.RUnlock()

    if exists {
        m.sendNotificationToWriter(writer, &notification)
        m.logger.InfoLog("Targeted message delivered successfully", ...)
    }
}
```

**測試狀態**:
- `TestMultiPod_SendToPlayer_CrossPod` 測試通過 ✅
- 所有多 Pod 測試 5/5 通過 ✅

**提交記錄**:
- `e98ea41` feat(sse): implement cross-pod individual push with handleTargetedMessage

---

## 📝 實作細節參考

完整的程式碼範例和任務細節請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (實作步驟章節)
- **總覽文檔**: [overview.md](overview.md)
- **測試檔案**:
  - `test/sse_integration_test.go` - 單 Pod 整合測試
  - `test/sse_multi_pod_test.go` - 多 Pod 整合測試
  - `test/sse_redis_test.go` - Redis 功能驗證測試

---

## 🔗 下一步

**當前狀態**: Phase 6 整合測試完成 ✅，前往 [Phase 7: 文檔撰寫](phase-7.md)

**已完成**:
1. ✅ 修復 `handleTargetedMessage` 已知問題 (2026-02-04)
2. ✅ Phase 6.1-6.2 整合測試 (18/18 通過)
3. ✅ Phase 6.4 Redis 功能驗證 (7/7 通過)

**延後執行**:
- ⏳ Phase 6.3 效能測試與基準測試（延後至需要時執行）
- ⏳ Phase 6.4-6.6 壓力測試與優化（選用）

**下一步行動**:
→ 進入 **Phase 7: 文檔撰寫**（API 文檔、部署指南、使用說明）

---

**維護者**: Development Team
**最後更新**: 2026-02-04
