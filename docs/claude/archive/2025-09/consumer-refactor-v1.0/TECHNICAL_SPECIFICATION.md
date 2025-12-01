# Fat Notification Cat Consumer 重構技術規格

## 文檔資訊
- **版本**: v2.0
- **建立日期**: 2025-09-23
- **最後更新**: 2025-09-24
- **技術架構**: 基於fat-identity-cat Consumer v2.0簡化實用版
- **目標**: 性能優化與代碼品質提升
- **重構狀態**: ✅ 完成
- **效能分析**: ✅ 完成 ([詳見效能分析文檔](PERFORMANCE_ANALYSIS.md))

---

## 📊 重構成果總覽

### 性能提升預期
- **基線吞吐量**: 1,400-2,000 筆/秒
- **優化後吞吐量**: 8,000-16,000 筆/秒 (4-8x 提升)
- **架構兼容性**: 100% 向後兼容
- **擴展能力**: 支援線性水平擴展

### 主要優化成果
- ✅ **批次處理**: Redis MGet/Pipeline 減少網路開銷 60-80%
- ✅ **Worker Pool**: 並行處理提升 CPU 利用率
- ✅ **智能退避**: 動態調整 API 調用頻率
- ✅ **可靠性**: 分片鎖 + 去重 + Checkpoint 機制

---

## 🏗️ 整體架構設計

### 當前架構分析
```
Current Consumer Architecture
├── cmd/consumer/consumer.go (297 lines)
│   ├── Service初始化
│   ├── Consumer循環控制
│   ├── 重試機制
│   └── Panic Recovery
└── internal/infrastructure/kds/consumer.go (576 lines)
    ├── ConsumeAllEvents (主要處理邏輯)
    ├── 單記錄順序處理
    ├── Redis單次操作
    └── 基礎退避策略
```

### 目標架構設計
```
Optimized Consumer Architecture (v2.0)
├── cmd/consumer/consumer.go (簡化後 ~200 lines)
│   ├── Service初始化
│   ├── Consumer循環控制
│   └── 統一錯誤處理
├── internal/infrastructure/kds/consumer.go (重構後 ~300 lines)
│   ├── ConsumeAllEvents (協調邏輯)
│   ├── Batch處理入口
│   └── Worker Pool管理
├── internal/infrastructure/kds/batch_processor.go (新增 ~200 lines)
│   ├── RecordBatch管理
│   ├── 批次去重檢查
│   └── 批次標記處理
├── internal/infrastructure/kds/worker_pool.go (新增 ~150 lines)
│   ├── Worker Pool實現
│   ├── Job分發機制
│   └── 結果收集
└── internal/infrastructure/kds/backoff_strategy.go (已存在)
    ├── 動態退避管理 ✅
    ├── BackoffManager 實現 ✅
    └── 智能頻率調整 ✅

### 實際重構結果 (2025-09-24)
```
Completed Refactored Architecture
├── internal/infrastructure/kds/consumer.go (重構完成 ~848 lines)
│   ├── ✅ ConsumeAllEvents (統一入口)
│   ├── ✅ acquireShardLock (分片鎖管理)
│   ├── ✅ consumeShardEvents (分片事件處理)
│   ├── ✅ processShardRecords (記錄處理)
│   ├── ✅ processRecordBatches (批次處理)
│   ├── ✅ processBatch (Worker Pool)
│   ├── ✅ batchCheckEventsProcessed (Redis MGet)
│   └── ✅ batchMarkEventsProcessed (Redis Pipeline)
├── internal/infrastructure/kds/backoff_strategy.go (已存在)
│   ├── ✅ BackoffManager 結構
│   ├── ✅ 動態退避調整
│   └── ✅ 記錄數量感知
└── internal/infrastructure/cache/redis/manager.go (擴充)
    ├── ✅ MGet 方法 (批次查詢)
    └── ✅ Pipeline 方法 (批次操作)
```

---

## 🔧 核心組件設計

### 1. RecordBatch 批次處理器

#### 結構定義
```go
// RecordBatch 批次記錄管理
type RecordBatch struct {
    Records     []types.Record
    BatchSize   int
    MaxWait     time.Duration
    ShardID     string
    CreatedAt   time.Time
}

// BatchProcessor 批次處理器
type BatchProcessor struct {
    batchSize       int
    maxWaitTime     time.Duration
    redisManager    *redis.Manager
    logger          infrastructure.Logger
    metrics         *BatchMetrics
}

