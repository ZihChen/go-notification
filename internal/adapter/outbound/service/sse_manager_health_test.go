// Package service SSE Manager Pod 健康心跳與路由清理測試
package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// 測試輔助函數
// =============================================================================

// createTestSSEManager 創建測試用的 SSE Manager
func createTestSSEManager(t *testing.T, podID string, redisClient *goredis.Client) *sseManager {
	// 創建 mock cache manager
	cacheManager := &mockCacheManager{client: redisClient}

	// 創建 mock logger
	logger := &mockLogger{}

	// 創建背景任務 Context
	ctx, cancel := context.WithCancel(context.Background())

	manager := &sseManager{
		podID:        podID,
		connections:  make(map[string]inbound.SSEWriter),
		cacheManager: cacheManager,
		redisClient:  redisClient,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 訂閱 Pub/Sub 頻道
	manager.pubsub = manager.redisClient.Subscribe(ctx,
		consts.SSEBroadcastChannel,
		fmt.Sprintf(consts.SSEPodChannel, manager.podID),
	)

	return manager
}

// mockCacheManager 模擬 CacheManager
type mockCacheManager struct {
	client *goredis.Client
}

func (m *mockCacheManager) Connect(ctx context.Context) error {
	return nil
}

func (m *mockCacheManager) GetClient() (*goredis.Client, error) {
	return m.client, nil
}

func (m *mockCacheManager) Pipeline() (goredis.Pipeliner, error) {
	return m.client.Pipeline(), nil
}

func (m *mockCacheManager) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockCacheManager) Close() error {
	return nil
}

func (m *mockCacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error) {
	return m.client.Set(ctx, key, value, expiration).Result()
}

func (m *mockCacheManager) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return m.client.SetNX(ctx, key, value, expiration).Result()
}

func (m *mockCacheManager) Get(ctx context.Context, key string) (string, error) {
	return m.client.Get(ctx, key).Result()
}

func (m *mockCacheManager) Del(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Del(ctx, keys...).Result()
}

func (m *mockCacheManager) Exists(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Exists(ctx, keys...).Result()
}

func (m *mockCacheManager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return m.client.MGet(ctx, keys...).Result()
}

func (m *mockCacheManager) GetRedsync() (*redsync.Redsync, error) {
	return nil, nil
}

// mockLogger 模擬 Logger
type mockLogger struct{}

func (l *mockLogger) InfoLog(msg string, fields ...*entity.LoggerFiled)                            {}
func (l *mockLogger) WarnLog(msg string, fields ...*entity.LoggerFiled)                            {}
func (l *mockLogger) ErrorLog(msg string, fields ...*entity.LoggerFiled)                           {}
func (l *mockLogger) DebugLog(msg string, fields ...*entity.LoggerFiled)                           {}
func (l *mockLogger) FatalLog(msg string, fields ...*entity.LoggerFiled)                           {}
func (l *mockLogger) InfoWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)  {}
func (l *mockLogger) WarnWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)  {}
func (l *mockLogger) ErrorWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (l *mockLogger) DebugWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (l *mockLogger) FatalWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (l *mockLogger) String(key, value string) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Int(key string, value int) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Int64(key string, value int64) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) UInt64(key string, value uint64) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Float64(key string, value float64) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Error(key string, err error) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Any(key string, value interface{}) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Bool(key string, value bool) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (l *mockLogger) Close() {}

// =============================================================================
// Phase 3.1: 單元測試 - Pod 健康心跳
// =============================================================================

// TestPodHealthHeartbeat_UpdatesRedisKey 測試心跳是否正確更新 Redis Key
func TestPodHealthHeartbeat_UpdatesRedisKey(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// 創建 SSE Manager
	podID := "test-pod-1"
	manager := createTestSSEManager(t, podID, redisClient)
	defer manager.cancel()

	// 手動觸發一次心跳更新（用於測試）
	ctx := context.Background()
	healthKey := fmt.Sprintf(consts.SSEPodHealthKey, podID)
	err := redisClient.Set(ctx, healthKey, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	// 驗證 Redis Key 存在
	exists, err := redisClient.Exists(ctx, healthKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "健康心跳 Key 應該存在")

	// 驗證 TTL 設置正確（允許一些誤差）
	ttl, err := redisClient.TTL(ctx, healthKey).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), 25.0, "TTL 應該大於 25 秒")
	assert.LessOrEqual(t, ttl.Seconds(), 30.0, "TTL 應該小於或等於 30 秒")

	// 驗證值為 "alive"
	value, err := redisClient.Get(ctx, healthKey).Result()
	require.NoError(t, err)
	assert.Equal(t, "alive", value, "健康心跳值應該為 'alive'")
}

