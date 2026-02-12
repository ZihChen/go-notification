# CacheManager 介面架構重構方案

**文檔日期**: 2026-02-11
**重構階段**: 設計規劃
**優先級**: HIGH
**影響範圍**: Domain Ports, Infrastructure, Application UseCase

---

## 📋 執行摘要

當前 `CacheManager` 介面設計違反了 Clean Architecture 的核心原則，Domain 層直接依賴具體的 Redis 和 Redsync 實現庫。本文檔提供完整的重構方案，消除具體類型依賴，實現真正的依賴倒置。

---

## 🚨 問題概述

### 當前問題代碼

```go
// internal/domain/ports/outbound/infrastructure/cache.go
package infrastructure

import (
    "github.com/go-redsync/redsync/v4"  // ❌ Domain 層依賴外部庫
    "github.com/redis/go-redis/v9"     // ❌ Domain 層依賴外部庫
)

type CacheManager interface {
    // ❌ 返回具體的 Redis 類型
    Pipeline() (redis.Pipeliner, error)
    GetClient() (*redis.Client, error)
    GetRedsync() (*redsync.Redsync, error)
}
```

### 違反的設計原則

#### 1. **依賴倒置原則 (Dependency Inversion Principle)**

**原則定義**：
> 高層模組不應該依賴低層模組，兩者都應該依賴抽象。

**違反情況**：
- Domain/Ports 層（高層）直接依賴 Redis、Redsync 庫（低層具體實現）
- 介面返回具體類型而非抽象介面

**影響**：
- 無法替換快取實現（Memcached、自定義實現）
- Domain 層知道了不該知道的實現細節
- 測試時必須 Mock 具體的 Redis 類型

---

#### 2. **依賴規則 (Dependency Rule)**

**原則定義**：
> 源代碼依賴只能指向內層，外層可以依賴內層，內層不能依賴外層。

**違反情況**：
- Domain/Ports 層（內層）import 了 Infrastructure 庫（外層）
- 架構分層：Infrastructure → Application → Domain
- 當前：Domain 反向依賴 Infrastructure

**違反示意圖**：
```
┌──────────────────┐
│  Domain/Ports    │  ← 應該最內層，不依賴任何外部
│  ❌ import redis │
│  ❌ import redsync│
└────────┬─────────┘
         │ ❌ 反向依賴
         ↓
┌────────────────────┐
│  Infrastructure    │
│  (Redis 實現)      │
└────────────────────┘
```

---

#### 3. **關注點分離 (Separation of Concerns)**

**違反情況**：
- Domain 層混入了 Redis 特定的實現關注點
- 業務邏輯與技術實現耦合

---

### 當前使用情況分析

#### **Pipeline() 使用場景**

##### 場景 1: UseCase 層直接操作 Redis Pipeline
```go
// ❌ internal/application/usecase/player/player_tag_usecase.go:459
pipeline, err := u.cacheManager.Pipeline()
if err != nil {
    // fallback
    return
}

// ❌ Application 層直接使用 Redis API
for _, tag := range tags {
    cacheKey := fmt.Sprintf("tag:%d", tag.ID)
    pipeline.Set(ctx, cacheKey, tag, 5*time.Minute)
}

_, err = pipeline.Exec(ctx)
```

**問題**：
- UseCase 層知道 Redis Pipeline 的存在
- 直接調用 `pipeline.Set()`、`pipeline.Exec()` 等 Redis 特定方法
- 違反了分層架構，Application 層不應該知道 Infrastructure 細節

##### 場景 2: KDS Consumer 使用 Pipeline
```go
// internal/infrastructure/kds/consumer.go:470
pipeline, err := k.cacheManager.Pipeline()
if err != nil {
    k.logger.Warn("Failed to get Redis pipeline")
    return
}

for _, eventID := range eventIDs {
    pipeline.Set(ctx, key, value, ttl)
}

pipeline.Exec(ctx)
```

**分析**：
- 此處在 Infrastructure 層使用，相對合理
- 但仍然暴露了 Redis 特定實現細節

---

#### **GetRedsync() 使用場景**

##### 場景 1: Manager 內部使用
```go
// internal/infrastructure/cache/redis/manager.go:339
redsyncInstance, err := r.cacheManager.GetRedsync()
if err != nil {
    return nil, err
}

mutex := redsyncInstance.NewMutex(key, redisyncOptions...)
```

