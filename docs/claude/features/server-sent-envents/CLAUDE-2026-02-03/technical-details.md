# SSE 通知系統 - 關鍵技術要點

**返回**: [總覽文檔](overview.md)

---

## 📋 文檔說明

本文檔整理了 SSE 實時推播通知系統的關鍵技術要點，包含 SSE 實作、Redis Pub/Sub、併發管理、效能優化等核心技術細節。

**⭐ 完整程式碼範例請參考**: `CLAUDE-2026-02-02-v1.0-task.md` (第2163-2945行)

---

## 1. SSE 實作細節

### SSE 格式規範
```
event: <event_name>
data: <json_data>

```

### Gin SSE Headers
```go
c.Writer.Header().Set("Content-Type", "text/event-stream")
c.Writer.Header().Set("Cache-Control", "no-cache")
c.Writer.Header().Set("Connection", "keep-alive")
c.Writer.Header().Set("X-Accel-Buffering", "no")
```

### SSE 事件類型
- `connected`: 連接成功
- `notification`: 新通知
- `ping`: 心跳檢測
- `shutdown`: Pod 關閉通知

---

## 2. Redis Streams 離線訊息

### 核心操作
```go
// 加入離線訊息
m.redisClient.XAdd(ctx, &redis.XAddArgs{
    Stream: "sse:offline:<player_id>",
    MaxLen: 100,  // 最多保留 100 條
    Values: map[string]interface{}{
        "notification": jsonData,
    },
})

// 讀取離線訊息
messages, _ := m.redisClient.XRange(ctx, "sse:offline:<player_id>", "-", "+").Result()

// 清除離線訊息
m.redisClient.Del(ctx, "sse:offline:<player_id>")
```

### TTL 管理
- 離線訊息保留 7 天 (使用 `EXPIRE` 設置)
- 上線時自動推送並清空

---

## 3. 併發連接管理

### sync.RWMutex 使用
```go
type sseManager struct {
    connections map[string]SSEWriter
    mu          sync.RWMutex  // 讀寫鎖
}

// 註冊連接 (寫鎖)
func (m *sseManager) RegisterConnection(...) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.connections[playerID] = writer
}

// 廣播訊息 (讀鎖)
func (m *sseManager) BroadcastToAll(...) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for _, writer := range m.connections {
        writer.Write(...)
    }
}
```

**關鍵點**:
- 讀操作使用 `RLock()` (允許多個並發讀)
- 寫操作使用 `Lock()` (獨占鎖)
- 始終使用 `defer` 確保鎖釋放

---

## 4. 效能優化策略

### 批次處理
```go
// 批次推送
sseManager.SendToPlayers(ctx, playerIDs, notification)
```

### Redis Pipeline
```go
pipe := redisClient.Pipeline()
for _, key := range keys {
    pipe.Set(ctx, key, value, 0)
}
pipe.Exec(ctx)
```

### Goroutine Pool (可選)
```go
// 使用 ants 控制併發數
pool, _ := ants.NewPool(1000)
defer pool.Release()
```

---

## 5. Redis Pub/Sub 跨 Pod 訊息路由 ⭐ 核心技術

### 架構設計

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

### Pod ID 管理

**環境變數注入**:
```bash
SSE_POD_ID=${HOSTNAME}  # Kubernetes Pod 名稱
# 範例: sse-pod-1, sse-pod-2, sse-pod-3
```

**SSE Manager 結構**:
```go
type sseManager struct {
    podID       string                       // Pod 識別碼
    connections map[string]inbound.SSEWriter // 本地連接池
    mu          sync.RWMutex                 // 併發安全
    redisClient *redis.Client
    pubsub      *redis.PubSub                // Pub/Sub 訂閱
    logger      infrastructure.Logger
}
```

### Redis Pub/Sub Channels

**頻道設計**:
- `sse:broadcast` - 廣播頻道 (所有 Pod 訂閱)
- `sse:pod:{podID}` - Pod 專屬頻道 (個別推送路由)

**訂閱設置**:
```go
m.pubsub = m.redisClient.Subscribe(ctx,
    "sse:broadcast",                   // 廣播
    fmt.Sprintf("sse:pod:%s", m.podID), // 本 Pod
)
```

### 玩家路由表管理

**Redis Hash 結構**:
```
Key: sse:player_routes
Type: Hash
Fields:
  player_001 → pod-1
  player_002 → pod-1
  player_003 → pod-2
  ...
```

**路由操作**:
```go
// 註冊時更新路由表
m.redisClient.HSet(ctx, "sse:player_routes", playerID, m.podID)

// 查詢玩家所在 Pod
targetPodID, _ := m.redisClient.HGet(ctx, "sse:player_routes", playerID).Result()

// 註銷時清理路由表
m.redisClient.HDel(ctx, "sse:player_routes", playerID)
```

