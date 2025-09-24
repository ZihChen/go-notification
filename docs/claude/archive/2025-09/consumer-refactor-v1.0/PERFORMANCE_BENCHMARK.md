# Consumer Performance Benchmark

**文件版本**: v1.0  
**創建日期**: 2025-09-24  
**最後更新**: 2025-09-24  
**狀態**: 📊 基準測試方案

## 📋 概述

本文檔定義 KDS Consumer 性能基準測試方法、測試環境、關鍵指標和驗收標準，用於驗證重構後的性能提升效果。

## 🎯 測試目標

### 性能基準驗證
- **基線測試**: 驗證重構後基礎性能 (1,400-2,000 筆/秒)
- **優化測試**: 驗證優化後性能提升 (5,000-10,000 筆/秒)
- **擴展測試**: 驗證水平擴展效果 (多 Pod 多 Shard)
- **穩定性測試**: 驗證長時間運行穩定性

### 對比測試目標
- **重構前 vs 重構後**: 驗證架構重構效果
- **基線配置 vs 優化配置**: 驗證參數調優效果  
- **單 Shard vs 多 Shard**: 驗證擴展效果
- **單 Pod vs 多 Pod**: 驗證並行效果

## 🔧 測試環境

### 硬體配置
```yaml
# Docker Container 資源配置
resources:
  limits:
    cpu: "4.0"
    memory: "4Gi" 
  requests:
    cpu: "2.0"
    memory: "2Gi"

# Host Machine 建議配置
cpu: 8 cores (Intel i7 or equivalent)
memory: 16GB RAM
storage: SSD 100GB+
network: 1Gbps
```

### 依賴服務配置
```yaml
# AWS Kinesis Data Stream
shards: 1, 2, 4, 8 (分別測試)
retention: 24 hours
encryption: 無 (測試環境)

# Redis Configuration  
memory: 2GB
max_connections: 1000
timeout: 300
tcp-keepalive: 60

# MySQL/DynamoDB
connection_pool: 100
timeout: 5s
max_idle: 25
```

### 測試數據生成
```go
// 測試事件結構
type TestEvent struct {
    ID               string                 `json:"id"`
    Type             string                 `json:"type"`
    Timestamp        time.Time              `json:"timestamp"`
    Data             map[string]interface{} `json:"data"`
    GlobalMerchantID string                 `json:"global_merchant_id,omitempty"`
}

// 測試數據特徵
EventSize: 500-2000 bytes (平均 1KB)
EventTypes: 6 種類型平均分布
MerchantIDs: 100 個商戶ID輪換
Payload: 隨機JSON數據
```

## 📊 測試方案

### 階段一: 基線性能測試

#### 測試 1.1: 單 Pod 單 Shard 基線測試
```bash
# 測試配置
PODS=1
SHARDS=1
DURATION=300s  # 5分鐘
EVENT_RATE=2000/s  # 目標吞吐量

# 環境變數 (預設配置)
CONSUMER_KDS_RECORD_LIMIT=1000
CONSUMER_WORKER_POOL_SIZE=10
CONSUMER_BATCH_SIZE=100
CONSUMER_MIN_BACKOFF=500ms
CONSUMER_MAX_BACKOFF=5s

# 執行命令
./scripts/benchmark_baseline.sh
```

**預期結果**: 1,400-2,000 筆/秒

#### 測試 1.2: 穩定性測試
```bash
# 長時間運行測試
DURATION=3600s  # 1小時
EVENT_RATE=1500/s  # 保守速率
MONITORING_INTERVAL=60s

./scripts/benchmark_stability.sh
```

**預期結果**: 
- 吞吐量穩定在 1,400+ 筆/秒
- 記憶體使用穩定 < 2GB
- CPU 使用率 < 60%
- 錯誤率 < 0.1%

### 階段二: 優化效果測試

#### 測試 2.1: 退避策略優化測試
```bash
# 修改退避邏輯後測試
CODE_VERSION=optimized_backoff
EVENT_RATE=5000/s

# 預期效果: 3-5x 提升
./scripts/benchmark_backoff_optimized.sh
```

**預期結果**: 4,200-10,000 筆/秒

#### 測試 2.2: 參數調優測試
```bash
# 優化參數配置
CONSUMER_KDS_RECORD_LIMIT=10000
CONSUMER_WORKER_POOL_SIZE=50
CONSUMER_BATCH_SIZE=500
CONSUMER_MIN_BACKOFF=50ms
CONSUMER_MAX_BACKOFF=1s
REDIS_POOL_SIZE=100
REDIS_READ_TIMEOUT=1s
REDIS_WRITE_TIMEOUT=1s

EVENT_RATE=8000/s
./scripts/benchmark_tuned_params.sh
```

