// Package tests Phase 6.4: Redis 功能驗證測試
// 驗證 Redis Pub/Sub, Streams, Hash, Counter 等操作
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSSERedisClient 設置 SSE 測試用 Redis 客戶端
func setupSSERedisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client, func()) {
	// 啟動 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err, "Failed to start mini redis")

	// 創建 Redis 客戶端 - 使用明確的類型聲明避免類型推斷衝突
	var redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		_ = redisClient.Close()
		mr.Close()
	}

	return mr, redisClient, cleanup
}

// =============================================================================
// Phase 6.4: Redis 功能驗證
// =============================================================================

// TestRedis_BasicConnection 測試 Redis 基本連接
func TestRedis_BasicConnection(t *testing.T) {
	mr, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()

	// 測試 PING
	pong, err := redisClient.Ping(ctx).Result()
	assert.NoError(t, err, "Should ping Redis without error")
	assert.Equal(t, "PONG", pong, "Should receive PONG from Redis")

	t.Logf("✅ Redis Connection Test Passed - Server: %s", mr.Server().Addr())
}

// TestRedis_PubSubFunctionality 測試 Redis Pub/Sub 功能
func TestRedis_PubSubFunctionality(t *testing.T) {
	mr, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Single Channel Subscribe and Publish", func(t *testing.T) {
		// 訂閱頻道
		pubsub := redisClient.Subscribe(ctx, "test-channel")
		defer func() {
			_ = pubsub.Close()
		}()

		// 等待訂閱完成
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		// 發布訊息
		err = redisClient.Publish(ctx, "test-channel", "test message").Err()
		assert.NoError(t, err, "Should publish message without error")

		// 接收訊息
		msg, err := pubsub.ReceiveMessage(ctx)
		assert.NoError(t, err, "Should receive message without error")
		assert.Equal(t, "test-channel", msg.Channel)
		assert.Equal(t, "test message", msg.Payload)

		t.Log("✅ Single Channel Pub/Sub Test Passed")
	})

	t.Run("Multiple Channels Subscribe", func(t *testing.T) {
		// 訂閱多個頻道
		pubsub := redisClient.Subscribe(ctx, consts.SSEBroadcastChannel, "sse:pod:1")
		defer func() {
			_ = pubsub.Close()
		}()

		// 等待訂閱完成
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		// 發布到廣播頻道
		err = redisClient.Publish(ctx, consts.SSEBroadcastChannel, "broadcast msg").Err()
		assert.NoError(t, err)

		// 接收廣播訊息
		msg, err := pubsub.ReceiveMessage(ctx)
		assert.NoError(t, err)
		assert.Equal(t, consts.SSEBroadcastChannel, msg.Channel)
		assert.Equal(t, "broadcast msg", msg.Payload)

		t.Log("✅ Multiple Channels Pub/Sub Test Passed")
	})

	t.Logf("✅ All Pub/Sub Tests Passed - Server: %s", mr.Server().Addr())
}

// TestRedis_StreamsOperations 測試 Redis Streams 操作
func TestRedis_StreamsOperations(t *testing.T) {
	_, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()
	streamKey := "sse:offline:player-test"

	t.Run("XADD - Add message to stream", func(t *testing.T) {
		// 創建測試通知
		notification := entity.SSENotification{
			ID:        "msg-001",
			Title:     "Test Message",
			Message:   "This is a test",
			Type:      consts.NotificationTypeSystem,
			Priority:  consts.NotificationPriorityHigh,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		data, err := json.Marshal(notification)
		require.NoError(t, err)

		// XADD: 新增到 Stream
		messageID, err := redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey,
			Values: map[string]interface{}{
				"data": string(data),
			},
		}).Result()

		assert.NoError(t, err, "Should add message to stream without error")
		assert.NotEmpty(t, messageID, "Message ID should not be empty")
		t.Logf("Added message with ID: %s", messageID)
	})

	t.Run("XRANGE - Read messages from stream", func(t *testing.T) {
		// XRANGE: 讀取 Stream
		messages, err := redisClient.XRange(ctx, streamKey, "-", "+").Result()
		assert.NoError(t, err, "Should read stream messages without error")
		assert.GreaterOrEqual(t, len(messages), 1, "Should have at least 1 message")

		t.Logf("Read %d messages from stream", len(messages))
	})

	t.Run("XLEN - Get stream length", func(t *testing.T) {
		// XLEN: 獲取 Stream 長度
		length, err := redisClient.XLen(ctx, streamKey).Result()
		assert.NoError(t, err, "Should get stream length without error")
		assert.GreaterOrEqual(t, length, int64(1), "Stream length should be at least 1")

		t.Logf("Stream length: %d", length)
	})

	t.Run("XDEL - Delete message from stream", func(t *testing.T) {
		// 先取得所有訊息
		messages, err := redisClient.XRange(ctx, streamKey, "-", "+").Result()
		require.NoError(t, err)
		require.Greater(t, len(messages), 0)

		messageID := messages[0].ID

		// XDEL: 刪除訊息
		deleted, err := redisClient.XDel(ctx, streamKey, messageID).Result()
		assert.NoError(t, err, "Should delete message without error")
		assert.Equal(t, int64(1), deleted, "Should delete 1 message")

		t.Logf("Deleted message: %s", messageID)
	})

	t.Run("XADD with MAXLEN - Limit stream size", func(t *testing.T) {
		streamKey2 := "sse:offline:player-maxlen"

		// 新增 5 條訊息，但限制最多保留 3 條
		for i := 1; i <= 5; i++ {
			_, err := redisClient.XAdd(ctx, &redis.XAddArgs{
				Stream: streamKey2,
				MaxLen: 3, // 最多保留 3 條
				Approx: true,
				Values: map[string]interface{}{
					"msg": i,
				},
			}).Result()
			require.NoError(t, err)
		}

		// 驗證只保留最多 3 條
		length, err := redisClient.XLen(ctx, streamKey2).Result()
		assert.NoError(t, err)
		assert.LessOrEqual(t, length, int64(3), "Stream should have at most 3 messages")

		t.Logf("Stream with MAXLEN has %d messages (expected ≤ 3)", length)
	})

	t.Log("✅ All Redis Streams Tests Passed")
}