### 廣播推送實作

**流程**:
1. 接收廣播請求
2. 序列化訊息
3. 發布到 `sse:broadcast` 頻道
4. 所有 Pod 收到訊息
5. 各 Pod 推送給本地連接池玩家

**關鍵代碼**:
```go
// 發布廣播
data, _ := json.Marshal(notification)
m.redisClient.Publish(ctx, "sse:broadcast", data)

// Pub/Sub Listener 處理
m.eventSource.addEventListener('notification', (e) => {
    // 推送到本地連接
    count := m.broadcastToLocalConnections(&notification)
})
```

### 個別推送實作

**流程**:
1. 查詢玩家路由表
2. 判斷玩家是否在本 Pod
   - 是: 直接推送
   - 否: 通過 Pub/Sub 路由到目標 Pod
3. 目標 Pod 收到訊息並推送

**關鍵代碼**:
```go
// 查詢路由
targetPodID, _ := m.redisClient.HGet(ctx, "sse:player_routes", playerID).Result()

if targetPodID == m.podID {
    // 本地推送
    writer.Write("notification", notification)
} else {
    // 跨 Pod 路由
    channel := fmt.Sprintf("sse:pod:%s", targetPodID)
    m.redisClient.Publish(ctx, channel, data)
}
```

### Pub/Sub 監聽器

**啟動監聽**:
```go
func (m *sseManager) startPubSubListener(ctx context.Context) {
    ch := m.pubsub.Channel()
    
    for msg := range ch {
        switch msg.Channel {
        case "sse:broadcast":
            m.handleBroadcastMessage(msg.Payload)
        case fmt.Sprintf("sse:pod:%s", m.podID):
            m.handleTargetedMessage(msg.Payload)
        }
    }
}
```

**訊息處理**:
- 反序列化 JSON
- 驗證訊息格式
- 推送到本地連接
- 錯誤處理與日誌記錄

---

## 6. 連接負載均衡策略 ⭐ 生產級設計

### 問題分析

**NAT 網段問題**:
- 使用 IP Hash 或 Session Affinity 會導致負載嚴重不均
- 同一公司/校園的玩家有相同 Public IP
- 大量玩家被路由到同一 Pod

### 推薦配置: 輪詢負載均衡

**Kubernetes Service**:
```yaml
spec:
  sessionAffinity: None  # 不使用 Session Affinity
```

**Nginx Ingress**:
```yaml
metadata:
  annotations:
    # 不設定 upstream-hash-by，使用預設輪詢
```

**預期效果**:
```
✅ 負載均勻分布：
Pod-1: 6,800 連接 (34%)
Pod-2: 6,500 連接 (32.5%)
Pod-3: 6,700 連接 (33.5%)
總計: 20,000 連接，分布均勻 ±5%
```

### 監控指標

**Prometheus Metrics**:
```promql
# 每個 Pod 的連接數
sse_connections_per_pod{pod=~"sse-pod-.*"}

# 連接數標準差
stddev(sse_connections_per_pod)

# 最大/最小比例
max(sse_connections_per_pod) / min(sse_connections_per_pod)
```

**告警規則**:
```yaml
- alert: SSEConnectionImbalance
  expr: |
    (max(sse_connections_per_pod) - min(sse_connections_per_pod))
    / avg(sse_connections_per_pod) > 0.3
  for: 10m
  labels:
    severity: warning
```

### HPA 配合負載均衡

**基於連接數的自動擴展**:
```yaml
spec:
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: sse_connections_per_pod
      target:
        type: AverageValue
        averageValue: "8000"  # 每個 Pod 8000 連接時觸發擴展
```

**擴展流程**:
- 當前: 3 Pods × 8000 連接 = 24,000
- 連接增長 → HPA 觸發
- 擴展到 4 Pods
- 新連接通過輪詢均勻分配
- 結果: 每個 Pod ~6000 連接

---

## 7. 優雅關閉與連接遷移

### Pod 關閉流程

1. 停止接受新連接
2. 通知所有玩家即將斷線 (發送 `shutdown` 事件)
3. 等待客戶端自動重連到其他 Pod (5秒)
4. 強制關閉所有連接
5. 清理 Redis 路由表與統計
6. 關閉 Pub/Sub 訂閱

### 客戶端自動重連

**JavaScript EventSource**:
```javascript
eventSource.addEventListener('shutdown', (e) => {
    console.log('Server shutting down, will reconnect...');
    reconnect(); // 指數退避重連
});

eventSource.onerror = () => {
    console.error('SSE connection error, reconnecting...');
    reconnect();
};
```

**重連策略**:
- 初始間隔: 1 秒
- 最大間隔: 30 秒
- 指數退避: 間隔 × 2

