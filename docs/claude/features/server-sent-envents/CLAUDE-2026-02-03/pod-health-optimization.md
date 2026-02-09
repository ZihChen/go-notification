# Pod 健康心跳與路由清理優化

**建立日期**: 2026-02-09
**最後更新**: 2026-02-09
**狀態**: 🚧 實施中 (Phase 1 完成 ✅ | Phase 2-4 待實施)
**優先級**: 🔴 HIGH
**返回**: [總覽文檔](overview.md)

---

## 📋 優化背景

### 問題描述

當前 SSE 服務在 Pod crash（非優雅關閉）時，存在**消息黑洞（Message Black Hole）**問題：

**場景示例**：
```
時間點 T0: Player123 連接到 Pod-A
           Redis: sse:player_routes["player123"] = "pod-a"

時間點 T1: Pod-A crash (沒有執行 Shutdown)
           Redis: sse:player_routes["player123"] = "pod-a" (殘留!)

時間點 T2: 後端服務發送消息給 Player123
           1. 查詢 Redis → 發現 player123 在 pod-a
           2. Pub/Sub 發送到 "sse:pod:pod-a" 頻道
           3. 但 pod-a 已死，沒有訂閱者
           4. ❌ 消息丟失（進入黑洞）

時間點 T3: Player123 重連（可能 5 分鐘後）
           1. Kubernetes Service 導向 Pod-B
           2. 更新 Redis: sse:player_routes["player123"] = "pod-b"
           3. ✅ 後續消息正常

結果: T1-T3 之間的所有消息永久丟失 ❌
```

### 根本原因分析

1. **Redis 路由殘留**
   - Pod crash 時無法執行 `UnregisterConnection()`
   - Redis `sse:player_routes` 中的路由記錄不會被清理
   - 指向死 Pod 的路由記錄持續存在

2. **Pub/Sub 發送無驗證**
   - 當前代碼發送前不檢查目標 Pod 是否存活
   - `PUBLISH` 命令返回成功只代表"發布成功"，不保證有訂閱者
   - 發送到死 Pod 的消息會靜默丟失

3. **缺乏失敗兜底機制**
   - 消息發送失敗不會自動加入離線隊列
   - 沒有重試或補償機制

### 影響範圍

- **用戶體驗**: 重要通知（訂單確認、支付結果、安全警告）可能丟失
- **影響時間**: Pod crash 到客戶端重連之間（可能 1-10 分鐘）
- **丟失量**: 與業務流量成正比，高峰期影響更大

---

## 🎯 優化目標

### 核心目標

1. **消息零丟失**: Pod crash 期間的所有消息都能進入離線隊列
2. **自動檢測**: 實時檢測 Pod 健康狀態，避免向死 Pod 發送消息
3. **自動清理**: 自動清理 Redis 中指向死 Pod 的殘留路由記錄

### 性能指標

- Pod crash 檢測延遲 < 30 秒
- 路由清理延遲 < 1 分鐘
- 消息遞送率 = 100%（包含離線消息）
- 額外性能開銷 < 1%

---

## 🏗️ 技術方案

### 方案 1: Pod 健康心跳機制

#### 設計思路

每個 Pod 定期向 Redis 更新健康狀態，發送消息前先檢查目標 Pod 是否健康。

#### Redis 數據結構

```
鍵: sse:pod_health:{podID}
值: "alive"
TTL: 30 秒

示例:
sse:pod_health:pod-a = "alive" (TTL 30s)
sse:pod_health:pod-b = "alive" (TTL 30s)
sse:pod_health:pod-c = "alive" (TTL 30s)
```

**特性**:
- Pod 每 10 秒更新一次心跳
- TTL 30 秒，Pod crash 後 30 秒自動過期
- 使用 `EXISTS` 命令快速檢查（O(1) 複雜度）

#### 實現步驟

##### Step 1.1: 新增心跳 Goroutine

**文件**: `internal/adapter/outbound/service/sse_manager.go`

