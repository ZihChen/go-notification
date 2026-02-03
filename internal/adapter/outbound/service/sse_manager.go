package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	outboundService "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
)

// sseManager SSE 連接管理器實作 (簡化版)
// TODO: Phase 3 完整實作將包含 Redis Pub/Sub 功能
type sseManager struct {
	podID       string                           // Pod ID (從環境變數注入)
	connections map[string]inbound.SSEWriter     // 本地連接池 (playerID → Writer)
	mu          sync.RWMutex                     // 併發安全鎖
	cacheManager infrastructure.CacheManager    // Redis 客戶端
	logger      infrastructure.Logger            // 日誌記錄器
}

// NewSSEManager 創建 SSE Manager 實例
func NewSSEManager(
	podID string,
	cacheManager infrastructure.CacheManager,
	logger infrastructure.Logger,
) outboundService.SSEManager {
	return &sseManager{
		podID:       podID,
		connections: make(map[string]inbound.SSEWriter),
		cacheManager: cacheManager,
		logger:      logger,
	}
}

// RegisterConnection 註冊玩家 SSE 連接
func (m *sseManager) RegisterConnection(ctx context.Context, playerID string, writer inbound.SSEWriter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 加入本地連接池
	m.connections[playerID] = writer

	// 2. 更新全局路由表 (Redis Hash)
	// TODO: 實作 Redis HSET sse:player_routes {playerID} {podID}

	// 3. 增加線上計數
	// TODO: 實作 Redis INCR sse:online_count

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
	// TODO: 實作 Redis HDEL sse:player_routes {playerID}

	// 3. 減少線上計數
	// TODO: 實作 Redis DECR sse:online_count

	m.logger.InfoWithContext(ctx, "Player disconnected",
		m.logger.String("player_id", playerID),
		m.logger.String("pod_id", m.podID))

	return nil
}

// IsPlayerOnline 檢查玩家是否在線上
func (m *sseManager) IsPlayerOnline(ctx context.Context, playerID string) (bool, error) {
	// TODO: 查詢 Redis Hash: HEXISTS sse:player_routes {playerID}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[playerID]
	return exists, nil
}

// GetOnlinePlayerCount 獲取線上玩家總數
func (m *sseManager) GetOnlinePlayerCount(ctx context.Context) (int, error) {
	// TODO: 從 Redis 讀取: GET sse:online_count
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections), nil
}

// GetOnlinePlayerIDs 獲取所有線上玩家 ID 清單
func (m *sseManager) GetOnlinePlayerIDs(ctx context.Context) ([]string, error) {
	// TODO: 從 Redis 讀取: HKEYS sse:player_routes
	m.mu.RLock()
	defer m.mu.RUnlock()

	playerIDs := make([]string, 0, len(m.connections))
	for playerID := range m.connections {
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs, nil
}

// BroadcastToAll 廣播訊息給所有線上玩家
func (m *sseManager) BroadcastToAll(ctx context.Context, notification *entity.SSENotification) (int, error) {
	// TODO: 發布到 Redis Pub/Sub: PUBLISH sse:broadcast {notification}

	m.mu.RLock()
	defer m.mu.RUnlock()

	successCount := 0
	for playerID, writer := range m.connections {
		if err := m.sendNotificationToWriter(writer, notification); err != nil {
			m.logger.WarnWithContext(ctx, "Failed to send notification to player",
				m.logger.Error("error", err),
				m.logger.String("player_id", playerID))
		} else {
			successCount++
		}
	}

	return successCount, nil
}

// SendToPlayer 推送訊息給單一玩家
func (m *sseManager) SendToPlayer(ctx context.Context, playerID string, notification *entity.SSENotification) error {
	// TODO: 查詢玩家所在 Pod，如果在其他 Pod 則通過 Pub/Sub 轉發

	m.mu.RLock()
	writer, exists := m.connections[playerID]
	m.mu.RUnlock()

	if !exists {
		// 玩家不在本地，加入離線佇列
		return m.EnqueueOfflineMessage(ctx, playerID, notification)
	}

	return m.sendNotificationToWriter(writer, notification)
}

// SendToPlayers 批次推送訊息給多位玩家
func (m *sseManager) SendToPlayers(ctx context.Context, playerIDs []string, notification *entity.SSENotification) (int, error) {
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
func (m *sseManager) EnqueueOfflineMessage(ctx context.Context, playerID string, notification *entity.SSENotification) error {
	// TODO: 使用 Redis Streams: XADD sse:offline:{playerID} MAXLEN ~ 100 * notification
	// TODO: 設置 TTL: EXPIRE sse:offline:{playerID} 604800 (7天)

	m.logger.InfoWithContext(ctx, "Enqueued offline message",
		m.logger.String("player_id", playerID),
		m.logger.String("notification_id", notification.ID))

	return nil
}

// GetOfflineMessages 獲取玩家的離線訊息清單
func (m *sseManager) GetOfflineMessages(ctx context.Context, playerID string) ([]*entity.SSENotification, error) {
	// TODO: 從 Redis Streams 讀取: XRANGE sse:offline:{playerID} - +

	// 暫時返回空列表
	return []*entity.SSENotification{}, nil
}

// ClearOfflineMessages 清除玩家的離線訊息
func (m *sseManager) ClearOfflineMessages(ctx context.Context, playerID string) error {
	// TODO: 刪除 Redis Stream: DEL sse:offline:{playerID}

	m.logger.InfoWithContext(ctx, "Cleared offline messages",
		m.logger.String("player_id", playerID))

	return nil
}

// ==================== 私有輔助方法 ====================

// sendNotificationToWriter 發送通知到 SSE Writer
func (m *sseManager) sendNotificationToWriter(writer inbound.SSEWriter, notification *entity.SSENotification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if err := writer.Write(consts.SSEEventTypeNotification, string(data)); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return writer.Flush()
}
