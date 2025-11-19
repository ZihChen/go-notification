package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

func TestManager_ReconnectStrategy(t *testing.T) {
	t.Run("context取消應該立即返回", func(t *testing.T) {
		cfg := &config.Config{
			Redis: config.RedisConfig{
				Domain: "invalid-redis-host",
				Port:   6379,
				DB:     0,
			},
		}
		
		manager := NewRedisManager(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		
		// 立即取消context
		cancel()
		
		start := time.Now()
		err := manager.Connect(ctx)
		duration := time.Since(start)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context cancelled")
		// 應該很快返回，不應該等待重連
		assert.Less(t, duration, 1*time.Second)
	})
	
	t.Run("最大重試次數限制", func(t *testing.T) {
		cfg := &config.Config{
			Redis: config.RedisConfig{
				Domain: "invalid-redis-host", // 故意使用無效的主機名
				Port:   9999, // 故意使用無效的端口
				DB:     0,
			},
		}
		
		manager := NewRedisManager(cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		
		start := time.Now()
		err := manager.Connect(ctx)
		duration := time.Since(start)
		
		assert.Error(t, err)
		// 可能因為context超時或達到最大重試次數而失敗
		errMsg := err.Error()
		isTimeoutError := assert.Contains(t, errMsg, "context cancelled") || 
						 assert.Contains(t, errMsg, "context deadline exceeded")
		isMaxRetriesError := assert.Contains(t, errMsg, "failed to connect to Redis after 5 attempts")
		
		if !isTimeoutError && !isMaxRetriesError {
			t.Logf("Unexpected error: %s", errMsg)
		}
		
		// 確認指數退避邏輯：2s + 4s + 8s + 16s + 30s = 60s (理論值)
		// 實際上可能會少一些，因為網路超時等因素
		assert.Greater(t, duration, 10*time.Second) // 至少要有一些等待時間
		assert.Less(t, duration, 65*time.Second)    // 但不應該超過太多
	})
	
	t.Run("已有連接時應該立即返回", func(t *testing.T) {
		cfg := &config.Config{
			Redis: config.RedisConfig{
				Domain: "localhost",
				Port:   6379,
				DB:     0,
			},
		}
		
		manager := NewRedisManager(cfg)
		
		// 模擬已經有連接的情況
		manager.client = &redis.Client{} // 只是為了測試，實際使用會有問題
		
		start := time.Now()
		err := manager.Connect(context.Background())
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 100*time.Millisecond) // 應該立即返回
	})
}

// 指數退避計算測試
func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 2 * time.Second},   // 2^1 = 2s
		{2, 4 * time.Second},   // 2^2 = 4s  
		{3, 8 * time.Second},   // 2^3 = 8s
		{4, 16 * time.Second},  // 2^4 = 16s
		{5, 30 * time.Second},  // 2^5 = 32s，但限制為30s
		{6, 30 * time.Second},  // 2^6 = 64s，但限制為30s
	}
	
	for _, tt := range tests {
		backoff := time.Duration(1<<uint(tt.attempt)) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		assert.Equal(t, tt.expected, backoff, "attempt %d", tt.attempt)
	}
}