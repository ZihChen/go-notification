package message

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// MessageUseCase 訊息用例
type MessageUseCase struct {
	campaignRepo      repository.MessageCampaignRepository
	merchantRepo      repository.MerchantRepository
	playerMessageRepo repository.PlayerMessageRepository
	playerRepo        repository.PlayerRepository
	logger            infrastructure.Logger
}

// NewMessageUseCase 創建訊息用例
func NewMessageUseCase(
	campaignRepo repository.MessageCampaignRepository,
	merchantRepo repository.MerchantRepository,
	playerMessageRepo repository.PlayerMessageRepository,
	playerRepo repository.PlayerRepository,
	logger infrastructure.Logger,
) inbound.MessageUseCase {
	return &MessageUseCase{
		campaignRepo:      campaignRepo,
		merchantRepo:      merchantRepo,
		playerMessageRepo: playerMessageRepo,
		playerRepo:        playerRepo,
		logger:            logger,
	}
}

// CreateMessageCampaign 創建會員訊息活動
func (u *MessageUseCase) CreateMessageCampaign(
	ctx context.Context,
	campaign *dto.CreateMessageCampaignRequest,
) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.CreateMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.Int("campaign.category", int(campaign.Category)),
		attribute.Int("campaign.target", int(campaign.Target)),
	)

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, campaign.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}
	tracing.TraceEvent(span, "Creating message campaign")

	// 轉換 DTO 到 Entity 物件
	campaignEntity := &entity.MessageCampaign{
		GlobalID:   uuid.NewString(),
		MerchantID: merchant.ID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = copier.Copy(campaignEntity, campaign)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("copy campaign failed: %w", err)
	}

	if err = u.campaignRepo.Create(ctx, campaignEntity); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("create campaign: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaignEntity.ID)))
	tracing.TraceEvent(span, "Message campaign created successfully")

	u.logger.InfoWithContext(ctx, "Message campaign created",
		u.logger.Int64("campaign_id", int64(campaignEntity.ID)),
		u.logger.String("title", campaignEntity.Title),
		u.logger.String("created_by", campaignEntity.CreatedBy))
	return nil
}

// UpdateMessageCampaign 更新會員訊息活動
func (u *MessageUseCase) UpdateMessageCampaign(
	ctx context.Context,
	campaign *dto.UpdateMessageCampaignRequest,
) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.UpdateMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("campaign.global_id", campaign.GlobalID),
		attribute.String("campaign.title", campaign.Title),
	)

	tracing.TraceEvent(span, "Updating message campaign")

	// 檢查活動是否存在
	existing, err := u.campaignRepo.FindByGlobalID(ctx, campaign.GlobalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	// 轉換 DTO 到 Entity 物件
	campaignEntity := &entity.MessageCampaign{
		ID:         existing.ID,
		MerchantID: existing.MerchantID,
		GlobalID:   existing.GlobalID,
		CreatedAt:  existing.CreatedAt,
		CreatedBy:  existing.CreatedBy,
		Status:     consts.MessageCampaignStatusDraft,
		UpdatedAt:  time.Now(),
		UpdatedBy:  &campaign.UpdatedBy,
	}

	// 先複製可更新的字段
	err = copier.Copy(campaignEntity, campaign)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("copy campaign failed: %w", err)
	}

	if err = u.campaignRepo.Update(ctx, campaignEntity); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("update campaign: %w", err)
	}

	tracing.TraceEvent(span, "Message campaign updated successfully")

	u.logger.InfoWithContext(ctx, "Message campaign updated",
		u.logger.String("campaign_global_id", campaign.GlobalID),
		u.logger.String("title", campaign.Title),
		u.logger.String("updated_by", *campaignEntity.UpdatedBy))
	return nil
}

// DeleteMessageCampaign 刪除會員訊息活動
func (u *MessageUseCase) DeleteMessageCampaign(ctx context.Context, globalID string) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.DeleteMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("campaign.global_id", globalID))

	tracing.TraceEvent(span, "Deleting message campaign")

	// 先查找記錄以獲取ID（用於後續更新狀態）
	existing, err := u.campaignRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	if err = u.campaignRepo.Delete(ctx, existing.ID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("delete campaign: %w", err)
	}
	setColumn := map[string]interface{}{
		"status": consts.MessageCampaignStatusCancelled,
	}

	if err = u.campaignRepo.UpdateFields(ctx, existing.ID, setColumn, true); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("update campaign status: %w", err)
	}

	tracing.TraceEvent(span, "Message campaign deleted successfully")
	u.logger.InfoWithContext(
		ctx,
		"Message campaign deleted",
		u.logger.String("campaign_global_id", globalID),
	)
	return nil
}