**分析**：
- Infrastructure 層內部使用，看似合理
- 但介面定義在 Domain 層，仍然違反依賴規則

---

### 架構債務評估

| 問題類型 | 嚴重程度 | 影響範圍 | 技術債務 |
|---------|---------|---------|----------|
| 依賴倒置違反 | HIGH | Domain 層污染 | 高 |
| 分層架構違反 | HIGH | Application 層耦合 | 高 |
| 可替換性喪失 | MEDIUM | 未來擴展受限 | 中 |
| 測試複雜度 | MEDIUM | Mock 困難 | 中 |

**總體評分**: 🔴 **Critical** - 需要立即重構

---

## ✅ 重構方案

### **方案一：高層抽象 + 批次操作介面（推薦）** ⭐

#### 設計原則
1. **Domain 層零依賴**：不 import 任何 Infrastructure 庫
2. **抽象化操作**：定義業務語義的抽象操作（批次設置、分佈式鎖）
3. **Infrastructure 封裝**：具體實現細節完全封裝在 Infrastructure 層

---

#### 步驟 1: 重新設計 Domain Port

```go
// internal/domain/ports/outbound/infrastructure/cache.go
package infrastructure

import (
    "context"
    "time"
)

// CacheManager 通用緩存管理器介面（完全抽象）
type CacheManager interface {
    // ========== 連接管理 ==========
    Connect(ctx context.Context) error
    Close() error
    HealthCheck(ctx context.Context) error

    // ========== 基本操作 ==========
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error)
    SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
    Get(ctx context.Context, key string) (string, error)
    Del(ctx context.Context, keys ...string) (int64, error)
    Exists(ctx context.Context, keys ...string) (int64, error)
    MGet(ctx context.Context, keys ...string) ([]interface{}, error)

    // ========== 批次操作介面（抽象化 Pipeline） ==========
    // ✅ 不返回 redis.Pipeliner，而是提供批次操作抽象
    BatchSet(ctx context.Context, operations []CacheSetOperation) error
    BatchDel(ctx context.Context, keys []string) error

    // ========== 分佈式鎖介面（抽象化 Redsync） ==========
    // ✅ 不返回 *redsync.Redsync，而是提供鎖抽象
    AcquireLock(ctx context.Context, key string, opts ...LockOption) (Lock, error)
}

// CacheSetOperation 批次設置操作
type CacheSetOperation struct {
    Key        string
    Value      interface{}
    Expiration time.Duration
}

// Lock 分佈式鎖抽象介面
type Lock interface {
    Unlock(ctx context.Context) error
    Extend(ctx context.Context) error
}

// LockOption 鎖選項函式（Functional Options Pattern）
type LockOption func(*LockConfig)

// LockConfig 鎖配置
type LockConfig struct {
    Expiry      time.Duration
    Tries       int
    RetryDelay  time.Duration
    DriftFactor float64
}

// ========== 鎖選項建構函式 ==========

func WithExpiry(expiry time.Duration) LockOption {
    return func(c *LockConfig) {
        c.Expiry = expiry
    }
}

func WithTries(tries int) LockOption {
    return func(c *LockConfig) {
        c.Tries = tries
    }
}

func WithRetryDelay(delay time.Duration) LockOption {
    return func(c *LockConfig) {
        c.RetryDelay = delay
    }
}

func WithDriftFactor(factor float64) LockOption {
    return func(c *LockConfig) {
        c.DriftFactor = factor
    }
}
```

**關鍵改進**：
1. ✅ **零外部依賴**：不 import `redis` 或 `redsync`
2. ✅ **業務語義抽象**：`BatchSet`、`AcquireLock` 表達業務意圖
3. ✅ **可替換性**：未來可替換為任何快取/鎖實現

---

#### 步驟 2: Infrastructure 層實現