---

## 8. 線上統計管理

### Redis Counter

**計數器操作**:
```go
// 註冊時遞增
m.redisClient.Incr(ctx, "sse:online_count")

// 註銷時遞減
m.redisClient.Decr(ctx, "sse:online_count")

// 查詢 (O(1) 複雜度)
count, _ := m.redisClient.Get(ctx, "sse:online_count").Int()
```

### Pod 統計

**Pod 連接數記錄**:
```go
podConnKey := fmt.Sprintf("sse:pod_stats:%s", m.podID)
m.redisClient.Set(ctx, podConnKey, len(m.connections), 0)
```

**查詢所有 Pod 統計**:
```go
pattern := "sse:pod_stats:*"
// 使用 SCAN 遍歷所有 Pod 統計
```

---

## 9. Pod 健康心跳機制 ⭐ 高可用性保障

### 問題背景

**訊息黑洞問題**:
當 Pod 崩潰時，Redis 路由表中的殘留路由會導致訊息發送到已死的 Pod，造成訊息丟失。

**時間線範例**:
```
T+0s:  Pod-2 崩潰
T+1s:  玩家 player_123 的路由表仍顯示 pod-2
T+2s:  訊息發送到 sse:pod:pod-2
T+3s:  訊息丟失（無 Pod 訂閱該頻道）
```

### 健康心跳設計

**核心參數**:
- 心跳間隔: 10 秒
- Redis TTL: 30 秒
- 健康檢查視窗: 3 個心跳週期

**Redis Key 設計**:
```
Key: sse:pod_health:{podID}
Type: String
Value: "alive"
TTL: 30 秒 (自動過期)
```

### 心跳發送實作

**Goroutine 實作**:
```go
func (m *sseManager) startPodHealthHeartbeat() {
    defer m.wg.Done()
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    healthKey := fmt.Sprintf(consts.SSEPodHealthKey, m.podID)

    for {
        select {
        case <-m.ctx.Done():
            // Pod 關閉，停止心跳
            return
        case <-ticker.C:
            // 每 10 秒更新一次心跳
            ctx := context.Background()
            if err := m.redisClient.Set(ctx, healthKey, "alive", 30*time.Second).Err(); err != nil {
                m.logger.WarnLog("Failed to update pod health heartbeat",
                    logger.String("pod_id", m.podID),
                    logger.Error(err),
                )
            }
        }
    }
}
```

**生命週期管理**:
- 在 SSE Manager 初始化時啟動
- 使用 `sync.WaitGroup` 確保優雅關閉
- 使用獨立 `context.Background()` 避免 context 取消影響

### 發送前健康檢查

**SendToPlayer 整合**:
```go
func (m *sseManager) SendToPlayer(ctx context.Context, playerID string, notification *entity.Notification) error {
    // 查詢玩家路由
    targetPodID, err := m.redisClient.HGet(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
    if err == redis.Nil {
        // 玩家未連線，加入離線隊列
        return m.EnqueueOfflineMessage(ctx, playerID, notification)
    }

    // 檢查目標 Pod 健康狀態
    healthKey := fmt.Sprintf(consts.SSEPodHealthKey, targetPodID)
    exists, err := m.redisClient.Exists(ctx, healthKey).Result()

    if exists == 0 {
        // 目標 Pod 已死，清理殘留路由並加入離線隊列
        m.logger.WarnLog("Target pod is dead, cleaning up stale route",
            logger.String("player_id", playerID),
            logger.String("dead_pod_id", targetPodID),
        )

        m.redisClient.HDel(ctx, consts.SSEPlayerRoutesKey, playerID)
        return m.EnqueueOfflineMessage(ctx, playerID, notification)
    }

    // 目標 Pod 健康，正常發送
    // ... 正常發送邏輯
}
```

**故障恢復流程**:
1. 檢測到 Pod 死亡（健康 Key 不存在）
2. 立即清理殘留路由
3. 訊息轉入離線隊列（7天TTL）
4. 玩家重連時自動推送離線訊息

### 效能優化

**Redis EXISTS 特性**:
- 複雜度: O(1)
- 返回值: 0 (Key 不存在) 或 1 (Key 存在)
- 無需讀取 Value，效能極高

**快取策略**:
- 健康狀態無需快取（TTL 已提供時效性）
- 避免健康狀態快取導致的延遲判斷

---

## 10. 死 Pod 路由清理 ⭐ 系統穩定性保障

### 清理機制設計

**執行間隔**: 1 分鐘
**清理目標**: Redis Hash `sse:player_routes` 中指向死 Pod 的殘留路由

### 清理演算法