```go
// NewSSEManager 初始化時啟動心跳
func NewSSEManager(...) outboundService.SSEManager {
    // ... 現有代碼 ...

    // 啟動 Pub/Sub 監聽器
    manager.wg.Add(1)
    go manager.startPubSubListener()

    // ✅ 新增：啟動 Pod 健康心跳
    manager.wg.Add(1)
    go manager.startPodHealthHeartbeat()

    return manager
}

// ✅ 新增方法：Pod 健康心跳
func (m *sseManager) startPodHealthHeartbeat() {
    defer m.wg.Done()

    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    healthKey := fmt.Sprintf("sse:pod_health:%s", m.podID)

    m.logger.InfoLog("Pod health heartbeat started",
        m.logger.String("pod_id", m.podID),
        m.logger.String("interval", "10s"),
        m.logger.String("ttl", "30s"))

    for {
        select {
        case <-m.ctx.Done():
            m.logger.InfoLog("Pod health heartbeat stopped",
                m.logger.String("pod_id", m.podID))
            return
        case <-ticker.C:
            // 更新健康狀態，TTL 30 秒
            ctx := context.Background()
            if err := m.redisClient.Set(ctx, healthKey, "alive", 30*time.Second).Err(); err != nil {
                m.logger.WarnLog("Failed to update pod health heartbeat",
                    m.logger.Error("error", err),
                    m.logger.String("pod_id", m.podID))
            }
        }
    }
}
```

##### Step 1.2: 發送消息前檢查 Pod 健康

**文件**: `internal/adapter/outbound/service/sse_manager.go`

**修改方法**: `SendToPlayer()`

```go
// SendToPlayer 推送訊息給單一玩家
func (m *sseManager) SendToPlayer(ctx context.Context, playerID string, notification *entity.SSENotification) error {
    // 先檢查玩家是否在本地連接池
    m.mu.RLock()
    writer, existsLocally := m.connections[playerID]
    m.mu.RUnlock()

    if existsLocally {
        // 玩家在本地，直接推送
        return m.sendNotificationToWriter(writer, notification)
    }

    // 玩家不在本地，查詢 Redis 路由表
    if m.redisClient != nil {
        targetPodID, err := m.redisClient.HGet(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
        if err == redis.Nil {
            // 玩家離線，加入離線佇列
            return m.EnqueueOfflineMessage(ctx, playerID, notification)
        } else if err != nil {
            m.logger.WarnWithContext(ctx, "Failed to get player route from Redis",
                m.logger.Error("error", err),
                m.logger.String("player_id", playerID))
            return fmt.Errorf("failed to query player route: %w", err)
        }

        // 玩家在其他 Pod，先檢查目標 Pod 是否健康
        if targetPodID != m.podID {
            // ✅ 新增：檢查目標 Pod 健康狀態
            healthKey := fmt.Sprintf("sse:pod_health:%s", targetPodID)
            exists, err := m.redisClient.Exists(ctx, healthKey).Result()

            if err != nil {
                m.logger.WarnWithContext(ctx, "Failed to check target pod health",
                    m.logger.Error("error", err),
                    m.logger.String("player_id", playerID),
                    m.logger.String("target_pod", targetPodID))
                // Redis 檢查失敗，為了安全起見加入離線隊列
                return m.EnqueueOfflineMessage(ctx, playerID, notification)
            }

            if exists == 0 {
                // ✅ 目標 Pod 已死，清理殘留路由並加入離線隊列
                m.logger.WarnWithContext(ctx, "Target pod is dead, cleaning route and enqueuing offline",
                    m.logger.String("player_id", playerID),
                    m.logger.String("dead_pod", targetPodID))

                // 清理殘留路由
                m.redisClient.HDel(ctx, consts.SSEPlayerRoutesKey, playerID)

                // 加入離線隊列
                return m.EnqueueOfflineMessage(ctx, playerID, notification)
            }

            // 目標 Pod 健康，通過 Pub/Sub 轉發
            // 設置目標玩家 ID 用於跨 Pod 路由
            notification.TargetPlayerID = playerID

            data, err := json.Marshal(notification)
            if err != nil {
                return fmt.Errorf("failed to marshal notification: %w", err)
            }

            channel := fmt.Sprintf(consts.SSEPodChannel, targetPodID)
            if err := m.redisClient.Publish(ctx, channel, data).Err(); err != nil {
                m.logger.WarnWithContext(ctx, "Failed to publish to target pod",
                    m.logger.Error("error", err),
                    m.logger.String("player_id", playerID),
                    m.logger.String("target_pod", targetPodID))

                // ✅ Pub/Sub 失敗也加入離線隊列
                return m.EnqueueOfflineMessage(ctx, playerID, notification)
            }

            m.logger.InfoWithContext(ctx, "Message routed to target pod",
                m.logger.String("player_id", playerID),
                m.logger.String("target_pod", targetPodID),
                m.logger.String("notification_id", notification.ID))
            return nil
        }
    }

    // 無 Redis 或其他錯誤，加入離線佇列
    return m.EnqueueOfflineMessage(ctx, playerID, notification)
}
```

##### Step 1.3: 新增 Redis Key 常量

