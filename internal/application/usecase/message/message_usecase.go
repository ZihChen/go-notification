package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

// MessageUseCase 訊息用例
type MessageUseCase struct {
	campaignRepo       repository.MessageCampaignRepository
	campaignTargetRepo repository.CampaignTargetRepository
	merchantRepo       repository.MerchantRepository
	playerMessageRepo  repository.PlayerMessageRepository
	playerRepo         repository.PlayerRepository
	levelRepo          repository.LevelRepository
	tagRepo            repository.TagRepository
	pushApiKeyRepo     repository.PushKeyRepository
	pushService        service.PushNotificationService
	logger             infrastructure.Logger
	tracingService     infrastructure.TracingService
}

// NewMessageUseCase 創建訊息用例
func NewMessageUseCase(
	campaignRepo repository.MessageCampaignRepository,
	campaignTargetRepo repository.CampaignTargetRepository,
	merchantRepo repository.MerchantRepository,
	playerMessageRepo repository.PlayerMessageRepository,
	playerRepo repository.PlayerRepository,
	levelRepo repository.LevelRepository,
	tagRepo repository.TagRepository,
	pushApiKeyRepo repository.PushKeyRepository,
	pushService service.PushNotificationService,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) inbound.MessageUseCase {
	return &MessageUseCase{
		campaignRepo:       campaignRepo,
		campaignTargetRepo: campaignTargetRepo,
		merchantRepo:       merchantRepo,
		playerMessageRepo:  playerMessageRepo,
		playerRepo:         playerRepo,
		levelRepo:          levelRepo,
		tagRepo:            tagRepo,
		pushApiKeyRepo:     pushApiKeyRepo,
		pushService:        pushService,
		logger:             logger,
		tracingService:     tracingService,
	}
}