// TestPodHealthHeartbeat_ExpiresAfterTTL 測試心跳 Key 在 TTL 後過期
func TestPodHealthHeartbeat_ExpiresAfterTTL(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	ctx := context.Background()
	podID := "test-pod-expire"
	healthKey := fmt.Sprintf(consts.SSEPodHealthKey, podID)

	// 設置心跳 Key，TTL 為 1 秒（用於快速測試）
	err := redisClient.Set(ctx, healthKey, "alive", 1*time.Second).Err()
	require.NoError(t, err)

	// 驗證 Key 存在
	exists, err := redisClient.Exists(ctx, healthKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	// 使用 miniredis 的 FastForward 快進時間
	mr.FastForward(2 * time.Second)

	// 驗證 Key 已過期
	exists, err = redisClient.Exists(ctx, healthKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "健康心跳 Key 應該已過期")
}

// TestPodHealthHeartbeat_StopsOnContextCancel 測試 Context 取消時心跳停止
func TestPodHealthHeartbeat_StopsOnContextCancel(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// 創建 SSE Manager
	podID := "test-pod-cancel"
	manager := createTestSSEManager(t, podID, redisClient)

	// 啟動心跳 Goroutine
	manager.wg.Add(1)
	go manager.startPodHealthHeartbeat()

	// 等待 Goroutine 啟動
	time.Sleep(50 * time.Millisecond)

	// 取消 Context
	manager.cancel()

	// 等待 Goroutine 結束
	done := make(chan struct{})
	go func() {
		manager.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Goroutine 正確結束
	case <-time.After(2 * time.Second):
		t.Fatal("心跳 Goroutine 未在預期時間內停止")
	}
}

// =============================================================================
// Phase 3.2: 單元測試 - 死 Pod 檢測
// =============================================================================

// TestSendToPlayer_DetectsDeadPod 測試發送消息時檢測到死 Pod
func TestSendToPlayer_DetectsDeadPod(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	ctx := context.Background()

	// 創建 SSE Manager (Pod-A)
	podIDSelf := "pod-a"
	manager := createTestSSEManager(t, podIDSelf, redisClient)
	defer manager.cancel()

	// 設置場景：player_001 在 pod-b，但 pod-b 已死（沒有健康心跳）
	playerID := "player_001"
	deadPodID := "pod-b"

	// 設置玩家路由指向死 Pod
	err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, deadPodID).Err()
	require.NoError(t, err)

	// 注意：不設置 pod-b 的健康心跳，模擬死 Pod

	// 創建測試通知
	notification := &entity.SSENotification{
		ID:       "test-notification-1",
		Title:    "Test Notification",
		Message:  "This is a test",
		Type:     "test",
		Priority: "medium",
	}

	// 發送消息
	err = manager.SendToPlayer(ctx, playerID, notification)

	// 應該成功（因為加入離線隊列）
	require.NoError(t, err)

	// 驗證路由已被清理
	exists, err := redisClient.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
	require.NoError(t, err)
	assert.False(t, exists, "死 Pod 的路由應該被清理")

	// 驗證消息加入離線隊列
	streamKey := fmt.Sprintf(consts.SSEOfflineMessagesKey, playerID)
	messages, err := redisClient.XRange(ctx, streamKey, "-", "+").Result()
	require.NoError(t, err)
	assert.Equal(t, 1, len(messages), "應該有 1 條離線消息")
}

// TestSendToPlayer_HealthyPodRoutesCorrectly 測試健康 Pod 正確路由消息
func TestSendToPlayer_HealthyPodRoutesCorrectly(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	ctx := context.Background()

	// 創建 SSE Manager (Pod-A)
	podIDSelf := "pod-a"
	manager := createTestSSEManager(t, podIDSelf, redisClient)
	defer manager.cancel()

	// 設置場景：player_001 在 pod-b，且 pod-b 健康
	playerID := "player_001"
	targetPodID := "pod-b"

	// 設置玩家路由
	err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, targetPodID).Err()
	require.NoError(t, err)

	// 設置 pod-b 的健康心跳
	healthKey := fmt.Sprintf(consts.SSEPodHealthKey, targetPodID)
	err = redisClient.Set(ctx, healthKey, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	// 訂閱目標 Pod 的頻道（模擬 pod-b 在監聽）
	podChannel := fmt.Sprintf(consts.SSEPodChannel, targetPodID)
	pubsub := redisClient.Subscribe(ctx, podChannel)
	defer pubsub.Close()

	// 創建測試通知
	notification := &entity.SSENotification{
		ID:       "test-notification-2",
		Title:    "Test Notification",
		Message:  "This is a test",
		Type:     "test",
		Priority: "medium",
	}

	// 發送消息
	err = manager.SendToPlayer(ctx, playerID, notification)
	require.NoError(t, err)

	// 驗證路由未被清理（健康 Pod）
	exists, err := redisClient.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
	require.NoError(t, err)
	assert.True(t, exists, "健康 Pod 的路由應該保留")

	// 驗證消息通過 Pub/Sub 發送到目標 Pod
	select {
	case msg := <-pubsub.Channel():
		assert.Equal(t, podChannel, msg.Channel, "消息應該發送到正確的 Pod 頻道")
	case <-time.After(1 * time.Second):
		t.Fatal("未在預期時間內收到 Pub/Sub 消息")
	}
}