// GetMessageCampaign 獲取會員訊息活動
func (u *MessageUseCase) GetMessageCampaign(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.GetMessageCampaign")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("campaign.global_id", globalID))

	campaign, err := u.campaignRepo.FindByGlobalID(ctx, globalID)
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
	req *dto.ListMessageCampaignsRequest,
) ([]*entity.MessageCampaign, int, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.ListMessageCampaigns")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.Int("page", req.Page),
		attribute.Int("page_size", req.PageSize),
	)

	// 轉換 DTO 到 Query 物件
	query := &entity.MessageCampaignsQuery{
		IncludeDeleted: true,
	}
	err := copier.Copy(query, req)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, 0, fmt.Errorf("copy query failed: %w", err)
	}

	campaigns, total, err := u.campaignRepo.FindAllWithOptions(ctx, query)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, 0, fmt.Errorf("list campaigns with options: %w", err)
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
		return consts.TargetNotActivity // 沒有活躍記錄視為不活躍
	}

	now := time.Now()
	daysSinceActive := int(now.Sub(*player.LastActiveAt).Hours() / 24)

	switch {
	case daysSinceActive <= 30:
		return consts.TargetHighActivity // 30天內
	case daysSinceActive <= 100:
		return consts.TargetLowActivity // 31-100天
	default:
		return consts.TargetNotActivity // 100天以上
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

// GetMerchantAutoSettings 獲取商戶自動設定
func (u *MessageUseCase) GetMerchantAutoSettings(
	ctx context.Context,
	globalMerchantID string,
) (*dto.MerchantAutoSettingsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.GetMerchantAutoSettings")
	defer tracing.SpanEnd(span)

	// 獲取商戶資訊
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 查找自動設定
	campaigns, err := u.campaignRepo.FindAutoSettingsByMerchantID(ctx, merchant.ID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find auto settings: %w", err)
	}

	return &dto.MerchantAutoSettingsResponse{
		GlobalMerchantID: globalMerchantID,
		Settings:         campaigns,
	}, nil
}

// CreateOrUpdateMerchantAutoSettings 建立或更新商戶自動設定
func (u *MessageUseCase) CreateOrUpdateMerchantAutoSettings(
	ctx context.Context,
	req *dto.MerchantAutoSettingsRequest,
) (*dto.AutoSettingsOperationResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.CreateOrUpdateMerchantAutoSettings")
	defer tracing.SpanEnd(span)

	// 獲取商戶資訊
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, req.GlobalMerchantID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 轉換 DTO 為 entity
	campaigns := make([]*entity.MessageCampaign, len(req.Settings))
	for i, setting := range req.Settings {
		campaign := &entity.MessageCampaign{
			MerchantID:    merchant.ID,
			GlobalID:      uuid.NewString(),
			Category:      setting.Category,
			Item:          setting.Item,
			TriggerType:   setting.TriggerType,
			Title:         setting.Title,
			Content:       setting.Content,
			Target:        consts.TargetAll,                      // 預設為所有玩家
			Status:        consts.MessageCampaignStatusScheduled, // 預設為已排程
			AutoSend:      true,
			RealSentCount: 0,
			CreatedBy:     "auto_system",
		}
		campaigns[i] = campaign
	}

	// 批量新增或更新
	if err = u.campaignRepo.UpsertAutoSettings(ctx, campaigns); err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("upsert auto settings: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Auto settings created/updated",
		u.logger.UInt64("merchant_id", merchant.ID),
		u.logger.Int("settings_count", len(req.Settings)))

	// 反回結果
	return &dto.AutoSettingsOperationResponse{
		GlobalMerchantID: req.GlobalMerchantID,
		CreatedCount:     len(req.Settings),
		UpdatedCount:     0, // 無法精確區分新增和更新的數量
		Operation:        "upsert",
	}, nil
}