**文件**: `internal/domain/consts/redis_keys.go`

```go
// ✅ 新增常量
const (
    // ... 現有常量 ...

    // SSEPodHealthKey Pod 健康心跳 Redis Key
    // 格式: sse:pod_health:{podID}
    // TTL: 30 秒
    // 示例: sse:pod_health:pod-a
    SSEPodHealthKey = "sse:pod_health:%s"
)
```

---

### 方案 2: 死 Pod 路由清理機制

#### 設計思路

定期掃描 Redis 路由表，清理指向死 Pod 的殘留路由記錄。

#### 實現步驟

##### Step 2.1: 新增路由清理 Goroutine

**文件**: `internal/adapter/outbound/service/sse_manager.go`

```go
// NewSSEManager 初始化時啟動清理任務
func NewSSEManager(...) outboundService.SSEManager {
    // ... 現有代碼 ...

    // 啟動 Pod 健康心跳
    manager.wg.Add(1)
    go manager.startPodHealthHeartbeat()

    // ✅ 新增：啟動路由清理任務
    manager.wg.Add(1)
    go manager.startRouteCleanupTask()

    return manager
}

// ✅ 新增方法：路由清理任務
func (m *sseManager) startRouteCleanupTask() {
    defer m.wg.Done()

    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    m.logger.InfoLog("Route cleanup task started",
        m.logger.String("pod_id", m.podID),
        m.logger.String("interval", "1m"))

    for {
        select {
        case <-m.ctx.Done():
            m.logger.InfoLog("Route cleanup task stopped",
                m.logger.String("pod_id", m.podID))
            return
        case <-ticker.C:
            m.cleanupDeadPodRoutes()
        }
    }
}

// ✅ 新增方法：清理死 Pod 路由
func (m *sseManager) cleanupDeadPodRoutes() {
    ctx := context.Background()

    // 1. 獲取所有玩家路由
    routes, err := m.redisClient.HGetAll(ctx, consts.SSEPlayerRoutesKey).Result()
    if err != nil {
        m.logger.WarnLog("Failed to get player routes for cleanup",
            m.logger.Error("error", err))
        return
    }

    if len(routes) == 0 {
        return
    }

    // 2. 收集所有唯一的 Pod ID
    podIDs := make(map[string]bool)
    for _, podID := range routes {
        podIDs[podID] = true
    }

    // 3. 檢查每個 Pod 是否健康
    deadPods := make(map[string]bool)
    for podID := range podIDs {
        healthKey := fmt.Sprintf(consts.SSEPodHealthKey, podID)
        exists, err := m.redisClient.Exists(ctx, healthKey).Result()
        if err != nil {
            m.logger.WarnLog("Failed to check pod health during cleanup",
                m.logger.Error("error", err),
                m.logger.String("pod_id", podID))
            continue
        }
        if exists == 0 {
            deadPods[podID] = true
        }
    }

    if len(deadPods) == 0 {
        // 沒有死 Pod，無需清理
        return
    }

    // 4. 清理指向死 Pod 的路由
    cleanedPlayers := make([]string, 0)
    for playerID, podID := range routes {
        if deadPods[podID] {
            if err := m.redisClient.HDel(ctx, consts.SSEPlayerRoutesKey, playerID).Err(); err != nil {
                m.logger.WarnLog("Failed to delete dead pod route",
                    m.logger.Error("error", err),
                    m.logger.String("player_id", playerID),
                    m.logger.String("dead_pod", podID))
            } else {
                cleanedPlayers = append(cleanedPlayers, playerID)
            }
        }
    }

    // 5. 記錄清理結果
    if len(cleanedPlayers) > 0 {
        m.logger.InfoLog("Cleaned up dead pod routes",
            m.logger.Int("cleaned_count", len(cleanedPlayers)),
            m.logger.Int("dead_pods_count", len(deadPods)),
            m.logger.String("dead_pods", fmt.Sprintf("%v", getMapKeys(deadPods))),
            m.logger.String("sample_players", fmt.Sprintf("%v", cleanedPlayers[:min(5, len(cleanedPlayers))])))
    }
}

// 輔助函數：獲取 map 的所有 key
func getMapKeys(m map[string]bool) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// 輔助函數：取最小值
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

---

## 📊 效果對比

### 優化前（當前狀態）

```
場景：Pod-A crash，player_001 在地鐵裡，5 分鐘後才重連

時間軸：
T0:     Pod-A crash
T1:     發送訊息 1 → ❌ 丟失（發送到死 Pod）
T2:     發送訊息 2 → ❌ 丟失
T3:     發送訊息 3 → ❌ 丟失
T4:     發送訊息 4 → ❌ 丟失
T5min:  player_001 重連 → 更新路由到 pod-b
T6:     發送訊息 5 → ✅ 成功