// BatchMetrics 批次處理指標
type BatchMetrics struct {
    ProcessedBatches   int64
    ProcessedRecords   int64
    DuplicateRecords   int64
    ProcessingTime     time.Duration
    RedisOperations    int64
}
```

#### 核心方法
```go
// ProcessBatch 處理批次記錄
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, batch RecordBatch) (*BatchResult, error) {
    // 1. 批次去重檢查 (Redis MGet)
    duplicates, err := bp.checkDuplicates(ctx, batch)
    if err != nil {
        return nil, fmt.Errorf("batch duplicate check failed: %w", err)
    }
    
    // 2. 過濾重複記錄
    uniqueRecords := bp.filterDuplicates(batch.Records, duplicates)
    
    // 3. 批次處理記錄
    results, err := bp.processRecords(ctx, uniqueRecords)
    if err != nil {
        return nil, fmt.Errorf("batch processing failed: %w", err)
    }
    
    // 4. 批次標記已處理 (Redis Pipeline)
    err = bp.markBatchProcessed(ctx, results)
    if err != nil {
        bp.logger.WarnWithContext(ctx, "Failed to mark batch as processed", 
            bp.logger.Error("err", err))
    }
    
    return &BatchResult{
        ProcessedCount: len(results),
        DuplicateCount: len(duplicates),
        ProcessingTime: time.Since(batch.CreatedAt),
    }, nil
}

// checkDuplicates 批次去重檢查
func (bp *BatchProcessor) checkDuplicates(ctx context.Context, batch RecordBatch) (map[string]bool, error) {
    if len(batch.Records) == 0 {
        return nil, nil
    }
    
    // 構建Redis keys
    keys := make([]string, len(batch.Records))
    for i, record := range batch.Records {
        eventID := bp.extractEventID(record.Data)
        keys[i] = processedEventKeyPrefix + eventID
    }
    
    // Redis MGet批次檢查
    values, err := bp.redisManager.MGet(ctx, keys...)
    if err != nil {
        return nil, fmt.Errorf("redis mget failed: %w", err)
    }
    
    duplicates := make(map[string]bool)
    for i, value := range values {
        if value != nil {
            eventID := bp.extractEventID(batch.Records[i].Data)
            duplicates[eventID] = true
        }
    }
    
    return duplicates, nil
}

