# SSE 通知系統 - 注意事項與最佳實踐

**返回**: [總覽文檔](overview.md)

---

## 📋 文檔說明

本文檔整理了 SSE 實時推播通知系統的開發注意事項與最佳實踐，涵蓋架構設計、安全性、效能、測試、監控等各個面向。

---

## 1. Clean Architecture 合規性

### 依賴方向規則

```
Infrastructure → Adapter → Application → Domain
       ↓            ↓            ↓          ↓
    實作細節    Ports實作    UseCase    純業務邏輯
```

### 各層職責

- ✅ **Domain Layer**: 
  - 不依賴任何外層
  - 純業務邏輯與實體定義
  - 定義 Ports 介面 (Inbound/Outbound)
  - 不包含技術實作細節

- ✅ **Application Layer**: 
  - 只依賴 Domain
  - 實作 UseCase 業務流程
  - 定義 DTO (資料傳輸物件)
  - 協調 Domain 實體與 Ports

- ✅ **Adapter Layer**: 
  - 實作 Domain Ports 介面
  - 處理外部依賴適配
  - HTTP Handler、Repository、SSE Manager
  - 不包含業務邏輯

- ✅ **Infrastructure Layer**: 
  - 提供技術實作細節
  - Database、Redis、KDS 配置
  - 依賴注入 (Wire)
  - 環境變數管理

### 驗證清單

- [ ] Domain Layer 無任何 import 外層套件
- [ ] Application Layer 只 import Domain
- [ ] Adapter Layer 實作所有 Ports 介面
- [ ] 依賴注入通過 Wire 自動生成
- [ ] 測試時可輕鬆 Mock 所有外部依賴

---

## 2. API 認證分離

### 認證機制設計

| 用途 | 路徑前綴 | 認證方式 | Middleware |
|------|---------|---------|-----------|
| **後端推播 API** | `/api/v1/admin/notifications/*` | API Key | `apiKeyAuth` |
| **前端接收 API** | `/api/v1/notifications/*` | JWT Token | `jwtAuth` |

### API Key 認證

**適用場景**: 後端服務、管理後台、第三方整合

**實作要點**:
```go
func apiKeyMiddleware(c *gin.Context) {
    apiKey := c.GetHeader("X-API-Key")
    
    if apiKey == "" {
        response.Unauthorized(c, "missing API key").Return()
        c.Abort()
        return
    }
    
    // 驗證 API Key
    if !validateAPIKey(apiKey) {
        response.Unauthorized(c, "invalid API key").Return()
        c.Abort()
        return
    }
    
    // 注入 merchant_id
    c.Set("global_merchant_id", extractMerchantID(apiKey))
    c.Next()
}
```

### JWT 認證

**適用場景**: 前端玩家 SSE 連接

**實作要點**:
```go
func jwtMiddleware(c *gin.Context) {
    token := extractToken(c)
    
    claims, err := validateJWT(token)
    if err != nil {
        response.Unauthorized(c, "invalid token").Return()
        c.Abort()
        return
    }
    
    // 注入 player_id
    c.Set("player_id", claims.PlayerID)
    c.Next()
}
```

### 安全建議

- ✅ API Key 儲存於安全環境變數或 Secret Manager
- ✅ JWT Token 設置合理過期時間 (例如 24 小時)
- ✅ 實施速率限制，防止暴力破解
- ✅ HTTPS 強制使用
- ✅ 定期輪換 API Key

---

## 3. 錯誤處理規範

### 錯誤定義

**位置**: `internal/domain/errmsg/errors.go`

**範例**:
```go
var (
    ErrPlayerNotFound = errors.New("player not found")
    ErrInvalidRequest = errors.New("invalid request")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
```

### 錯誤包裝

**使用 fmt.Errorf 保留錯誤鏈**:
```go
if err := repo.Create(ctx, notification); err != nil {
    return fmt.Errorf("create notification: %w", err)
}
```

### HTTP 錯誤回應

**統一使用 response 工具包**:
```go
// 成功回應
response.OK(c, "推送成功", result).Return()

// 錯誤回應
response.BadRequest(c, "invalid request", err.Error()).Return()
response.InternalServerError(c, "server error", err.Error()).Return()
response.Unauthorized(c, "unauthorized").Return()
```

### 日誌記錄

**使用 infrastructure.Logger**:
```go
logger.InfoWithContext(ctx, "notification sent",
    logger.String("notification_id", id),
    logger.Int("player_count", count))

logger.ErrorWithContext(ctx, "failed to send notification",
    logger.Error("err", err),
    logger.String("notification_id", id))
```

### 錯誤處理最佳實踐