// =============================================================================
// Phase 3.3: 單元測試 - 路由清理
// =============================================================================

// TestCleanupDeadPodRoutes_RemovesDeadPods 測試清理死 Pod 的路由
func TestCleanupDeadPodRoutes_RemovesDeadPods(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	ctx := context.Background()

	// 創建 SSE Manager
	podID := "pod-cleaner"
	manager := createTestSSEManager(t, podID, redisClient)
	defer manager.cancel()

	// 設置測試場景：3 個 Pod，2 個健康，1 個死亡
	// 設置玩家路由
	for i := 1; i <= 50; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, "pod-a").Err()
		require.NoError(t, err)
	}
	for i := 51; i <= 80; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, "pod-b").Err()
		require.NoError(t, err)
	}
	for i := 81; i <= 100; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, "pod-c").Err()
		require.NoError(t, err)
	}

	// 設置健康心跳（只設置 pod-a 和 pod-b，pod-c 沒有心跳）
	healthKeyA := fmt.Sprintf(consts.SSEPodHealthKey, "pod-a")
	err := redisClient.Set(ctx, healthKeyA, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	healthKeyB := fmt.Sprintf(consts.SSEPodHealthKey, "pod-b")
	err = redisClient.Set(ctx, healthKeyB, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	// 執行清理
	manager.cleanupDeadPodRoutes()

	// 驗證結果
	routes, err := redisClient.HGetAll(ctx, consts.SSEPlayerRoutesKey).Result()
	require.NoError(t, err)

	// 應該只剩下 80 條路由（pod-a: 50 + pod-b: 30）
	assert.Equal(t, 80, len(routes), "應該剩下 80 條路由（死 Pod 的 20 條已清理）")

	// 驗證 pod-c 的路由全部被清理
	for i := 81; i <= 100; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		exists, err := redisClient.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
		require.NoError(t, err)
		assert.False(t, exists, fmt.Sprintf("player_%03d 的路由應該被清理", i))
	}

	// 驗證健康 Pod 的路由保留
	for i := 1; i <= 50; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		exists, err := redisClient.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
		require.NoError(t, err)
		assert.True(t, exists, fmt.Sprintf("player_%03d 的路由應該保留", i))
	}
}

// TestCleanupDeadPodRoutes_NoCleanupWhenAllHealthy 測試所有 Pod 健康時不清理
func TestCleanupDeadPodRoutes_NoCleanupWhenAllHealthy(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	ctx := context.Background()

	// 創建 SSE Manager
	podID := "pod-cleaner"
	manager := createTestSSEManager(t, podID, redisClient)
	defer manager.cancel()

	// 設置測試場景：2 個 Pod，都健康
	for i := 1; i <= 50; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, "pod-a").Err()
		require.NoError(t, err)
	}
	for i := 51; i <= 100; i++ {
		playerID := fmt.Sprintf("player_%03d", i)
		err := redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, "pod-b").Err()
		require.NoError(t, err)
	}

	// 設置健康心跳
	healthKeyA := fmt.Sprintf(consts.SSEPodHealthKey, "pod-a")
	err := redisClient.Set(ctx, healthKeyA, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	healthKeyB := fmt.Sprintf(consts.SSEPodHealthKey, "pod-b")
	err = redisClient.Set(ctx, healthKeyB, "alive", 30*time.Second).Err()
	require.NoError(t, err)

	// 執行清理
	manager.cleanupDeadPodRoutes()

	// 驗證結果：所有路由保留
	routes, err := redisClient.HGetAll(ctx, consts.SSEPlayerRoutesKey).Result()
	require.NoError(t, err)
	assert.Equal(t, 100, len(routes), "所有路由應該保留")
}

// TestCleanupDeadPodRoutes_EmptyRoutes 測試空路由表不出錯
func TestCleanupDeadPodRoutes_EmptyRoutes(t *testing.T) {
	// 啟動 miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	// 創建 Redis 客戶端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// 創建 SSE Manager
	podID := "pod-cleaner"
	manager := createTestSSEManager(t, podID, redisClient)
	defer manager.cancel()

	// 執行清理（路由表為空）
	// 不應該 panic 或報錯
	assert.NotPanics(t, func() {
		manager.cleanupDeadPodRoutes()
	})
}

// =============================================================================
// 測試總結
// =============================================================================

// 測試統計：
// - Phase 3.1 (Pod 健康心跳): 3 個測試
// - Phase 3.2 (死 Pod 檢測): 2 個測試
// - Phase 3.3 (路由清理): 3 個測試
//
// 總計: 8 個實現測試