// markBatchProcessed 批次標記已處理
func (bp *BatchProcessor) markBatchProcessed(ctx context.Context, results []ProcessResult) error {
    if len(results) == 0 {
        return nil
    }
    
    // Redis Pipeline批次操作
    pipe := bp.redisManager.Pipeline()
    for _, result := range results {
        key := processedEventKeyPrefix + result.EventID
        pipe.Set(ctx, key, "1", eventProcessedTTL)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}
```

### 2. Worker Pool 並行處理器

#### 結構定義
```go
// WorkerPool Worker Pool管理器
type WorkerPool struct {
    workerCount     int
    jobQueue        chan WorkJob
    resultQueue     chan WorkResult
    wg              sync.WaitGroup
    ctx             context.Context
    cancel          context.CancelFunc
    logger          infrastructure.Logger
    kdsService      *KDSService
}

// WorkJob 工作任務
type WorkJob struct {
    Record          types.Record
    EventID         string
    EventType       string
    ShardID         string
    SequenceNumber  string
    RetryCount      int
}

// WorkResult 工作結果
type WorkResult struct {
    Job             WorkJob
    Success         bool
    Error           error
    ProcessingTime  time.Duration
    EnqueueSuccess  bool
}
```

#### 核心方法
```go
// Start 啟動Worker Pool
func (wp *WorkerPool) Start() error {
    wp.ctx, wp.cancel = context.WithCancel(context.Background())
    
    // 啟動workers
    for i := 0; i < wp.workerCount; i++ {
        wp.wg.Add(1)
        go wp.worker(i)
    }
    
    wp.logger.InfoWithContext(wp.ctx, "Worker pool started",
        wp.logger.Int("worker_count", wp.workerCount))
    
    return nil
}

// worker Worker執行函數
func (wp *WorkerPool) worker(workerID int) {
    defer wp.wg.Done()
    
    for {
        select {
        case <-wp.ctx.Done():
            return
        case job := <-wp.jobQueue:
            result := wp.processJob(job, workerID)
            wp.resultQueue <- result
        }
    }
}

// processJob 處理單個任務
func (wp *WorkerPool) processJob(job WorkJob, workerID int) WorkResult {
    startTime := time.Now()
    
    result := WorkResult{
        Job:            job,
        ProcessingTime: 0,
        Success:        false,
    }
    
    // 創建job專用context
    jobCtx := context.WithValue(wp.ctx, constants.EventIDKey, job.EventID)
    
    // 處理事件入隊
    err := wp.kdsService.eventEnqueueProcess(jobCtx, job.EventType, job.Record.Data)
    if err != nil {
        result.Error = err
        wp.logger.ErrorWithContext(jobCtx, "Worker failed to process job",
            wp.logger.Int("worker_id", workerID),
            wp.logger.String("event_id", job.EventID),
            wp.logger.Error("err", err))
        return result
    }
    
    result.Success = true
    result.EnqueueSuccess = true
    result.ProcessingTime = time.Since(startTime)
    
    wp.logger.InfoWithContext(jobCtx, "Worker processed job successfully",
        wp.logger.Int("worker_id", workerID),
        wp.logger.String("event_id", job.EventID),
        wp.logger.Duration("processing_time", result.ProcessingTime))
    
    return result
}

// SubmitJob 提交工作任務
func (wp *WorkerPool) SubmitJob(job WorkJob) error {
    select {
    case wp.jobQueue <- job:
        return nil
    case <-wp.ctx.Done():
        return fmt.Errorf("worker pool is shutting down")
    default:
        return fmt.Errorf("job queue is full")
    }
}

// Stop 停止Worker Pool
func (wp *WorkerPool) Stop() error {
    wp.cancel()
    wp.wg.Wait()
    wp.logger.InfoWithContext(context.Background(), "Worker pool stopped")
    return nil
}
```

### 3. BackoffStrategy 智能退避策略

#### 結構定義
```go
// BackoffManager 退避策略管理器
type BackoffManager struct {
    MinBackoff     time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
    Jitter         bool
    MaxRetries     int
    currentBackoff time.Duration
    retryCount     int
    mu             sync.RWMutex
}

// PanicRecovery Panic恢復管理器
type PanicRecovery struct {
    initialBufferSize int
    maxBufferSize     int
    currentBufferSize int
    expandMultiplier  int
    logger            infrastructure.Logger
}
```

#### 核心方法
```go
// NewBackoffManager 創建退避管理器
func NewBackoffManager(minBackoff, maxBackoff time.Duration, multiplier float64) *BackoffManager {
    return &BackoffManager{
        MinBackoff:     minBackoff,
        MaxBackoff:     maxBackoff,
        Multiplier:     multiplier,
        Jitter:         true,
        MaxRetries:     5,
        currentBackoff: minBackoff,
        retryCount:     0,
    }
}

// NextBackoff 計算下一次退避時間
func (bm *BackoffManager) NextBackoff() time.Duration {
    bm.mu.Lock()
    defer bm.mu.Unlock()
    
    if bm.retryCount >= bm.MaxRetries {
        return bm.MaxBackoff
    }
    
    backoff := bm.currentBackoff
    bm.currentBackoff = time.Duration(float64(bm.currentBackoff) * bm.Multiplier)
    
    if bm.currentBackoff > bm.MaxBackoff {
        bm.currentBackoff = bm.MaxBackoff
    }
    
    bm.retryCount++
    
    // 添加隨機抖動
    if bm.Jitter {
        jitter := time.Duration(rand.Float64() * float64(backoff) * 0.1)
        backoff += jitter
    }
    
    return backoff
}

// Reset 重置退避狀態
func (bm *BackoffManager) Reset() {
    bm.mu.Lock()
    defer bm.mu.Unlock()
    
    bm.currentBackoff = bm.MinBackoff
    bm.retryCount = 0
}

// RecoverWithDynamicBuffer 動態Buffer Panic恢復
func (pr *PanicRecovery) RecoverWithDynamicBuffer() error {
    if r := recover(); r != nil {
        // 動態分配buffer
        bufferSize := pr.getCurrentBufferSize()
        buf := make([]byte, bufferSize)
        buf = buf[:runtime.Stack(buf, false)]
        
        // 如果buffer不足，擴展buffer
        if len(buf) == bufferSize && bufferSize < pr.maxBufferSize {
            pr.expandBuffer()
        }
        
        var err error
        if e, ok := r.(error); ok {
            err = fmt.Errorf("panic recovered: %w\n%s", e, buf)
        } else {
            err = fmt.Errorf("panic recovered: %v\n%s", r, buf)
        }
        
        pr.logger.ErrorLog("Panic recovered with dynamic buffer",
            pr.logger.Error("err", err),
            pr.logger.Int("buffer_size", bufferSize))
        
        return err
    }
    return nil
}

// getCurrentBufferSize 獲取當前buffer大小
func (pr *PanicRecovery) getCurrentBufferSize() int {
    if pr.currentBufferSize == 0 {
        pr.currentBufferSize = pr.initialBufferSize
    }
    return pr.currentBufferSize
}

// expandBuffer 擴展buffer大小
func (pr *PanicRecovery) expandBuffer() {
    newSize := pr.currentBufferSize * pr.expandMultiplier
    if newSize > pr.maxBufferSize {
        newSize = pr.maxBufferSize
    }
    pr.currentBufferSize = newSize
}
```

---

## 📊 性能優化策略

### 1. 批次處理優化
- **批次大小**: 100 records/batch (可配置)
- **等待時間**: 最大100ms等待批次填滿
- **Redis操作**: MGet批次檢查，Pipeline批次寫入
- **預期效果**: Redis網絡請求減少80%

### 2. 並行處理優化
- **Worker數量**: 10個goroutine (可配置)
- **Job Queue**: 有界channel防止記憶體溢出
- **Result Collection**: 非阻塞結果收集
- **預期效果**: CPU利用率提升，吞吐量提升50-100%

### 3. 記憶體管理優化
- **Buffer Pool**: 重用buffer減少GC壓力
- **Context Pool**: 重用context降低分配開銷
- **Dynamic Buffer**: 根據需要動態調整stack trace buffer
- **預期效果**: 記憶體使用穩定，GC頻率降低

### 4. 錯誤處理優化
- **分級錯誤處理**: 區分可重試和不可重試錯誤
- **Graceful Degradation**: 部分失敗不影響整體處理
- **Smart Retry**: 指數退避+隨機抖動
- **預期效果**: 系統穩定性提升，錯誤恢復能力增強

---

## 🔍 配置管理

### 新增配置項
```yaml
consumer:
  # 批次處理配置
  batch:
    size: 100                    # 批次大小
    max_wait_ms: 100            # 最大等待時間
    enable: true                # 啟用批次處理
  
  # Worker Pool配置
  worker_pool:
    worker_count: 10            # Worker數量
    job_queue_size: 1000        # Job隊列大小
    result_queue_size: 1000     # Result隊列大小
    enable: true                # 啟用Worker Pool
  
  # 退避策略配置
  backoff:
    min_backoff_ms: 500         # 最小退避時間
    max_backoff_ms: 5000        # 最大退避時間
    multiplier: 1.5             # 退避倍數
    jitter: true                # 啟用抖動
    max_retries: 5              # 最大重試次數
  
  # Panic恢復配置
  panic_recovery:
    initial_buffer_size: 4096   # 初始buffer大小 (4KB)
    max_buffer_size: 1048576    # 最大buffer大小 (1MB)
    expand_multiplier: 2        # 擴展倍數
```

### 配置熱更新
- 支援動態調整批次大小和Worker數量
- 配置變更不需要重啟服務
- 配置驗證確保參數合理性

---

## 🧪 測試策略

### 1. 單元測試
- **BatchProcessor測試**: 批次處理邏輯驗證
- **WorkerPool測試**: 並行處理邏輯驗證
- **BackoffManager測試**: 退避策略邏輯驗證
- **目標覆蓋率**: 90%

### 2. 整合測試
- **Redis整合測試**: 批次操作正確性驗證
- **KDS整合測試**: 端到端流程驗證
- **錯誤場景測試**: 異常情況處理驗證

### 3. 性能測試
- **吞吐量測試**: 模擬高負載場景
- **延遲測試**: P50/P95/P99延遲測量
- **穩定性測試**: 長期運行測試
- **壓力測試**: 極限負載測試

### 4. 兼容性測試
- **向後兼容性**: 確保API不變
- **配置兼容性**: 舊配置正常工作
- **部署兼容性**: 灰度部署驗證

---

## 📈 監控與指標

### 關鍵指標
```go
type ConsumerMetrics struct {
    // 吞吐量指標
    RecordsPerSecond    float64
    BatchesPerSecond    float64
    
    // 延遲指標
    ProcessingLatencyP50 time.Duration
    ProcessingLatencyP95 time.Duration
    ProcessingLatencyP99 time.Duration
    
    // 錯誤指標
    ErrorRate           float64
    RetryRate           float64
    PanicCount          int64
    
    // 資源指標
    MemoryUsage         int64
    CPUUsage            float64
    GoroutineCount      int
    
    // 業務指標
    DuplicateRate       float64
    EnqueueSuccessRate  float64
    CheckpointLag       time.Duration
}
```

### 告警規則
- 吞吐量下降超過30%
- P95延遲超過閾值
- 錯誤率超過5%
- 記憶體使用異常增長
- Worker Pool隊列積壓

---

**文檔狀態**: 技術規格完成  
**下一步**: 開始Phase 1實施  
**負責人**: Claude Code  
**審核日期**: 待定  