# Consumer Performance Analysis

**文件版本**: v1.0  
**創建日期**: 2025-09-24  
**最後更新**: 2025-09-24  
**狀態**: ✅ 重構完成 + 效能分析完成

## 📋 概述

本文檔分析基於 fat-identity-cat 架構重構後的 KDS Consumer 效能表現，包含單 Pod 單 Shard 吞吐量評估、水平擴展能力分析，以及達到高吞吐量目標的優化方案。

## 🔍 當前架構分析

### 重構後架構特點

基於 fat-identity-cat 重構完成的架構具備以下特性：

#### **1. 模組化函數設計**
```go
// 主要處理函數
- ConsumeAllEvents()           // 主入口點
- acquireShardLock()          // 分片鎖管理  
- consumeShardEvents()        // 單分片事件處理循環
- processShardRecords()       // 分片記錄處理
- handleGetRecordsError()     // 錯誤處理
- processRecordBatches()      // 批次記錄處理

// 批次優化函數  
- processBatch()              // Worker Pool 並行處理
- processRecord()             // 單個記錄處理
- batchCheckEventsProcessed() // 批次去重檢查 (Redis MGet)
- batchMarkEventsProcessed()  // 批次標記已處理 (Redis Pipeline)
```

#### **2. 關鍵性能參數**
```go
// 預設配置參數
KDSRecordLimit: 1000       // 每次 GetRecords 最大筆數
BatchSize: 100             // Worker Pool 批次大小  
WorkerPoolSize: 10         // 並行 Worker 數量
WorkerBufferSize: 200      // Worker 緩衝區大小
MinBackoff: 500ms          // 最小退避時間
MaxBackoff: 5s             // 最大退避時間
MaxShardConcurrency: 8     // 最大分片並行數
```

#### **3. 架構優勢**
- ✅ **批次處理優化**: Redis MGet/Pipeline 減少網絡開銷
- ✅ **智能退避策略**: 動態調整 API 調用頻率
- ✅ **Worker Pool 並行**: 充分利用多核 CPU
- ✅ **可靠性保證**: 分片鎖 + Checkpoint + 去重機制
- ✅ **水平擴展能力**: 支援多 Pod 多 Shard 線性擴展

## 📊 性能基線評估

### 單 Pod 單 Shard 吞吐量分析

#### **處理流程延遲分解**
```
1. KDS GetRecords API:     ~10-50ms/調用
2. JSON 解析:              ~0.1ms/筆  
3. Redis 去重檢查 (MGet):   ~1-2ms/批次
4. 事件入隊 (Asynq):        ~0.5-1ms/筆
5. Redis 標記處理 (Pipeline): ~1-2ms/批次  
6. DynamoDB Checkpoint:     ~5-10ms/更新

總處理延遲: ~3-5ms/筆 (包含所有 I/O)
```

#### **吞吐量計算**
```
基礎參數:
- GetRecords 調用: 1-2次/秒 (受退避策略影響)
- 每次獲取記錄: 最多 1000 筆
- Worker Pool 並行: 10x 加速
- 處理效率: 70% (考慮 I/O 等待和競爭)

保守估算:
單 Pod 單 Shard 吞吐量 = 1,400-2,000 筆/秒
```

### 水平擴展能力分析

#### **Pod 擴展效果**
```
分片鎖機制保證每個 Shard 只能被一個 Pod 消費:
- Pod ≤ Shard 數量: 線性擴展
- Pod > Shard 數量: 多餘 Pod 空閒

預期效果:
2 Pod + 1 Shard = 2,000 筆/秒 (無提升)  
1 Pod + 2 Shard = 2,800-4,000 筆/秒 (線性提升)
```

#### **Shard 擴展效果**  
```
每個 Shard 獨立處理，近線性擴展:
1 Shard = 1,400-2,000 筆/秒
2 Shard = 2,800-4,000 筆/秒
4 Shard = 5,600-8,000 筆/秒
8 Shard = 11,200-16,000 筆/秒
```

## 🎯 高吞吐量目標分析

### 挑戰: 達到 5,000-10,000 筆/秒

**當前基線**: 1,400-2,000 筆/秒  
**目標吞吐量**: 5,000-10,000 筆/秒  
**所需提升**: 2.5-7x 性能提升

### 性能瓶頸識別

#### **1. 主要瓶頸：串行等待**
```go
// 當前問題代碼
for {
    // 處理記錄
    newIterator, shouldContinue := k.processShardRecords(...)
    currentIterator = newIterator
    time.Sleep(backoffManager.GetCurrentBackoff()) // 🔥 性能殺手
}
```

**問題**: 即使有大量記錄待處理，仍然強制等待退避時間 (500ms-5s)

#### **2. 次要瓶頸：資源限制**
- Worker Pool 大小限制 (預設 10)
- Redis 連接池限制 (預設 20)
- 批次處理大小限制 (預設 100)
- KDS Record Limit 限制 (預設 1000)

## 🚀 優化方案

### 方案 1: 消除退避瓶頸 ⭐⭐⭐⭐⭐

#### **核心問題**
當前退避邏輯在**有記錄時仍然等待**，這是最大的性能殺手。