// CreateMessageCampaign 創建會員訊息活動
func (u *MessageUseCase) CreateMessageCampaign(
	ctx context.Context,
	campaign *dto.CreateMessageCampaignRequest,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.CreateMessageCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.String("campaign.category", campaign.Category),
		attribute.String("campaign.target", campaign.Target),
	)

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, campaign.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}
	u.tracingService.TraceEvent(span, "Creating message campaign")

	// 轉換 DTO 到 Entity 物件
	campaignEntity := &entity.MessageCampaign{
		GlobalID:   uuid.NewString(),
		MerchantID: merchant.ID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = copier.Copy(campaignEntity, campaign)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("copy campaign failed: %w", err)
	}

	// 使用領域方法驗證通知類型
	if err := campaignEntity.ValidateNotificationTypes(true); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("invalid notification type: %w", err)
	}

	// 使用領域方法處理不同的目標類型
	var targetIds []uint64
	switch campaign.Target {
	case consts.TargetPlayer:
		if len(campaign.TargetDetail) > 0 {
			// 驗證玩家帳號存在性並獲取player IDs
			playerIDs, err := u.validatePlayerAccounts(ctx, campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate player accounts: %w", err)
			}
			targetIds = playerIDs // 儲存以供優化使用
			if err := campaignEntity.SetPlayerTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set player target detail: %w", err)
			}
		}
	case consts.TargetLevel:
		if len(campaign.TargetDetail) > 0 {
			// 驗證等級ID存在性
			levelIDs, err := convertStringIDsToUint64(campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("convert level IDs: %w", err)
			}
			err = u.validateLevelIDs(ctx, levelIDs)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate level IDs: %w", err)
			}
			// 使用領域方法設置目標詳情
			if err := campaignEntity.SetLevelTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set level target detail: %w", err)
			}
			targetIds = levelIDs
		}
	case consts.TargetTag:
		if len(campaign.TargetDetail) > 0 {
			// 驗證標籤ID存在性
			tagIDs, err := convertStringIDsToUint64(campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("convert tag IDs: %w", err)
			}
			err = u.validateTagIDs(ctx, tagIDs)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate tag IDs: %w", err)
			}
			// 使用領域方法設置目標詳情
			if err := campaignEntity.SetTagTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set tag target detail: %w", err)
			}
			targetIds = tagIDs
		}
	}

	if err = u.campaignRepo.Create(ctx, campaignEntity); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("create campaign: %w", err)
	}

	err = u.createCampaignTargets(ctx, campaignEntity, targetIds)
	if err != nil {
		u.logger.WarnLog("Failed to create campaign targets (dual-write)",
			u.logger.UInt64("campaign_id", campaignEntity.ID),
			u.logger.Error("err", err))
	}

	u.tracingService.RecordSpanAttributes(
		span,
		attribute.Int64("campaign.id", int64(campaignEntity.ID)),
	)
	u.tracingService.TraceEvent(span, "Message campaign created successfully")

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
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.UpdateMessageCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("campaign.global_id", campaign.GlobalID),
		attribute.String("campaign.title", campaign.Title),
	)

	u.tracingService.TraceEvent(span, "Updating message campaign")

	// 檢查活動是否存在
	existing, err := u.campaignRepo.FindByGlobalID(ctx, campaign.GlobalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	// 轉換 DTO 到 Entity 物件
	campaignEntity := &entity.MessageCampaign{
		ID:          existing.ID,
		MerchantID:  existing.MerchantID,
		GlobalID:    existing.GlobalID,
		CreatedAt:   existing.CreatedAt,
		CreatedBy:   existing.CreatedBy,
		Status:      campaign.Status,
		TriggerType: "success",
		UpdatedAt:   time.Now(),
		UpdatedBy:   &campaign.UpdatedBy,
	}

	// 先複製可更新的字段
	err = copier.Copy(campaignEntity, campaign)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("copy campaign failed: %w", err)
	}

	// 處理 NotificationTypes
	if campaign.NotificationTypes != nil {
		campaignEntity.NotificationTypes = *campaign.NotificationTypes
	} else {
		campaignEntity.NotificationTypes = existing.NotificationTypes // 保持原有值
	}

	// 使用領域方法驗證通知類型
	if err := campaignEntity.ValidateNotificationTypes(false); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("invalid notification type: %w", err)
	}

	// 使用領域方法處理不同的目標類型
	var targetIds []uint64 // 用於優化的player IDs
	switch campaign.Target {
	case consts.TargetPlayer:
		if len(campaign.TargetDetail) > 0 {
			// 驗證玩家帳號存在性並獲取player IDs
			playerIDs, err := u.validatePlayerAccounts(ctx, campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate player accounts: %w", err)
			}
			if err := campaignEntity.SetPlayerTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set player target detail: %w", err)
			}
			targetIds = playerIDs
		}
	case consts.TargetLevel:
		if len(campaign.TargetDetail) > 0 {
			// 驗證等級ID存在性
			levelIDs, err := convertStringIDsToUint64(campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("convert level IDs: %w", err)
			}
			err = u.validateLevelIDs(ctx, levelIDs)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate level IDs: %w", err)
			}
			// 使用領域方法設置目標詳情
			if err := campaignEntity.SetLevelTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set level target detail: %w", err)
			}
			targetIds = levelIDs
		}
	case consts.TargetTag:
		if len(campaign.TargetDetail) > 0 {
			// 驗證標籤ID存在性
			tagIDs, err := convertStringIDsToUint64(campaign.TargetDetail)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("convert tag IDs: %w", err)
			}
			err = u.validateTagIDs(ctx, tagIDs)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("validate tag IDs: %w", err)
			}
			// 使用領域方法設置目標詳情
			if err := campaignEntity.SetTagTargetDetail(campaign.TargetDetail); err != nil {
				u.tracingService.RecordSpanError(span, err)
				return fmt.Errorf("set tag target detail: %w", err)
			}
			targetIds = tagIDs
		}
	}

	if err = u.campaignRepo.Update(ctx, campaignEntity); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update campaign: %w", err)
	}

	err = u.updateCampaignTargets(ctx, campaignEntity, targetIds)
	if err != nil {
		u.logger.WarnLog("Failed to update campaign targets (dual-write)",
			u.logger.UInt64("campaign_id", campaignEntity.ID),
			u.logger.Error("err", err))
	}

	u.tracingService.TraceEvent(span, "Message campaign updated successfully")

	u.logger.InfoWithContext(ctx, "Message campaign updated",
		u.logger.String("campaign_global_id", campaign.GlobalID),
		u.logger.String("title", campaign.Title),
		u.logger.String("updated_by", *campaignEntity.UpdatedBy))
	return nil
}

