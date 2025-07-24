package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// MessageUseCase 訊息用例
type MessageUseCase struct {
	campaignRepo      repositoryport.MessageCampaignRepository
	playerMessageRepo repositoryport.PlayerMessageRepository
	playerRepo        repositoryport.PlayerRepository
	logger            infraport.Logger
}

// NewMessageUseCase 創建訊息用例
func NewMessageUseCase(
	campaignRepo repositoryport.MessageCampaignRepository,
	playerMessageRepo repositoryport.PlayerMessageRepository,
	playerRepo repositoryport.PlayerRepository,
	logger infraport.Logger,
) usecaseport.MessageUseCase {
	return &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerMessageRepo: playerMessageRepo,
		playerRepo:        playerRepo,
		logger:            logger,
	}
}

// CreateMessageCampaign 創建會員訊息活動
func (u *MessageUseCase) CreateMessageCampaign(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.CreateMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.Int("campaign.category", int(campaign.Category)),
		attribute.Int("campaign.focus", int(campaign.Focus)),
	)

	tracing.TraceEvent(span, "Creating message campaign")

	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()

	if err := u.campaignRepo.Create(ctx, campaign); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("create campaign: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaign.ID)))
	tracing.TraceEvent(span, "Message campaign created successfully")

	u.logger.InfoLog("Message campaign created",
		u.logger.Int64("campaign_id", int64(campaign.ID)),
		u.logger.String("title", campaign.Title),
		u.logger.String("created_by", campaign.CreatedBy))

	return nil
}

// UpdateMessageCampaign 更新會員訊息活動
func (u *MessageUseCase) UpdateMessageCampaign(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.UpdateMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("campaign.title", campaign.Title),
	)

	tracing.TraceEvent(span, "Updating message campaign")

	// 檢查活動是否存在
	existing, err := u.campaignRepo.FindByID(ctx, campaign.ID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	// 保持創建時間和創建者不變
	campaign.CreatedAt = existing.CreatedAt
	campaign.CreatedBy = existing.CreatedBy
	campaign.UpdatedAt = time.Now()

	if err = u.campaignRepo.Update(ctx, campaign); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("update campaign: %w", err)
	}

	tracing.TraceEvent(span, "Message campaign updated successfully")

	u.logger.InfoLog("Message campaign updated",
		u.logger.Int64("campaign_id", int64(campaign.ID)),
		u.logger.String("title", campaign.Title),
		u.logger.String("updated_by", *campaign.UpdatedBy))

	return nil
}

// DeleteMessageCampaign 刪除會員訊息活動
func (u *MessageUseCase) DeleteMessageCampaign(ctx context.Context, id uint64) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.DeleteMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	tracing.TraceEvent(span, "Deleting message campaign")

	if err := u.campaignRepo.Delete(ctx, id); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("delete campaign: %w", err)
	}

	tracing.TraceEvent(span, "Message campaign deleted successfully")

	u.logger.InfoLog("Message campaign deleted", u.logger.Int64("campaign_id", int64(id)))

	return nil
}

// GetMessageCampaign 獲取會員訊息活動
func (u *MessageUseCase) GetMessageCampaign(
	ctx context.Context,
	id uint64,
) (*entity.MessageCampaign, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.GetMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	campaign, err := u.campaignRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find campaign: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.String("campaign.title", campaign.Title))

	return campaign, nil
}

// ListMessageCampaigns 列出會員訊息活動
func (u *MessageUseCase) ListMessageCampaigns(
	ctx context.Context,
	page, pageSize int,
) ([]*entity.MessageCampaign, int, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.ListMessageCampaigns")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	campaigns, total, err := u.campaignRepo.FindAll(ctx, page, pageSize)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.Int("total_campaigns", total),
		attribute.Int("returned_campaigns", len(campaigns)),
	)

	return campaigns, total, nil
}