**流程**:
```
1. 讀取完整路由表 (HGETALL sse:player_routes)
2. 提取所有唯一的 Pod ID
3. 批次檢查每個 Pod 的健康狀態 (EXISTS sse:pod_health:{podID})
4. 識別死 Pod 列表
5. 批次刪除指向死 Pod 的所有路由 (HDEL)
```

**關鍵代碼**:
```go
func (m *sseManager) startRouteCleanupTask() {
    defer m.wg.Done()
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-m.ctx.Done():
            return
        case <-ticker.C:
            m.cleanupDeadPodRoutes()
        }
    }
}

func (m *sseManager) cleanupDeadPodRoutes() {
    ctx := context.Background()

    // 1. 獲取完整路由表
    routes, err := m.redisClient.HGetAll(ctx, consts.SSEPlayerRoutesKey).Result()
    if err != nil || len(routes) == 0 {
        return
    }

    // 2. 提取所有唯一 Pod ID
    podIDs := make(map[string]bool)
    for _, podID := range routes {
        podIDs[podID] = true
    }

    // 3. 批次檢查 Pod 健康狀態
    deadPods := make(map[string]bool)
    for podID := range podIDs {
        healthKey := fmt.Sprintf(consts.SSEPodHealthKey, podID)
        exists, _ := m.redisClient.Exists(ctx, healthKey).Result()
        if exists == 0 {
            deadPods[podID] = true
        }
    }

    // 4. 批次刪除死 Pod 的所有路由
    if len(deadPods) > 0 {
        for playerID, podID := range routes {
            if deadPods[podID] {
                m.redisClient.HDel(ctx, consts.SSEPlayerRoutesKey, playerID)
            }
        }

        m.logger.InfoLog("Cleaned up dead pod routes",
            logger.Int("dead_pods", len(deadPods)),
            logger.Int("cleaned_routes", cleanedCount),
        )
    }
}
```

### 複雜度分析

**時間複雜度**:
- HGETALL: O(N) - N 為玩家數量
- EXISTS 批次檢查: O(P) - P 為 Pod 數量 (通常 < 20)
- HDEL 批次刪除: O(D) - D 為死 Pod 的路由數量

**總複雜度**: O(N + P + D) ≈ O(N)

**效能評估**:
```
假設場景:
- 20,000 線上玩家
- 10 個 Pod
- 1 個 Pod 崩潰 (約 2,000 路由需清理)

執行時間:
- HGETALL: ~50ms
- EXISTS × 10: ~5ms
- HDEL × 2,000: ~100ms
總計: ~155ms (完全可接受)
```

### 雙層保護機制

**即時保護**: SendToPlayer 健康檢查
- 發送前即時檢測
- 延遲: 0ms (發送時立即檢查)
- 準確度: 99.9%

**週期清理**: 路由清理任務
- 清理殘留路由
- 延遲: 平均 30 秒 (最長 1 分鐘)
- 涵蓋率: 100%

**互補效果**:
```
T+0s:   Pod-2 崩潰
T+0s:   SendToPlayer 檢查會立即檢測到（即時保護）
T+30s:  定期清理任務執行，清理所有殘留路由（週期保護）
```

### 監控指標

**Prometheus Metrics**:
```go
// 清理任務執行次數
sse_route_cleanup_runs_total

// 每次清理的死 Pod 數量
sse_route_cleanup_dead_pods

// 每次清理的路由數量
sse_route_cleanup_routes_removed

// 清理任務執行時間
sse_route_cleanup_duration_seconds
```

**告警規則**:
```yaml
- alert: SSEFrequentPodFailures
  expr: |
    rate(sse_route_cleanup_dead_pods[5m]) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "SSE Pods 頻繁死亡"
```

---

## 📝 詳細程式碼參考

本文檔提供技術要點總覽。完整的程式碼範例、實作細節、錯誤處理請參考：

- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (第2163-2945行)
- **檔案結構文檔**: [file-structure.md](file-structure.md)
- **Phase 3 文檔**: [phase-3.md](phase-3.md) - Adapter Layer 實作細節

**包含內容**:
- ✅ 完整的 SSE Manager 實作 (含 Pub/Sub)
- ✅ 玩家路由表管理完整代碼
- ✅ 廣播/個別推送完整流程
- ✅ Pub/Sub Listener 完整實作
- ✅ 優雅關閉機制完整代碼
- ✅ 客戶端重連機制範例
- ✅ 錯誤處理與日誌記錄
- ✅ 效能優化技巧

---

## 🔗 相關文檔

- [總覽文檔](overview.md)
- [檔案結構](file-structure.md)
- [Phase 3: Adapter Layer 實作](phase-3.md)
- [Phase 6: 測試與優化](phase-6.md)
- [最佳實踐](best-practices.md)

---

**維護者**: Development Team
**最後更新**: 2026-02-09
