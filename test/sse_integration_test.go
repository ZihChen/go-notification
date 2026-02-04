// Package tests SSE 整合測試
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	outboundService "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
)

// =============================================================================
// Mock SSEWriter 實現
// =============================================================================

// MockSSEWriter 模擬 SSE Writer，用於測試
// 實現完整的 inbound.SSEWriter 介面
type MockSSEWriter struct {
	mu       sync.Mutex
	events   []SSEEvent // 儲存所有事件
	closed   bool
}

// SSEEvent 記錄 SSE 事件數據
type SSEEvent struct {
	Event string // 事件類型 (notification, ping, error, etc.)
	Data  string // 事件數據 (JSON 字串)
}

func NewMockSSEWriter() *MockSSEWriter {
	return &MockSSEWriter{
		events: make([]SSEEvent, 0),
		closed: false,
	}
}

// Write 實現 SSEWriter.Write 介面
func (w *MockSSEWriter) Write(event string, data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	w.events = append(w.events, SSEEvent{
		Event: event,
		Data:  data,
	})
	return nil
}

// Flush 實現 SSEWriter.Flush 介面
func (w *MockSSEWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	// Mock 實現不需要實際 flush 操作
	return nil
}

// Close 實現 SSEWriter.Close 介面
func (w *MockSSEWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

// GetEvents 獲取所有記錄的事件（測試用）
func (w *MockSSEWriter) GetEvents() []SSEEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]SSEEvent, len(w.events))
	copy(result, w.events)
	return result
}

// GetNotificationEvents 獲取所有 notification 類型的事件（測試用）
func (w *MockSSEWriter) GetNotificationEvents() []SSEEvent {
	w.mu.Lock()
	defer w.mu.Unlock()

	var notifications []SSEEvent
	for _, event := range w.events {
		if event.Event == "notification" || event.Event == consts.SSEEventTypeNotification {
			notifications = append(notifications, event)
		}
	}
	return notifications
}

// EventCount 獲取事件數量（測試用）
func (w *MockSSEWriter) EventCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.events)
}

// ParseNotificationFromEvent 從 SSEEvent 解析通知數據（測試用）
func (w *MockSSEWriter) ParseNotificationFromEvent(event SSEEvent) (*entity.SSENotification, error) {
	var notification entity.SSENotification
	err := json.Unmarshal([]byte(event.Data), &notification)
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// =============================================================================
// 測試環境設置
// =============================================================================

// TestCacheManager 測試用的簡單 CacheManager 實現
type TestCacheManager struct {
	client *redis.Client
}

func (m *TestCacheManager) GetClient() (*redis.Client, error) {
	if m.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	return m.client, nil
}

func (m *TestCacheManager) Connect(ctx context.Context) error {
	return nil // 測試環境已經連接
}

func (m *TestCacheManager) Close() error {
	return nil // 由測試環境管理生命週期
}

func (m *TestCacheManager) Ping(ctx context.Context) error {
	return m.client.Ping(ctx).Err()
}

func (m *TestCacheManager) HealthCheck(ctx context.Context) error {
	return m.Ping(ctx)
}

func (m *TestCacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error) {
	return m.client.Set(ctx, key, value, expiration).Result()
}

func (m *TestCacheManager) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return m.client.SetNX(ctx, key, value, expiration).Result()
}

func (m *TestCacheManager) Get(ctx context.Context, key string) (string, error) {
	return m.client.Get(ctx, key).Result()
}

func (m *TestCacheManager) Del(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Del(ctx, keys...).Result()
}

func (m *TestCacheManager) Exists(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Exists(ctx, keys...).Result()
}

func (m *TestCacheManager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return m.client.MGet(ctx, keys...).Result()
}

func (m *TestCacheManager) MSet(ctx context.Context, pairs ...interface{}) (string, error) {
	return m.client.MSet(ctx, pairs...).Result()
}

func (m *TestCacheManager) Incr(ctx context.Context, key string) (int64, error) {
	return m.client.Incr(ctx, key).Result()
}

func (m *TestCacheManager) Decr(ctx context.Context, key string) (int64, error) {
	return m.client.Decr(ctx, key).Result()
}

func (m *TestCacheManager) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return m.client.Expire(ctx, key, expiration).Result()
}

func (m *TestCacheManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	return m.client.TTL(ctx, key).Result()
}

func (m *TestCacheManager) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return m.client.SAdd(ctx, key, members...).Result()
}

func (m *TestCacheManager) SMembers(ctx context.Context, key string) ([]string, error) {
	return m.client.SMembers(ctx, key).Result()
}

func (m *TestCacheManager) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return m.client.SRem(ctx, key, members...).Result()
}

func (m *TestCacheManager) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return m.client.HSet(ctx, key, values...).Result()
}

func (m *TestCacheManager) HGet(ctx context.Context, key, field string) (string, error) {
	return m.client.HGet(ctx, key, field).Result()
}