// DeleteMessageCampaign 刪除會員訊息活動
func (u *MessageUseCase) DeleteMessageCampaign(ctx context.Context, globalID string) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.DeleteMessageCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("campaign.global_id", globalID))

	u.tracingService.TraceEvent(span, "Deleting message campaign")

	// 先查找記錄以獲取ID（用於後續更新狀態）
	existing, err := u.campaignRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	if err = u.campaignRepo.Delete(ctx, existing.ID); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("delete campaign: %w", err)
	}

	// 雙寫：清理關聯記錄
	if err = u.campaignTargetRepo.DeleteByCampaignID(ctx, existing.ID); err != nil {
		u.logger.WarnLog("Failed to delete campaign targets (dual-write)",
			u.logger.UInt64("campaign_id", existing.ID),
			u.logger.Error("err", err))
		// 如果關聯記錄刪除失敗，記錄警告但不影響主流程
	}
	setColumn := map[string]interface{}{
		"status": consts.MessageCampaignStatusCancelled,
	}

	if err = u.campaignRepo.UpdateFields(ctx, existing.ID, setColumn, true); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update campaign status: %w", err)
	}

	u.tracingService.TraceEvent(span, "Message campaign deleted successfully")
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
) (*dto.MessageCampaignResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.GetMessageCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("campaign.global_id", globalID))

	campaign, err := u.campaignRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find campaign: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span, attribute.String("campaign.title", campaign.Title))

	// 轉換為 DTO
	response := &dto.MessageCampaignResponse{
		ID:                campaign.ID,
		GlobalID:          campaign.GlobalID,
		MerchantID:        campaign.MerchantID,
		Title:             campaign.Title,
		Content:           campaign.Content,
		AppContent:        campaign.AppContent,
		NotificationTypes: campaign.NotificationTypes,
		TargetType:        campaign.Target,
		TargetDetail: func() []string {
			if campaign.TargetDetail == nil {
				return []string{}
			}

			switch campaign.Target {
			case consts.TargetPlayer:
				// 對於 player，直接解析為 []string
				var playerAccounts []string
				if err := json.Unmarshal([]byte(*campaign.TargetDetail), &playerAccounts); err != nil {
					u.logger.WarnLog("Failed to unmarshal player target detail",
						u.logger.String("target_detail", *campaign.TargetDetail),
						u.logger.Error("err", err))
					return []string{}
				}
				return playerAccounts

			case consts.TargetLevel:
				// 對於 level，透過 ID 查詢並回傳 level name
				var levelIDStrings []string
				if err := json.Unmarshal([]byte(*campaign.TargetDetail), &levelIDStrings); err != nil {
					u.logger.WarnLog("Failed to unmarshal level target detail",
						u.logger.String("target_detail", *campaign.TargetDetail),
						u.logger.Error("err", err))
					return []string{}
				}

				levelIDs, err := convertStringIDsToUint64(levelIDStrings)
				if err != nil {
					u.logger.WarnLog("Failed to convert level IDs",
						u.logger.Any("level_id_strings", levelIDStrings),
						u.logger.Error("err", err))
					return []string{}
				}

				levels, err := u.levelRepo.FindByIDs(ctx, levelIDs)
				if err != nil {
					u.logger.WarnLog("Failed to find levels by IDs",
						u.logger.Any("level_ids", levelIDs),
						u.logger.Error("err", err))
					return []string{}
				}

				levelNames := make([]string, len(levels))
				for i, level := range levels {
					levelNames[i] = level.Name
				}
				return levelNames

			case consts.TargetTag:
				// 對於 tag，透過 ID 查詢並回傳 tag name
				var tagIDStrings []string
				if err := json.Unmarshal([]byte(*campaign.TargetDetail), &tagIDStrings); err != nil {
					u.logger.WarnLog("Failed to unmarshal tag target detail",
						u.logger.String("target_detail", *campaign.TargetDetail),
						u.logger.Error("err", err))
					return []string{}
				}

				tagIDs, err := convertStringIDsToUint64(tagIDStrings)
				if err != nil {
					u.logger.WarnLog("Failed to convert tag IDs",
						u.logger.Any("tag_id_strings", tagIDStrings),
						u.logger.Error("err", err))
					return []string{}
				}

				tags, err := u.tagRepo.FindByIDs(ctx, tagIDs)
				if err != nil {
					u.logger.WarnLog("Failed to find tags by IDs",
						u.logger.Any("tag_ids", tagIDs),
						u.logger.Error("err", err))
					return []string{}
				}

				tagNames := make([]string, len(tags))
				for i, tag := range tags {
					tagNames[i] = tag.Name
				}
				return tagNames

			default:
				// 其他類型無需特殊處理
				return []string{}
			}
		}(),
		ScheduledAt: campaign.SendStartTime,
		Status:      campaign.Status,
		SentCount:   int(campaign.RealSentCount),
		ReadCount:   0,
		IsScheduled: campaign.IsScheduled(),
		ProcessedAt: campaign.SendEndTime,
		CreatedAt:   campaign.CreatedAt,
		UpdatedAt:   campaign.UpdatedAt,
	}

	return response, nil
}