**預期結果**: 6,000-12,000 筆/秒

#### 測試 2.3: 本地快取測試
```bash
# 啟用本地記憶體快取
LOCAL_CACHE_ENABLED=true
CACHE_TTL=24h
EVENT_RATE=10000/s

./scripts/benchmark_local_cache.sh
```

**預期結果**: 8,000-16,000 筆/秒

### 階段三: 水平擴展測試

#### 測試 3.1: 多 Shard 擴展測試
```bash
# 測試不同 Shard 數量
for SHARDS in 1 2 4 8; do
    PODS=1
    EVENT_RATE=$((2000 * $SHARDS))
    ./scripts/benchmark_multi_shard.sh $SHARDS $PODS
done
```

**預期結果**:
- 1 Shard: 2,000 筆/秒
- 2 Shard: 4,000 筆/秒  
- 4 Shard: 8,000 筆/秒
- 8 Shard: 16,000 筆/秒

#### 測試 3.2: 多 Pod 擴展測試  
```bash
# 測試 Pod 與 Shard 的組合效果
SCENARIOS=(
    "1,1"   # 1 Pod, 1 Shard
    "2,2"   # 2 Pod, 2 Shard
    "4,4"   # 4 Pod, 4 Shard
    "4,2"   # 4 Pod, 2 Shard (過配置)
)

for scenario in "${SCENARIOS[@]}"; do
    IFS=',' read -r PODS SHARDS <<< "$scenario"
    ./scripts/benchmark_pod_shard_combo.sh $PODS $SHARDS
done
```

**預期結果**:
- 1P1S: 2,000 筆/秒
- 2P2S: 4,000 筆/秒  
- 4P4S: 8,000 筆/秒
- 4P2S: 4,000 筆/秒 (Pod 過配置無效果)

## 📈 關鍵指標

### 性能指標 (Primary KPIs)
```go
type PerformanceBenchmark struct {
    // 吞吐量指標
    ThroughputPerSecond    float64   // 每秒處理事件數
    PeakThroughput        float64   // 峰值吞吐量
    AverageThroughput     float64   // 平均吞吐量
    
    // 延遲指標  
    ProcessingLatencyP50  time.Duration  // 中位數延遲
    ProcessingLatencyP95  time.Duration  // 95百分位延遲
    ProcessingLatencyP99  time.Duration  // 99百分位延遲
    EndToEndLatencyP95    time.Duration  // 端到端延遲
    
    // 可靠性指標
    SuccessRate          float64   // 成功率 %
    ErrorRate            float64   // 錯誤率 %  
    DuplicateRate        float64   // 重複處理率 %
    MessageLossRate      float64   // 消息丟失率 %
}
```

### 資源指標 (Resource KPIs)
```go
type ResourceBenchmark struct {
    // CPU 使用率
    CPUUsageAvg    float64  // 平均 CPU 使用率 %
    CPUUsagePeak   float64  // 峰值 CPU 使用率 %
    
    // 記憶體使用
    MemoryUsageAvg   int64   // 平均記憶體使用 MB
    MemoryUsagePeak  int64   // 峰值記憶體使用 MB
    GCPauseTime      time.Duration  // GC 暫停時間
    
    // 網絡 I/O
    NetworkInbound   int64   // 入站流量 MB/s
    NetworkOutbound  int64   // 出站流量 MB/s
    
    // 存儲 I/O
    RedisConnections int     // Redis 連接數
    RedisLatency     time.Duration  // Redis 操作延遲
    DynamoDBLatency  time.Duration  // DynamoDB 操作延遲
}
```

### 業務指標 (Business KPIs)
```go
type BusinessBenchmark struct {
    // 快取效率
    LocalCacheHitRate   float64  // 本地快取命中率 %
    RedisCacheHitRate   float64  // Redis 快取命中率 %
    
    // 批次效率  
    AverageBatchSize    float64  // 平均批次大小
    BatchUtilization    float64  // 批次利用率 %
    
    // 退避效率
    BackoffFrequency    float64  // 退避觸發頻率
    IdleWaitTime        time.Duration  // 空閒等待時間
    
    // 分片效率
    ShardLoadBalance    float64  // 分片負載平衡度
    CheckpointLag       time.Duration  // Checkpoint 延遲
}
```

## 🧪 測試腳本

