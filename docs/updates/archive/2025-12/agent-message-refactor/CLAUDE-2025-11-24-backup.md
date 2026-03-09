# 代理訊息系統效能優化 v1.0

**日期**: 2025-11-24  
**版本**: v1.0  
**狀態**: 規劃中  
**類型**: 效能重構

## 概述

本文檔概述了代理同步系統的全面效能優化計劃，專門針對 `SyncAgentRelationshipsUpsert` 和 `BackfillMissedMessages` 方法中的資料庫操作複雜性問題，以解決 Asynq 任務堆積導致的 Redis OOM 問題。

## 問題分析

### 根本原因
1. **代理同步處理緩慢**: 複雜的資料庫操作拖慢整體同步速度
2. **Asynq 任務堆積**: 生產速度 > 消費速度，導致 Redis 記憶體溢出
3. **低效的資料庫查詢**: N+1 查詢問題和全量掃描瓶頸

### 現有效能指標
| 指標 | 當前數值 | 目標數值 | 改善幅度 |
|------|---------|---------|---------|
| 代理同步時間 | 2-5秒 | 200-800ms | 80% 提升 |
| 每次同步的DB查詢 | 3-100次查詢 | 1-5次查詢 | 90% 減少 |
| Redis記憶體使用量 | 無限制增長 | 穩定循環 | 70% 減少 |
| Worker處理速度 | 10 tasks/sec | 50+ tasks/sec | 400% 增長 |

## 詳細瓶頸分析

### 1. SyncAgentRelationshipsUpsert 瓶頸

**位置**: `internal/application/service/agent_service.go:50`

**問題**: BatchGetOrCreateAgentsByGlobalIDs 執行 3 次資料庫操作

```go
// 當前實現問題
func (r *AgentRepository) BatchGetOrCreateAgentsByGlobalIDs(/*...*/) {
    // 1. 查詢已存在的代理 - O(n)
    SELECT * FROM agents WHERE global_agent_id IN (...) AND merchant_id = ?
    
    // 2. 批量建立不存在的代理 - O(m) 
    INSERT INTO agents (...) VALUES (...) ON CONFLICT DO NOTHING
    
    // 3. 重新查詢已建立的代理 - O(m)
    SELECT * FROM agents WHERE global_agent_id IN (...) AND merchant_id = ?
}
```

**影響**:
- 每次代理同步需要 3 次 DB roundtrip
- 深層 ancestry 會建立多個代理，放大查詢次數
- 網路延遲累積影響整體效能

### 2. BackfillMissedMessages 瓶頸

**位置**: `internal/application/usecase/agent/agent_usecase.go:1187`

**問題**: 無條件全量掃描 + 分頁循環查詢

```sql
-- 每次補派發都執行的重查詢
SELECT * FROM agent_campaigns 
WHERE merchant_id = ? 
  AND target_type IN ('all', 'specific', 'line') 
  AND status = 'sent' 
  AND created_at < NOW()
ORDER BY created_at DESC 
LIMIT 100 OFFSET ?
```

**影響**:
- 商戶有 10,000 個活動 → 需要 100 次分頁查詢
- 每次代理同步都觸發全量掃描
- 沒有時間視窗過濾，處理大量過期資料

## 優化解決方案設計

### 階段 1: 緊急修復 (當天實施)

#### 1.1 BackfillMissedMessages 異步化

**目標**: 防止補派發阻塞主要同步流程

```go
// 修改: internal/application/usecase/agent/agent_usecase.go:63
func (u *AgentUseCase) SyncAgentDataWithRelationships(
    ctx context.Context,
    agentEvent *event.AgentSyncEvent,
) error {
    // 1. 同步當前代理資料（保持同步）
    if err := u.SyncCurrentAgentUpsert(ctx, agentEvent); err != nil {
        return fmt.Errorf("sync current agent: %w", err)
    }

    // 2. 同步代理關係（保持同步）
    if err := u.agentService.SyncAgentRelationshipsUpsert(ctx, agentEvent); err != nil {
        return fmt.Errorf("sync agent relationships: %w", err)
    }

    // 3. 補派發變為異步（關鍵優化）
    select {
    case u.backfillQueue <- &BackfillTask{
        AgentEvent: agentEvent,
        Priority:   u.calculateBackfillPriority(agentEvent),
        CreatedAt:  time.Now(),
    }:
        u.logger.DebugLog("Backfill task queued", 
            u.logger.String("agent_id", agentEvent.GlobalAgentID))
    default:
        // 佇列滿時跳過，避免記憶體壓力
        u.logger.WarnLog("Backfill queue full, skipping", 
            u.logger.String("agent_id", agentEvent.GlobalAgentID))
    }

    return nil // 主流程立即返回
}
```

