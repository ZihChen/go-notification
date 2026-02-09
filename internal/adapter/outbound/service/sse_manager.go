package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	outboundService "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/redis/go-redis/v9"
)

// sseManager SSE 連接管理器實作 (完整版 with Redis Pub/Sub)
type sseManager struct {
	podID        string                       // Pod ID (從環境變數注入)
	connections  map[string]inbound.SSEWriter // 本地連接池 (playerID → Writer)
	mu           sync.RWMutex                 // 併發安全鎖
	cacheManager infrastructure.CacheManager  // Cache Manager
	redisClient  *redis.Client                // Redis 客戶端 (用於 Pub/Sub 和 Hash/Streams 操作)
	pubsub       *redis.PubSub                // Pub/Sub 訂閱
	logger       infrastructure.Logger        // 日誌記錄器
	ctx          context.Context              // 背景任務 Context
	cancel       context.CancelFunc           // 取消函數
	wg           sync.WaitGroup               // 等待 Goroutine 結束
}

// NewSSEManager 創建 SSE Manager 實例並啟動 Pub/Sub 監聽器
func NewSSEManager(
	podID string,
	cacheManager infrastructure.CacheManager,
	logger infrastructure.Logger,
) outboundService.SSEManager {
	// 獲取 Redis 客戶端
	redisClient, err := cacheManager.GetClient()
	if err != nil {
		logger.ErrorLog("Failed to get Redis client", logger.Error("error", err))
		// 如果無法獲取 Redis 客戶端，返回簡化版本（僅本地功能）
		return &sseManager{
			podID:        podID,
			connections:  make(map[string]inbound.SSEWriter),
			cacheManager: cacheManager,
			logger:       logger,
		}
	}

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
		consts.SSEBroadcastChannel,                       // sse:broadcast
		fmt.Sprintf(consts.SSEPodChannel, manager.podID), // sse:pod:{podID}
	)

	// 啟動 Pub/Sub 監聽器
	manager.wg.Add(1)
	go manager.startPubSubListener()

	logger.InfoLog("SSE Manager initialized",
		logger.String("pod_id", podID),
		logger.String("broadcast_channel", consts.SSEBroadcastChannel),
		logger.String("pod_channel", fmt.Sprintf(consts.SSEPodChannel, podID)))

	return manager
}

// RegisterConnection 註冊玩家 SSE 連接
func (m *sseManager) RegisterConnection(
	ctx context.Context,
	playerID string,
	writer inbound.SSEWriter,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 加入本地連接池
	m.connections[playerID] = writer

	// 2. 更新全局路由表 (Redis Hash)
	if m.redisClient != nil {
		if err := m.redisClient.HSet(ctx, consts.SSEPlayerRoutesKey, playerID, m.podID).
			Err(); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to update player routes in Redis",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
		}

		// 3. 增加線上計數
		if err := m.redisClient.Incr(ctx, consts.SSEOnlineCountKey).Err(); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to increment online count",
				m.logger.Error("error", err))
		}
	}

	m.logger.InfoWithContext(ctx, "Player connected",
		m.logger.String("player_id", playerID),
		m.logger.String("pod_id", m.podID))

	return nil
}

// UnregisterConnection 取消註冊玩家 SSE 連接
func (m *sseManager) UnregisterConnection(ctx context.Context, playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 從本地連接池移除
	delete(m.connections, playerID)

	// 2. 從全局路由表移除 (Redis Hash)
	// 使用獨立的 context 進行 cleanup 操作，避免因原始 context 取消而失敗
	if m.redisClient != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.redisClient.HDel(cleanupCtx, consts.SSEPlayerRoutesKey, playerID).
			Err(); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to remove player route from Redis",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
		}

		// 3. 減少線上計數
		if err := m.redisClient.Decr(cleanupCtx, consts.SSEOnlineCountKey).Err(); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to decrement online count",
				m.logger.Error("error", err))
		}
	}

	m.logger.InfoWithContext(ctx, "Player disconnected",
		m.logger.String("player_id", playerID),
		m.logger.String("pod_id", m.podID))

	return nil
}

