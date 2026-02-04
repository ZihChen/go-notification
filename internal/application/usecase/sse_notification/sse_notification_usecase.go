package sse_notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
)

// sseNotificationUseCase SSE 通知使用案例實作（完整版，包含 KDS 審計）
type sseNotificationUseCase struct {
	sseManager   service.SSEManager        // SSE 連接管理與訊息推送
	eventService service.EventProducer     // KDS 事件發布服務（用於審計日誌）
	logger       infrastructure.Logger     // 日誌記錄器
}

// NewSSENotificationUseCase 創建 SSE 通知使用案例實例
func NewSSENotificationUseCase(
	sseManager service.SSEManager,
	eventService service.EventProducer,
	logger infrastructure.Logger,
) inbound.SSENotificationUseCase {
	return &sseNotificationUseCase{
		sseManager:   sseManager,
		eventService: eventService,
		logger:       logger,
	}
}

// BroadcastNotification 廣播推送通知給所有線上玩家
func (u *sseNotificationUseCase) BroadcastNotification(
	ctx context.Context,
	req *dto.BroadcastRequest,
) (*dto.BroadcastResponse, error) {
	// 1. 創建通知實體（純記憶體對象，不持久化）
	notification := u.createNotificationEntity(req.Title, req.Message, req.Type, req.Priority, req.IconURL, req.ActionURL)

	// 2. 獲取線上玩家總數
	totalOnline, err := u.sseManager.GetOnlinePlayerCount(ctx)
	if err != nil {
		u.logger.ErrorWithContext(ctx, "Failed to get online player count",
			u.logger.Error("error", err))
		return nil, fmt.Errorf("failed to get online player count: %w", err)
	}

	// 3. 透過 SSEManager 廣播到所有 Pod
	successCount, err := u.sseManager.BroadcastToAll(ctx, notification)
	if err != nil {
		u.logger.ErrorWithContext(ctx, "Failed to broadcast notification",
			u.logger.Error("error", err),
			u.logger.String("notification_id", notification.ID))
		return nil, fmt.Errorf("failed to broadcast notification: %w", err)
	}

	// 4. 返回推送統計
	return &dto.BroadcastResponse{
		NotificationID:  notification.ID,
		TotalOnline:     totalOnline,
		SuccessfulSends: successCount,
		FailedSends:     totalOnline - successCount,
	}, nil
}

// SendNotification 個別推送通知給特定玩家清單
func (u *sseNotificationUseCase) SendNotification(
	ctx context.Context,
	req *dto.SendNotificationRequest,
) (*dto.SendNotificationResponse, error) {
	// 1. 驗證玩家 ID 數量（最多 1000 位）
	if len(req.PlayerIDs) > consts.NotificationMaxBatchSize {
		return nil, fmt.Errorf("player IDs count exceeds maximum batch size: %d > %d",
			len(req.PlayerIDs), consts.NotificationMaxBatchSize)
	}

	// 2. 創建通知實體
	notification := u.createNotificationEntity(req.Title, req.Message, req.Type, req.Priority, req.IconURL, req.ActionURL)

	// 3. 批次推送給目標玩家
	successCount, err := u.sseManager.SendToPlayers(ctx, req.PlayerIDs, notification)
	if err != nil {
		u.logger.ErrorWithContext(ctx, "Failed to send notification to players",
			u.logger.Error("error", err),
			u.logger.String("notification_id", notification.ID),
			u.logger.Int("target_count", len(req.PlayerIDs)))
		return nil, fmt.Errorf("failed to send notification to players: %w", err)
	}

	// 4. 返回推送統計
	return &dto.SendNotificationResponse{
		NotificationID:  notification.ID,
		TotalTargets:    len(req.PlayerIDs),
		SuccessfulSends: successCount,
		FailedSends:     len(req.PlayerIDs) - successCount,
	}, nil
}

// StreamNotifications 建立 SSE 長連接，持續推送實時通知
func (u *sseNotificationUseCase) StreamNotifications(
	ctx context.Context,
	playerID string,
	writer inbound.SSEWriter,
) error {
	// 1. 註冊玩家連接
	if err := u.sseManager.RegisterConnection(ctx, playerID, writer); err != nil {
		u.logger.ErrorWithContext(ctx, "Failed to register player connection",
			u.logger.Error("error", err),
			u.logger.String("player_id", playerID))
		return fmt.Errorf("failed to register connection: %w", err)
	}

	// 確保連接取消時清理
	defer func() {
		if err := u.sseManager.UnregisterConnection(ctx, playerID); err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to unregister player connection",
				u.logger.Error("error", err),
				u.logger.String("player_id", playerID))
		}
	}()

	// 2. 推送連接成功事件
	if err := writer.Write(consts.SSEEventTypeConnected, `{"status":"connected"}`); err != nil {
		u.logger.ErrorWithContext(ctx, "Failed to send connected event",
			u.logger.Error("error", err),
			u.logger.String("player_id", playerID))
		return fmt.Errorf("failed to send connected event: %w", err)
	}
	writer.Flush()

	// 3. 推送離線訊息（Redis Streams）
	offlineMessages, err := u.sseManager.GetOfflineMessages(ctx, playerID)
	if err != nil {
		u.logger.WarnWithContext(ctx, "Failed to get offline messages",
			u.logger.Error("error", err),
			u.logger.String("player_id", playerID))
	} else if len(offlineMessages) > 0 {
		for _, msg := range offlineMessages {
			if err := u.sendNotificationToWriter(writer, msg); err != nil {
				u.logger.WarnWithContext(ctx, "Failed to send offline message",
					u.logger.Error("error", err),
					u.logger.String("player_id", playerID),
					u.logger.String("notification_id", msg.ID))
			}
		}
		// 清除已推送的離線訊息
		if err := u.sseManager.ClearOfflineMessages(ctx, playerID); err != nil {
			u.logger.WarnWithContext(ctx, "Failed to clear offline messages",
				u.logger.Error("error", err),
				u.logger.String("player_id", playerID))
		}
	}

	// 4. 保持連接直到 context 取消
	// 定期發送心跳以保持連接
	ticker := time.NewTicker(consts.SSEHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context 取消，連接關閉
			return nil
		case <-ticker.C:
			// 發送心跳
			if err := writer.Write(consts.SSEEventTypePing, `{"timestamp":"`+time.Now().Format(time.RFC3339)+`"}`); err != nil {
				u.logger.WarnWithContext(ctx, "Failed to send heartbeat",
					u.logger.Error("error", err),
					u.logger.String("player_id", playerID))
				return fmt.Errorf("failed to send heartbeat: %w", err)
			}
			writer.Flush()
		}
	}
}

// ==================== 私有輔助方法 ====================

// createNotificationEntity 創建通知實體
func (u *sseNotificationUseCase) createNotificationEntity(
	title, message, notifType, priority string,
	iconURL, actionURL *string,
) *entity.SSENotification {
	// 設定預設優先級
	if priority == "" {
		priority = consts.NotificationPriorityMedium
	}

	now := time.Now()
	return &entity.SSENotification{
		ID:        uuid.New().String(),
		Title:     title,
		Message:   message,
		Type:      notifType,
		Priority:  priority,
		IconURL:   iconURL,
		ActionURL: actionURL,
		CreatedAt: now,
		ExpiresAt: now.Add(consts.NotificationDefaultTTL),
	}
}

// sendNotificationToWriter 發送通知到 SSE Writer
func (u *sseNotificationUseCase) sendNotificationToWriter(
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