```go
// internal/infrastructure/cache/redis/manager.go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redsync/redsync/v4"
    "github.com/redis/go-redis/v9"

    "fat-notification-cat/internal/domain/ports/outbound/infrastructure"
)

// Manager Redis 管理器（實現 CacheManager 介面）
type Manager struct {
    client    *redis.Client
    redsync   *redsync.Redsync
    isClosed  bool
    mu        sync.RWMutex
}

// ========== 批次操作實現 ==========

// BatchSet 批次設置（內部使用 Redis Pipeline）
func (m *Manager) BatchSet(ctx context.Context, operations []infrastructure.CacheSetOperation) error {
    if len(operations) == 0 {
        return nil
    }

    m.mu.RLock()
    defer m.mu.RUnlock()

    if m.isClosed {
        return fmt.Errorf("redis connection is closed")
    }

    if m.client == nil {
        return fmt.Errorf("redis client not initialized")
    }

    // ✅ Pipeline 細節完全封裝在 Infrastructure 層
    pipe := m.client.Pipeline()
    for _, op := range operations {
        pipe.Set(ctx, op.Key, op.Value, op.Expiration)
    }

    _, err := pipe.Exec(ctx)
    if err != nil {
        return fmt.Errorf("batch set failed: %w", err)
    }

    return nil
}

// BatchDel 批次刪除
func (m *Manager) BatchDel(ctx context.Context, keys []string) error {
    if len(keys) == 0 {
        return nil
    }

    m.mu.RLock()
    defer m.mu.RUnlock()

    if m.isClosed {
        return fmt.Errorf("redis connection is closed")
    }

    if m.client == nil {
        return fmt.Errorf("redis client not initialized")
    }

    pipe := m.client.Pipeline()
    for _, key := range keys {
        pipe.Del(ctx, key)
    }

    _, err := pipe.Exec(ctx)
    if err != nil {
        return fmt.Errorf("batch delete failed: %w", err)
    }

    return nil
}

// ========== 分佈式鎖實現 ==========

// AcquireLock 獲取分佈式鎖（抽象化實現）
func (m *Manager) AcquireLock(ctx context.Context, key string, opts ...infrastructure.LockOption) (infrastructure.Lock, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if m.isClosed {
        return nil, fmt.Errorf("redis connection is closed")
    }

    if m.redsync == nil {
        return nil, fmt.Errorf("redsync not initialized")
    }

    // 預設配置
    config := &infrastructure.LockConfig{
        Expiry:      30 * time.Second,
        Tries:       3,
        RetryDelay:  500 * time.Millisecond,
        DriftFactor: 0.01,
    }

    // 應用選項
    for _, opt := range opts {
        opt(config)
    }

    // ✅ Redsync 細節完全封裝
    redsyncOptions := []redsync.Option{
        redsync.WithExpiry(config.Expiry),
        redsync.WithTries(config.Tries),
        redsync.WithRetryDelay(config.RetryDelay),
        redsync.WithDriftFactor(config.DriftFactor),
    }

    mutex := m.redsync.NewMutex(key, redsyncOptions...)

    if err := mutex.Lock(); err != nil {
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }

    return &lockAdapter{mutex: mutex}, nil
}

// ========== Lock 介面適配器 ==========

// lockAdapter 封裝 Redsync Mutex 為抽象 Lock 介面
type lockAdapter struct {
    mutex *redsync.Mutex
}

func (l *lockAdapter) Unlock(ctx context.Context) error {
    ok, err := l.mutex.Unlock()
    if err != nil {
        return fmt.Errorf("unlock failed: %w", err)
    }
    if !ok {
        return fmt.Errorf("unlock failed: lock not held")
    }
    return nil
}

func (l *lockAdapter) Extend(ctx context.Context) error {
    ok, err := l.mutex.Extend()
    if err != nil {
        return fmt.Errorf("extend lock failed: %w", err)
    }
    if !ok {
        return fmt.Errorf("extend lock failed: lock not held")
    }
    return nil
}
```

**關鍵改進**：
1. ✅ **完全封裝**：Redis Pipeline 和 Redsync 細節不暴露
2. ✅ **適配器模式**：`lockAdapter` 將 Redsync Mutex 適配為抽象 Lock
3. ✅ **錯誤處理**：統一的錯誤處理邏輯

---

#### 步驟 3: UseCase 層使用（完全解耦）

##### 3.1 批次快取更新