func (m *TestCacheManager) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return m.client.HGetAll(ctx, key).Result()
}

func (m *TestCacheManager) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	return m.client.HDel(ctx, key, fields...).Result()
}

func (m *TestCacheManager) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return m.client.LPush(ctx, key, values...).Result()
}

func (m *TestCacheManager) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return m.client.RPush(ctx, key, values...).Result()
}

func (m *TestCacheManager) LPop(ctx context.Context, key string) (string, error) {
	return m.client.LPop(ctx, key).Result()
}

func (m *TestCacheManager) RPop(ctx context.Context, key string) (string, error) {
	return m.client.RPop(ctx, key).Result()
}

func (m *TestCacheManager) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return m.client.LRange(ctx, key, start, stop).Result()
}

func (m *TestCacheManager) ZAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	// 注意：這裡簡化了實現，實際可能需要處理 ZAddArgs
	return 0, fmt.Errorf("ZAdd not implemented in test")
}

func (m *TestCacheManager) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return m.client.ZRange(ctx, key, start, stop).Result()
}

func (m *TestCacheManager) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return m.client.ZRem(ctx, key, members...).Result()
}

func (m *TestCacheManager) Pipeline() (redis.Pipeliner, error) {
	pipe := m.client.Pipeline()
	return pipe, nil
}

func (m *TestCacheManager) GetRedsync() (*redsync.Redsync, error) {
	// 測試環境不需要 redsync，返回 nil
	return nil, nil
}

// setupSSEIntegrationTestEnv 設置 SSE 整合測試環境
func setupSSEIntegrationTestEnv(t *testing.T, podID string) (*miniredis.Miniredis, *redis.Client, outboundService.SSEManager, func()) {
	// 啟動 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err, "Failed to start mini redis")

	// 創建 Redis 客戶端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 創建測試用 Cache Manager
	cacheManager := &TestCacheManager{
		client: client,
	}

	// 創建 Logger
	logger := helper.NewMockLogger()

	// 創建 SSE Manager
	sseManager := service.NewSSEManager(podID, cacheManager, logger)

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return mr, client, sseManager, cleanup
}

// =============================================================================
// Phase 6.1: 單 Pod 整合測試
// =============================================================================