// IsPlayerOnline 檢查玩家是否在線上
func (m *sseManager) IsPlayerOnline(ctx context.Context, playerID string) (bool, error) {
	if m.redisClient != nil {
		// 查詢 Redis Hash: HEXISTS sse:player_routes {playerID}
		exists, err := m.redisClient.HExists(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
		if err != nil {
			m.logger.WarnWithContext(ctx, "Failed to check player online status in Redis",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
			// 降級到本地檢查
		} else {
			return exists, nil
		}
	}

	// 降級：僅檢查本地連接池
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[playerID]
	return exists, nil
}

// GetOnlinePlayerCount 獲取線上玩家總數
func (m *sseManager) GetOnlinePlayerCount(ctx context.Context) (int, error) {
	if m.redisClient != nil {
		// 從 Redis 讀取: GET sse:online_count
		count, err := m.redisClient.Get(ctx, consts.SSEOnlineCountKey).Int()
		if err != nil && err != redis.Nil {
			m.logger.WarnWithContext(ctx, "Failed to get online count from Redis",
				m.logger.Error("error", err))
			// 降級到本地計數
		} else if err == nil {
			return count, nil
		}
	}

	// 降級：僅返回本地連接數
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections), nil
}

// GetOnlinePlayerIDs 獲取所有線上玩家 ID 清單
func (m *sseManager) GetOnlinePlayerIDs(ctx context.Context) ([]string, error) {
	if m.redisClient != nil {
		// 從 Redis 讀取: HKEYS sse:player_routes
		playerIDs, err := m.redisClient.HKeys(ctx, consts.SSEPlayerRoutesKey).Result()
		if err != nil {
			m.logger.WarnWithContext(ctx, "Failed to get player IDs from Redis",
				m.logger.Error("error", err))
			// 降級到本地查詢
		} else {
			return playerIDs, nil
		}
	}

	// 降級：僅返回本地連接的玩家 ID
	m.mu.RLock()
	defer m.mu.RUnlock()

	playerIDs := make([]string, 0, len(m.connections))
	for playerID := range m.connections {
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs, nil
}

// BroadcastToAll 廣播訊息給所有線上玩家
func (m *sseManager) BroadcastToAll(
	ctx context.Context,
	notification *entity.SSENotification,
) (int, error) {
	// 發布到 Redis Pub/Sub: PUBLISH sse:broadcast {notification}
	if m.redisClient != nil {
		data, err := json.Marshal(notification)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal notification: %w", err)
		}

		if err := m.redisClient.Publish(ctx, consts.SSEBroadcastChannel, data).Err(); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to publish broadcast to Redis",
				m.logger.Error("error", err))
			// 降級：僅推送到本地連接
		} else {
			m.logger.InfoWithContext(ctx, "Broadcast published to Redis",
				m.logger.String("notification_id", notification.ID))
			// Pub/Sub 會自動推送到所有 Pod（包括本 Pod）
			// 但為了降低延遲，同時也直接推送到本地連接
		}
	}

	// 推送到本地連接池（無論 Redis 是否成功）
	return m.broadcastToLocalConnections(ctx, notification), nil
}

// SendToPlayer 推送訊息給單一玩家
func (m *sseManager) SendToPlayer(
	ctx context.Context,
	playerID string,
	notification *entity.SSENotification,
) error {
	// 先檢查玩家是否在本地連接池
	m.mu.RLock()
	writer, existsLocally := m.connections[playerID]
	m.mu.RUnlock()

	if existsLocally {
		// 玩家在本地，直接推送
		return m.sendNotificationToWriter(writer, notification)
	}

	// 玩家不在本地，查詢 Redis 路由表
	if m.redisClient != nil {
		targetPodID, err := m.redisClient.HGet(ctx, consts.SSEPlayerRoutesKey, playerID).Result()
		if err == redis.Nil {
			// 玩家離線，加入離線佇列
			return m.EnqueueOfflineMessage(ctx, playerID, notification)
		} else if err != nil {
			m.logger.WarnWithContext(ctx, "Failed to get player route from Redis",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
			return fmt.Errorf("failed to query player route: %w", err)
		}

		// 玩家在其他 Pod，通過 Pub/Sub 轉發
		if targetPodID != m.podID {
			// 設置目標玩家 ID 用於跨 Pod 路由
			notification.TargetPlayerID = playerID

			data, err := json.Marshal(notification)
			if err != nil {
				return fmt.Errorf("failed to marshal notification: %w", err)
			}

			channel := fmt.Sprintf(consts.SSEPodChannel, targetPodID)
			if err := m.redisClient.Publish(ctx, channel, data).Err(); err != nil {
				m.logger.WarnWithContext(ctx, "Failed to publish to target pod",
					m.logger.Error("error", err),
					m.logger.String("player_id", playerID),
					m.logger.String("target_pod", targetPodID))
				return fmt.Errorf("failed to publish to pod %s: %w", targetPodID, err)
			}

			m.logger.InfoWithContext(ctx, "Message routed to target pod",
				m.logger.String("player_id", playerID),
				m.logger.String("target_pod", targetPodID),
				m.logger.String("notification_id", notification.ID))
			return nil
		}
	}

	// 無 Redis 或其他錯誤，加入離線佇列
	return m.EnqueueOfflineMessage(ctx, playerID, notification)
}

// SendToPlayers 批次推送訊息給多位玩家
func (m *sseManager) SendToPlayers(
	ctx context.Context,
	playerIDs []string,
	notification *entity.SSENotification,
) (int, error) {
	successCount := 0

	for _, playerID := range playerIDs {
		if err := m.SendToPlayer(ctx, playerID, notification); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to send notification to player",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
		} else {
			successCount++
		}
	}

	return successCount, nil
}

