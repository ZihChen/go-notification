package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var sfGroup singleflight.Group

// QueryWithCache 泛型快取查詢，整合 singleflight 防止快取雪崩
// - 快取命中：直接回傳反序列化結果
// - 快取未命中 (redis.Nil)：回退到 DB 查詢
// - 快取錯誤（非 redis.Nil）：記錄日誌後回退到 DB 查詢（靜默降級）
// - singleflight 防護：同一鍵同一時間只有 1 個 DB 查詢
// - 非同步寫入快取，不阻塞主流程
func QueryWithCache[T any](
	ctx context.Context,
	cache infrastructure.CacheManager,
	logger infrastructure.Logger,
	cacheKey string,
	ttl time.Duration,
	entityName string,
	dbQuery func(ctx context.Context) (T, error),
) (T, error) {
	var zero T

	// 嘗試從快取獲取
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		var entity T
		if unmarshalErr := json.Unmarshal([]byte(cachedData), &entity); unmarshalErr == nil {
			return entity, nil
		}
		// 反序列化失敗，繼續查詢 DB
	} else if !errors.Is(err, redis.Nil) {
		// 快取服務錯誤（非 key 不存在），記錄但不中斷，繼續查詢 DB
		logger.WarnWithContext(ctx, "Cache get error",
			logger.String("entity", entityName),
			logger.String("cache_key", cacheKey),
			logger.Error("err", err))
	}

	// 快取未命中或錯誤，使用 singleflight 確保同一鍵只有一個併發查詢
	v, err, _ := sfGroup.Do(cacheKey, func() (interface{}, error) {
		result, err := dbQuery(ctx)
		if err != nil {
			return nil, err
		}

		// 非同步寫入快取（不阻塞主流程）
		// 使用 context.Background() 避免 request context 取消後寫入失敗
		go func() {
			if entityData, err := json.Marshal(result); err == nil {
				if _, err := cache.Set(
					context.Background(),
					cacheKey,
					string(entityData),
					ttl,
				); err != nil {
					// 非同步路徑無 request context，使用 WarnLog（不帶 context）
					logger.WarnLog("Cache set error",
						logger.String("entity", entityName),
						logger.String("cache_key", cacheKey),
						logger.Error("err", err))
				}
			}
		}()

		return result, nil
	})
	if err != nil {
		return zero, err
	}

	return v.(T), nil
}

// ExecuteWithLock 使用分佈式鎖執行函數的共用方法
func ExecuteWithLock(
	ctx context.Context,
	lockManager infrastructure.DistributedLockManager,
	logger infrastructure.Logger,
	mutexKey string,
	entityID uint64,
	entityType string,
	fn func() error,
) error {
	// 智能重試機制：3次嘗試，每次使用遞增參數
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 智能遞增參數策略
		expiry := time.Duration(10+attempt*5) * time.Second
		tries := 5 + attempt*2
		baseDelay := time.Duration(50*attempt) * time.Millisecond

		lockOptions := infrastructure.LockOptions{
			Expiry:     expiry,
			Tries:      tries,
			RetryDelay: baseDelay,
		}

		mutex, err := lockManager.GetLockWithOptions(ctx, mutexKey, lockOptions)
		if err != nil {
			logger.ErrorWithContext(ctx, "Failed to create mutex",
				logger.String("mutex_key", mutexKey),
				logger.String("entity_type", entityType),
				logger.UInt64("entity_id", entityID),
				logger.Int("attempt", attempt),
				logger.Error("err", err),
			)
			if attempt < maxAttempts {
				// 指數退避策略: 500ms → 2s → 4.5s
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to create mutex for %s %d after %d attempts: %w",
				entityType,
				entityID,
				maxAttempts,
				err,
			)
		}

		if err = mutex.Lock(); err != nil {
			logger.WarnWithContext(ctx, "Failed to acquire lock",
				logger.String("mutex_key", mutexKey),
				logger.String("entity_type", entityType),
				logger.UInt64("entity_id", entityID),
				logger.Int("attempt", attempt),
				logger.String("expiry", expiry.String()),
				logger.Int("tries", tries),
				logger.Error("err", err),
			)

			if attempt < maxAttempts {
				// 指數退避策略
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				logger.InfoWithContext(ctx, "Retrying after backoff",
					logger.String("entity_type", entityType),
					logger.UInt64("entity_id", entityID),
					logger.String("backoff_time", backoffTime.String()),
					logger.Int("next_attempt", attempt+1),
				)
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to acquire lock for %s %d after %d attempts: %w",
				entityType,
				entityID,
				maxAttempts,
				err,
			)
		}

		// 成功獲取鎖
		logger.InfoWithContext(ctx, "Successfully acquired lock",
			logger.String("entity_type", entityType),
			logger.UInt64("entity_id", entityID),
			logger.Int("attempt", attempt),
			logger.String("expiry", expiry.String()),
			logger.Int("tries", tries),
		)

		defer func() {
			ok, unlockErr := mutex.Unlock()
			if !ok || unlockErr != nil {
				logger.ErrorWithContext(ctx, "Failed to unlock mutex",
					logger.String("mutex_key", mutexKey),
					logger.String("entity_type", entityType),
					logger.UInt64("entity_id", entityID),
					logger.Error("err", unlockErr),
					logger.Bool("unlock_success", ok),
				)
			}
		}()

		return fn()
	}

	return fmt.Errorf(
		"failed to execute with lock for %s %d after %d attempts",
		entityType,
		entityID,
		maxAttempts,
	)
}