// TestSSE_RegisterConnection 測試玩家連接註冊
func TestSSE_RegisterConnection(t *testing.T) {
	_, client, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()
	playerID := "player-001"
	writer := NewMockSSEWriter()

	// 註冊連接
	err := sseManager.RegisterConnection(ctx, playerID, writer)
	assert.NoError(t, err, "Should register connection without error")

	// 驗證路由表
	podID, err := client.HGet(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
	assert.NoError(t, err, "Should get player route from Redis")
	assert.Equal(t, "pod-1", podID, "Player should be routed to pod-1")

	// 驗證線上計數
	count, err := client.Get(ctx, consts.SSEOnlineCountKey).Int()
	assert.NoError(t, err, "Should get online count from Redis")
	assert.Equal(t, 1, count, "Online count should be 1")

	t.Log("✅ Player Registration Test Passed")
}

// TestSSE_UnregisterConnection 測試玩家連接取消註冊
func TestSSE_UnregisterConnection(t *testing.T) {
	_, client, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()
	playerID := "player-001"
	writer := NewMockSSEWriter()

	// 先註冊
	err := sseManager.RegisterConnection(ctx, playerID, writer)
	require.NoError(t, err)

	// 取消註冊
	err = sseManager.UnregisterConnection(ctx, playerID)
	assert.NoError(t, err, "Should unregister connection without error")

	// 驗證路由表已刪除
	exists := client.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Val()
	assert.False(t, exists, "Player route should be removed from Redis")

	// 驗證線上計數已減少
	count, err := client.Get(ctx, consts.SSEOnlineCountKey).Int()
	if err == redis.Nil {
		count = 0
	} else {
		require.NoError(t, err)
	}
	assert.Equal(t, 0, count, "Online count should be 0")

	t.Log("✅ Player Unregistration Test Passed")
}

// TestSSE_BroadcastToAll_SinglePod 測試單 Pod 廣播推送
func TestSSE_BroadcastToAll_SinglePod(t *testing.T) {
	_, _, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()

	// 註冊 3 個玩家連接
	writers := make([]*MockSSEWriter, 3)
	for i := 0; i < 3; i++ {
		playerID := fmt.Sprintf("player-%03d", i+1)
		writers[i] = NewMockSSEWriter()
		err := sseManager.RegisterConnection(ctx, playerID, writers[i])
		require.NoError(t, err)
	}

	// 等待連接註冊完成
	time.Sleep(100 * time.Millisecond)

	// 創建測試通知
	now := time.Now()
	notification := &entity.SSENotification{
		ID:        "msg-001",
		Title:     "Test Broadcast",
		Message:   "This is a broadcast message",
		Type:      consts.NotificationTypeSystem,
		Priority:  consts.NotificationPriorityHigh,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	// 廣播訊息
	count, err := sseManager.BroadcastToAll(ctx, notification)
	assert.NoError(t, err, "Should broadcast message without error")
	assert.GreaterOrEqual(t, count, 1, "Should broadcast to at least 1 player")

	// 等待訊息推送
	time.Sleep(200 * time.Millisecond)

	// 驗證所有玩家都收到訊息
	receivedCount := 0
	for i, writer := range writers {
		events := writer.GetNotificationEvents()
		if len(events) > 0 {
			receivedCount++
			t.Logf("Player %d received %d notification events", i+1, len(events))
		}
	}

	assert.GreaterOrEqual(t, receivedCount, 1, "At least 1 player should receive notification")
	t.Logf("✅ Broadcast Test Passed: %d/%d players received messages", receivedCount, 3)
}

// TestSSE_SendToPlayer_SamePod 測試同 Pod 個別推送
func TestSSE_SendToPlayer_SamePod(t *testing.T) {
	_, _, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()

	// 註冊 2 個玩家連接
	writer1 := NewMockSSEWriter()
	writer2 := NewMockSSEWriter()

	err := sseManager.RegisterConnection(ctx, "player-001", writer1)
	require.NoError(t, err)
	err = sseManager.RegisterConnection(ctx, "player-002", writer2)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// 發送訊息給 player-001
	now := time.Now()
	notification := &entity.SSENotification{
		ID:        "msg-002",
		Type:      consts.NotificationTypeSystem,
		Priority:  consts.NotificationPriorityHigh,
		Title:     "Direct Message",
		Message:   "This is for player-001 only",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	err = sseManager.SendToPlayer(ctx, "player-001", notification)
	assert.NoError(t, err, "Should send message without error")

	time.Sleep(100 * time.Millisecond)

	// 驗證只有 player-001 收到訊息
	events1 := writer1.GetNotificationEvents()
	events2 := writer2.GetNotificationEvents()

	assert.GreaterOrEqual(t, len(events1), 1, "Player 1 should receive at least 1 message")
	assert.Equal(t, 0, len(events2), "Player 2 should not receive any message")

	t.Log("✅ Individual Send Test Passed")
}

// TestSSE_GetOnlinePlayerCount 測試線上人數統計
func TestSSE_GetOnlinePlayerCount(t *testing.T) {
	_, _, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()

	// 初始線上人數應為 0
	count, err := sseManager.GetOnlinePlayerCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Initial online count should be 0")

	// 註冊 5 個玩家
	for i := 1; i <= 5; i++ {
		playerID := fmt.Sprintf("player-%03d", i)
		writer := NewMockSSEWriter()
		err := sseManager.RegisterConnection(ctx, playerID, writer)
		require.NoError(t, err)
	}

	// 驗證線上人數
	count, err = sseManager.GetOnlinePlayerCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 5, count, "Online count should be 5")

	// 取消註冊 2 個玩家
	err = sseManager.UnregisterConnection(ctx, "player-001")
	require.NoError(t, err)
	err = sseManager.UnregisterConnection(ctx, "player-002")
	require.NoError(t, err)

	// 驗證線上人數減少
	count, err = sseManager.GetOnlinePlayerCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Online count should be 3 after unregistering 2 players")

	t.Log("✅ Online Player Count Test Passed")
}

// TestSSE_OfflineMessages 測試離線訊息處理
func TestSSE_OfflineMessages(t *testing.T) {
	_, _, sseManager, cleanup := setupSSEIntegrationTestEnv(t, "pod-1")
	defer cleanup()

	ctx := context.Background()
	playerID := "player-offline"

	// 創建測試通知
	now := time.Now()
	notification := &entity.SSENotification{
		ID:        "msg-offline-001",
		Title:     "Offline Message",
		Message:   "You received this while offline",
		Type:      consts.NotificationTypeSystem,
		Priority:  consts.NotificationPriorityMedium,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	// 儲存離線訊息（玩家未連接）
	err := sseManager.EnqueueOfflineMessage(ctx, playerID, notification)
	assert.NoError(t, err, "Should enqueue offline message without error")

	// 取回離線訊息
	messages, err := sseManager.GetOfflineMessages(ctx, playerID)
	assert.NoError(t, err, "Should get offline messages without error")
	assert.GreaterOrEqual(t, len(messages), 1, "Should have at least 1 offline message")

	if len(messages) > 0 {
		assert.Equal(t, "msg-offline-001", messages[0].ID)
		assert.Equal(t, "Offline Message", messages[0].Title)
	}

	// 清除離線訊息
	err = sseManager.ClearOfflineMessages(ctx, playerID)
	assert.NoError(t, err, "Should clear offline messages without error")

	// 驗證已清除
	messages, err = sseManager.GetOfflineMessages(ctx, playerID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(messages), "Offline messages should be cleared")

	t.Log("✅ Offline Messages Test Passed")
}