// GetPlayerMessages 獲取玩家訊息列表
func (u *MessageUseCase) GetPlayerMessages(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) (*entity.MessageListResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.GetPlayerMessages")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("global_player_id", globalPlayerID),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	// 檢查玩家是否需要新增訊息
	if err := u.processPlayerMessages(ctx, globalPlayerID); err != nil {
		u.logger.WarnLog("Failed to process player messages",
			u.logger.String("global_player_id", globalPlayerID),
			u.logger.Error("err", err))
	}

	// 獲取統計資訊
	stats, err := u.playerMessageRepo.GetPlayerMessageStats(ctx, globalPlayerID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("get player message stats: %w", err)
	}

	// 獲取訊息列表
	messages, total, err := u.playerMessageRepo.FindByPlayerID(ctx, globalPlayerID, page, pageSize)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player messages: %w", err)
	}

	// 轉換為摘要格式
	summaries := make([]entity.MessageSummary, len(messages))
	for i, message := range messages {
		summaries[i] = entity.MessageSummary{
			ID:        message.ID,
			Title:     message.Title,
			Summary:   generateSummary(message.Content),
			IsRead:    message.IsRead,
			CreatedAt: message.CreatedAt,
		}
	}

	tracing.RecordSpanAttributes(span,
		attribute.Int("stats.total_count", stats.TotalCount),
		attribute.Int("stats.read_count", stats.ReadCount),
		attribute.Int("stats.unread_count", stats.UnreadCount),
		attribute.Int("returned_messages", len(summaries)),
	)

	response := &entity.MessageListResponse{
		Stats:    *stats,
		Messages: summaries,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}

	return response, nil
}

// MarkMessageAsRead 標記訊息為已讀
func (u *MessageUseCase) MarkMessageAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.MarkMessageAsRead")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("global_player_id", globalPlayerID),
		attribute.Int64("message.id", int64(messageID)),
	)

	tracing.TraceEvent(span, "Marking message as read")

	if err := u.playerMessageRepo.MarkAsRead(ctx, globalPlayerID, messageID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("mark message as read: %w", err)
	}

	tracing.TraceEvent(span, "Message marked as read successfully")

	u.logger.InfoLog("Message marked as read",
		u.logger.String("global_player_id", globalPlayerID),
		u.logger.Int64("message_id", int64(messageID)))

	return nil
}

// processPlayerMessages 處理玩家訊息（檢查是否需要新增訊息）
func (u *MessageUseCase) processPlayerMessages(ctx context.Context, globalPlayerID string) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.processPlayerMessages")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("global_player_id", globalPlayerID))

	// 獲取玩家資訊
	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 判斷玩家的活躍狀態
	focusType := u.determinePlayerFocus(player)

	tracing.RecordSpanAttributes(span,
		attribute.Int("player.focus_type", int(focusType)),
		attribute.String("player.account", player.Account),
	)

	// 獲取符合條件的活動
	campaigns, err := u.campaignRepo.FindActiveByFocus(ctx, focusType)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find active campaigns: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.Int("matching_campaigns", len(campaigns)))

	// 為每個活動創建訊息（如果尚未存在）
	var newMessages []*entity.PlayerMessage
	for _, campaign := range campaigns {
		// 檢查是否已經存在該活動的訊息
		exists, err := u.playerMessageRepo.CheckMessageExists(ctx, globalPlayerID, campaign.ID)
		if err != nil {
			u.logger.WarnLog("Failed to check message existence",
				u.logger.String("global_player_id", globalPlayerID),
				u.logger.Int64("campaign_id", int64(campaign.ID)),
				u.logger.Error("err", err))
			continue
		}

		if !exists {
			message := &entity.PlayerMessage{
				GlobalPlayerID: globalPlayerID,
				CampaignID:     campaign.ID,
				Title:          campaign.Title,
				Content:        campaign.Content,
				IsRead:         false,
				CreatedAt:      time.Now(),
			}
			newMessages = append(newMessages, message)
		}
	}

	// 批量創建新訊息
	if len(newMessages) > 0 {
		if err = u.playerMessageRepo.CreateBatch(ctx, newMessages); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("create batch messages: %w", err)
		}

		tracing.TraceEvent(span, "New messages created",
			attribute.Int("new_messages_count", len(newMessages)))

		u.logger.InfoLog("New messages created for player",
			u.logger.String("global_player_id", globalPlayerID),
			u.logger.Int("new_messages_count", len(newMessages)))
	}

	return nil
}

// determinePlayerFocus 根據玩家最後活躍時間判斷焦點類型
func (u *MessageUseCase) determinePlayerFocus(player *entity.Player) uint8 {
	if player.LastActiveAt == nil {
		return entity.FocusNotActivity // 沒有活躍記錄視為不活躍
	}

	now := time.Now()
	daysSinceActive := int(now.Sub(*player.LastActiveAt).Hours() / 24)

	switch {
	case daysSinceActive <= 30:
		return entity.FocusInThirty // 30天內
	case daysSinceActive <= 100:
		return entity.FocusLowActivity // 31-100天
	default:
		return entity.FocusNotActivity // 100天以上
	}
}

// generateSummary 生成內容摘要
func generateSummary(content string) string {
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")

	runes := []rune(content)
	if len(runes) <= 100 {
		return content
	}
	return string(runes[:100]) + "..."
}
