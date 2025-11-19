package infrastructure

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager 通用緩存管理器介面 (基於現有功能)
type CacheManager interface {
	// 連接管理
	Connect(ctx context.Context) error
	Close() error

	// 基本操作 (基於現有Set和MGet方法)
	Set(
		ctx context.Context,
		key string,
		value interface{},
		expiration time.Duration,
	) (string, error)
	SetNX(
		ctx context.Context,
		key string,
		value interface{},
		expiration time.Duration,
	) (bool, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// Pipeline操作 (修復後的版本)
	Pipeline() (redis.Pipeliner, error)

	// 低層級客戶端存取 (保持現有功能)
	GetClient() (*redis.Client, error)

	// 健康檢查 (新增)
	HealthCheck(ctx context.Context) error
}
