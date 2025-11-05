package redis

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
)

// DistributedMutex Redis分佈式鎖實現 (Infrastructure層)
type DistributedMutex struct {
	mutex *redsync.Mutex
}

// Lock 取得鎖
func (r *DistributedMutex) Lock() error {
	return r.mutex.Lock()
}

// Unlock 釋放鎖
func (r *DistributedMutex) Unlock() (bool, error) {
	return r.mutex.Unlock()
}

// TryLock 嘗試取得鎖，不阻塞
func (r *DistributedMutex) TryLock() error {
	return r.mutex.TryLock()
}

// Extend 延長鎖的過期時間
func (r *DistributedMutex) Extend() (bool, error) {
	return r.mutex.Extend()
}

// RedisDistributedLockManager Redis分佈式鎖管理器 (Infrastructure層)
type RedisDistributedLockManager struct {
	manager *Manager
}

// NewRedisDistributedLockManager 創建Redis分佈式鎖管理器
func NewRedisDistributedLockManager(manager *Manager) infrastructure.DistributedLockManager {
	return &RedisDistributedLockManager{
		manager: manager,
	}
}

// GetLock 獲取分佈式鎖實例
func (r *RedisDistributedLockManager) GetLock(ctx context.Context, key string) (infrastructure.DistributedMutex, error) {
	defaultOptions := infrastructure.LockOptions{
		Expiry:     30 * time.Second,
		Tries:      5,
		RetryDelay: 100 * time.Millisecond,
	}
	return r.GetLockWithOptions(ctx, key, defaultOptions)
}

// GetLockWithOptions 獲取帶選項的分佈式鎖實例
func (r *RedisDistributedLockManager) GetLockWithOptions(ctx context.Context, key string, options infrastructure.LockOptions) (infrastructure.DistributedMutex, error) {
	// 轉換為 redsync 選項
	redisyncOptions := []redsync.Option{
		redsync.WithExpiry(options.Expiry),
		redsync.WithTries(options.Tries),
		redsync.WithRetryDelay(options.RetryDelay),
	}

	if options.DriftFactor > 0 {
		redisyncOptions = append(redisyncOptions, redsync.WithDriftFactor(options.DriftFactor))
	}

	if options.TimeoutFactor > 0 {
		redisyncOptions = append(redisyncOptions, redsync.WithTimeoutFactor(options.TimeoutFactor))
	}

	mutex, err := r.manager.GetMutexWithOption(key, redisyncOptions...)
	if err != nil {
		return nil, err
	}

	return &DistributedMutex{mutex: mutex}, nil
}

// IsAvailable 檢查分佈式鎖服務是否可用
func (r *RedisDistributedLockManager) IsAvailable() bool {
	if r.manager == nil {
		return false
	}

	// 嘗試獲取 Redis 客戶端來檢查連接狀態
	_, err := r.manager.GetClient()
	return err == nil
}