結果：
✅ 訊息 5 成功
❌ 訊息 1-4 永久丟失（4 條訊息）
📉 消息遞送率：20% (1/5)
😞 用戶體驗：極差
```

### 優化後（實施方案）

```
場景：Pod-A crash，player_001 在地鐵裡，5 分鐘後才重連

時間軸：
T0:      Pod-A crash
T0+30s:  Pod-A 健康心跳過期（自動）
T1:      發送訊息 1 → ✅ 檢測到 Pod 已死 → 加入離線隊列
T2:      發送訊息 2 → ✅ 加入離線隊列
T1+1m:   路由清理任務執行 → 清理 player_001 路由
T3:      發送訊息 3 → ✅ 查詢路由失敗 → 加入離線隊列
T4:      發送訊息 4 → ✅ 加入離線隊列
T5min:   player_001 重連 → 推送離線訊息 1-4
T6:      發送訊息 5 → ✅ 實時推送

結果：
✅ 訊息 1-5 全部成功送達
✅ 離線訊息 1-4 在重連時推送
✅ 實時訊息 5 即時推送
📈 消息遞送率：100% (5/5)
😄 用戶體驗：完美
```

---

## 🧪 測試計劃

### 單元測試

**文件**: `internal/adapter/outbound/service/sse_manager_test.go`

```go
// ✅ 新增測試
func TestSSEManager_PodHealthHeartbeat(t *testing.T) {
    // 測試心跳正常更新
}

func TestSSEManager_SendToPlayer_DeadPodDetection(t *testing.T) {
    // 測試檢測到死 Pod 後加入離線隊列
}

func TestSSEManager_CleanupDeadPodRoutes(t *testing.T) {
    // 測試路由清理功能
}
```

### 整合測試

**場景 1: Pod Crash 檢測**
```
1. 啟動 Pod-A 和 Pod-B
2. Player123 連接到 Pod-A
3. 停止 Pod-A（模擬 crash）
4. 等待 30 秒（心跳過期）
5. 發送消息給 Player123
6. ✅ 驗證：消息進入離線隊列，不會丟失
```

**場景 2: 路由清理**
```
1. 啟動 Pod-A 和 Pod-B
2. 多個玩家連接到 Pod-A
3. 停止 Pod-A（模擬 crash）
4. 等待 1 分鐘（清理任務執行）
5. ✅ 驗證：Redis 中 Pod-A 的所有路由已清理
```

**場景 3: 消息零丟失**
```
1. 啟動 Pod-A
2. Player123 連接到 Pod-A
3. 停止 Pod-A（模擬 crash）
4. 連續發送 100 條消息給 Player123
5. Player123 重連到 Pod-B
6. ✅ 驗證：所有 100 條消息都在離線隊列中
```

---

## 📝 實施檢查清單

### Phase 1: Pod 健康心跳機制 ✅ **已完成 (2026-02-09)**

- [x] **Step 1.1**: 實現 `startPodHealthHeartbeat()` 方法
  - [x] 每 10 秒更新 Redis `sse:pod_health:{podID}`
  - [x] TTL 設置為 30 秒
  - [x] 在 `NewSSEManager()` 中啟動 Goroutine
  - [x] 優雅關閉時停止心跳

- [x] **Step 1.2**: 修改 `SendToPlayer()` 方法
  - [x] 新增目標 Pod 健康檢查邏輯
  - [x] 檢測到死 Pod 時清理路由
  - [x] 檢測到死 Pod 時加入離線隊列
  - [x] Pub/Sub 失敗時加入離線隊列

- [x] **Step 1.3**: 新增 Redis Key 常量
  - [x] 在 `consts/sse_notification.go` 新增 `SSEPodHealthKey`
  - [x] 更新相關文檔

### Phase 2: 死 Pod 路由清理機制

- [ ] **Step 2.1**: 實現 `startRouteCleanupTask()` 方法
  - [ ] 每 1 分鐘執行一次清理
  - [ ] 實現 `cleanupDeadPodRoutes()` 邏輯
  - [ ] 在 `NewSSEManager()` 中啟動 Goroutine
  - [ ] 優雅關閉時停止清理任務

- [ ] **Step 2.2**: 實現清理邏輯
  - [ ] 獲取所有玩家路由
  - [ ] 檢查每個 Pod 健康狀態
  - [ ] 清理指向死 Pod 的路由
  - [ ] 記錄清理統計日誌

### Phase 3: 測試與驗證

- [ ] **單元測試**
  - [ ] 心跳更新測試
  - [ ] 死 Pod 檢測測試
  - [ ] 路由清理測試
  - [ ] 離線隊列兜底測試

- [ ] **整合測試**
  - [ ] Pod Crash 檢測場景
  - [ ] 路由清理場景
  - [ ] 消息零丟失場景
  - [ ] 高併發壓力測試

- [ ] **生產驗證**
  - [ ] Staging 環境驗證
  - [ ] 性能指標驗證
  - [ ] 監控告警驗證

### Phase 4: 文檔更新

- [ ] **技術文檔**
  - [ ] 更新 `technical-details.md`
  - [ ] 更新架構圖

- [ ] **運維文檔**
  - [ ] 更新 `OPERATIONS_GUIDE.md`
  - [ ] 新增監控指標說明

- [ ] **API 文檔**
  - [ ] 更新錯誤處理說明
  - [ ] 新增最佳實踐建議

---

## 📈 監控指標

### 新增監控指標

```yaml
# Pod 健康狀態
sse_pod_health_heartbeat_success_total
sse_pod_health_heartbeat_failure_total
sse_pod_health_check_total
sse_pod_health_dead_pod_detected_total

