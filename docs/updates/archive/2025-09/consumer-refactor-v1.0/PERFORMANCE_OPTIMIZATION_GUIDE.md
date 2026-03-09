# Consumer Performance Optimization Implementation Guide

**文件版本**: v1.0  
**創建日期**: 2025-09-24  
**最後更新**: 2025-09-24  
**狀態**: 📋 實施指南

## 📋 概述

本文檔提供 KDS Consumer 性能優化的具體實施步驟，旨在將單 Pod 單 Shard 吞吐量從當前的 **1,400-2,000 筆/秒** 提升至 **8,000-16,000 筆/秒**。

## 🎯 優化目標

- **短期目標**: 達到 5,000-8,000 筆/秒 (階段一)
- **長期目標**: 達到 10,000-16,000 筆/秒 (階段二) 
- **架構要求**: 不大幅修改現有架構
- **可靠性要求**: 維持所有安全機制

## 🚀 階段一: 立即優化 (預期提升 4-6x)

### 1. 消除退避瓶頸 ⭐⭐⭐⭐⭐

#### **問題診斷**
當前代碼在有記錄時仍然強制等待退避時間，這是最大的性能殺手。

#### **修改步驟**

**步驟 1: 修改 processShardRecords 返回值**
```go
// 文件: internal/infrastructure/kds/consumer.go
// 修改函數簽名，返回記錄數量
func (k *KDSService) processShardRecords(
    ctx context.Context,
    shardId, currentIterator string,
    backoffManager *BackoffManager,
    shardErrs chan<- error,
) (string, bool, int) {  // 新增 int 返回記錄數量
    // ... 現有代碼 ...
    
    // 調整退避時間
    recordsCount := len(recordsOutput.Records)
    backoffManager.AdjustBackoffByRecordCount(recordsCount)
    
    // 處理記錄批次
    if recordsCount > 0 {
        k.processRecordBatches(ctx, shardId, recordsOutput.Records)
    }
    
    // 獲取下一個迭代器
    if recordsOutput.NextShardIterator != nil {
        return *recordsOutput.NextShardIterator, true, recordsCount  // 返回記錄數量
    }
    
    k.logger.InfoWithContext(ctx, "Shard has been closed", k.logger.String("shard_id", shardId))
    return "", false, recordsCount  // 返回記錄數量
}
```

**步驟 2: 修改 consumeShardEvents 等待邏輯**
```go
// 文件: internal/infrastructure/kds/consumer.go
// 修改主循環邏輯
for {
    select {
    case <-shardCtx.Done():
        return
    default:
        // 獲取和處理記錄
        newIterator, shouldContinue, recordCount := k.processShardRecords(
            shardCtx, shardId, currentIterator, backoffManager, shardErrs)
        if !shouldContinue {
            return
        }
        currentIterator = newIterator
        
        // 🔥 關鍵優化：只在無記錄時才等待
        if recordCount == 0 {
            time.Sleep(backoffManager.GetCurrentBackoff())
        }
        // 有記錄時立即繼續，實現連續拉取
    }
}
```

**步驟 3: 更新相關調用**
```go
// 文件: internal/infrastructure/kds/consumer.go
// 更新 handleGetRecordsError 函數
func (k *KDSService) handleGetRecordsError(
    ctx context.Context,
    shardId string,
    getRecordsErr error,
    backoffManager *BackoffManager,
    shardErrs chan<- error,
) (string, bool, int) {  // 更新返回值
    // ... 現有錯誤處理邏輯 ...
    
    if newIterator, ok := iterators[shardId]; ok {
        return newIterator, true, 0  // 錯誤情況返回 0 記錄
    }
    
    k.logger.ErrorWithContext(ctx, "Shard no longer available", k.logger.String("shard_id", shardId))
    return "", false, 0
}
```

**預期效果**: **3-5x** 吞吐量提升

### 2. 關鍵參數調優 ⭐⭐⭐⭐

#### **環境變數配置**
建立 `config/performance.env` 檔案:

