package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
)

func TestManager_Pipeline_ErrorHandling(t *testing.T) {
	t.Run("未初始化client", func(t *testing.T) {
		manager := &Manager{}
		pipeline, err := manager.Pipeline()
		assert.Error(t, err)
		assert.Nil(t, pipeline)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})

	t.Run("連接已關閉", func(t *testing.T) {
		manager := &Manager{
			isClosed: true,
		}
		pipeline, err := manager.Pipeline()
		assert.Error(t, err)
		assert.Nil(t, pipeline)
		assert.Contains(t, err.Error(), "redis connection is closed")
	})

	// 注意：正常情況的測試需要真實的Redis連接，
	// 在實際環境中可以通過setupTestManager來設置
}

func TestManager_HealthCheck(t *testing.T) {
	t.Run("未初始化client", func(t *testing.T) {
		manager := &Manager{}
		err := manager.HealthCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})

	t.Run("連接已關閉", func(t *testing.T) {
		manager := &Manager{
			isClosed: true,
		}
		err := manager.HealthCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection is closed")
	})

	t.Run("context cancelled", func(t *testing.T) {
		manager := &Manager{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消context

		err := manager.HealthCheck(ctx)
		assert.Error(t, err)
	})

	// 注意：健康連接的測試需要真實的Redis連接
	// 可以在整合測試中進行驗證
}

func TestManager_ImplementsCacheManager(t *testing.T) {
	// 確保Manager實現了CacheManager介面
	var _ infrastructure.CacheManager = (*Manager)(nil)
	
	t.Run("介面方法可用性檢查", func(t *testing.T) {
		manager := &Manager{}
		ctx := context.Background()
		
		// 測試所有介面方法都存在（即使會返回錯誤）
		// 注意：跳過Connect測試因為它需要有效的config
		
		// Close
		err := manager.Close()
		assert.NoError(t, err) // Close在未初始化時不會出錯
		
		// Set (會出錯因為client未初始化)
		_, err = manager.Set(ctx, "test_key", "test_value", time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		
		// SetNX
		_, err = manager.SetNX(ctx, "test_key", "value", time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		
		// MGet
		_, err = manager.MGet(ctx, "test_key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		
		// Pipeline
		_, err = manager.Pipeline()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get redis client for pipeline")
		
		// HealthCheck
		err = manager.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		
		// GetClient
		_, err = manager.GetClient()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})
}

// 以下是測試用的輔助函數，在有真實Redis環境時使用
/*
func setupTestManager(t *testing.T) *Manager {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain: "localhost",
			Port:   6379,
			DB:     0,
		},
	}

	manager := NewRedisManager(cfg)
	err := manager.Connect(context.Background())
	if err != nil {
		t.Skipf("Skipping test requiring Redis: %v", err)
	}

	t.Cleanup(func() {
		_ = manager.Close()
	})

	return manager
}

func TestManager_Pipeline_Integration(t *testing.T) {
	manager := setupTestManager(t)

	t.Run("正常Pipeline操作", func(t *testing.T) {
		pipeline, err := manager.Pipeline()
		assert.NoError(t, err)
		assert.NotNil(t, pipeline)

		// 測試Pipeline基本功能
		pipeline.Set(context.Background(), "test_key_1", "value1", time.Minute)
		pipeline.Set(context.Background(), "test_key_2", "value2", time.Minute)

		results, err := pipeline.Exec(context.Background())
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestManager_HealthCheck_Integration(t *testing.T) {
	manager := setupTestManager(t)

	t.Run("健康連接", func(t *testing.T) {
		err := manager.HealthCheck(context.Background())
		assert.NoError(t, err)
	})

	t.Run("健康檢查超時", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// 給context一點時間過期
		time.Sleep(1 * time.Millisecond)

		err := manager.HealthCheck(ctx)
		assert.Error(t, err)
	})
}
*/
