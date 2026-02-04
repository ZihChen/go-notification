// Package tests SSE 多 Pod 整合測試
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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
// 多 Pod 測試工具
// =============================================================================

// PodInstance 代表一個 SSE Service Pod 實例
type PodInstance struct {
	ID        string                      // Pod ID (pod-1, pod-2, pod-3)
	Manager   outboundService.SSEManager  // SSE Manager
	Writers   map[string]*MockSSEWriter   // playerID -> MockSSEWriter
	WritersMu sync.RWMutex                // 保護 Writers map
}

// NewPodInstance 創建一個新的 Pod 實例
func NewPodInstance(podID string, redisClient *redis.Client, cacheManager *TestCacheManager) *PodInstance {
	logger := helper.NewMockLogger()
	manager := service.NewSSEManager(podID, cacheManager, logger)

	return &PodInstance{
		ID:      podID,
		Manager: manager,
		Writers: make(map[string]*MockSSEWriter),
	}
}

// RegisterPlayer 在此 Pod 註冊玩家連接
func (p *PodInstance) RegisterPlayer(ctx context.Context, playerID string) (*MockSSEWriter, error) {
	p.WritersMu.Lock()
	defer p.WritersMu.Unlock()

	writer := NewMockSSEWriter()
	p.Writers[playerID] = writer

	err := p.Manager.RegisterConnection(ctx, playerID, writer)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

// UnregisterPlayer 取消註冊玩家連接
func (p *PodInstance) UnregisterPlayer(ctx context.Context, playerID string) error {
	p.WritersMu.Lock()
	defer p.WritersMu.Unlock()

	err := p.Manager.UnregisterConnection(ctx, playerID)
	if err != nil {
		return err
	}

	delete(p.Writers, playerID)
	return nil
}

// GetWriter 獲取玩家的 Writer
func (p *PodInstance) GetWriter(playerID string) *MockSSEWriter {
	p.WritersMu.RLock()
	defer p.WritersMu.RUnlock()
	return p.Writers[playerID]
}

// Cleanup 清理 Pod 資源
func (p *PodInstance) Cleanup(ctx context.Context) {
	// 取消註冊所有連接
	p.WritersMu.Lock()
	defer p.WritersMu.Unlock()

	for playerID := range p.Writers {
		_ = p.Manager.UnregisterConnection(ctx, playerID)
	}
}

// MultiPodTestSetup 多 Pod 測試環境
type MultiPodTestSetup struct {
	Redis        *miniredis.Miniredis
	RedisClient  *redis.Client
	CacheManager *TestCacheManager
	Pods         []*PodInstance
	Cleanup      func()
}

// SetupMultiPodTest 設置多 Pod 測試環境
func SetupMultiPodTest(t *testing.T, podCount int) *MultiPodTestSetup {
	// 啟動 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// 創建 Redis 客戶端
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 創建 CacheManager
	cacheManager := &TestCacheManager{client: redisClient}

	// 創建多個 Pod 實例
	pods := make([]*PodInstance, podCount)
	for i := 0; i < podCount; i++ {
		podID := fmt.Sprintf("pod-%d", i+1)
		pods[i] = NewPodInstance(podID, redisClient, cacheManager)
	}

	// 清理函數
	cleanup := func() {
		ctx := context.Background()
		for _, pod := range pods {
			pod.Cleanup(ctx)
		}
		redisClient.Close()
		mr.Close()
	}

	return &MultiPodTestSetup{
		Redis:        mr,
		RedisClient:  redisClient,
		CacheManager: cacheManager,
		Pods:         pods,
		Cleanup:      cleanup,
	}
}

// =============================================================================
// 測試案例: 跨 Pod 廣播
// =============================================================================

func TestMultiPod_BroadcastToAll(t *testing.T) {
	setup := SetupMultiPodTest(t, 3)
	defer setup.Cleanup()

	ctx := context.Background()

	// 在不同 Pod 上註冊玩家
	// Pod-1: player-001, player-002
	// Pod-2: player-003, player-004
	// Pod-3: player-005, player-006

	_, err := setup.Pods[0].RegisterPlayer(ctx, "player-001")
	require.NoError(t, err)
	_, err = setup.Pods[0].RegisterPlayer(ctx, "player-002")
	require.NoError(t, err)

	_, err = setup.Pods[1].RegisterPlayer(ctx, "player-003")
	require.NoError(t, err)
	_, err = setup.Pods[1].RegisterPlayer(ctx, "player-004")
	require.NoError(t, err)

	_, err = setup.Pods[2].RegisterPlayer(ctx, "player-005")
	require.NoError(t, err)
	_, err = setup.Pods[2].RegisterPlayer(ctx, "player-006")
	require.NoError(t, err)

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 從 Pod-1 發送廣播
	notification := &entity.SSENotification{
		Type:     consts.NotificationTypeSystem,
		Title:    "系統廣播",
		Message:  "這是一則跨 Pod 廣播測試",
		Priority: consts.NotificationPriorityMedium,
	}

	sentCount, err := setup.Pods[0].Manager.BroadcastToAll(ctx, notification)
	require.NoError(t, err)
	// BroadcastToAll 只返回本地推送數量（Pod-1 有 2 個玩家）
	assert.Equal(t, 2, sentCount, "Pod-1 應該發送給 2 個本地玩家")

	// 等待訊息傳遞
	time.Sleep(300 * time.Millisecond)

	// 驗證所有 Pod 上的玩家都收到訊息
	allPlayers := []string{"player-001", "player-002", "player-003", "player-004", "player-005", "player-006"}
	podIndices := []int{0, 0, 1, 1, 2, 2}

	for i, playerID := range allPlayers {
		podIdx := podIndices[i]
		writer := setup.Pods[podIdx].GetWriter(playerID)
		require.NotNil(t, writer, "玩家 %s 的 Writer 應該存在", playerID)

		// 獲取接收到的通知事件
		writer.mu.Lock()
		notificationEvents := []SSEEvent{}
		for _, event := range writer.events {
			if event.Event == "notification" {
				notificationEvents = append(notificationEvents, event)
			}
		}
		writer.mu.Unlock()

		assert.GreaterOrEqual(t, len(notificationEvents), 1, "玩家 %s 應該至少收到 1 個通知", playerID)

		// 驗證訊息內容
		if len(notificationEvents) > 0 {
			var receivedNotif entity.SSENotification
			err := json.Unmarshal([]byte(notificationEvents[0].Data), &receivedNotif)
			assert.NoError(t, err)
			assert.Equal(t, notification.Title, receivedNotif.Title)
			assert.Equal(t, notification.Message, receivedNotif.Message)
		}
	}

	t.Log("✅ 跨 Pod 廣播測試通過")
}

// =============================================================================
// 測試案例: 跨 Pod 個別推送
// =============================================================================

func TestMultiPod_SendToPlayer_CrossPod(t *testing.T) {
	setup := SetupMultiPodTest(t, 3)
	defer setup.Cleanup()

	ctx := context.Background()

	// Pod-1: player-001
	// Pod-2: player-002
	// Pod-3: player-003

	_, err := setup.Pods[0].RegisterPlayer(ctx, "player-001")
	require.NoError(t, err)

	_, err = setup.Pods[1].RegisterPlayer(ctx, "player-002")
	require.NoError(t, err)

	_, err = setup.Pods[2].RegisterPlayer(ctx, "player-003")
	require.NoError(t, err)

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 從 Pod-1 發送訊息給 Pod-2 上的玩家
	notification := &entity.SSENotification{
		Type:     consts.NotificationTypeSystem,
		Title:    "個別推送",
		Message:  "這是一則跨 Pod 個別推送測試",
		Priority: consts.NotificationPriorityHigh,
	}

	err = setup.Pods[0].Manager.SendToPlayer(ctx, "player-002", notification)
	require.NoError(t, err)

	// 等待訊息傳遞（跨 Pod 需要更長時間）
	time.Sleep(600 * time.Millisecond)

	// 驗證 player-002 收到訊息
	writer2 := setup.Pods[1].GetWriter("player-002")
	require.NotNil(t, writer2)

	writer2.mu.Lock()
	notificationEvents := []SSEEvent{}
	for _, event := range writer2.events {
		if event.Event == "notification" {
			notificationEvents = append(notificationEvents, event)
		}
	}
	writer2.mu.Unlock()

	assert.GreaterOrEqual(t, len(notificationEvents), 1, "player-002 應該收到訊息")

	// 驗證 player-001 和 player-003 沒有收到訊息
	writer1 := setup.Pods[0].GetWriter("player-001")
	writer3 := setup.Pods[2].GetWriter("player-003")

	writer1.mu.Lock()
	count1 := 0
	for _, event := range writer1.events {
		if event.Event == "notification" {
			count1++
		}
	}
	writer1.mu.Unlock()

	writer3.mu.Lock()
	count3 := 0
	for _, event := range writer3.events {
		if event.Event == "notification" {
			count3++
		}
	}
	writer3.mu.Unlock()

	assert.Equal(t, 0, count1, "player-001 不應該收到訊息")
	assert.Equal(t, 0, count3, "player-003 不應該收到訊息")

	t.Log("✅ 跨 Pod 個別推送測試通過")
}

// =============================================================================
// 測試案例: 玩家路由表一致性
// =============================================================================

func TestMultiPod_PlayerRoutingConsistency(t *testing.T) {
	setup := SetupMultiPodTest(t, 3)
	defer setup.Cleanup()

	ctx := context.Background()

	// 在不同 Pod 上註冊玩家
	_, err := setup.Pods[0].RegisterPlayer(ctx, "player-001")
	require.NoError(t, err)
	_, err = setup.Pods[1].RegisterPlayer(ctx, "player-002")
	require.NoError(t, err)
	_, err = setup.Pods[2].RegisterPlayer(ctx, "player-003")
	require.NoError(t, err)

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 驗證每個 Pod 的線上玩家數量一致
	for i, pod := range setup.Pods {
		count, err := pod.Manager.GetOnlinePlayerCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "Pod-%d 的線上玩家數量應該是 3", i+1)
	}

	// 取消註冊一個玩家
	err = setup.Pods[1].UnregisterPlayer(ctx, "player-002")
	require.NoError(t, err)

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 再次驗證每個 Pod 的線上玩家數量一致
	for i, pod := range setup.Pods {
		count, err := pod.Manager.GetOnlinePlayerCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "Pod-%d 的線上玩家數量應該是 2", i+1)
	}

	t.Log("✅ 玩家路由表一致性測試通過")
}