```bash
# KDS 參數優化
CONSUMER_KDS_RECORD_LIMIT=10000      # AWS Kinesis 單次最大獲取記錄數

# Worker Pool 優化
CONSUMER_WORKER_POOL_SIZE=50         # 增加並行 Worker 數量  
CONSUMER_WORKER_BUFFER_SIZE=1000     # 增大 Worker 緩衝區

# 批次處理優化
CONSUMER_BATCH_SIZE=500              # 增大批次處理大小

# 退避策略優化
CONSUMER_MIN_BACKOFF=50ms            # 減少最小退避時間
CONSUMER_MAX_BACKOFF=1s              # 減少最大退避時間

# 分片處理優化
CONSUMER_MAX_SHARD_CONCURRENCY=16   # 增加最大分片並行數
CONSUMER_SHARD_LOCK_TIMEOUT=30s      # 縮短分片鎖超時時間
```

#### **Docker Compose 配置更新**
```yaml
# docker-compose.yml
services:
  consumer:
    environment:
      - CONSUMER_KDS_RECORD_LIMIT=10000
      - CONSUMER_WORKER_POOL_SIZE=50
      - CONSUMER_WORKER_BUFFER_SIZE=1000
      - CONSUMER_BATCH_SIZE=500
      - CONSUMER_MIN_BACKOFF=50ms
      - CONSUMER_MAX_BACKOFF=1s
```

**預期效果**: **1.5-2x** 吞吐量提升

### 3. Redis 連接池優化 ⭐⭐⭐

#### **Redis 配置優化**
```bash
# 連接池大小優化
REDIS_POOL_SIZE=100              # 從 20 提升到 100
REDIS_MIN_IDLE_CONNS=20          # 從 5 提升到 20

# 超時時間優化
REDIS_READ_TIMEOUT=1s            # 從 3s 縮短到 1s
REDIS_WRITE_TIMEOUT=1s           # 從 3s 縮短到 1s  
REDIS_POOL_TIMEOUT=2s            # 從 4s 縮短到 2s

# 連接生命週期優化
REDIS_IDLE_TIMEOUT=5m            # 連接空閒超時
REDIS_MAX_CONN_AGE=30m           # 連接最大存活時間
REDIS_MAX_RETRIES=5              # 增加重試次數
```

#### **資源配置建議**
```bash
# Redis 服務器配置 (如果可控)
maxclients 10000                 # 最大客戶端連接數
timeout 300                      # 客戶端超時時間 (秒)
tcp-keepalive 60                 # TCP keepalive 時間
```

**預期效果**: **1.2-1.5x** 吞吐量提升

### 4. 系統資源優化

#### **容器資源配置**
```yaml
# docker-compose.yml
services:
  consumer:
    deploy:
      resources:
        limits:
          cpus: '4.0'           # 增加 CPU 配額
          memory: 4G            # 增加記憶體配額
        reservations:
          cpus: '2.0'           # CPU 預留
          memory: 2G            # 記憶體預留
```

#### **JVM 參數優化 (如果適用)**
```bash
# Go 程序環境變數
GOGC=100                        # 垃圾回收目標百分比
GOMAXPROCS=4                    # 最大 CPU 使用數
```

## 🚀 階段二: 進階優化 (預期額外提升 2-3x)

### 1. 本地記憶體快取 ⭐⭐⭐

#### **快取實現**

