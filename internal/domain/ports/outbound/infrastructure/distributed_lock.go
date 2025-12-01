package infrastructure

import (
	"context"
	"time"
)

// DistributedMutex 分佈式鎖介面 (Domain層抽象)
type DistributedMutex interface {
	// Lock 取得鎖
	Lock() error

	// Unlock 釋放鎖
	Unlock() (bool, error)

	// TryLock 嘗試取得鎖，不阻塞
	TryLock() error

	// Extend 延長鎖的過期時間
	Extend() (bool, error)
}

// DistributedLockManager 分佈式鎖管理器介面 (Domain層抽象)
type DistributedLockManager interface {
	// GetLock 獲取分佈式鎖實例
	GetLock(ctx context.Context, key string) (DistributedMutex, error)

	// GetLockWithOptions 獲取帶選項的分佈式鎖實例
	GetLockWithOptions(
		ctx context.Context,
		key string,
		options LockOptions,
	) (DistributedMutex, error)

	// GetMutex 獲取簡單的分佈式鎖
	GetMutex(key string, expireTime time.Duration) (DistributedMutex, error)

	// IsAvailable 檢查分佈式鎖服務是否可用
	IsAvailable() bool
}

// LockOptions 鎖配置選項
type LockOptions struct {
	// Expiry 鎖的過期時間
	Expiry time.Duration

	// Tries 最大嘗試次數
	Tries int

	// RetryDelay 重試間隔
	RetryDelay time.Duration

	// DriftFactor 時鐘漂移係數 (0-1之間)
	DriftFactor float64

	// TimeoutFactor 超時係數
	TimeoutFactor float64
}