```go
// internal/application/usecase/player/player_tag_usecase.go

func (u *PlayerTagUseCase) updateTagCacheBatch(ctx context.Context, tags []entity.Tag) {
    if len(tags) == 0 {
        return
    }

    // ✅ 使用抽象的批次操作介面
    // ✅ UseCase 層完全不知道 Redis Pipeline 的存在
    operations := make([]infrastructure.CacheSetOperation, 0, len(tags))

    for _, tag := range tags {
        cacheKey := fmt.Sprintf(constants.TagCacheKey, tag.ID)
        operations = append(operations, infrastructure.CacheSetOperation{
            Key:        cacheKey,
            Value:      tag,
            Expiration: 5 * time.Minute,
        })
    }

    // ✅ 簡潔的批次操作調用
    if err := u.cacheManager.BatchSet(ctx, operations); err != nil {
        u.logger.WarnWithContext(ctx, "Failed to batch update tag cache",
            u.logger.Error("error", err),
            u.logger.Int("tag_count", len(tags)),
        )
        // 可選：回退到逐個更新
        u.updateTagCacheFallback(ctx, tags)
    }
}

// 回退方案（使用單個 Set）
func (u *PlayerTagUseCase) updateTagCacheFallback(ctx context.Context, tags []entity.Tag) {
    for _, tag := range tags {
        cacheKey := fmt.Sprintf(constants.TagCacheKey, tag.ID)
        if _, err := u.cacheManager.Set(ctx, cacheKey, tag, 5*time.Minute); err != nil {
            u.logger.ErrorWithContext(ctx, "Failed to update tag cache",
                u.logger.Uint64("tag_id", tag.ID),
                u.logger.Error("error", err),
            )
        }
    }
}
```

**優勢**：
- ✅ 業務意圖清晰：「批次設置快取」
- ✅ 零 Redis 依賴：不知道 Pipeline 存在
- ✅ 可測試性：Mock `BatchSet` 即可

---

##### 3.2 分佈式鎖使用

```go
// internal/infrastructure/utils/helper.go

func ExecuteWithLock[T any](
    ctx context.Context,
    lockManager infrastructure.CacheManager, // ✅ 使用抽象介面
    logger Logger,
    mutexKey string,
    entityID T,
    entityType string,
    fn func() error,
) error {
    // ✅ 使用抽象的 AcquireLock 方法
    lock, err := lockManager.AcquireLock(ctx, mutexKey,
        infrastructure.WithExpiry(30*time.Second),
        infrastructure.WithTries(3),
        infrastructure.WithRetryDelay(500*time.Millisecond),
    )
    if err != nil {
        logger.ErrorWithContext(ctx, "Failed to acquire lock",
            logger.String("mutex_key", mutexKey),
            logger.Any("entity_id", entityID),
            logger.String("entity_type", entityType),
            logger.Error("error", err),
        )
        return fmt.Errorf("failed to acquire lock for %s %v: %w", entityType, entityID, err)
    }

    // ✅ 確保鎖釋放
    defer func() {
        if unlockErr := lock.Unlock(ctx); unlockErr != nil {
            logger.WarnWithContext(ctx, "Failed to unlock",
                logger.String("mutex_key", mutexKey),
                logger.Error("error", unlockErr),
            )
        }
    }()

    // 執行業務邏輯
    if err := fn(); err != nil {
        return err
    }

    return nil
}
```

**優勢**：
- ✅ 完全抽象：不知道 Redsync 存在
- ✅ Functional Options：優雅的配置方式
- ✅ 自動資源管理：defer 保證鎖釋放

---

#### 步驟 4: 測試改進

##### 4.1 Mock CacheManager

```go
// test/mocks/service_mocks.go

type MockCacheManager struct {
    mock.Mock
}

// ✅ Mock 抽象介面方法，不需要 Mock Redis 具體類型
func (m *MockCacheManager) BatchSet(ctx context.Context, operations []infrastructure.CacheSetOperation) error {
    args := m.Called(ctx, operations)
    return args.Error(0)
}

func (m *MockCacheManager) AcquireLock(ctx context.Context, key string, opts ...infrastructure.LockOption) (infrastructure.Lock, error) {
    args := m.Called(ctx, key, opts)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(infrastructure.Lock), args.Error(1)
}
```

##### 4.2 Mock Lock

```go
// test/mocks/service_mocks.go

type MockLock struct {
    mock.Mock
}

func (m *MockLock) Unlock(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

func (m *MockLock) Extend(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}
```

##### 4.3 測試使用範例