**步驟 1: 建立快取結構**
```go
// 文件: internal/infrastructure/kds/cache.go (新建)
package kds

import (
    "sync"
    "time"
)

type EventCache struct {
    cache    sync.Map           // 線程安全 map
    ttl      time.Duration      // 24小時 TTL
    lastGC   time.Time          // 上次垃圾回收時間
    gcMutex  sync.Mutex         // 垃圾回收鎖
}

func NewEventCache() *EventCache {
    return &EventCache{
        ttl:    24 * time.Hour,
        lastGC: time.Now(),
    }
}

type cacheEntry struct {
    timestamp time.Time
}

func (ec *EventCache) Set(eventID string) {
    ec.cache.Store(eventID, cacheEntry{timestamp: time.Now()})
    ec.maybeGC()
}

func (ec *EventCache) Exists(eventID string) bool {
    if entry, ok := ec.cache.Load(eventID); ok {
        if time.Since(entry.(cacheEntry).timestamp) < ec.ttl {
            return true
        }
        ec.cache.Delete(eventID) // 過期刪除
    }
    return false
}

func (ec *EventCache) maybeGC() {
    if time.Since(ec.lastGC) > time.Hour {
        ec.gcMutex.Lock()
        defer ec.gcMutex.Unlock()
        
        if time.Since(ec.lastGC) > time.Hour {
            ec.runGC()
            ec.lastGC = time.Now()
        }
    }
}

func (ec *EventCache) runGC() {
    now := time.Now()
    ec.cache.Range(func(key, value interface{}) bool {
        if now.Sub(value.(cacheEntry).timestamp) > ec.ttl {
            ec.cache.Delete(key)
        }
        return true
    })
}
```

**步驟 2: 集成到 KDSService**
```go
// 文件: internal/infrastructure/kds/kds.go
type KDSService struct {
    // ... 現有欄位 ...
    localCache *EventCache  // 新增快取欄位
}

func NewKDSService(...) (*KDSService, error) {
    // ... 現有代碼 ...
    
    return &KDSService{
        // ... 現有初始化 ...
        localCache: NewEventCache(),  // 初始化快取
    }, nil
}
```

**步驟 3: 優化批次檢查邏輯**
```go
// 文件: internal/infrastructure/kds/consumer.go
// 替換 batchCheckEventsProcessed 的調用
func (k *KDSService) optimizedCheckEventsProcessed(
    ctx context.Context,
    eventIDs []string,
) map[string]bool {
    processedMap := make(map[string]bool)
    redisCheckIDs := make([]string, 0, len(eventIDs))
    
    // 1. 本地快取過濾
    for _, eventID := range eventIDs {
        if k.localCache.Exists(eventID) {
            processedMap[eventID] = true
        } else {
            redisCheckIDs = append(redisCheckIDs, eventID)
        }
    }
    
    // 2. Redis 批次查詢 (僅查未命中的)
    if len(redisCheckIDs) > 0 {
        redisResults := k.batchCheckEventsProcessed(ctx, redisCheckIDs)
        for eventID, processed := range redisResults {
            processedMap[eventID] = processed
            if processed {
                k.localCache.Set(eventID) // 更新本地快取
            }
        }
    }
    
    return processedMap
}
```

**預期效果**: **1.3-1.8x** 吞吐量提升

### 2. JSON 解析優化 ⭐⭐

#### **使用高性能 JSON 庫**

**步驟 1: 引入 jsoniter**
```bash
go get github.com/json-iterator/go
```

**步驟 2: 替換 JSON 操作**
```go
// 文件: internal/infrastructure/kds/consumer.go
import (
    jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (k *KDSService) parseEvent(data []byte) (EventPayload, error) {
    var jsonData map[string]interface{}
    if err := json.Unmarshal(data, &jsonData); err != nil {
        return EventPayload{}, err
    }
    
    // ... 其餘邏輯不變 ...
}
```

**預期效果**: **1.1-1.2x** 吞吐量提升

## 📊 監控與驗證

### 性能監控指標

#### **1. 關鍵效能指標 (KPIs)**
```go
// 建議在 consumer 中添加 metrics
type ConsumerMetrics struct {
    ProcessedEventsTotal     prometheus.Counter   // 總處理事件數
    ProcessingDurationHist   prometheus.Histogram // 處理時間直方圖
    BatchSizeHist           prometheus.Histogram // 批次大小直方圖
    ErrorsTotal             prometheus.Counter   // 錯誤總數
    RedisHitRatio           prometheus.Gauge     // Redis 快取命中率
    LocalCacheHitRatio      prometheus.Gauge     // 本地快取命中率
}
```