# 路由清理
sse_route_cleanup_execution_total
sse_route_cleanup_cleaned_routes_total
sse_route_cleanup_dead_pods_detected_total
sse_route_cleanup_duration_seconds

# 消息遞送
sse_message_offline_queue_total
sse_message_dead_pod_recovered_total
sse_message_delivery_success_rate
```

### 告警規則

```yaml
# Pod 心跳失敗告警
- alert: SSEPodHeartbeatFailure
  expr: rate(sse_pod_health_heartbeat_failure_total[5m]) > 0.1
  for: 2m
  annotations:
    summary: "SSE Pod 健康心跳失敗率過高"

# 死 Pod 檢測告警
- alert: SSEDeadPodDetected
  expr: increase(sse_pod_health_dead_pod_detected_total[5m]) > 5
  for: 1m
  annotations:
    summary: "檢測到大量死 Pod，可能存在系統問題"

# 消息遞送率告警
- alert: SSEMessageDeliveryRateLow
  expr: sse_message_delivery_success_rate < 0.99
  for: 5m
  annotations:
    summary: "SSE 消息遞送率低於 99%"
```

---

## 🎯 預期效果

### 性能指標

| 指標 | 優化前 | 優化後 | 改善 |
|------|--------|--------|------|
| 消息遞送率 | ~80-90% | 100% | +10-20% |
| Pod Crash 檢測延遲 | N/A | < 30s | 新功能 |
| 路由清理延遲 | 永不清理 | < 1min | 新功能 |
| 額外 Redis 操作 | 0 | +2/10s/pod | 可忽略 |
| 額外記憶體開銷 | 0 | ~100KB/pod | 可忽略 |

### 業務價值

- ✅ **零消息丟失**: 關鍵業務通知（訂單、支付）保證送達
- ✅ **自動恢復**: Pod crash 後自動檢測並補償
- ✅ **運維簡化**: 無需手動清理殘留路由
- ✅ **用戶體驗**: 重連後立即收到所有錯過的消息

---

## 🔄 回滾計劃

### 風險評估

- **風險等級**: 🟡 MEDIUM
- **影響範圍**: SSE Manager 消息發送邏輯
- **回滾複雜度**: 簡單（移除新增的 Goroutine）

### 回滾步驟

如果優化導致問題，可以快速回滾：

1. **停用心跳檢查**
   ```go
   // 註釋掉健康檢查邏輯
   // if exists == 0 { ... }
   ```

2. **停止清理任務**
   ```go
   // 註釋掉清理任務啟動
   // go manager.startRouteCleanupTask()
   ```

3. **重新部署**
   - 部署不包含優化的版本
   - Redis 中的健康心跳會自動過期（30秒）

---

## 📚 參考資料

### 相關文檔

- [Phase 1: Domain Layer](phase-1.md)
- [Phase 3: Adapter Layer](phase-3.md)
- [技術細節](technical-details.md)
- [運維指南](OPERATIONS_GUIDE.md)

### 外部參考

- [Redis SET 命令文檔](https://redis.io/commands/set/)
- [Redis EXISTS 命令文檔](https://redis.io/commands/exists/)
- [Go Context 最佳實踐](https://go.dev/blog/context)

---

**維護者**: Development Team
**最後更新**: 2026-02-09
**下一次審查**: 實施完成後