// ListMessageCampaigns 列出會員訊息活動
func (u *MessageUseCase) ListMessageCampaigns(
	ctx context.Context,
	req *dto.ListMessageCampaignsRequest,
) (*dto.MessageCampaignListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.ListMessageCampaigns")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("page", req.Page),
		attribute.Int("page_size", req.PageSize),
	)

	// 轉換 DTO 到 Query 物件
	query := &dto.MessageCampaignsQuery{
		IncludeDeleted: true,
	}
	err := copier.Copy(query, req)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("copy query failed: %w", err)
	}

	campaigns, total, err := u.campaignRepo.FindAllWithOptions(ctx, query)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("list campaigns with options: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("total_campaigns", total),
		attribute.Int("returned_campaigns", len(campaigns)),
	)

	// 轉換為 DTO
	campaignResponses := make([]dto.MessageCampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		campaignResponses[i] = dto.MessageCampaignResponse{
			ID:                campaign.ID,
			GlobalID:          campaign.GlobalID,
			MerchantID:        campaign.MerchantID,
			Title:             campaign.Title,
			Content:           campaign.Content,
			AppContent:        campaign.AppContent,
			NotificationTypes: campaign.NotificationTypes,
			TargetType:        campaign.Target,
			ScheduledAt:       campaign.SendStartTime,
			Status:            campaign.Status,
			SentCount:         int(campaign.RealSentCount),
			IsScheduled:       campaign.IsScheduled(),
			ProcessedAt:       campaign.SendEndTime,
			CreatedAt:         campaign.CreatedAt,
			UpdatedAt:         campaign.UpdatedAt,
		}
	}

	response := &dto.MessageCampaignListResponse{
		Campaigns: campaignResponses,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
	return response, nil
}

// GetPlayerMessages 獲取玩家訊息列表
func (u *MessageUseCase) GetPlayerMessages(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) (*dto.MessageListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.GetPlayerMessages")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("global_player_id", globalPlayerID),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	// 先檢查 player 是否存在
	_, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player by global ID: %w", err)
	}

	// 獲取統計資訊
	stats, err := u.playerMessageRepo.GetPlayerMessageStats(ctx, globalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get player message stats: %w", err)
	}

	// 使用 JOIN 查詢直接獲取訊息列表和活動資訊
	messageAggregates, total, err := u.playerMessageRepo.FindByPlayerIDWithCampaign(
		ctx,
		globalPlayerID,
		page,
		pageSize,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player messages with campaign: %w", err)
	}

	// 轉換為摘要格式
	summaries := make([]dto.MessageSummary, len(messageAggregates))
	for i, aggregate := range messageAggregates {
		summaries[i] = dto.MessageSummary{
			ID:        aggregate.ID,
			Title:     aggregate.CampaignTitle,
			Summary:   generateSummary(aggregate.CampaignContent),
			IsRead:    aggregate.IsRead,
			CreatedAt: aggregate.CreatedAt,
		}
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("stats.total_count", stats.TotalCount),
		attribute.Int("stats.read_count", stats.ReadCount),
		attribute.Int("stats.unread_count", stats.UnreadCount),
		attribute.Int("returned_messages", len(summaries)),
	)

	dtoStats := *stats

	return &dto.MessageListResponse{
		Stats:    dtoStats,
		Messages: summaries,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// MarkMessageAsRead 標記訊息為已讀
func (u *MessageUseCase) MarkMessageAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.MarkMessageAsRead")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("global_player_id", globalPlayerID),
		attribute.Int64("message.id", int64(messageID)),
	)

	u.tracingService.TraceEvent(span, "Marking message as read")

	if err := u.playerMessageRepo.MarkAsRead(ctx, globalPlayerID, messageID); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("mark message as read: %w", err)
	}

	u.tracingService.TraceEvent(span, "Message marked as read successfully")

	u.logger.InfoLog("Message marked as read",
		u.logger.String("global_player_id", globalPlayerID),
		u.logger.Int64("message_id", int64(messageID)))

	return nil
}