#### 1.2 異步處理機制

```go
// 新增: internal/application/usecase/agent/backfill_worker.go
type BackfillWorker struct {
    queue       chan *BackfillTask
    workerCount int
    useCase     *AgentUseCase
    ctx         context.Context
}

type BackfillTask struct {
    AgentEvent *event.AgentSyncEvent
    Priority   int
    CreatedAt  time.Time
    RetryCount int
}

func (w *BackfillWorker) Start(ctx context.Context) {
    for i := 0; i < w.workerCount; i++ {
        go w.processBackfillTasks(ctx, i)
    }
}

func (w *BackfillWorker) processBackfillTasks(ctx context.Context, workerID int) {
    for {
        select {
        case task := <-w.queue:
            if err := w.processTask(ctx, task); err != nil {
                w.handleTaskError(task, err)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

**預期效果**:
- 主同步時間: 2-5秒 → 200-500ms（80% 改善）
- 立即緩解 Redis 任務堆積

#### 1.3 Redis 記憶體管理

```bash
# 立即執行的 Redis 配置修正
redis-cli CONFIG SET maxmemory 4gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru

# 清理失敗任務以減少記憶體壓力
redis-cli EVAL "
local failed_keys = redis.call('keys', 'asynq:*:failed')
local deleted = 0
for i = 1, #failed_keys do
    redis.call('del', failed_keys[i])
    deleted = deleted + 1
end
return deleted
" 0
```

### 階段 2: 查詢優化 (本週實施)

#### 2.1 時間視窗過濾優化

**目標**: 減少 BackfillMissedMessages 掃描範圍 90%

```go
// 修改: FindSentCampaignsForBackfillPaginated 方法簽名
func (r *AgentCampaignRepository) FindSentCampaignsForBackfillPaginated(
    ctx context.Context,
    merchantID uint64,
    targetTypes []string,
    startTime time.Time, // 新增時間過濾
    batchSize, offset int,
) ([]*entity.AgentCampaign, error) {
    var campaignModels []models.AgentCampaign

    if err := r.db.WithContext(ctx).
        Where(`merchant_id = ? AND 
               target_type IN ? AND 
               status = ? AND 
               created_at BETWEEN ? AND ?`, // 時間視窗過濾
            merchantID, targetTypes, "sent", startTime, time.Now()).
        Order("created_at DESC").
        Limit(batchSize).
        Offset(offset).
        Find(&campaignModels).Error; err != nil {
        return nil, fmt.Errorf("find sent campaigns for backfill paginated failed: %w", err)
    }

    // ... 轉換邏輯保持不變
}
```

#### 2.2 智能時間視窗計算

```go
// 新增: internal/application/usecase/agent/time_calculator.go
func (u *AgentUseCase) calculateBackfillTimeWindow(
    agent *entity.Agent,
) time.Time {
    // 策略 1: 基於代理最後登入時間
    if agent.CurrentSignInAt != nil {
        lastLogin := *agent.CurrentSignInAt
        // 最多回溯30天
        maxBacktrack := time.Now().AddDate(0, 0, -30)
        if lastLogin.After(maxBacktrack) {
            return lastLogin
        }
        return maxBacktrack
    }

    // 策略 2: 新代理或無登入記錄，回溯7天
    return time.Now().AddDate(0, 0, -7)
}