### 基準測試腳本框架
```bash
#!/bin/bash
# 文件: scripts/benchmark_framework.sh

set -e

# 配置參數
TEST_NAME=${1:-"baseline"}
DURATION=${2:-"300"}
EVENT_RATE=${3:-"2000"}
WARMUP_TIME=${4:-"60"}

echo "🚀 開始 Consumer 性能測試: $TEST_NAME"
echo "⏱️  測試時長: ${DURATION}s"
echo "📊 目標吞吐量: ${EVENT_RATE}/s"

# 1. 環境準備
echo "📋 準備測試環境..."
docker-compose down
docker-compose up -d redis mysql
sleep 30

# 2. 啟動 Consumer
echo "🔄 啟動 Consumer..."  
docker-compose up -d consumer
sleep $WARMUP_TIME

# 3. 開始數據生成
echo "📡 開始數據生成..."
./scripts/data_generator.sh $EVENT_RATE $DURATION &
GENERATOR_PID=$!

# 4. 監控數據收集
echo "📈 開始監控..."
./scripts/metrics_collector.sh $DURATION &
METRICS_PID=$!

# 5. 等待測試完成
wait $GENERATOR_PID
sleep 30  # 等待處理完成

# 6. 停止監控
kill $METRICS_PID 2>/dev/null || true

# 7. 收集結果
echo "📊 收集測試結果..."
./scripts/results_analyzer.sh > "results/${TEST_NAME}_$(date +%Y%m%d_%H%M%S).json"

# 8. 清理環境
docker-compose down

echo "✅ 測試完成: $TEST_NAME"
```

### 數據生成器
```bash
#!/bin/bash
# 文件: scripts/data_generator.sh

RATE=${1:-2000}
DURATION=${2:-300}
TOTAL_EVENTS=$((RATE * DURATION))

echo "📡 生成 $TOTAL_EVENTS 個測試事件 (${RATE}/s for ${DURATION}s)"

python3 scripts/event_generator.py \
    --rate $RATE \
    --duration $DURATION \
    --stream-name $KDS_STREAM_NAME \
    --event-types merchant,player,manager,level,tags,tag \
    --merchants 100 \
    --payload-size-min 500 \
    --payload-size-max 2000
```

### 指標收集器
```bash  
#!/bin/bash
# 文件: scripts/metrics_collector.sh

DURATION=${1:-300}
INTERVAL=5
SAMPLES=$((DURATION / INTERVAL))

echo "📊 收集性能指標 (${SAMPLES} samples, ${INTERVAL}s interval)"

# 收集 Consumer 指標
for i in $(seq 1 $SAMPLES); do
    timestamp=$(date +%s)
    
    # Container 指標
    docker stats --no-stream --format "table {{.Name}},{{.CPUPerc}},{{.MemUsage}}" consumer >> metrics/container_stats.csv
    
    # Redis 指標  
    redis-cli info stats | grep -E "instantaneous_ops_per_sec|used_memory:" >> metrics/redis_stats.txt
    
    # 應用指標 (如果有 Prometheus endpoint)
    curl -s http://localhost:8080/metrics >> metrics/app_metrics.txt
    
    sleep $INTERVAL
done

echo "✅ 指標收集完成"
```

### 結果分析器
```python
#!/usr/bin/env python3
# 文件: scripts/results_analyzer.py

import json
import pandas as pd
from datetime import datetime
import sys

def analyze_performance():
    """分析性能測試結果"""
    
    # 讀取原始數據
    container_stats = pd.read_csv('metrics/container_stats.csv')
    redis_stats = parse_redis_stats('metrics/redis_stats.txt')
    app_metrics = parse_app_metrics('metrics/app_metrics.txt')
    
    # 計算性能指標
    results = {
        'timestamp': datetime.now().isoformat(),
        'test_config': get_test_config(),
        'performance': {
            'throughput_avg': calculate_avg_throughput(app_metrics),
            'throughput_peak': calculate_peak_throughput(app_metrics),
            'latency_p50': calculate_latency_percentile(app_metrics, 0.5),
            'latency_p95': calculate_latency_percentile(app_metrics, 0.95),
            'latency_p99': calculate_latency_percentile(app_metrics, 0.99),
            'success_rate': calculate_success_rate(app_metrics),
            'error_rate': calculate_error_rate(app_metrics),
        },
        'resources': {
            'cpu_avg': container_stats['CPUPerc'].str.rstrip('%').astype(float).mean(),
            'cpu_peak': container_stats['CPUPerc'].str.rstrip('%').astype(float).max(),
            'memory_avg': parse_memory_usage(container_stats['MemUsage']).mean(),
            'memory_peak': parse_memory_usage(container_stats['MemUsage']).max(),
        },
        'business': {
            'cache_hit_rate': calculate_cache_hit_rate(redis_stats),
            'avg_batch_size': calculate_avg_batch_size(app_metrics),
            'backoff_frequency': calculate_backoff_frequency(app_metrics),
        }
    }
    
    return results

if __name__ == '__main__':
    results = analyze_performance()
    print(json.dumps(results, indent=2))
```

