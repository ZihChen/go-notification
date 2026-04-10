# Known Issue: Redis reconnect() 持有 Write Lock 阻塞所有操作

**發現日期**: 2026-04-10  
**嚴重性**: 中（日常無影響；Redis 長時間異常時才觸發）  
**狀態**: 待修復（下次迭代）  
**相關 commit**: `8fc378949a238770713656e0d05edfc2cdc74221`

---

## 問題描述

`internal/infrastructure/cache/redis/manager.go` 的 `reconnect()` 方法在持有 **write lock**（`m.mu.Lock()`）期間，呼叫 `doConnect()` 執行帶有指數退避重試的連線邏輯，最長可執行 **30 秒**（受 `redisReconnectTimeout = 30s` 限制）。

在此期間，所有 Redis 操作（`Get`/`Set`/`Del`/`MGet` 等）都必須透過 `GetClient()` 取得 `m.mu.RLock()`，但 `sync.RWMutex.RLock()` **不接受 context，無法設 timeout**，會無限期阻塞直到 write lock 釋放。

## 觸發條件

```
健康檢查連續失敗 3 次（間隔 30s × 3 = 約 90 秒）
→ startHealthChecker() 呼叫 reconnect()
→ write lock 持有最長 30 秒
→ 所有 Redis 操作在此期間全部阻塞
```

**只在 Redis 持續異常 90 秒以上時觸發。**  
Redis 正常運作（99.9% 情況）下完全不影響任何功能。

## 問題程式碼

```go
// internal/infrastructure/cache/redis/manager.go

func (m *Manager) reconnect() error {
    // ...
    m.mu.Lock()          // 取得 write lock
    defer m.mu.Unlock()  // 釋放（最長 30s 後）

    ctx, cancel := context.WithTimeout(context.Background(), redisReconnectTimeout) // 30s
    defer cancel()

    return m.doConnect(ctx) // 在持有 write lock 期間執行，含重試退避，最長 30s
}

// 所有 Redis 操作的必經路徑
func (m *Manager) GetClient() (*redis.Client, error) {
    m.mu.RLock()   // ← 若 write lock 被持有，此處無限期阻塞（無 context 支援）
    defer m.mu.RUnlock()
    // ...
}
```

## 建議修法

將「建立新連線」的耗時操作移到鎖外，連線成功後再用最短時間持有 write lock 進行 swap：

```go
func (m *Manager) reconnect() error {
    select {
    case <-m.isClose:
        return errors.New("redis manager is closed, skipping reconnect")
    default:
    }

    // 1. 在鎖外建立新連線（耗時部分，不阻塞其他操作）
    ctx, cancel := context.WithTimeout(context.Background(), redisReconnectTimeout)
    defer cancel()

    newClient, newRedsync, err := buildRedisClient(ctx, m.config)
    if err != nil {
        return err
    }

    // 2. 連線成功後，只用最短時間持有 write lock 進行 swap
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.client != nil {
        _ = m.client.Close()
    }
    m.client, m.redsync, m.isClosed = newClient, newRedsync, false
    return nil
}
```

## 臨時緩解方案（如需立即緩解）

縮短 `redisReconnectTimeout`，減少 write lock 持有時間：

```go
// manager.go
redisReconnectTimeout = 5 * time.Second  // 原本 30s
```

write lock 最壞持有時間從 30s 縮至 5s，幾乎無副作用，但 reconnect 重試次數會受限。

---

**Code Review by**: Claude Code  
**修復優先級**: P2（下次迭代修，非緊急）