// =============================================================================
// 測試案例: Pod 間訊息轉發延遲
// =============================================================================

func TestMultiPod_MessageLatency(t *testing.T) {
	setup := SetupMultiPodTest(t, 2)
	defer setup.Cleanup()

	ctx := context.Background()

	// Pod-1: player-001
	// Pod-2: player-002

	_, err := setup.Pods[0].RegisterPlayer(ctx, "player-001")
	require.NoError(t, err)
	_, err = setup.Pods[1].RegisterPlayer(ctx, "player-002")
	require.NoError(t, err)

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 測試 10 次跨 Pod 推送，記錄延遲
	latencies := make([]time.Duration, 10)

	for i := 0; i < 10; i++ {
		notification := &entity.SSENotification{
			Type:     consts.NotificationTypeSystem,
			Title:    fmt.Sprintf("延遲測試 #%d", i+1),
			Message:  "測試訊息",
			Priority: consts.NotificationPriorityMedium,
		}

		start := time.Now()
		err := setup.Pods[0].Manager.SendToPlayer(ctx, "player-002", notification)
		require.NoError(t, err)

		// 等待訊息到達
		time.Sleep(100 * time.Millisecond)
		latency := time.Since(start)
		latencies[i] = latency
	}

	// 計算平均延遲
	var totalLatency time.Duration
	for _, latency := range latencies {
		totalLatency += latency
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	t.Logf("平均跨 Pod 推送延遲: %v", avgLatency)
	t.Logf("最小延遲: %v", minDuration(latencies))
	t.Logf("最大延遲: %v", maxDuration(latencies))

	// 驗證平均延遲在合理範圍內 (< 300ms，包含 100ms 等待時間 + Pub/Sub 傳遞延遲)
	assert.Less(t, avgLatency.Milliseconds(), int64(300), "平均延遲應該小於 300ms")

	t.Log("✅ Pod 間訊息轉發延遲測試通過")
}

// =============================================================================
// 測試案例: 高併發跨 Pod 推送
// =============================================================================

func TestMultiPod_ConcurrentBroadcast(t *testing.T) {
	setup := SetupMultiPodTest(t, 3)
	defer setup.Cleanup()

	ctx := context.Background()

	// 在 3 個 Pod 上各註冊 5 個玩家（總共 15 個玩家）
	for podIdx := 0; podIdx < 3; podIdx++ {
		for i := 1; i <= 5; i++ {
			playerID := fmt.Sprintf("player-%d-%d", podIdx+1, i)
			_, err := setup.Pods[podIdx].RegisterPlayer(ctx, playerID)
			require.NoError(t, err)
		}
	}

	// 等待路由表同步和 Pub/Sub listener 啟動
	time.Sleep(500 * time.Millisecond)

	// 併發發送 20 則廣播
	var wg sync.WaitGroup
	errorsChan := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			notification := &entity.SSENotification{
				Type:     consts.NotificationTypeSystem,
				Title:    fmt.Sprintf("併發廣播 #%d", idx+1),
				Message:  "測試併發推送",
				Priority: consts.NotificationPriorityMedium,
			}

			// 從隨機 Pod 發送
			podIdx := idx % 3
			sentCount, err := setup.Pods[podIdx].Manager.BroadcastToAll(ctx, notification)
			if err != nil {
				errorsChan <- err
				return
			}

			// 每個 Pod 有 5 個本地玩家，BroadcastToAll 只返回本地推送數量
			if sentCount != 5 {
				errorsChan <- fmt.Errorf("expected 5 local players for pod-%d, got %d", podIdx+1, sentCount)
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	// 檢查是否有錯誤
	errors := []error{}
	for err := range errorsChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors, "不應該有錯誤發生")

	t.Log("✅ 高併發跨 Pod 推送測試通過")
}

// =============================================================================
// 工具函數
// =============================================================================

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}