// processPlayerMessages 處理玩家訊息（檢查是否需要新增訊息）
func (u *MessageUseCase) processPlayerMessages(ctx context.Context, globalPlayerID string) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.processPlayerMessages")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(
		span,
		attribute.String("global_player_id", globalPlayerID),
	)

	// 獲取玩家資訊
	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 使用領域方法判斷玩家的活躍狀態
	focusType := player.DetermineFocusType()

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("player.focus_type", focusType),
		attribute.String("player.account", player.Account),
	)

	// 獲取符合條件的活動
	campaigns, err := u.campaignRepo.FindActiveByFocus(ctx, focusType)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find active campaigns: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span, attribute.Int("matching_campaigns", len(campaigns)))

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
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("create batch messages: %w", err)
		}

		u.tracingService.TraceEvent(span, "New messages created",
			attribute.Int("new_messages_count", len(newMessages)))

		u.logger.InfoLog("New messages created for player",
			u.logger.String("global_player_id", globalPlayerID),
			u.logger.Int("new_messages_count", len(newMessages)))
	}

	return nil
}

// generateSummary 生成內容摘要：最多只顯示 300個字符，避免過長
func generateSummary(content string) string {
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	maxLength := 300
	runes := []rune(content)
	if len(runes) <= maxLength {
		return content
	}
	return string(runes[:maxLength]) + "..."
}