- ✅ 每一層都要處理錯誤，不要忽略
- ✅ 使用 `%w` 包裝錯誤，保留堆疊資訊
- ✅ 記錄關鍵錯誤到日誌
- ✅ 返回使用者友善的錯誤訊息
- ✅ 敏感資訊不要暴露在錯誤訊息中

---

## 4. 測試策略

### 測試金字塔

```
      /\
     /  \    E2E Tests (少量)
    /────\
   /      \  Integration Tests (適量)
  /────────\
 /          \ Unit Tests (大量)
/____________\
```

### 單元測試

**目標**: UseCase 層 100% 覆蓋

**工具**: `testing`, `testify/assert`, `testify/mock`

**範例**:
```go
func TestBroadcastNotification(t *testing.T) {
    mockSSEManager := new(mocks.MockSSEManager)
    mockEventService := new(mocks.MockEventService)
    
    useCase := NewSSENotificationUseCase(
        mockSSEManager,
        mockEventService,
        mockLogger,
    )
    
    mockSSEManager.On("BroadcastToAll", mock.Anything, mock.Anything).
        Return(1000, nil)
    
    result, err := useCase.BroadcastNotification(ctx, req)
    
    assert.NoError(t, err)
    assert.Equal(t, 1000, result.TotalPlayers)
    mockSSEManager.AssertExpectations(t)
}
```

### 整合測試

**目標**: 驗證多個組件協作

**範例**:
- Handler + UseCase + Mock Repository
- SSE Manager + Redis (使用 miniredis)
- 完整 HTTP 請求測試

### 壓力測試

**工具**: wrk, ab, locust

**範例**:
```bash
# 廣播吞吐量測試
wrk -t 10 -c 1000 -d 30s --latency \
    -H "X-API-Key: test-key" \
    -s broadcast.lua \
    http://localhost:8080/api/v1/admin/notifications/broadcast
```

### Mock 策略

**統一 Mock 架構**:
```go
// test/mocks/repository_mocks.go
type MockSSEManager struct {
    mock.Mock
}

func (m *MockSSEManager) BroadcastToAll(ctx context.Context, notification *entity.SSENotification) (int, error) {
    args := m.Called(ctx, notification)
    return args.Int(0), args.Error(1)
}
```

### 測試覆蓋率目標

| 層級 | 目標覆蓋率 | 優先級 |
|------|-----------|--------|
| Domain Layer | 100% | 🔥 最高 |
| Application Layer | 100% | 🔥 最高 |
| Adapter Layer | 80%+ | ⭐ 高 |
| Infrastructure Layer | 60%+ | ✅ 中 |

---

## 5. 效能目標

### 關鍵指標

| 指標 | 目標 | 驗收標準 | 測試工具 |
|------|------|----------|---------|
| **廣播吞吐量** | > 10,000 QPS | wrk 測試 30s | wrk |
| **訊息入隊延遲** | < 100ms (P99) | 壓力測試 | wrk + pprof |
| **並發連接數** | 10,000+ | SSE 連接測試 | 自定義測試腳本 |
| **訊息遞送率** | 99%+ | 統計資料驗證 | Prometheus |
| **跨 Pod 路由延遲** | < 100ms (P99) | 多 Pod 測試 | 自定義測試 |

### 效能優化技巧

#### 批次處理
- ✅ 批次推送 (SendToPlayers)
- ✅ Redis Pipeline
- ✅ GORM CreateInBatches

#### 連接池優化
```go
// GORM 連接池
db.DB().SetMaxOpenConns(100)
db.DB().SetMaxIdleConns(10)
db.DB().SetConnMaxLifetime(time.Hour)

// Redis 連接池
redisClient := redis.NewClient(&redis.Options{
    PoolSize:     100,
    MinIdleConns: 10,
})
```

#### Goroutine Pool
```go
// 使用 ants 控制併發數
pool, _ := ants.NewPool(1000)
defer pool.Release()
```

#### 避免 N+1 查詢
- ✅ 使用 Preload 預載關聯資料
- ✅ 批次查詢取代循環查詢
- ✅ Redis Pipeline 減少往返次數

### 效能監控

**Prometheus Metrics**:
```go
// 訊息推送計數
sse_message_sent_total{type="broadcast",status="success"}

// 推送延遲
sse_message_latency_seconds{quantile="0.99"}

// 連接數
sse_connections_total{pod="sse-pod-1"}
```

---

## 6. 安全性要求

### 認證授權

- ✅ API Key 驗證正確實施
- ✅ JWT Token 驗證與過期處理
- ✅ 不同 API 使用不同認證機制
- ✅ 認證失敗返回 401 Unauthorized

### 審計日誌