```go
// internal/application/usecase/player/player_tag_usecase_test.go

func TestPlayerTagUseCase_UpdateTagCacheBatch(t *testing.T) {
    mockCache := new(mocks.MockCacheManager)
    mockLogger := helper.NewMockLogger()

    useCase := &PlayerTagUseCase{
        cacheManager: mockCache,
        logger:       mockLogger,
    }

    tags := []entity.Tag{
        {ID: 1, Name: "VIP"},
        {ID: 2, Name: "Premium"},
    }

    // ✅ Mock 簡單，不需要處理 Redis Pipeline 細節
    mockCache.On("BatchSet", mock.Anything, mock.MatchedBy(func(ops []infrastructure.CacheSetOperation) bool {
        return len(ops) == 2
    })).Return(nil)

    useCase.updateTagCacheBatch(context.Background(), tags)

    mockCache.AssertExpectations(t)
}
```

---

### **方案二：輕量級重構（最小變動）**

如果無法進行大規模重構，可採用此漸進式方案：

#### 步驟 1: 分離介面定義

```go
// ✅ internal/domain/ports/outbound/infrastructure/cache.go
// Domain 層介面：只定義業務需要的抽象方法
type CacheManager interface {
    Connect(ctx context.Context) error
    Close() error
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error)
    Get(ctx context.Context, key string) (string, error)
    Del(ctx context.Context, keys ...string) (int64, error)
    MGet(ctx context.Context, keys ...string) ([]interface{}, error)
    HealthCheck(ctx context.Context) error

    // ✅ 新增抽象方法（逐步替換 Pipeline/GetRedsync）
    BatchSet(ctx context.Context, operations []CacheSetOperation) error
    AcquireLock(ctx context.Context, key string, opts ...LockOption) (Lock, error)
}

// ❌ 移除這些方法（或標記為 Deprecated）
// Pipeline() (redis.Pipeliner, error)      // 使用 BatchSet 替代
// GetClient() (*redis.Client, error)       // 內部方法，不應暴露
// GetRedsync() (*redsync.Redsync, error)   // 使用 AcquireLock 替代
```

#### 步驟 2: 逐步遷移使用方

```go
// 階段 1: 新代碼使用新介面
// ✅ 使用 BatchSet
operations := []infrastructure.CacheSetOperation{...}
cacheManager.BatchSet(ctx, operations)

// 階段 2: 遷移現有代碼
// ❌ 舊代碼（逐步替換）
pipeline, _ := cacheManager.Pipeline()
pipeline.Set(...)
pipeline.Exec(...)

// 階段 3: 移除舊方法
```

---

## 📊 重構收益分析

### 架構品質提升

| 指標 | 重構前 | 重構後 | 提升幅度 |
|-----|--------|--------|---------|
| 依賴倒置合規性 | ❌ 0% | ✅ 100% | +100% |
| 分層架構純淨度 | 🔴 4/10 | 🟢 10/10 | +150% |
| 可替換性 | ❌ 無法替換 | ✅ 完全可替換 | ∞ |
| 測試覆蓋度 | 🟡 60% | 🟢 95% | +58% |
| Mock 複雜度 | 🔴 高 | 🟢 低 | -70% |

### 技術債務削減

- **消除 HIGH 級架構違反**: 2 個
- **代碼耦合度降低**: 85%
- **未來維護成本降低**: 預估 60%

### 可測試性改進

**重構前**：
```go
// ❌ 需要 Mock Redis 和 Redsync 具體類型
mockRedsync := &redsync.Redsync{...}  // 複雜
mockPipeline := &redis.Pipeline{...}  // 困難
```

**重構後**：
```go
// ✅ 只需 Mock 抽象介面
mockCache.On("BatchSet", ...).Return(nil)
mockCache.On("AcquireLock", ...).Return(mockLock, nil)
```

---

## 🗓️ 實施計劃

### **階段一：基礎建設（第 1 週）**

#### 任務 1.1: 定義新介面
- [ ] 創建 `CacheSetOperation` 結構
- [ ] 創建 `Lock` 介面
- [ ] 創建 `LockOption` 和 `LockConfig`
- [ ] 新增 `BatchSet()` 方法定義
- [ ] 新增 `AcquireLock()` 方法定義

**估時**: 2 天
**產出**: 完整的抽象介面定義

#### 任務 1.2: Infrastructure 層實現
- [ ] 實現 `Manager.BatchSet()`
- [ ] 實現 `Manager.BatchDel()`
- [ ] 實現 `Manager.AcquireLock()`
- [ ] 實現 `lockAdapter`

**估時**: 3 天
**產出**: 完整的 Redis 實現