func (u *AgentUseCase) BackfillMissedMessages(
    ctx context.Context,
    agentEvent *event.AgentSyncEvent,
) error {
    // ... 現有邏輯 ...

    // 計算智能時間視窗
    startTime := u.calculateBackfillTimeWindow(agent)
    
    // 使用時間過濾的查詢
    campaigns, err := u.agentCampaignRepo.FindSentCampaignsForBackfillPaginated(
        ctx,
        merchant.ID,
        targetTypes,
        startTime, // 加入時間過濾
        batchSize,
        offset,
    )
    
    // ... 後續處理邏輯保持不變
}
```

**預期效果**:
- 掃描資料量減少 85-95%
- 查詢時間從秒級降到毫秒級

### 階段 3: 資料庫操作優化 (下週實施)

#### 3.1 BatchGetOrCreateAgentsByGlobalIDs 重構

**目標**: 3次查詢 → 1次複合查詢

```go
// 新實現: internal/adapter/outbound/repository/agent/agent_repository.go
func (r *AgentRepository) UpsertAgentsAndGetIDs(
    ctx context.Context,
    globalIDs []string,
    merchantID uint64,
) (map[string]uint64, error) {
    if len(globalIDs) == 0 {
        return make(map[string]uint64), nil
    }

    // 建構批量 UPSERT 查詢
    placeholders := make([]string, len(globalIDs))
    args := make([]interface{}, 0, len(globalIDs)*2+1)
    
    for i, globalID := range globalIDs {
        placeholders[i] = "(?, ?)"
        args = append(args, globalID, merchantID)
    }
    args = append(args, merchantID)

    // 單次 UPSERT 並回傳所有 ID
    query := fmt.Sprintf(`
        WITH upserted AS (
            INSERT INTO agents (global_agent_id, merchant_id, created_at, updated_at)
            VALUES %s
            ON DUPLICATE KEY UPDATE 
                updated_at = VALUES(updated_at)
            RETURNING global_agent_id, id
        )
        SELECT global_agent_id, id FROM agents 
        WHERE global_agent_id IN (%s) AND merchant_id = ?
    `, 
        strings.Join(placeholders, ","),
        strings.Repeat("?,", len(globalIDs)-1)+"?")

    rows, err := r.db.WithContext(ctx).Raw(query, args...).Rows()
    if err != nil {
        return nil, fmt.Errorf("upsert agents and get IDs failed: %w", err)
    }
    defer rows.Close()

    result := make(map[string]uint64)
    for rows.Next() {
        var globalID string
        var id uint64
        if err := rows.Scan(&globalID, &id); err != nil {
            return nil, fmt.Errorf("scan upsert result failed: %w", err)
        }
        result[globalID] = id
    }

    return result, nil
}
```

#### 3.2 資料庫索引優化

```sql
-- 針對補派發查詢優化
CREATE INDEX IF NOT EXISTS idx_agent_campaigns_backfill_optimized 
ON agent_campaigns (merchant_id, status, target_type, created_at)
INCLUDE (id, title, content, target_details);

-- 針對代理關係查詢優化
CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_global_merchant_unique
ON agents (global_agent_id, merchant_id);

-- 針對批次訊息存在性檢查優化
CREATE INDEX IF NOT EXISTS idx_agent_messages_campaign_agent
ON agent_messages (agent_campaign_id, agent_id)
INCLUDE (id, is_read, created_at);
```

### 階段 4: 快取策略實施 (下週)

#### 4.1 活動查詢結果快取

```go
// 新增: internal/application/service/campaign_cache_service.go
type CampaignCacheService struct {
    redis  *redis.Client
    logger infrastructure.Logger
}