// GetMerchantAutoSettings 獲取商戶自動設定
func (u *MessageUseCase) GetMerchantAutoSettings(
	ctx context.Context,
	globalMerchantID string,
) (*dto.MerchantAutoSettingsResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.GetMerchantAutoSettings")
	defer u.tracingService.SpanEnd(span)

	// 獲取商戶資訊
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 查找自動設定
	campaigns, err := u.campaignRepo.FindAutoSettingsByMerchantID(ctx, merchant.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
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
	ctx, span := u.tracingService.StartSpan(
		ctx,
		"MessageUseCase.CreateOrUpdateMerchantAutoSettings",
	)
	defer u.tracingService.SpanEnd(span)

	// 獲取商戶資訊
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, req.GlobalMerchantID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
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
		u.tracingService.RecordSpanError(span, err)
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

// validateLevelIDs 驗證等級ID是否存在
func (u *MessageUseCase) validateLevelIDs(ctx context.Context, levelIDs []uint64) error {
	// 直接根據ID獲取等級
	levels, err := u.levelRepo.FindByIDs(ctx, levelIDs)
	if err != nil {
		return fmt.Errorf("find levels by IDs: %w", err)
	}

	// 檢查是否所有ID都找到了對應的等級
	if len(levels) != len(levelIDs) {
		return fmt.Errorf(
			"some level IDs do not exist: expected %d, found %d",
			len(levelIDs),
			len(levels),
		)
	}

	return nil
}

// validateTagIDs 驗證標籤ID是否存在
func (u *MessageUseCase) validateTagIDs(ctx context.Context, tagIDs []uint64) error {
	// 直接根據ID獲取標籤
	tags, err := u.tagRepo.FindByIDs(ctx, tagIDs)
	if err != nil {
		return fmt.Errorf("find tags by IDs: %w", err)
	}

	// 檢查是否所有ID都找到了對應的標籤
	if len(tags) != len(tagIDs) {
		return fmt.Errorf(
			"some tag IDs do not exist: expected %d, found %d",
			len(tagIDs),
			len(tags),
		)
	}

	return nil
}

// convertStringIDsToUint64 將字符串ID數組轉換為uint64數組
func convertStringIDsToUint64(stringIDs []string) ([]uint64, error) {
	if len(stringIDs) == 0 {
		return nil, nil
	}

	ids := make([]uint64, len(stringIDs))
	for i, strID := range stringIDs {
		id, err := strconv.ParseUint(strID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format '%s': %w", strID, err)
		}
		ids[i] = id
	}
	return ids, nil
}

// createCampaignTargets 創建活動目標關聯記錄（雙寫功能）
func (u *MessageUseCase) createCampaignTargets(
	ctx context.Context,
	campaign *entity.MessageCampaign,
	targetIds []uint64,
) error {
	targets, err := u.buildCampaignTargets(campaign.Target, targetIds, campaign.ID)
	if err != nil {
		return fmt.Errorf("build campaign targets: %w", err)
	}

	if len(targets) == 0 {
		return nil
	}

	if err := u.campaignTargetRepo.CreateBatch(ctx, targets); err != nil {
		return fmt.Errorf("create campaign targets: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Created campaign targets for dual-write",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("target_type", campaign.Target),
		u.logger.Int("targets_count", len(targets)))

	return nil
}

// buildCampaignTargets 根據目標類型和詳情構建關聯記錄
func (u *MessageUseCase) buildCampaignTargets(
	targetType string,
	targetIds []uint64,
	campaignID uint64,
) ([]*entity.CampaignTarget, error) {
	var campaignTargets []*entity.CampaignTarget

	if len(targetIds) == 0 {
		return nil, fmt.Errorf("targetIds is required for campaign target")
	}

	for _, targetId := range targetIds {
		campaignTargets = append(campaignTargets, &entity.CampaignTarget{
			CampaignID: campaignID,
			TargetType: targetType,
			TargetID:   targetId,
		})
	}

	return campaignTargets, nil
}

// updateCampaignTargets 更新活動目標關聯記錄（雙寫功能）
func (u *MessageUseCase) updateCampaignTargets(
	ctx context.Context,
	campaign *entity.MessageCampaign,
	targetIds []uint64,
) error {
	// 先刪除現有的關聯記錄
	if err := u.campaignTargetRepo.DeleteByCampaignID(ctx, campaign.ID); err != nil {
		return fmt.Errorf("delete existing campaign targets: %w", err)
	}

	// 重新創建關聯記錄
	targets, err := u.buildCampaignTargets(campaign.Target, targetIds, campaign.ID)
	if err != nil {
		return fmt.Errorf("build campaign targets: %w", err)
	}

	if len(targets) == 0 {
		return nil
	}

	if err = u.campaignTargetRepo.CreateBatch(ctx, targets); err != nil {
		return fmt.Errorf("create campaign targets: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Updated campaign targets for dual-write",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("target_type", campaign.Target),
		u.logger.Int("targets_count", len(targets)))

	return nil
}

// ProcessPlayer 處理玩家重新上線時同步符合條件的message_campaign，使用campaign_targets表進行高效能查詢
func (u *MessageUseCase) ProcessPlayer(ctx context.Context, globalPlayerID string) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.ProcessPlayer")
	defer u.tracingService.SpanEnd(span)

	// 獲取玩家資訊
	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 獲取玩家標籤IDs
	playerTagIDs, err := u.playerRepo.GetPlayerTagIDs(ctx, player.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("get player tag IDs: %w", err)
	}

	// 一次性查詢所有符合條件的 campaigns
	eligibleCampaignIDs, err := u.campaignTargetRepo.FindCampaignIDsByPlayerCriteria(ctx,
		player.MerchantID,
		player.ID,
		player.LevelID,
		playerTagIDs,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find eligible campaigns: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("eligible_campaigns", len(eligibleCampaignIDs)))

	// 批量處理 player_message 創建
	return u.processCampaignMessages(ctx, player, eligibleCampaignIDs)
}

// processCampaignMessages 批量處理campaign訊息創建
func (u *MessageUseCase) processCampaignMessages(
	ctx context.Context,
	player *entity.Player,
	campaignIDs []uint64,
) error {
	if len(campaignIDs) == 0 {
		return nil
	}

	// 檢查已存在的訊息
	existingMessageCampaignIDs, err := u.playerMessageRepo.FindExistingCampaignIDs(
		ctx, player.GlobalPlayerID, campaignIDs,
	)
	if err != nil {
		return fmt.Errorf("find existing messages: %w", err)
	}

	// 計算需要創建的新訊息
	newCampaignIDs := make([]uint64, 0, len(campaignIDs))
	existingSet := make(map[uint64]bool)
	for _, id := range existingMessageCampaignIDs {
		existingSet[id] = true
	}

	for _, id := range campaignIDs {
		if !existingSet[id] {
			newCampaignIDs = append(newCampaignIDs, id)
		}
	}

	if len(newCampaignIDs) == 0 {
		return nil
	}

	// 批量創建新訊息
	newMessages := make([]*entity.PlayerMessage, len(newCampaignIDs))
	for i, campaignID := range newCampaignIDs {
		newMessages[i] = &entity.PlayerMessage{
			GlobalPlayerID: player.GlobalPlayerID,
			PlayerID:       player.ID,
			CampaignID:     campaignID,
			IsRead:         false,
			CreatedAt:      time.Now(),
		}
	}

	if err = u.playerMessageRepo.CreateBatch(ctx, newMessages); err != nil {
		return fmt.Errorf("create batch messages: %w", err)
	}

	u.logger.InfoWithContext(ctx, "New messages created for returning player",
		u.logger.String("global_player_id", player.GlobalPlayerID),
		u.logger.Int("new_messages_count", len(newMessages)))

	return nil
}

// validatePlayerAccounts 驗證玩家帳號是否存在並回傳存在的player IDs
func (u *MessageUseCase) validatePlayerAccounts(
	ctx context.Context,
	accounts []string,
) ([]uint64, error) {
	if len(accounts) == 0 {
		return []uint64{}, nil
	}

	// 批量查詢所有帳號
	foundPlayers, err := u.playerRepo.FindByAccounts(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to query player accounts: %w", err)
	}

	// 建立已找到的帳號集合
	foundAccountsSet := make(map[string]bool)
	playerIDs := make([]uint64, 0, len(foundPlayers))
	for _, player := range foundPlayers {
		foundAccountsSet[player.Account] = true
		playerIDs = append(playerIDs, player.ID)
	}

	// 檢查哪些帳號不存在
	var invalidAccounts []string
	for _, account := range accounts {
		if !foundAccountsSet[account] {
			invalidAccounts = append(invalidAccounts, account)
		}
	}

	if len(invalidAccounts) > 0 {
		return nil, fmt.Errorf("invalid player accounts: %v", invalidAccounts)
	}

	return playerIDs, nil
}

// SendAutoNotification 發送系統自動推播訊息給特定玩家
func (u *MessageUseCase) SendAutoNotification(
	ctx context.Context,
	req *dto.SendAutoNotificationRequest,
) (*dto.SendAutoNotificationResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.SendAutoNotification")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("global_player_id", req.GlobalPlayerID),
		attribute.String("category", req.Category),
		attribute.String("item", req.Item),
		attribute.String("trigger_type", req.TriggerType),
	)

	// 1. 根據 global_player_id 查找玩家
	player, err := u.playerRepo.FindByGlobalID(ctx, req.GlobalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return &dto.SendAutoNotificationResponse{
			PlayerID:    req.GlobalPlayerID,
			Category:    req.Category,
			Item:        req.Item,
			TriggerType: req.TriggerType,
			Status:      "failed",
			Message:     "player not found",
		}, nil
	}

	// 2. 根據 merchant_id、category、item、trigger_type、auto_send = true 查找 message_campaign
	campaign, err := u.campaignRepo.FindAutoSettingByCategoryItemTrigger(
		ctx,
		player.MerchantID,
		req.Category,
		req.Item,
		req.TriggerType,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return &dto.SendAutoNotificationResponse{
			PlayerID:    req.GlobalPlayerID,
			Category:    req.Category,
			Item:        req.Item,
			TriggerType: req.TriggerType,
			Status:      "not_found",
			Message:     "no matching auto notification setting found",
		}, nil
	}

	// 3. 根據 notification_type 決定發送渠道，使用統一的業務邏輯處理
	var sentChannels []string

	// 將 notification_types 轉換為業務邏輯類型
	notificationType, err := consts.NewNotificationType(campaign.NotificationTypes)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return &dto.SendAutoNotificationResponse{
			PlayerID:    req.GlobalPlayerID,
			Category:    req.Category,
			Item:        req.Item,
			TriggerType: req.TriggerType,
			Status:      "failed",
			Message:     "invalid notification type configuration",
		}, nil
	}

	// 檢查是否發送站內信
	if notificationType.HasInApp() {
		// 創建玩家訊息記錄，使用時間窗口防重複（5分鐘）
		playerMessage := &entity.PlayerMessage{
			PlayerID:       player.ID,
			GlobalPlayerID: req.GlobalPlayerID,
			CampaignID:     campaign.ID,
			IsRead:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err = u.playerMessageRepo.CreateAutoNotification(ctx, playerMessage, 5); err != nil {
			// 檢查是否為時間窗口重複錯誤
			if strings.Contains(err.Error(), "duplicate auto notification within") {
				u.logger.WarnWithContext(ctx, "Auto notification skipped due to time window",
					u.logger.String("global_player_id", req.GlobalPlayerID),
					u.logger.UInt64("campaign_id", campaign.ID),
					u.logger.Error("error", err),
				)
				return &dto.SendAutoNotificationResponse{
					PlayerID:    req.GlobalPlayerID,
					Category:    req.Category,
					Item:        req.Item,
					TriggerType: req.TriggerType,
					Status:      "skipped",
					Message:     "duplicate notification within time window",
				}, nil
			}

			u.logger.ErrorWithContext(ctx, "Failed to create player message",
				u.logger.Error("error", err),
				u.logger.String("global_player_id", req.GlobalPlayerID),
				u.logger.UInt64("campaign_id", campaign.ID),
			)
			// 其他創建失敗錯誤
			u.tracingService.RecordSpanError(span, err)
			return &dto.SendAutoNotificationResponse{
				PlayerID:    req.GlobalPlayerID,
				Category:    req.Category,
				Item:        req.Item,
				TriggerType: req.TriggerType,
				Status:      "failed",
				Message:     "failed to create in-app message",
			}, nil
		}

		sentChannels = append(sentChannels, "in_app")
		u.logger.InfoWithContext(ctx, "Auto notification in-app message sent",
			u.logger.String("global_player_id", req.GlobalPlayerID),
			u.logger.String("category", req.Category),
			u.logger.String("item", req.Item),
			u.logger.String("trigger_type", req.TriggerType),
		)
	}

	// 檢查是否發送App推播
	if notificationType.HasAppPush() && campaign.AppContent != nil {
		// 獲取商戶推播API金鑰
		pushApiKey, err := u.pushApiKeyRepo.FindByMerchantID(ctx, player.MerchantID)
		if err != nil {
			u.logger.WarnWithContext(ctx, "Push API key not found for merchant",
				u.logger.UInt64("merchant_id", player.MerchantID),
				u.logger.Error("error", err),
			)
		} else {
			// 發送App推播
			pushReq := &service.PushNotificationRequest{
				Accounts: []string{player.Account},
				MsgTitle: campaign.Title,
				MsgBody:  *campaign.AppContent,
			}

			_, err = u.pushService.SendPushNotification(ctx, pushApiKey.Key, pushReq)
			if err != nil {
				u.logger.ErrorWithContext(ctx, "Failed to send push notification",
					u.logger.Error("error", err),
					u.logger.String("global_player_id", req.GlobalPlayerID),
					u.logger.UInt64("campaign_id", campaign.ID),
				)
			} else {
				sentChannels = append(sentChannels, "push")
				u.logger.InfoWithContext(ctx, "Auto notification push message sent",
					u.logger.String("global_player_id", req.GlobalPlayerID),
					u.logger.String("category", req.Category),
					u.logger.String("item", req.Item),
					u.logger.String("trigger_type", req.TriggerType),
				)
			}
		}
	}

	// 更新發送計數
	if len(sentChannels) > 0 {
		if err := u.campaignRepo.UpdateSentCount(ctx, campaign.ID, campaign.RealSentCount+1); err != nil {
			u.logger.WarnWithContext(ctx, "Failed to update sent count",
				u.logger.Error("error", err),
				u.logger.UInt64("campaign_id", campaign.ID),
			)
		}
	}

	return &dto.SendAutoNotificationResponse{
		PlayerID:     req.GlobalPlayerID,
		Category:     req.Category,
		Item:         req.Item,
		TriggerType:  req.TriggerType,
		Status:       "sent",
		SentChannels: sentChannels,
		Message:      fmt.Sprintf("notification sent via %v", sentChannels),
	}, nil
}
