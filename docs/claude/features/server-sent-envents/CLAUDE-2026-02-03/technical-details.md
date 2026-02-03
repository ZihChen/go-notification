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
**最後更新**: 2026-02-02