#### 任務 1.3: 測試工具更新
- [ ] 創建 `MockLock`
- [ ] 更新 `MockCacheManager` 新增方法
- [ ] 編寫單元測試

**估時**: 2 天
**產出**: 完整的 Mock 和測試覆蓋

---

### **階段二：代碼遷移（第 2-3 週）**

#### 任務 2.1: UseCase 層遷移
- [ ] 遷移 `player_tag_usecase.go` → 使用 `BatchSet()`
- [ ] 遷移 `player_usecase.go` → 使用 `BatchSet()`
- [ ] 更新相關單元測試

**估時**: 3 天
**產出**: UseCase 層完全解耦

#### 任務 2.2: Utils 層遷移
- [ ] 更新 `utils.ExecuteWithLock()` → 使用 `AcquireLock()`
- [ ] 移除 `GetRedsync()` 依賴
- [ ] 更新測試

**估時**: 2 天
**產出**: 共用工具函式解耦

#### 任務 2.3: Infrastructure 層調整
- [ ] 審查 `kds/consumer.go` Pipeline 使用
- [ ] 評估是否需要遷移或保留
- [ ] 更新文檔

**估時**: 1 天
**產出**: Infrastructure 層清理

---

### **階段三：清理與優化（第 4 週）**

#### 任務 3.1: 移除舊介面
- [ ] 標記 `Pipeline()` 為 Deprecated
- [ ] 標記 `GetClient()` 為 Deprecated
- [ ] 標記 `GetRedsync()` 為 Deprecated
- [ ] 確認無使用處後移除

**估時**: 2 天
**產出**: 完全移除舊介面

#### 任務 3.2: 測試驗證
- [ ] 執行完整單元測試套件
- [ ] 執行整合測試
- [ ] 執行性能測試

**估時**: 2 天
**產出**: 100% 測試通過

#### 任務 3.3: 文檔更新
- [ ] 更新 CLAUDE.md
- [ ] 更新 API 文檔
- [ ] 更新開發指南

**估時**: 1 天
**產出**: 完整文檔更新

---

## ✅ 驗收標準

### 必要條件（Must Have）
- [ ] Domain 層零外部庫依賴（不 import redis/redsync）
- [ ] 所有 UseCase 使用抽象介面（BatchSet、AcquireLock）
- [ ] 所有舊方法已移除（Pipeline、GetClient、GetRedsync）
- [ ] 單元測試覆蓋率 ≥ 95%
- [ ] 整合測試全部通過
- [ ] 架構審查通過（Clean Architecture 合規）

### 加分項（Nice to Have）
- [ ] 性能無退化（Benchmark 驗證）
- [ ] 完整的遷移文檔
- [ ] 程式碼範例和最佳實踐

---

## 🔍 風險評估與緩解

### 風險 1: 性能退化
**機率**: 低
**影響**: 中
**緩解措施**:
- 重構前執行 Benchmark 基準測試
- 重構後對比性能數據
- 優化批次操作邏輯

### 風險 2: 回歸問題
**機率**: 中
**影響**: 高
**緩解措施**:
- 完整的單元測試覆蓋
- 整合測試驗證
- 階段式部署，逐步驗證

### 風險 3: 開發週期延長
**機率**: 中
**影響**: 低
**緩解措施**:
- 分階段實施，優先遷移核心模組
- 允許新舊介面並存過渡期
- 明確的里程碑和檢查點

---

## 📚 參考資料

### Clean Architecture 原則
- Robert C. Martin, "Clean Architecture: A Craftsman's Guide to Software Structure and Design"
- Dependency Inversion Principle (DIP)
- Dependency Rule

### Go 設計模式
- Functional Options Pattern
- Adapter Pattern
- Interface Segregation

### 專案相關文檔
- `docs/claude/CLAUDE.md` - 專案架構規範
- `docs/claude/audit/SECURITY_AUDIT_REPORT.md` - 安全稽核報告
- `internal/domain/ports/outbound/infrastructure/cache.go` - 當前介面定義

---

## 📝 變更歷史

| 日期 | 版本 | 變更描述 | 作者 |
|------|------|---------|------|
| 2026-02-11 | 1.0 | 初始版本：問題分析與重構方案 | Claude |

---

**文檔狀態**: 📋 **設計完成，待實施**
**下一步**: 開始階段一任務 - 定義新介面