#### **優化實現**
```go
// 修改 consumeShardEvents 中的等待邏輯
for {
    select {
    case <-shardCtx.Done():
        return
    default:
        newIterator, shouldContinue, recordCount := k.processShardRecords(...)
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

**預期提升**: **3-5x** 吞吐量 (在高流量情況下)

#### **實施方式**
1. 修改 `processShardRecords` 返回記錄數量
2. 僅在無記錄時應用退避策略
3. 有記錄時立即繼續下一次拉取

### 方案 2: 參數調優 ⭐⭐⭐⭐

#### **環境變數優化**
```bash
# KDS 參數優化
CONSUMER_KDS_RECORD_LIMIT=10000      # AWS 允許的最大值 (預設 1000)

# Worker Pool 優化  
CONSUMER_WORKER_POOL_SIZE=50         # 大幅增加並行度 (預設 10)
CONSUMER_WORKER_BUFFER_SIZE=1000     # 增大緩衝區 (預設 200)

# 批次處理優化
CONSUMER_BATCH_SIZE=500              # 增大批次處理 (預設 100)

# 退避策略優化
CONSUMER_MIN_BACKOFF=50ms            # 減少最小退避 (預設 500ms)
CONSUMER_MAX_BACKOFF=1s              # 減少最大退避 (預設 5s)
```

**預期提升**: **1.5-2x** 吞吐量

### 方案 3: Redis 連接池優化 ⭐⭐⭐

#### **Redis 配置優化**
```bash
# 連接池優化
REDIS_POOL_SIZE=100              # 預設 20 → 100
REDIS_MIN_IDLE_CONNS=20          # 預設 5 → 20

# 超時優化  
REDIS_READ_TIMEOUT=1s            # 預設 3s → 1s
REDIS_WRITE_TIMEOUT=1s           # 預設 3s → 1s
REDIS_POOL_TIMEOUT=2s            # 預設 4s → 2s

# 連接生命週期優化
REDIS_IDLE_TIMEOUT=5m            # 連接空閒時間
REDIS_MAX_CONN_AGE=30m           # 連接最大生命週期
```

**預期提升**: **1.2-1.5x** 吞吐量

### 方案 4: 記憶體快取優化 ⭐⭐⭐

#### **本地快取設計**
```go
type EventCache struct {
    cache sync.Map              // 線程安全的 map
    ttl   time.Duration         // TTL = 24小時 (與 Redis 一致)
}

// 混合快取策略
func (k *KDSService) optimizedCheckEventsProcessed(
    ctx context.Context, 
    eventIDs []string,
) map[string]bool {
    localFiltered := make([]string, 0, len(eventIDs))
    processedMap := make(map[string]bool)
    
    // 1. 本地快取過濾
    for _, eventID := range eventIDs {
        if k.localCache.Exists(eventID) {
            processedMap[eventID] = true
        } else {
            localFiltered = append(localFiltered, eventID)
        }
    }
    
    // 2. Redis 批次查詢 (僅查詢本地快取未命中的)
    if len(localFiltered) > 0 {
        redisResults := k.batchCheckEventsProcessed(ctx, localFiltered)
        // 合併結果並更新本地快取
        for eventID, processed := range redisResults {
            processedMap[eventID] = processed
            if processed {
                k.localCache.Set(eventID)
            }
        }
    }
    
    return processedMap
}
```

**優勢**:
- 減少 Redis 查詢次數 60-80%
- 降低網絡延遲
- 提升批次處理速度

**預期提升**: **1.3-1.8x** 吞吐量

## 📈 綜合優化效果

### 保守估算 (方案 1 + 2)
```
基準吞吐量: 2,000 筆/秒
× 消除退避瓶頸: 3x
× 參數調優: 1.5x
= 9,000 筆/秒 ✅ 達到目標範圍
```

### 積極優化 (所有方案)
```
基準吞吐量: 2,000 筆/秒  
× 消除退避瓶頸: 4x
× 參數調優: 2x
× Redis 優化: 1.3x  
× 記憶體快取: 1.5x
= 15,600 筆/秒 ✅ 超越目標
```

### 實際部署建議

#### **階段一: 立即優化**
1. **消除串行等待** (程式碼修改)
2. **關鍵參數調優** (環境變數)  
3. **Redis 連接池優化** (環境變數)

**預期結果**: 8,000-12,000 筆/秒

#### **階段二: 進階優化** 
4. **本地記憶體快取** (程式碼開發)
5. **JSON 解析優化** (使用更快的 JSON 庫)

**預期結果**: 12,000-16,000 筆/秒

## 🏆 結論

### 可行性評估
**在不大幅修改架構的前提下，達到 5,000-10,000 筆/秒是完全可行的！**

### 關鍵成功因素
1. **消除串行等待** - 最大的性能解鎖點
2. **合理參數調優** - 充分利用硬體資源
3. **I/O 路徑優化** - Redis/DynamoDB 連接優化  
4. **智能快取策略** - 減少重複查詢

### 最終性能預期
- **保守估算**: 8,000-10,000 筆/秒 ✅
- **積極優化**: 12,000-16,000 筆/秒 ✅

### 架構影響評估
- ✅ **架構友好**: 不破壞現有設計模式
- ✅ **可靠性保持**: 維持所有安全機制
- ✅ **擴展性增強**: 為多 Shard 擴展奠定基礎
- ✅ **維護性提升**: 優化後的代碼更清晰

## 📝 相關文檔

- [優化實施指南](PERFORMANCE_OPTIMIZATION_GUIDE.md)
- [效能基準測試](PERFORMANCE_BENCHMARK.md)  
- [技術規格文檔](TECHNICAL_SPECIFICATION.md)
- [重構計畫文檔](CONSUMER_REFACTOR_PLAN.md)