// EnqueueOfflineMessage 將訊息加入玩家的離線佇列
func (m *sseManager) EnqueueOfflineMessage(
	ctx context.Context,
	playerID string,
	notification *entity.SSENotification,
) error {
	if m.redisClient == nil {
		return fmt.Errorf("redis client not available for offline message storage")
	}

	// 使用 Redis Streams: XADD sse:offline:{playerID} MAXLEN ~ 100 * notification
	streamKey := fmt.Sprintf(consts.SSEOfflineMessagesKey, playerID)
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if err := m.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: 100, // 最多保留 100 條離線訊息
		Approx: true,
		Values: map[string]interface{}{
			"notification": string(data),
		},
	}).Err(); err != nil {
		return fmt.Errorf("failed to add offline message: %w", err)
	}

	// 設置 TTL: EXPIRE sse:offline:{playerID} 604800 (7天)
	if err := m.redisClient.Expire(ctx, streamKey, 7*24*time.Hour).Err(); err != nil {
		m.logger.WarnWithContext(ctx, "Failed to set TTL for offline messages",
			m.logger.Error("error", err),
			m.logger.String("player_id", playerID))
	}

	m.logger.InfoWithContext(ctx, "Enqueued offline message",
		m.logger.String("player_id", playerID),
		m.logger.String("notification_id", notification.ID))

	return nil
}

// GetOfflineMessages 獲取玩家的離線訊息清單
func (m *sseManager) GetOfflineMessages(
	ctx context.Context,
	playerID string,
) ([]*entity.SSENotification, error) {
	if m.redisClient == nil {
		return []*entity.SSENotification{}, nil
	}

	// 從 Redis Streams 讀取: XRANGE sse:offline:{playerID} - +
	streamKey := fmt.Sprintf(consts.SSEOfflineMessagesKey, playerID)
	messages, err := m.redisClient.XRange(ctx, streamKey, "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get offline messages: %w", err)
	}

	notifications := make([]*entity.SSENotification, 0, len(messages))
	for _, msg := range messages {
		if notificationData, ok := msg.Values["notification"].(string); ok {
			var notification entity.SSENotification
			if err := json.Unmarshal([]byte(notificationData), &notification); err != nil {
				m.logger.WarnWithContext(ctx, "Failed to unmarshal offline notification",
					m.logger.Error("error", err),
					m.logger.String("player_id", playerID))
				continue
			}
			notifications = append(notifications, &notification)
		}
	}

	return notifications, nil
}

// ClearOfflineMessages 清除玩家的離線訊息
func (m *sseManager) ClearOfflineMessages(ctx context.Context, playerID string) error {
	if m.redisClient == nil {
		return nil
	}

	// 刪除 Redis Stream: DEL sse:offline:{playerID}
	streamKey := fmt.Sprintf(consts.SSEOfflineMessagesKey, playerID)
	if err := m.redisClient.Del(ctx, streamKey).Err(); err != nil {
		return fmt.Errorf("failed to clear offline messages: %w", err)
	}

	m.logger.InfoWithContext(ctx, "Cleared offline messages",
		m.logger.String("player_id", playerID))

	return nil
}