func (c *CampaignCacheService) GetCachedCampaigns(
    ctx context.Context,
    merchantID uint64,
    targetTypes []string,
    startTime time.Time,
) ([]*entity.AgentCampaign, bool) {
    // 產生快取鍵值
    key := fmt.Sprintf("backfill:campaigns:%d:%s:%d", 
        merchantID,
        strings.Join(targetTypes, ","),
        startTime.Unix())

    // 嘗試從 Redis 取得
    cached, err := c.redis.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, false // 快取未命中
    }
    if err != nil {
        c.logger.WarnLog("Failed to get cached campaigns", 
            c.logger.Error("error", err))
        return nil, false
    }

    // 反序列化
    var campaigns []*entity.AgentCampaign
    if err := json.Unmarshal([]byte(cached), &campaigns); err != nil {
        c.logger.WarnLog("Failed to unmarshal cached campaigns", 
            c.logger.Error("error", err))
        return nil, false
    }

    return campaigns, true
}
```

### 階段 5: 監控和告警 (持續改進)

#### 5.1 效能監控指標

```go
// 新增: internal/infrastructure/monitoring/agent_metrics.go
type AgentSyncMetrics struct {
    // 同步時間指標
    SyncDuration     prometheus.Histogram
    BackfillDuration prometheus.Histogram
    
    // 查詢數量指標
    DatabaseQueries prometheus.Counter
    CacheHitRate   prometheus.Gauge
    
    // 佇列狀態指標
    BackfillQueueSize prometheus.Gauge
    FailedTasks      prometheus.Counter
}
```

## 實施計劃

### 第一週: 緊急修復 (2025-11-25 ~ 2025-11-29)

**第1-2天**: 階段 1 實施
- [ ] BackfillMissedMessages 異步化
- [ ] Redis 記憶體管理配置
- [ ] 異步處理機制

**第3-5天**: 階段 2 實施  
- [ ] 時間視窗過濾優化
- [ ] 智能時間視窗計算
- [ ] 基本效能測試

### 第二週: 深度優化 (2025-12-02 ~ 2025-12-06)

**第1-3天**: 階段 3 實施
- [ ] BatchGetOrCreateAgentsByGlobalIDs 重構
- [ ] 資料庫索引優化
- [ ] 查詢效能驗證

**第4-5天**: 階段 4 實施
- [ ] 活動快取策略
- [ ] 記憶體池重用
- [ ] 整體效能測試

### 第三週: 監控完善 (2025-12-09 ~ 2025-12-13)

**第1-2天**: 階段 5 實施
- [ ] 效能監控指標
- [ ] Redis 記憶體告警
- [ ] 例外處理機制

**第3-5天**: 生產部署
- [ ] 漸進式發佈
- [ ] 效能指標驗證
- [ ] 生產穩定性確認

## 驗證指標

### 效能指標

| 指標 | 優化前 | 目標 | 測量方法 |
|------|--------|------|----------|
| 平均代理同步時間 | 2-5秒 | 200-800ms | Prometheus histogram |
| 每次同步DB查詢數 | 3-100次查詢 | 1-5次查詢 | SQL query logging |
| Redis記憶體使用量 | 無限制增長 | 穩定4GB | Redis INFO monitoring |
| Worker處理速度 | 10 tasks/sec | 50+ tasks/sec | Asynq metrics |
| BackfillMissedMessages時間 | 10-60秒 | 1-5秒 | 自訂metrics |

### 穩定性指標

| 指標 | 優化前 | 目標 | 測量方法 |
|------|--------|------|----------|
| Redis OOM頻率 | 每日2-3次 | 0次 | 監控告警 |
| 任務堆積數量 | >10,000 | <1,000 | Asynq dashboard |
| 系統錯誤率 | 5-10% | <1% | 錯誤率監控 |
| P99回應時間 | 10+秒 | <2秒 | 回應時間histogram |

## 回滾計劃

### 快速回滾觸發條件
- 系統錯誤率超過 5%
- 代理同步失敗率超過 10%
- Redis 記憶體使用量超過 6GB
- 任務堆積數量持續增長

### 回滾步驟
1. **立即**: 禁用異步 BackfillMissedMessages，回復同步模式
2. **5分鐘內**: 回滾資料庫索引變更
3. **10分鐘內**: 恢復原始查詢邏輯
4. **15分鐘內**: 重啟所有 worker 服務
5. **20分鐘內**: 驗證系統穩定性恢復

## 風險評估

### 高風險項目
| 風險 | 影響 | 機率 | 緩解措施 |
|------|------|------|----------|
| 異步處理資料一致性 | 高 | 中 | 強化錯誤處理與重試機制 |
| 資料庫索引影響寫入效能 | 中 | 低 | 分階段建立，監控寫入延遲 |
| 快取失效導致效能下降 | 中 | 中 | 雙重快取策略 + 降級機制 |

## 成功標準

### 關鍵效能指標 (KPIs)
1. **效能提升**: 代理同步時間減少 80%
2. **穩定性提升**: Redis OOM 事件減少至 0
3. **擴展性提升**: Worker 處理能力提升 400%
4. **維護性提升**: 資料庫查詢複雜度降低 67%

### 業務價值
- **使用者體驗**: 大幅改善代理同步回應速度
- **營運成本**: 控制 Redis 記憶體使用，減少擴容成本
- **系統穩定性**: 消除 OOM 風險，提升服務可用性
- **開發效率**: 簡化查詢邏輯，減少維護成本

---

**文檔狀態**: ✅ 已完成  
**編碼檢查**: 
- [x] 正確的編碼格式 (UTF-8)
- [x] 中文字符正常顯示
- [x] 程式碼區塊格式正確
- [x] 表格對齊正確
- [x] 連結格式有效

**變更紀錄**:
- 2025-11-24: 初版完成，包含全面的優化計劃和實施時程