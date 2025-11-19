package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	client   *redis.Client
	redsync  *redsync.Redsync
	config   *config.Config
	mu       sync.RWMutex
	isClosed bool
}

// 確保Manager實現CacheManager介面
var _ infrastructure.CacheManager = (*Manager)(nil)

func NewRedisManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return nil // Redis connection 還存在
	}

	retryCount := 0
	maxRetries := 5

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while trying to connect to Redis: %w", ctx.Err())
		default:
			client := redis.NewClient(&redis.Options{
				Addr:            fmt.Sprintf("%s:%d", m.config.Redis.Domain, m.config.Redis.Port),
				Password:        m.config.Redis.Password,
				DB:              m.config.Redis.DB,
				PoolSize:        m.config.Redis.PoolSize,     // 連線池大小
				MinIdleConns:    m.config.Redis.MinIdleConns, // 最小空閒連線數
				MaxRetries:      m.config.Redis.MaxRetries,   // 最大重試次數
				DialTimeout:     m.config.Redis.DialTimeout,  // 連線超時
				ReadTimeout:     m.config.Redis.ReadTimeout,  // 讀取超時
				WriteTimeout:    m.config.Redis.WriteTimeout, // 寫入超時
				PoolTimeout:     m.config.Redis.PoolTimeout,  // 連線池等待超時
				ConnMaxIdleTime: m.config.Redis.IdleTimeout,  // 連線最大空閒時間
				ConnMaxLifetime: m.config.Redis.MaxConnAge,   // 連線最大生命週期
			})

			// 初始化 redsync 互斥鎖
			pool := goredis.NewPool(client)
			rs := redsync.New(pool)

			// 測試連接
			if err := client.Ping(ctx).Err(); err != nil {
				retryCount++
				if retryCount >= maxRetries {
					_ = client.Close()
					return fmt.Errorf(
						"failed to connect to Redis after %d attempts: %w",
						maxRetries,
						err,
					)
				}

				// 指數退避，最大30秒
				backoff := time.Duration(1<<uint(retryCount)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				log.Printf("Failed to connect to Redis (attempt %d/%d): %v, retrying in %v...",
					retryCount, maxRetries, err, backoff)

				_ = client.Close()
				time.Sleep(backoff)
				continue
			}

			// 連接成功
			m.client, m.redsync, m.isClosed = client, rs, false
			log.Printf(
				"Successfully connected to Redis at %s:%d",
				m.config.Redis.Domain,
				m.config.Redis.Port,
			)
			return nil
		}
	}
}

func (m *Manager) GetClient() (*redis.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.isClosed {
		return nil, errors.New("redis connection is closed")
	}

	if m.client == nil {
		return nil, errors.New("redis client not initialized")
	}

	return m.client, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed || m.client == nil {
		return nil
	}

	err := m.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	m.client, m.isClosed = nil, true
	return nil
}

func (m *Manager) GetRedsync() (*redsync.Redsync, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isClosed {
		return nil, errors.New("redis connection is closed")
	}
	if m.redsync == nil {
		return nil, errors.New("redsync not initialized")
	}
	return m.redsync, nil
}

func (m *Manager) SetNX(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (bool, error) {
	client, err := m.GetClient()
	if err != nil {
		return false, err
	}
	success, err := client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func (m *Manager) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (string, error) {
	client, err := m.GetClient()
	if err != nil {
		return "", err
	}
	status, err := client.Set(ctx, key, value, expiration).Result()
	if err != nil {
		return status, err
	}
	return status, nil
}

// Pipeline 返回 Redis Pipeline 用於批次操作
func (m *Manager) Pipeline() (redis.Pipeliner, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis client for pipeline: %w", err)
	}
	return client.Pipeline(), nil
}

// MGet 批次獲取多個 key 的值
func (m *Manager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	return client.MGet(ctx, keys...).Result()
}

// HealthCheck 檢查Redis連接健康狀態
func (m *Manager) HealthCheck(ctx context.Context) error {
	client, err := m.GetClient()
	if err != nil {
		return err
	}
	return client.Ping(ctx).Err()
}

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
func (r *RedisDistributedLockManager) GetLock(
	ctx context.Context,
	key string,
) (infrastructure.DistributedMutex, error) {
	defaultOptions := infrastructure.LockOptions{
		Expiry:     30 * time.Second,
		Tries:      5,
		RetryDelay: 100 * time.Millisecond,
	}
	return r.GetLockWithOptions(ctx, key, defaultOptions)
}

// GetLockWithOptions 獲取帶選項的分佈式鎖實例
func (r *RedisDistributedLockManager) GetLockWithOptions(
	ctx context.Context,
	key string,
	options infrastructure.LockOptions,
) (infrastructure.DistributedMutex, error) {
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

	redsyncInstance, err := r.manager.GetRedsync()
	if err != nil {
		return nil, err
	}

	mutex := redsyncInstance.NewMutex(key, redisyncOptions...)
	return &DistributedMutex{mutex: mutex}, nil
}

// IsAvailable 檢查分佈式鎖服務是否可用
func (r *RedisDistributedLockManager) IsAvailable() bool {
	if r.manager == nil {
		return false
	}

	// 使用實際連通性檢查
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return r.manager.HealthCheck(ctx) == nil
}

// GetMutex 獲取簡單的分佈式鎖
func (r *RedisDistributedLockManager) GetMutex(
	key string,
	expireTime time.Duration,
) (infrastructure.DistributedMutex, error) {
	redsyncInstance, err := r.manager.GetRedsync()
	if err != nil {
		return nil, err
	}
	mutex := redsyncInstance.NewMutex(key, redsync.WithExpiry(expireTime))
	return &DistributedMutex{mutex: mutex}, nil
}