// Shutdown 優雅關閉 SSE Manager
func (m *sseManager) Shutdown(ctx context.Context) error {
	m.logger.InfoLog("SSE Manager shutting down", m.logger.String("pod_id", m.podID))

	// 1. 發送關閉通知給所有連接
	shutdownNotification := &entity.SSENotification{
		ID:       fmt.Sprintf("shutdown-%d", time.Now().Unix()),
		Title:    "Server Shutdown",
		Message:  "Server is shutting down, please reconnect",
		Type:     "system",
		Priority: "high",
	}

	m.mu.RLock()
	for playerID, writer := range m.connections {
		if err := m.sendNotificationToWriter(writer, shutdownNotification); err != nil {
			m.logger.WarnLog("Failed to send shutdown notification",
				m.logger.String("player_id", playerID),
				m.logger.Error("error", err))
		}
	}
	m.mu.RUnlock()

	// 等待客戶端接收關閉通知
	time.Sleep(2 * time.Second)

	// 2. 關閉所有連接並清理 Redis 路由表
	m.mu.Lock()
	for playerID, writer := range m.connections {
		_ = writer.Close()
		if m.redisClient != nil {
			m.redisClient.HDel(ctx, consts.SSEPlayerRoutesKey, playerID)
		}
	}
	m.connections = make(map[string]inbound.SSEWriter)
	m.mu.Unlock()

	// 3. 重置線上計數 (從 Redis 減去本 Pod 的連接數)
	// 注意：這裡簡化處理，實際應該維護 Pod 級別的計數

	// 4. 關閉 Pub/Sub 訂閱
	if m.pubsub != nil {
		if err := m.pubsub.Close(); err != nil {
			m.logger.WarnLog("Failed to close pubsub", m.logger.Error("error", err))
		}
	}

	// 5. 取消背景任務並等待 Goroutine 結束
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	m.logger.InfoLog("SSE Manager shutdown complete", m.logger.String("pod_id", m.podID))
	return nil
}

// ==================== 私有方法 ====================

// startPubSubListener 啟動 Pub/Sub 監聽器 Goroutine
func (m *sseManager) startPubSubListener() {
	defer m.wg.Done()

	ch := m.pubsub.Channel()
	m.logger.InfoLog("Pub/Sub listener started", m.logger.String("pod_id", m.podID))

	for {
		select {
		case <-m.ctx.Done():
			m.logger.InfoLog("Pub/Sub listener stopped", m.logger.String("pod_id", m.podID))
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}

			switch msg.Channel {
			case consts.SSEBroadcastChannel:
				m.handleBroadcastMessage(msg.Payload)
			case fmt.Sprintf(consts.SSEPodChannel, m.podID):
				m.handleTargetedMessage(msg.Payload)
			}
		}
	}
}

// handleBroadcastMessage 處理廣播訊息
func (m *sseManager) handleBroadcastMessage(payload string) {
	var notification entity.SSENotification
	if err := json.Unmarshal([]byte(payload), &notification); err != nil {
		m.logger.WarnLog("Failed to unmarshal broadcast notification",
			m.logger.Error("error", err))
		return
	}

	count := m.broadcastToLocalConnections(context.Background(), &notification)
	m.logger.InfoLog("Broadcast message delivered",
		m.logger.String("notification_id", notification.ID),
		m.logger.Int("local_recipients", count))
}

// handleTargetedMessage 處理個別推送訊息
func (m *sseManager) handleTargetedMessage(payload string) {
	var notification entity.SSENotification
	if err := json.Unmarshal([]byte(payload), &notification); err != nil {
		m.logger.WarnLog("Failed to unmarshal targeted notification",
			m.logger.Error("error", err))
		return
	}

	// 檢查是否有目標玩家 ID
	if notification.TargetPlayerID == "" {
		m.logger.WarnLog("Targeted message missing target player ID",
			m.logger.String("notification_id", notification.ID))
		return
	}

	// 查找本地連接
	m.mu.RLock()
	writer, exists := m.connections[notification.TargetPlayerID]
	m.mu.RUnlock()

	if !exists {
		m.logger.WarnLog("Target player not found in local connections",
			m.logger.String("notification_id", notification.ID),
			m.logger.String("target_player_id", notification.TargetPlayerID))
		return
	}

	// 發送訊息到對應 Writer
	if err := m.sendNotificationToWriter(writer, &notification); err != nil {
		m.logger.WarnLog("Failed to send targeted message to player",
			m.logger.Error("error", err),
			m.logger.String("notification_id", notification.ID),
			m.logger.String("target_player_id", notification.TargetPlayerID))
		return
	}

	m.logger.InfoLog("Targeted message delivered successfully",
		m.logger.String("notification_id", notification.ID),
		m.logger.String("target_player_id", notification.TargetPlayerID))
}

// broadcastToLocalConnections 推送訊息到本地所有連接
func (m *sseManager) broadcastToLocalConnections(
	ctx context.Context,
	notification *entity.SSENotification,
) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	successCount := 0
	for playerID, writer := range m.connections {
		if err := m.sendNotificationToWriter(writer, notification); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to send notification to local player",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
		} else {
			successCount++
		}
	}

	return successCount
}

// sendNotificationToWriter 發送通知到 SSE Writer
func (m *sseManager) sendNotificationToWriter(
	writer inbound.SSEWriter,
	notification *entity.SSENotification,
) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if err := writer.Write(consts.SSEEventTypeNotification, string(data)); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return writer.Flush()
}