**所有推播事件記錄至 KDS**:
```go
eventService.PublishEvent(ctx, "notification.broadcast", map[string]interface{}{
    "broadcast_id": broadcastID,
    "merchant_id":  req.GlobalMerchantID,
    "queued_count": queuedCount,
    "type":         notification.Type,
    "priority":     notification.Priority,
    "timestamp":    now.Unix(),
})
```

### 速率限制

**防止濫用**:
```go
// 使用 Redis + Lua 實現速率限制
func rateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := fmt.Sprintf("rate_limit:%s", c.ClientIP())
        
        count, _ := redisClient.Incr(ctx, key).Result()
        if count == 1 {
            redisClient.Expire(ctx, key, window)
        }
        
        if count > int64(limit) {
            response.TooManyRequests(c, "rate limit exceeded").Return()
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### 資料驗證

- ✅ 使用 Gin Binding 驗證請求參數
- ✅ 驗證訊息長度限制
- ✅ 驗證玩家 ID 數量限制
- ✅ 防止 SQL 注入 (使用 GORM)
- ✅ 防止 XSS 攻擊 (HTML Escape)

### HTTPS 強制使用

```go
// Gin Middleware
router.Use(func(c *gin.Context) {
    if c.Request.Header.Get("X-Forwarded-Proto") != "https" {
        c.Redirect(301, "https://"+c.Request.Host+c.Request.RequestURI)
        return
    }
    c.Next()
})
```

---

## 7. 監控與告警

### 關鍵指標

**應用指標**:
- 線上玩家數 (`sse_connections_total`)
- 訊息推送成功率 (`sse_message_sent_total / sse_message_total`)
- SSE 連接異常斷開率 (`sse_connection_errors_total / sse_connections_total`)
- API 回應時間 (P50/P95/P99)

**基礎設施指標**:
- Redis 連接池使用率
- Redis Pub/Sub 延遲
- CPU/Memory 使用率
- Goroutine 數量

### Prometheus 告警規則

```yaml
groups:
- name: sse_alerts
  rules:
  # 連接數異常
  - alert: SSEHighConnectionCount
    expr: sse_connections_total > 50000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "SSE 連接數過高"
      
  # 訊息推送失敗率高
  - alert: SSEHighFailureRate
    expr: |
      rate(sse_message_failed_total[5m])
      / rate(sse_message_sent_total[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "訊息推送失敗率超過 5%"
      
  # Pod 連接數分布不均
  - alert: SSEConnectionImbalance
    expr: |
      (max(sse_connections_per_pod) - min(sse_connections_per_pod))
      / avg(sse_connections_per_pod) > 0.3
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "SSE Pod 連接數分布不均"
```

### Grafana Dashboard

**建議面板**:
1. **連接監控**:
   - 總連接數趨勢
   - 各 Pod 連接數分布
   - 連接增長率

2. **訊息監控**:
   - 訊息推送 QPS
   - 訊息成功/失敗率
   - 訊息延遲分布 (P50/P95/P99)

3. **系統監控**:
   - CPU/Memory 使用率
   - Goroutine 數量
   - Redis/MySQL 連接池

4. **告警面板**:
   - 當前告警列表
   - 告警歷史趨勢

### 日誌管理

**結構化日誌**:
```go
logger.InfoWithContext(ctx, "broadcast completed",
    logger.String("broadcast_id", id),
    logger.Int("total_players", count),
    logger.Duration("duration", duration))
```

**日誌級別**:
- `DEBUG`: 詳細除錯資訊
- `INFO`: 一般資訊 (連接建立、訊息推送)
- `WARN`: 警告 (推送失敗、重試)
- `ERROR`: 錯誤 (系統錯誤、連接異常)

---

## 8. 多 Pod 部署注意事項

### 環境變數配置

**Kubernetes Deployment**:
```yaml
env:
- name: SSE_POD_ID
  valueFrom:
    fieldRef:
      fieldPath: metadata.name  # 使用 Pod 名稱
- name: REDIS_ADDR
  value: "redis-service:6379"
```

### 負載均衡配置

**使用輪詢，避免 Session Affinity**:
```yaml
spec:
  sessionAffinity: None  # 關鍵配置
```

### 優雅關閉

**Kubernetes lifecycle hooks**:
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 15"]
```

### Redis Pub/Sub 可靠性

- ✅ 實作自動重連機制
- ✅ 訊息去重 (使用 Notification ID)
- ✅ 錯誤處理與日誌記錄
- ✅ 監控 Pub/Sub 延遲

---

## 🔗 相關文檔

- [總覽文檔](overview.md)
- [技術要點](technical-details.md)
- [檔案結構](file-structure.md)
- [Phase 6: 測試與優化](phase-6.md)
- [Phase 8: Kubernetes 部署配置](phase-8.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
