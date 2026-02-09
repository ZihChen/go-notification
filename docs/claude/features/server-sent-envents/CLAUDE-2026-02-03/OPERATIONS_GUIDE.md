# SSE 通知系統運維指南

**版本**: v1.1
**最後更新**: 2026-02-09
**目標讀者**: DevOps 工程師、系統管理員

---

## 📋 目錄

1. [環境變數配置](#環境變數配置)
2. [Redis 配置最佳實踐](#redis-配置最佳實踐)
3. [多 Pod 部署指南](#多-pod-部署指南)
4. [效能調優](#效能調優)
5. [監控與日誌](#監控與日誌)
6. [故障排查](#故障排查)
7. [安全性建議](#安全性建議)

---

## 環境變數配置

### 必要環境變數

```bash
# 服務配置
SSE_SERVICE_PORT=8081                    # SSE Service 監聽埠號
SSE_POD_ID=sse-pod-1                     # Pod 唯一識別碼（多 Pod 部署時必須唯一）

# Redis 配置
REDIS_HOST=localhost                      # Redis 主機地址
REDIS_PORT=6379                          # Redis 端口
REDIS_PASSWORD=                          # Redis 密碼（選填）
REDIS_DB=0                               # Redis 資料庫編號

# Redis 連接池配置
REDIS_POOL_SIZE=100                      # 最大連接數
REDIS_MIN_IDLE_CONNS=10                  # 最小空閒連接數
REDIS_MAX_RETRIES=3                      # 最大重試次數
REDIS_DIAL_TIMEOUT=5s                    # 連接超時
REDIS_READ_TIMEOUT=3s                    # 讀取超時
REDIS_WRITE_TIMEOUT=3s                   # 寫入超時

# JWT 配置
JWT_SECRET=your-jwt-secret-key           # JWT 簽名密鑰
JWT_EXPIRATION=24h                       # JWT 過期時間

# 日誌配置
LOG_LEVEL=info                           # 日誌級別：debug|info|warn|error
LOG_FORMAT=json                          # 日誌格式：json|text

# OpenTelemetry 配置（選填）
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=sse-notification-service
```

### 開發環境範例 (.env.development)

```bash
# Service
SSE_SERVICE_PORT=8081
SSE_POD_ID=sse-dev-1

# Redis (本地)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Connection Pool (開發環境可較小)
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2

# JWT
JWT_SECRET=dev-secret-key-change-in-production
JWT_EXPIRATION=24h

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

### 生產環境範例 (.env.production)

```bash
# Service
SSE_SERVICE_PORT=8081
SSE_POD_ID=${HOSTNAME}                   # 使用 Kubernetes Pod 名稱作為 ID

# Redis (生產集群)
REDIS_HOST=redis-cluster.production.svc.cluster.local
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD_SECRET}  # 從 Kubernetes Secret 注入
REDIS_DB=0

# Connection Pool (生產環境需較大)
REDIS_POOL_SIZE=200
REDIS_MIN_IDLE_CONNS=50
REDIS_MAX_RETRIES=5
REDIS_DIAL_TIMEOUT=10s
REDIS_READ_TIMEOUT=5s
REDIS_WRITE_TIMEOUT=5s

# JWT
JWT_SECRET=${JWT_SECRET_KEY}             # 從 Kubernetes Secret 注入
JWT_EXPIRATION=24h

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_SERVICE_NAME=sse-notification-service
```

---

## Redis 配置最佳實踐

### 1. Redis 部署模式

#### 開發環境：Standalone
```bash
# Docker Compose
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
```

#### 生產環境：Redis Cluster 或 Sentinel

**推薦**: Redis Cluster（高可用 + 自動分片）

```yaml
# Kubernetes: Redis Cluster
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-cluster
spec:
  serviceName: redis-cluster
  replicas: 6  # 3 master + 3 replica
  template:
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command:
        - redis-server
        - --cluster-enabled yes
        - --cluster-config-file /data/nodes.conf
        - --cluster-node-timeout 5000
        - --appendonly yes
```

### 2. Redis 效能調優

#### 核心參數

```bash
# redis.conf

# 記憶體配置
maxmemory 4gb                              # 最大記憶體使用量
maxmemory-policy allkeys-lru              # 記憶體淘汰策略

# 持久化配置（根據需求選擇）
# RDB: 適合災難恢復
save 900 1                                 # 900秒內至少1次寫入則保存
save 300 10
save 60 10000

# AOF: 適合高可靠性要求
appendonly yes                             # 啟用 AOF
appendfsync everysec                       # 每秒同步一次

# 網路配置
timeout 300                                # 客戶端閒置超時（秒）
tcp-keepalive 60                           # TCP keepalive

# Pub/Sub 配置
client-output-buffer-limit pubsub 32mb 8mb 60  # Pub/Sub 緩衝區限制
```

### 3. Redis 監控指標

**關鍵指標**:
- `connected_clients`: 當前連接數
- `used_memory`: 記憶體使用量
- `keyspace_hits / keyspace_misses`: 快取命中率
- `pubsub_channels`: Pub/Sub 頻道數量
- `pubsub_patterns`: Pub/Sub 模式數量

**監控命令**:
```bash
# 查看 Pub/Sub 狀態
redis-cli PUBSUB CHANNELS
redis-cli PUBSUB NUMSUB sse:broadcast

# 查看記憶體使用
redis-cli INFO MEMORY

# 查看連接數
redis-cli INFO CLIENTS

# 查看 Streams 狀態
redis-cli XINFO STREAM sse:offline:player-001
```

---

## 多 Pod 部署指南

### 架構概覽

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  SSE Pod 1  │     │  SSE Pod 2  │     │  SSE Pod 3  │
│  (8081)     │     │  (8081)     │     │  (8081)     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
                  ┌────────────────┐
                  │  Redis Cluster │
                  │  (Pub/Sub Bus) │
                  └────────────────┘
```

### Kubernetes 部署配置

#### 1. Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sse-notification-service
  labels:
    app: sse-notification
spec:
  replicas: 3                              # 3 個 Pod 實例
  selector:
    matchLabels:
      app: sse-notification
  template:
    metadata:
      labels:
        app: sse-notification
    spec:
      containers:
      - name: sse-service
        image: your-registry/sse-notification-service:latest
        ports:
        - containerPort: 8081
          name: http
        env:
        - name: SSE_SERVICE_PORT
          value: "8081"
        - name: SSE_POD_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name      # 使用 Pod 名稱作為 Pod ID
        - name: REDIS_HOST
          value: "redis-cluster"
        - name: REDIS_PORT
          value: "6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: jwt-secret
              key: secret-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 10
```

#### 2. Service (LoadBalancer)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sse-notification-service
spec:
  type: LoadBalancer
  selector:
    app: sse-notification
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  sessionAffinity: ClientIP                # 重要：使用 Client IP 親和性
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600                 # SSE 連接親和性超時
```

#### 3. HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: sse-notification-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sse-notification-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 重要注意事項

#### ⚠️  Pod ID 必須唯一

每個 SSE Pod 必須有唯一的 `SSE_POD_ID`，建議使用：
- Kubernetes Pod Name: `metadata.name`
- Hostname: `${HOSTNAME}`

#### ⚠️  Session Affinity 配置

SSE 連接需要保持與同一 Pod 的連接，必須配置：
```yaml
sessionAffinity: ClientIP
```

#### ⚠️  Graceful Shutdown

確保 Pod 終止時優雅關閉 SSE 連接：
```go
// 處理 SIGTERM 信號
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
<-c

// 關閉 SSE Manager（會自動斷開所有連接）
sseManager.Shutdown(context.Background())
```

---

## 效能調優

### 1. Redis 連接池調優

**指標監控**:
```go
stats := redisClient.PoolStats()
log.Printf("Pool Stats:")
log.Printf("  TotalConns: %d", stats.TotalConns)
log.Printf("  IdleConns: %d", stats.IdleConns)
log.Printf("  Hits: %d", stats.Hits)
log.Printf("  Misses: %d", stats.Misses)
```

**調優建議**:
- `PoolSize`: 設置為預期最大併發連接數的 1.5-2 倍
- `MinIdleConns`: 設置為平均連接數
- 監控 `Hits / (Hits + Misses)` 比率，應 > 95%

### 2. SSE 連接數優化

**單 Pod 連接數上限建議**:
- 開發環境: 1,000 連接
- 生產環境: 10,000 連接

**計算 Pod 數量**:
```
所需 Pod 數 = 預期線上玩家數 / 單 Pod 連接上限
```

範例：100,000 線上玩家，單 Pod 上限 10,000
```
所需 Pod 數 = 100,000 / 10,000 = 10 個 Pod
```

### 3. Pub/Sub 訂閱優化

**避免過多訂閱**:
- 每個 Pod 只訂閱 2 個頻道：
  - `sse:broadcast` (廣播頻道)
  - `sse:pod:{podID}` (Pod 專屬頻道)

### 4. 離線訊息 TTL 優化

**Redis Streams MAXLEN 設置**:
```go
// 單個玩家最多保留 100 條離線訊息
client.XAdd(ctx, &redis.XAddArgs{
    Stream: fmt.Sprintf("sse:offline:%s", playerID),
    MaxLen: 100,
    Approx: true,  // 使用近似修剪以提升效能
    Values: map[string]interface{}{
        "notification": data,
    },
})
```

---

## 監控與日誌

### 關鍵監控指標

#### 1. 服務健康指標

| 指標 | 說明 | 告警閾值 |
|------|------|----------|
| `sse_connections_total` | 當前 SSE 連接總數 | > 單 Pod 上限的 90% |
| `sse_online_players_total` | 線上玩家總數 | - |
| `sse_broadcast_messages_total` | 廣播訊息總數 | - |
| `sse_individual_messages_total` | 個別推送訊息總數 | - |
| `sse_offline_messages_total` | 離線訊息總數 | - |

#### 2. Redis 指標

| 指標 | 說明 | 告警閾值 |
|------|------|----------|
| `redis_connected_clients` | Redis 連接數 | > 最大連接數的 80% |
| `redis_used_memory_bytes` | Redis 記憶體使用 | > 配置上限的 85% |
| `redis_pubsub_channels` | Pub/Sub 頻道數 | > 100 |

#### 3. 效能指標

| 指標 | 說明 | 告警閾值 |
|------|------|----------|
| `sse_broadcast_latency_ms` | 廣播延遲（P99） | > 100ms |
| `sse_routing_latency_ms` | 路由延遲（P99） | > 50ms |
| `sse_message_delivery_rate` | 訊息遞送率 | < 99% |

#### 4. Pod 健康與路由清理指標 ⭐ 新增

| 指標 | 說明 | 告警閾值 |
|------|------|----------|
| `sse_pod_health_heartbeat_errors_total` | Pod 健康心跳更新失敗次數 | > 10 次/分鐘 |
| `sse_route_cleanup_runs_total` | 路由清理任務執行次數 | - |
| `sse_route_cleanup_dead_pods` | 每次清理檢測到的死 Pod 數量 | > 1（持續 5 分鐘） |
| `sse_route_cleanup_routes_removed` | 每次清理移除的路由數量 | - |
| `sse_route_cleanup_duration_seconds` | 路由清理任務執行時間 | > 1 秒 |
| `sse_dead_pod_detected_total` | 發送訊息時檢測到死 Pod 次數 | > 100 次/分鐘 |

**Prometheus 查詢範例**:
```promql
# 檢測 Pod 頻繁死亡
rate(sse_route_cleanup_dead_pods[5m]) > 0.1

# 路由清理效率
histogram_quantile(0.99, sse_route_cleanup_duration_seconds)

# 死 Pod 檢測頻率（可能表示 Pod 不穩定）
rate(sse_dead_pod_detected_total[5m])
```

**告警規則範例**:
```yaml
# Pod 頻繁死亡告警
- alert: SSEFrequentPodFailures
  expr: |
    rate(sse_route_cleanup_dead_pods[5m]) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "SSE Pods 頻繁死亡"
    description: "過去 5 分鐘內平均每分鐘檢測到 {{ $value }} 個死 Pod"

# 健康心跳失敗告警
- alert: SSEPodHealthHeartbeatFailing
  expr: |
    rate(sse_pod_health_heartbeat_errors_total[5m]) > 10
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Pod 健康心跳更新失敗"
    description: "Pod {{ $labels.pod_id }} 過去 5 分鐘內健康心跳更新失敗率過高"

# 路由清理任務耗時告警
- alert: SSERouteCleanupSlow
  expr: |
    histogram_quantile(0.99, sse_route_cleanup_duration_seconds) > 1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "路由清理任務執行緩慢"
    description: "路由清理任務 P99 執行時間超過 1 秒"
```

### 日誌最佳實踐

#### 結構化日誌格式

```json
{
  "timestamp": "2026-02-04T10:00:00Z",
  "level": "info",
  "message": "Player connected",
  "player_id": "player-001",
  "pod_id": "sse-pod-1",
  "connection_count": 1523
}
```

#### 重要日誌事件

1. **連接事件**:
   - `Player connected`
   - `Player disconnected`

2. **訊息事件**:
   - `Broadcast published to Redis`
   - `Message routed to target pod`
   - `Targeted message delivered successfully`

3. **錯誤事件**:
   - `Failed to publish to Redis`
   - `Failed to send notification to player`
   - `Redis connection error`

---

## 故障排查

### 常見問題

#### 1. 玩家無法建立 SSE 連接

**症狀**: 前端無法建立 EventSource 連接

**檢查步驟**:
```bash
# 1. 檢查 JWT Token 是否有效
curl -H "Authorization: Bearer <token>" \
  http://localhost:8081/api/v1/notifications/stream

# 2. 檢查 Pod 健康狀態
kubectl get pods -l app=sse-notification

# 3. 檢查日誌
kubectl logs -f <pod-name> | grep "SSE connection"
```

**常見原因**:
- JWT Token 過期或無效
- Pod 未就緒（ReadinessProbe 失敗）
- Load Balancer 超時設置過短

#### 2. 訊息推送失敗

**症狀**: API 呼叫成功但玩家未收到通知

**檢查步驟**:
```bash
# 1. 檢查 Redis Pub/Sub
redis-cli PUBSUB CHANNELS

# 2. 檢查玩家路由表
redis-cli HGET sse:player_routes player-001

# 3. 檢查離線訊息佇列
redis-cli XLEN sse:offline:player-001
```

**常見原因**:
- 玩家離線（訊息進入離線佇列）
- Redis Pub/Sub 連接中斷
- 目標 Pod 已終止

#### 3. Redis 連接池耗盡

**症狀**: 日誌出現 "connection pool timeout"

**解決方案**:
```bash
# 1. 增加連接池大小
REDIS_POOL_SIZE=200

# 2. 檢查是否有連接洩漏
redis-cli CLIENT LIST | wc -l

# 3. 重啟 Pod 重置連接池
kubectl rollout restart deployment sse-notification-service
```

#### 4. Pod 健康心跳失敗 ⭐ 新增

**症狀**: 日誌出現 "Failed to update pod health heartbeat"

**檢查步驟**:
```bash
# 1. 檢查 Pod 到 Redis 的網路連接
kubectl exec -it <pod-name> -- redis-cli -h redis-cluster ping

# 2. 檢查 Redis 記憶體使用情況
redis-cli INFO MEMORY | grep used_memory_human

# 3. 檢查 Pod 健康心跳 Key
redis-cli GET sse:pod_health:sse-pod-1

# 4. 檢查 Pod 日誌中的錯誤
kubectl logs -f <pod-name> | grep "heartbeat"
```

**常見原因**:
- Redis 連接中斷或超時
- Redis 記憶體不足導致寫入失敗
- Pod 的 CPU 或記憶體資源不足，導致 Goroutine 阻塞
- 網路分區導致 Pod 無法連接到 Redis

**解決方案**:
```bash
# 1. 如果 Redis 連接問題，檢查網路或重啟 Redis
kubectl get pods -l app=redis-cluster

# 2. 如果記憶體不足，增加 Redis maxmemory 或清理舊數據
redis-cli CONFIG SET maxmemory 8gb

# 3. 如果 Pod 資源不足，調整 resources limits
kubectl edit deployment sse-notification-service
```

#### 5. 訊息發送到死 Pod（路由殘留）⭐ 新增

**症狀**:
- 日誌出現 "Target pod is dead, cleaning up stale route"
- 玩家明明在線但收不到訊息
- `sse_dead_pod_detected_total` 指標持續增加

**檢查步驟**:
```bash
# 1. 檢查路由表中是否有死 Pod 的路由
redis-cli HGETALL sse:player_routes

# 2. 檢查所有 Pod 的健康狀態
for pod in pod-1 pod-2 pod-3; do
  redis-cli EXISTS sse:pod_health:$pod
done

# 3. 檢查路由清理任務日誌
kubectl logs -f <pod-name> | grep "Cleaned up dead pod routes"

# 4. 手動觸發路由清理（等待最多 1 分鐘自動執行）
# 路由清理任務每分鐘自動執行一次
```

**常見原因**:
- Pod 崩潰後路由表未及時清理
- 路由清理任務執行失敗或被阻塞
- Pod 健康心跳 TTL 過長（30 秒）

**解決方案**:
```bash
# 1. 手動清理特定玩家的死 Pod 路由
redis-cli HDEL sse:player_routes player-001

# 2. 如果路由清理任務停止，重啟 Pod
kubectl rollout restart deployment sse-notification-service

# 3. 檢查並修復 Redis 連接問題
kubectl logs <pod-name> | grep "Redis"
```

**預防措施**:
- 監控 `sse_route_cleanup_dead_pods` 指標
- 設置告警規則檢測 Pod 頻繁死亡
- 確保 Pod 有足夠的資源配置
- 使用 Readiness Probe 確保 Pod 健康後才接收流量

#### 6. 路由清理任務執行緩慢 ⭐ 新增

**症狀**:
- `sse_route_cleanup_duration_seconds` P99 > 1 秒
- 日誌中路由清理任務執行時間過長

**檢查步驟**:
```bash
# 1. 檢查路由表大小
redis-cli HLEN sse:player_routes

# 2. 檢查 Redis 延遲
redis-cli --latency

# 3. 檢查 Pod 數量（影響健康檢查次數）
kubectl get pods -l app=sse-notification | wc -l

# 4. 檢查 Redis CPU 使用率
kubectl top pods -l app=redis-cluster
```

**效能分析**:
```
假設場景:
- 100,000 線上玩家路由
- 20 個 Pod
- 5 個死 Pod

預期執行時間:
- HGETALL 100,000 entries: ~500ms
- EXISTS × 20: ~10ms
- HDEL × 25,000 (5 dead pods): ~1250ms
總計: ~1.76 秒

如果超過 3 秒，表示有效能問題
```

**解決方案**:
```bash
# 1. 優化 Redis 性能（增加記憶體、使用 Pipeline）
# 2. 考慮分批清理（每次清理部分路由）
# 3. 增加 Redis 節點（使用 Redis Cluster 分片）
# 4. 監控並移除異常路由（單個玩家不應該有多個路由）
```

---

## 安全性建議

### 1. API Key 管理

- ✅ 使用 Kubernetes Secrets 儲存 API Key
- ✅ 定期輪換 API Key（建議每 90 天）
- ✅ 每個商戶使用獨立的 API Key
- ✅ 記錄所有 API Key 使用日誌

### 2. JWT Token 安全

- ✅ 使用強加密演算法（HS256 或 RS256）
- ✅ 設置合理的過期時間（建議 24 小時）
- ✅ 在 Refresh Token 機制中包含 Revocation List

### 3. Redis 安全

- ✅ 啟用 Redis AUTH（設置密碼）
- ✅ 使用 TLS 加密 Redis 連接（生產環境）
- ✅ 限制 Redis 網路訪問（防火牆規則）
- ✅ 定期備份 Redis 數據

### 4. 網路安全

- ✅ 使用 HTTPS（TLS/SSL）傳輸
- ✅ 配置 CORS 白名單
- ✅ 實施 Rate Limiting
- ✅ 啟用 DDoS 防護

---

## 災難恢復

### 備份策略

#### Redis 數據備份

```bash
# RDB 自動備份（每小時）
0 * * * * redis-cli BGSAVE

# AOF 備份（即時）
# 在 redis.conf 中啟用
appendonly yes
appendfsync everysec
```

#### Kubernetes 配置備份

```bash
# 匯出所有 SSE 相關資源
kubectl get deployment,service,hpa,secret \
  -l app=sse-notification \
  -o yaml > sse-backup.yaml
```

### 恢復流程

#### 1. Pod 故障恢復

Kubernetes 自動重啟故障 Pod：
```bash
# 檢查 Pod 狀態
kubectl get pods -l app=sse-notification

# 查看 Pod 重啟次數
kubectl describe pod <pod-name>
```

#### 2. Redis 數據恢復

```bash
# 從 RDB 備份恢復
cp dump.rdb /var/lib/redis/
redis-server --loadmodule /path/to/dump.rdb
```

#### 3. 完全災難恢復

```bash
# 1. 恢復 Redis
kubectl apply -f redis-cluster.yaml

# 2. 恢復 SSE Service
kubectl apply -f sse-backup.yaml

# 3. 驗證服務
kubectl get pods,svc -l app=sse-notification
```

---

**文檔維護**: DevOps Team
**聯繫方式**: [GitHub Issues](https://github.com/your-org/ms-notification-cat/issues)