## 📋 驗收標準

### 基線性能標準
```yaml
baseline_requirements:
  throughput:
    min: 1400  # 筆/秒
    target: 2000  # 筆/秒
  latency:
    p95: <100ms  # 95百分位延遲
    p99: <200ms  # 99百分位延遲
  reliability:
    success_rate: >99.9%
    error_rate: <0.1%
  resources:
    cpu_avg: <60%
    memory_peak: <2GB
```

### 優化性能標準
```yaml
optimized_requirements:
  throughput:
    min: 5000   # 筆/秒
    target: 10000  # 筆/秒
  latency:
    p95: <100ms  # 保持延遲不增加
    p99: <200ms
  reliability:
    success_rate: >99.9%  # 維持可靠性
    error_rate: <0.1%
  resources:
    cpu_avg: <80%   # 允許更高 CPU 使用
    memory_peak: <4GB  # 允許更大記憶體使用
```

### 擴展性標準
```yaml
scalability_requirements:
  shard_scaling:
    efficiency: >90%  # 擴展效率
    linear_factor: >0.9  # 線性因子
  pod_scaling:
    max_effective_pods: equals_shard_count
  load_balancing:
    shard_variance: <20%  # 分片負載差異
```

### 穩定性標準
```yaml
stability_requirements:
  duration: 3600s  # 1小時持續測試
  throughput_variance: <10%  # 吞吐量變異性
  memory_leak_rate: 0  # 無記憶體洩漏
  connection_leak_rate: 0  # 無連接洩漏
  gc_pause_p99: <50ms  # GC 暫停時間
```

## 📈 結果報告範例

### 測試報告模板
```json
{
  "test_report": {
    "metadata": {
      "test_name": "baseline_performance",
      "timestamp": "2025-09-24T10:30:00Z",
      "duration": "300s",
      "environment": "docker_compose_local"
    },
    "configuration": {
      "pods": 1,
      "shards": 1,
      "consumer_config": {
        "kds_record_limit": 1000,
        "worker_pool_size": 10,
        "batch_size": 100
      }
    },
    "results": {
      "performance": {
        "throughput_avg": 1850,
        "throughput_peak": 2100,
        "latency_p95": 85,
        "success_rate": 99.95,
        "error_rate": 0.05
      },
      "verdict": "PASS",
      "score": 85
    }
  }
}
```

## 🔄 持續測試流程

### CI/CD 集成
```yaml
# .github/workflows/performance_test.yml
name: Performance Benchmark

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

jobs:
  performance_test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Test Environment
        run: ./scripts/setup_test_env.sh
        
      - name: Run Baseline Test
        run: ./scripts/benchmark_baseline.sh
        
      - name: Run Optimized Test  
        run: ./scripts/benchmark_optimized.sh
        
      - name: Compare Results
        run: ./scripts/compare_results.sh
        
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: results/
```

### 定期性能監控
```bash
# 每日性能監控 cron job
0 2 * * * /opt/consumer/scripts/daily_performance_check.sh
```

## 🎯 成功標準總結

### 必要條件 (MUST HAVE)
- ✅ 基線性能: ≥1,400 筆/秒
- ✅ 可靠性: 99.9% 成功率
- ✅ 資源效率: CPU <80%, Memory <4GB
- ✅ 穩定性: 1小時無性能衰減

### 期望條件 (SHOULD HAVE)  
- 🎯 優化性能: ≥5,000 筆/秒
- 🎯 低延遲: P95 <100ms
- 🎯 高效擴展: >90% 擴展效率
- 🎯 快取效率: >60% 命中率

### 卓越條件 (NICE TO HAVE)
- 🏆 極限性能: ≥10,000 筆/秒  
- 🏆 超低延遲: P95 <50ms
- 🏆 完美擴展: >95% 擴展效率
- 🏆 超高效率: >80% 快取命中率

## 📚 相關資源

- **[性能分析文檔](PERFORMANCE_ANALYSIS.md)** - 詳細性能分析
- **[優化實施指南](PERFORMANCE_OPTIMIZATION_GUIDE.md)** - 優化實施步驟
- **[技術規格文檔](TECHNICAL_SPECIFICATION.md)** - 技術架構規格
- **[AWS Kinesis 性能最佳實踐](https://docs.aws.amazon.com/kinesis/latest/dev/troubleshooting-consumers.html)**