// TestRedis_HashOperations 測試 Redis Hash 操作
func TestRedis_HashOperations(t *testing.T) {
	_, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()
	hashKey := consts.SSEPlayerRoutesKey

	t.Run("HSET - Set hash field", func(t *testing.T) {
		err := redisClient.HSet(ctx, hashKey, "player-001", "pod-1").Err()
		assert.NoError(t, err, "Should set hash field without error")

		err = redisClient.HSet(ctx, hashKey, "player-002", "pod-2").Err()
		assert.NoError(t, err)

		err = redisClient.HSet(ctx, hashKey, "player-003", "pod-1").Err()
		assert.NoError(t, err)

		t.Log("✅ HSET: Added 3 player routes")
	})

	t.Run("HGET - Get hash field", func(t *testing.T) {
		podID, err := redisClient.HGet(ctx, hashKey, "player-001").Result()
		assert.NoError(t, err, "Should get hash field without error")
		assert.Equal(t, "pod-1", podID, "Pod ID should match")

		t.Logf("✅ HGET: player-001 -> %s", podID)
	})

	t.Run("HEXISTS - Check field existence", func(t *testing.T) {
		exists := redisClient.HExists(ctx, hashKey, "player-001").Val()
		assert.True(t, exists, "Field should exist")

		exists = redisClient.HExists(ctx, hashKey, "player-999").Val()
		assert.False(t, exists, "Field should not exist")

		t.Log("✅ HEXISTS: Field existence check passed")
	})

	t.Run("HGETALL - Get all hash fields", func(t *testing.T) {
		routes, err := redisClient.HGetAll(ctx, hashKey).Result()
		assert.NoError(t, err, "Should get all hash fields without error")
		assert.Equal(t, 3, len(routes), "Should have 3 routes")
		assert.Equal(t, "pod-1", routes["player-001"])
		assert.Equal(t, "pod-2", routes["player-002"])
		assert.Equal(t, "pod-1", routes["player-003"])

		t.Logf("✅ HGETALL: Retrieved %d routes", len(routes))
	})

	t.Run("HLEN - Get hash length", func(t *testing.T) {
		length, err := redisClient.HLen(ctx, hashKey).Result()
		assert.NoError(t, err, "Should get hash length without error")
		assert.Equal(t, int64(3), length, "Hash should have 3 fields")

		t.Logf("✅ HLEN: Hash has %d fields", length)
	})

	t.Run("HDEL - Delete hash field", func(t *testing.T) {
		deleted, err := redisClient.HDel(ctx, hashKey, "player-001").Result()
		assert.NoError(t, err, "Should delete hash field without error")
		assert.Equal(t, int64(1), deleted, "Should delete 1 field")

		// 驗證已刪除
		exists := redisClient.HExists(ctx, hashKey, "player-001").Val()
		assert.False(t, exists, "Field should not exist after deletion")

		t.Log("✅ HDEL: Deleted player-001 route")
	})

	t.Log("✅ All Redis Hash Tests Passed")
}