#### **2. 監控 Dashboard**
```yaml
# Grafana Dashboard 配置
panels:
  - title: "Events Per Second"
    targets:
      - expr: "rate(consumer_processed_events_total[1m])"
  - title: "Processing Latency"  
    targets:
      - expr: "histogram_quantile(0.95, consumer_processing_duration_seconds)"
  - title: "Cache Hit Rates"
    targets:
      - expr: "consumer_redis_hit_ratio"
      - expr: "consumer_local_cache_hit_ratio"
```

### 效能測試方案

#### **負載測試腳本**
```bash
#!/bin/bash
# 文件: scripts/performance_test.sh

echo "開始 Consumer 效能測試..."

# 1. 基線測試
echo "測試階段 1: 基線效能"
docker-compose up -d consumer
sleep 60
./test_throughput.sh 60  # 測試 60 秒

# 2. 優化後測試  
echo "測試階段 2: 優化後效能"
docker-compose down
docker-compose -f docker-compose.optimized.yml up -d consumer
sleep 60
./test_throughput.sh 60

echo "效能測試完成，查看結果:"
echo "基線吞吐量: $(cat baseline_result.txt)"
echo "優化後吞吐量: $(cat optimized_result.txt)"
```

## 🚨 注意事項與風險

### 風險評估

#### **1. 資源消耗風險**
- **CPU 使用率**: Worker Pool 增大可能導致 CPU 使用率上升
- **記憶體使用**: 本地快取和更大的緩衝區會增加記憶體消耗
- **網絡連接**: Redis 連接池增大需要確保 Redis 服務器可以承受

#### **2. 穩定性風險**
- **連接數過多**: 可能導致 Redis 連接數超限
- **記憶體洩漏**: 本地快取如果沒有正確 GC 可能導致記憶體洩漏
- **錯誤處理**: 高速處理時錯誤處理邏輯要更加健壯

### 緩解措施

#### **1. 監控告警**
```yaml
# 告警規則
alerts:
  - alert: ConsumerHighCPU
    expr: rate(container_cpu_usage_seconds_total{name="consumer"}[5m]) > 0.8
  - alert: ConsumerHighMemory  
    expr: container_memory_usage_bytes{name="consumer"} > 3GB
  - alert: ConsumerLowThroughput
    expr: rate(consumer_processed_events_total[5m]) < 1000
```

#### **2. 逐步部署**
1. **開發環境驗證**: 完整測試所有優化
2. **測試環境部署**: 進行壓力測試
3. **灰度部署**: 生產環境小流量測試
4. **全量部署**: 確認無問題後全量上線

#### **3. 回滾計劃**
- 保持原始配置備份
- 準備快速回滾腳本
- 監控關鍵指標，異常時立即回滾

## 📅 實施時間表

### 第一週: 立即優化
- **Day 1-2**: 修改退避邏輯程式碼
- **Day 3-4**: 參數調優和測試
- **Day 5-7**: Redis 優化和整合測試

### 第二週: 進階優化  
- **Day 8-10**: 實現本地快取
- **Day 11-12**: JSON 解析優化
- **Day 13-14**: 整合測試和效能驗證

### 第三週: 部署上線
- **Day 15-17**: 測試環境完整驗證
- **Day 18-19**: 生產環境灰度部署  
- **Day 20-21**: 監控和調優

## 🏆 成功標準

### 效能目標
- ✅ **階段一目標**: 達到 8,000-12,000 筆/秒
- ✅ **階段二目標**: 達到 12,000-16,000 筆/秒
- ✅ **穩定性要求**: 99.9% 可用性
- ✅ **資源使用**: CPU < 80%, Memory < 4GB

### 驗收條件
1. 通過 1 小時持續負載測試
2. 錯誤率 < 0.1%
3. P95 處理延遲 < 100ms
4. 快取命中率 > 60%
5. 所有監控指標正常

## 📚 相關資源

- [性能分析文檔](PERFORMANCE_ANALYSIS.md)
- [效能基準測試](PERFORMANCE_BENCHMARK.md)
- [技術規格文檔](TECHNICAL_SPECIFICATION.md)
- [AWS Kinesis 最佳實踐](https://docs.aws.amazon.com/kinesis/latest/dev/kinesis-record-processor-scaling.html)