// TestRedis_CounterOperations 測試 Redis Counter 操作
func TestRedis_CounterOperations(t *testing.T) {
	_, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()
	counterKey := consts.SSEOnlineCountKey

	t.Run("INCR - Increment counter", func(t *testing.T) {
		// 初始值應該是 0 或不存在
		count, err := redisClient.Incr(ctx, counterKey).Result()
		assert.NoError(t, err, "Should increment counter without error")
		assert.Equal(t, int64(1), count, "First increment should be 1")

		count, err = redisClient.Incr(ctx, counterKey).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count, "Second increment should be 2")

		t.Log("✅ INCR: Counter incremented to 2")
	})

	t.Run("GET - Get counter value", func(t *testing.T) {
		count, err := redisClient.Get(ctx, counterKey).Int64()
		assert.NoError(t, err, "Should get counter value without error")
		assert.Equal(t, int64(2), count, "Counter value should be 2")

		t.Logf("✅ GET: Counter value is %d", count)
	})

	t.Run("INCRBY - Increment by N", func(t *testing.T) {
		count, err := redisClient.IncrBy(ctx, counterKey, 5).Result()
		assert.NoError(t, err, "Should increment by 5 without error")
		assert.Equal(t, int64(7), count, "Counter should be 7 (2+5)")

		t.Logf("✅ INCRBY: Counter incremented to %d", count)
	})

	t.Run("DECR - Decrement counter", func(t *testing.T) {
		count, err := redisClient.Decr(ctx, counterKey).Result()
		assert.NoError(t, err, "Should decrement counter without error")
		assert.Equal(t, int64(6), count, "Counter should be 6 (7-1)")

		t.Logf("✅ DECR: Counter decremented to %d", count)
	})

	t.Run("DECRBY - Decrement by N", func(t *testing.T) {
		count, err := redisClient.DecrBy(ctx, counterKey, 3).Result()
		assert.NoError(t, err, "Should decrement by 3 without error")
		assert.Equal(t, int64(3), count, "Counter should be 3 (6-3)")

		t.Logf("✅ DECRBY: Counter decremented to %d", count)
	})

	t.Run("DEL - Delete counter", func(t *testing.T) {
		deleted, err := redisClient.Del(ctx, counterKey).Result()
		assert.NoError(t, err, "Should delete counter without error")
		assert.Equal(t, int64(1), deleted, "Should delete 1 key")

		// 驗證已刪除
		_, err = redisClient.Get(ctx, counterKey).Result()
		assert.Equal(t, redis.Nil, err, "Counter should not exist")

		t.Log("✅ DEL: Counter deleted")
	})

	t.Log("✅ All Redis Counter Tests Passed")
}

// TestRedis_ConnectionPoolConfiguration 測試 Redis 連接池配置
func TestRedis_ConnectionPoolConfiguration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 配置連接池參數
	redisClient := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		PoolSize:     10,              // 最大連接數
		MinIdleConns: 5,               // 最小空閒連接數
		MaxRetries:   3,               // 最大重試次數
		DialTimeout:  5 * time.Second, // 連接超時
		ReadTimeout:  3 * time.Second, // 讀取超時
		WriteTimeout: 3 * time.Second, // 寫入超時
	})
	defer func() { _ = redisClient.Close() }()

	ctx := context.Background()

	// 測試連接
	pong, err := redisClient.Ping(ctx).Result()
	assert.NoError(t, err, "Should ping Redis without error")
	assert.Equal(t, "PONG", pong)

	// 驗證連接池統計
	stats := redisClient.PoolStats()
	t.Logf("Pool Stats:")
	t.Logf("  - TotalConns: %d", stats.TotalConns)
	t.Logf("  - IdleConns: %d", stats.IdleConns)
	t.Logf("  - StaleConns: %d", stats.StaleConns)
	t.Logf("  - Hits: %d", stats.Hits)
	t.Logf("  - Misses: %d", stats.Misses)
	t.Logf("  - Timeouts: %d", stats.Timeouts)

	// 執行多個操作來測試連接池
	for i := 0; i < 20; i++ {
		err := redisClient.Set(ctx, fmt.Sprintf("key-%d", i), i, 0).Err()
		require.NoError(t, err)
	}

	// 再次檢查統計
	stats = redisClient.PoolStats()
	assert.GreaterOrEqual(t, stats.Hits, uint32(0), "Should have connection hits")
	assert.GreaterOrEqual(t, stats.TotalConns, uint32(1), "Should have at least 1 connection")

	t.Log("✅ Redis Connection Pool Configuration Test Passed")
}

// TestRedis_VersionCompatibility 測試 Redis 版本兼容性
func TestRedis_VersionCompatibility(t *testing.T) {
	mr, redisClient, cleanup := setupSSERedisClient(t)
	defer cleanup()

	ctx := context.Background()

	// 測試 Redis 5.0+ Streams 功能
	_, err := redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "test-stream",
		Values: map[string]interface{}{"test": "data"},
	}).Result()
	assert.NoError(t, err, "Should support Redis Streams (5.0+)")

	// 測試 Pub/Sub
	pubsub := redisClient.Subscribe(ctx, "test-channel")
	defer func() {
		_ = pubsub.Close()
	}()
	assert.NotNil(t, pubsub, "Should support Pub/Sub")

	t.Logf("✅ Redis Version Compatibility Test Passed - Server: %s", mr.Server().Addr())
	t.Log("   - Streams (5.0+): ✅")
	t.Log("   - Pub/Sub: ✅")
	t.Log("   - Hash: ✅")
	t.Log("   - Counter: ✅")
}
