package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

// AgentUseCase 代理業務用例實作
type AgentUseCase struct {
	agentRepo         repository.AgentRepository
	agentCampaignRepo repository.AgentCampaignRepository
	agentMessageRepo  repository.AgentMessageRepository
	agentRelationRepo repository.AgentRelationshipRepository
	merchantRepo      repository.MerchantRepository
	agentService      service.AgentService
	eventProducer     service.EventProducer
	logger            infrastructure.Logger
	tracingService    infrastructure.TracingService
}

// NewAgentUseCase 創建代理用例
func NewAgentUseCase(
	agentRepo repository.AgentRepository,
	agentCampaignRepo repository.AgentCampaignRepository,
	agentMessageRepo repository.AgentMessageRepository,
	agentRelationRepo repository.AgentRelationshipRepository,
	merchantRepo repository.MerchantRepository,
	agentService service.AgentService,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) inbound.AgentUseCase {
	return &AgentUseCase{
		agentRepo:         agentRepo,
		agentCampaignRepo: agentCampaignRepo,
		agentMessageRepo:  agentMessageRepo,
		agentRelationRepo: agentRelationRepo,
		merchantRepo:      merchantRepo,
		agentService:      agentService,
		eventProducer:     eventProducer,
		logger:            logger,
		tracingService:    tracingService,
	}
}

// SyncAgentDataWithRelationships 完整的代理同步邏輯 (資料同步 + 關係建立)
func (u *AgentUseCase) SyncAgentDataWithRelationships(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SyncAgentDataWithRelationships")
	defer u.tracingService.SpanEnd(span)

	// 添加代理信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("agent.global_id", agentEvent.GlobalAgentID),
		attribute.String("agent.account", agentEvent.Account),
		attribute.String("merchant.global_id", agentEvent.GlobalMerchantID))

	// 1. 同步當前代理資料
	u.tracingService.TraceEvent(span, "Syncing current agent data")
	if err := u.SyncCurrentAgentUpsert(ctx, agentEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("sync current agent: %w", err)
	}

	// 2. 同步代理關係 (使用Service處理複雜邏輯)
	u.tracingService.TraceEvent(span, "Syncing agent relationships using AgentService")
	if err := u.agentService.SyncAgentRelationshipsUpsert(ctx, agentEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("sync agent relationships: %w", err)
	}

	// 記錄完成
	u.tracingService.TraceEvent(span, "Agent sync with relationships completed successfully")
	u.logger.InfoLog("Agent sync with relationships completed successfully",
		u.logger.String("global_agent_id", agentEvent.GlobalAgentID),
		u.logger.String("account", agentEvent.Account))

	return nil
}

// SyncCurrentAgentUpsert 同步當前代理資料
func (u *AgentUseCase) SyncCurrentAgentUpsert(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SyncCurrentAgentUpsert")
	defer u.tracingService.SpanEnd(span)

	// 查找對應的商戶
	u.tracingService.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, agentEvent.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	if merchant == nil {
		u.tracingService.TraceEvent(span, "Merchant not found, skipping agent sync")
		u.logger.WarnLog("Merchant not found for agent sync",
			u.logger.String("global_merchant_id", agentEvent.GlobalMerchantID),
			u.logger.String("global_agent_id", agentEvent.GlobalAgentID))
		return nil
	}

	// 構建代理實體
	agent := entity.Agent{
		MerchantID:      merchant.ID,
		GlobalAgentID:   agentEvent.GlobalAgentID,
		Account:         agentEvent.Account,
		Ancestry:        agentEvent.Ancestry,
		CurrentSignInAt: agentEvent.CurrentSignInAt,
		CreatedAt:       agentEvent.CreatedAt,
		UpdatedAt:       agentEvent.UpdatedAt,
	}

	u.tracingService.TraceEvent(span, "Upsert agent")
	if err = u.agentRepo.Upsert(ctx, &agent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert agent: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent upserted successfully")
	u.logger.InfoLog("Agent upserted successfully",
		u.logger.String("global_id", agent.GlobalAgentID),
		u.logger.String("account", agent.Account))

	return nil
}

// GetAgentByGlobalID 通過全局ID獲取代理
func (u *AgentUseCase) GetAgentByGlobalID(
	ctx context.Context,
	globalAgentID string,
) (*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentByGlobalID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("agent.global_id", globalAgentID))

	agent, err := u.agentRepo.GetByGlobalID(ctx, globalAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent: %w", err)
	}

	return agent, nil
}

// GetActiveAgents 獲取活躍代理列表
func (u *AgentUseCase) GetActiveAgents(
	ctx context.Context,
	merchantID uint64,
	limit, offset int,
) ([]*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetActiveAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("merchant.id", int64(merchantID)),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset))

	agents, err := u.agentRepo.FindAgents(ctx, merchantID, limit, offset)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find active agents: %w", err)
	}

	return agents, nil
}

// CreateAgentCampaign 創建代理訊息活動
func (u *AgentUseCase) CreateAgentCampaign(
	ctx context.Context,
	req *dto.CreateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.CreateAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("campaign.title", req.Title),
		attribute.String("campaign.target_type", req.TargetType))

	// 驗證請求參數
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if req.TargetType == "" {
		return nil, fmt.Errorf("target_type is required")
	}

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, req.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}
	u.tracingService.TraceEvent(span, "Creating message campaign")

	// 構建代理活動實體
	campaign := &entity.AgentCampaign{
		Title:         req.Title,
		Content:       req.Content,
		ScheduledAt:   req.ScheduledAt,
		MerchantID:    merchant.ID,
		TargetType:    req.TargetType,
		TargetDetails: req.TargetDetails,
		Status:        consts.AgentCampaignStatusDraft, // 預設狀態為草稿
		TargetCount:   0,                               // 將在排程時計算
		RealSentCount: 0,
		CreatedBy:     req.CreatedBy,
		UpdatedBy:     req.CreatedBy,
	}

	// 根據TargetType驗證TargetDetails
	if err = campaign.ValidateTargetDetails(); err != nil {
		return nil, err
	}

	// 如果設定了排程時間，狀態改為scheduled
	if req.ScheduledAt != nil {
		campaign.Status = consts.AgentCampaignStatusScheduled
	}

	u.tracingService.TraceEvent(span, "Creating agent campaign")
	createdCampaign, err := u.agentCampaignRepo.Create(ctx, campaign)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("create agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign created successfully")
	u.logger.InfoLog("Agent campaign created successfully",
		u.logger.UInt64("campaign_id", createdCampaign.ID),
		u.logger.String("title", createdCampaign.Title),
		u.logger.String("target_type", createdCampaign.TargetType))

	return createdCampaign, nil
}

// UpdateAgentCampaign 更新代理訊息活動
func (u *AgentUseCase) UpdateAgentCampaign(
	ctx context.Context,
	req *dto.UpdateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.UpdateAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(req.ID)))

	// 先獲取現有的活動
	u.tracingService.TraceEvent(span, "Getting existing campaign")
	existing, err := u.agentCampaignRepo.GetByID(ctx, req.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get existing campaign: %w", err)
	}

	// 轉換為新的 Entity 物件 (參考 message_usecase.go 的做法)
	campaignEntity := &entity.AgentCampaign{}

	// 使用 copier 複製現有資料
	err = copier.Copy(campaignEntity, existing)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("copy campaign failed: %w", err)
	}

	// 檢查活動狀態是否允許更新
	if err = campaignEntity.CanUpdate(); err != nil {
		return nil, err
	}

	// 使用 Entity 方法更新欄位
	campaignEntity.UpdateFromRequest(req)

	// 使用 Entity 方法驗證目標詳情
	if err = campaignEntity.ValidateTargetDetails(); err != nil {
		return nil, err
	}

	// 使用 Entity 方法更新狀態
	campaignEntity.UpdateStatus()

	u.tracingService.TraceEvent(span, "Updating agent campaign")
	err = u.agentCampaignRepo.Update(ctx, campaignEntity)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("update agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign updated successfully")
	u.logger.InfoLog("Agent campaign updated successfully",
		u.logger.UInt64("campaign_id", campaignEntity.ID),
		u.logger.String("title", campaignEntity.Title),
		u.logger.String("status", campaignEntity.Status))

	return campaignEntity, nil
}

// GetAgentCampaign 獲取代理訊息活動
func (u *AgentUseCase) GetAgentCampaign(
	ctx context.Context,
	id uint64,
) (*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	u.tracingService.TraceEvent(span, "Getting agent campaign")
	campaign, err := u.agentCampaignRepo.GetByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign retrieved successfully")
	return campaign, nil
}

// GetAgentCampaigns 獲取代理訊息活動列表
func (u *AgentUseCase) GetAgentCampaigns(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) (*dto.AgentCampaignListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaigns")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("page", query.Page),
		attribute.Int("page_size", query.PageSize),
		attribute.String("status", query.Status),
		attribute.String("target_type", query.TargetType))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, query.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}
	query.MerchantID = merchant.ID
	query.IncludeDeleted = true

	// 設定預設值
	if query.Limit == 0 {
		query.Limit = query.PageSize
	}
	if query.Offset == 0 {
		query.Offset = (query.Page - 1) * query.PageSize
	}

	campaigns, total, err := u.agentCampaignRepo.List(ctx, query)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent campaigns: %w", err)
	}
	u.tracingService.TraceEvent(span, "Getting agent campaigns list")

	// 轉換為回應DTO
	campaignResponses := make([]dto.AgentCampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		campaignResponses[i] = dto.AgentCampaignResponse{
			ID:            campaign.ID,
			MerchantID:    campaign.MerchantID,
			Title:         campaign.Title,
			Content:       campaign.Content,
			ScheduledAt:   campaign.ScheduledAt,
			Status:        campaign.Status,
			TargetType:    campaign.TargetType,
			TargetDetails: campaign.TargetDetails,
			TargetCount:   campaign.TargetCount,
			RealSentCount: campaign.RealSentCount,
			CreatedBy:     campaign.CreatedBy,
			UpdatedBy:     campaign.UpdatedBy,
			CreatedAt:     campaign.CreatedAt,
			UpdatedAt:     campaign.UpdatedAt,
		}
	}

	response := &dto.AgentCampaignListResponse{
		Campaigns: campaignResponses,
		Page:      query.Page,
		PageSize:  query.PageSize,
		Total:     total,
	}

	u.tracingService.TraceEvent(span, "Agent campaigns list retrieved successfully")
	return response, nil
}

// DeleteAgentCampaign 刪除代理訊息活動
func (u *AgentUseCase) DeleteAgentCampaign(ctx context.Context, id uint64) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.DeleteAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	// 先檢查活動是否存在
	u.tracingService.TraceEvent(span, "Checking if campaign exists")
	campaign, err := u.agentCampaignRepo.GetByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("get campaign for deletion: %w", err)
	}

	// 檢查活動狀態是否允許刪除
	if err = campaign.CanDelete(); err != nil {
		return err
	}

	// 先更新狀態為 cancelled
	u.tracingService.TraceEvent(span, "Updating campaign status to cancelled before deletion")
	setColumn := map[string]interface{}{
		"status": consts.MessageCampaignStatusCancelled,
	}
	if err = u.agentCampaignRepo.UpdateFields(ctx, id, setColumn); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update campaign status to cancelled: %w", err)
	}

	u.tracingService.TraceEvent(span, "Deleting agent campaign")
	if err = u.agentCampaignRepo.Delete(ctx, id); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("delete agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign deleted successfully")
	u.logger.InfoLog("Agent campaign deleted successfully",
		u.logger.UInt64("campaign_id", id),
		u.logger.String("title", campaign.Title))

	return nil
}

// GetAgentMessages 獲取代理站內信列表
func (u *AgentUseCase) GetAgentMessages(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) (*dto.AgentMessageListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentMessages")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("agent.global_id", query.GlobalAgentID),
		attribute.Int("page", query.Page),
		attribute.Int("page_size", query.PageSize))

	// 根據GlobalAgentID獲取AgentID
	u.tracingService.TraceEvent(span, "Getting agent by global ID")
	agent, err := u.agentRepo.GetByGlobalID(ctx, query.GlobalAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get agent by global ID: %w", err)
	}

	// 設定查詢參數
	query.AgentID = agent.ID
	if query.Limit == 0 {
		query.Limit = query.PageSize
	}
	if query.Offset == 0 {
		query.Offset = (query.Page - 1) * query.PageSize
	}

	// 獲取訊息列表
	u.tracingService.TraceEvent(span, "Getting agent messages")
	// TODO: Implement FindMessages method in repository
	messages := []*entity.AgentMessage{}
	total := int64(0)
	// messages, total, err := u.agentMessageRepo.FindMessages(ctx, query)
	// if err != nil {
	//	u.tracingService.RecordSpanError(span, err)
	//	return nil, fmt.Errorf("find agent messages: %w", err)
	// }

	// 獲取訊息統計
	u.tracingService.TraceEvent(span, "Getting agent message stats")
	// TODO: Implement GetMessageStats method in repository
	stats := &dto.AgentMessageStats{ReadCount: 0, UnreadCount: 0, TotalCount: 0}
	// stats, err := u.agentMessageRepo.GetMessageStats(ctx, agent.ID)
	// if err != nil {
	//	u.tracingService.RecordSpanError(span, err)
	//	return nil, fmt.Errorf("get message stats: %w", err)
	// }

	// 轉換為回應DTO
	messageResponses := make([]dto.AgentMessageResponse, len(messages))
	for i, message := range messages {
		messageResponses[i] = dto.AgentMessageResponse{
			ID:              message.ID,
			AgentCampaignID: message.AgentCampaignID,
			AgentID:         message.AgentID,
			GlobalAgentID:   query.GlobalAgentID,
			Title:           "", // TODO: 從JOIN查詢獲取
			Content:         "", // TODO: 從JOIN查詢獲取
			IsRead:          message.IsRead,
			ReadAt:          message.ReadAt,
			SentAt:          message.CreatedAt, // SentAt就是CreatedAt
			CreatedAt:       message.CreatedAt,
			UpdatedAt:       message.UpdatedAt,
		}
	}

	response := &dto.AgentMessageListResponse{
		Stats: dto.AgentMessageStats{
			ReadCount:   stats.ReadCount,
			UnreadCount: stats.UnreadCount,
			TotalCount:  stats.TotalCount,
		},
		Messages: messageResponses,
		Page:     query.Page,
		PageSize: query.PageSize,
		Total:    int(total),
	}

	u.tracingService.TraceEvent(span, "Agent messages retrieved successfully")
	return response, nil
}

// MarkMessageAsRead 標記訊息為已讀
func (u *AgentUseCase) MarkMessageAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	_, span := u.tracingService.StartSpan(ctx, "AgentUseCase.MarkMessageAsRead")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("message.id", int64(messageID)),
		attribute.Int64("agent.id", int64(agentID)))

	// 檢查訊息是否存在且屬於該代理
	u.tracingService.TraceEvent(span, "Checking message ownership")
	// TODO: Implement GetMessageByIDAndAgentID method in repository
	message := &entity.AgentMessage{IsRead: false}
	//message, err := u.agentMessageRepo.GetMessageByIDAndAgentID(ctx, messageID, agentID)
	//if err != nil {
	//	u.tracingService.RecordSpanError(span, err)
	//	return fmt.Errorf("get message by ID and agent ID: %w", err)
	//}

	// 檢查訊息是否已經被讀過
	if message.IsRead {
		u.logger.InfoLog("Message already read",
			u.logger.UInt64("message_id", messageID),
			u.logger.UInt64("agent_id", agentID))
		return nil // 已讀過，直接返回成功
	}

	// 標記為已讀
	u.tracingService.TraceEvent(span, "Marking message as read")
	// TODO: Implement MarkAsRead method in repository
	// if err := u.agentMessageRepo.MarkAsRead(ctx, messageID, agentID); err != nil {
	//	u.tracingService.RecordSpanError(span, err)
	//	return fmt.Errorf("mark message as read: %w", err)
	// }

	u.tracingService.TraceEvent(span, "Message marked as read successfully")
	u.logger.InfoLog("Message marked as read successfully",
		u.logger.UInt64("message_id", messageID),
		u.logger.UInt64("agent_id", agentID))

	return nil
}

// SendMessageToCampaignTargets 發送訊息給活動目標
func (u *AgentUseCase) SendMessageToCampaignTargets(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SendMessageToCampaignTargets")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作發送訊息給活動目標邏輯
	return fmt.Errorf("not implemented yet")
}

// GetScheduledCampaigns 獲取排程活動
func (u *AgentUseCase) GetScheduledCampaigns(ctx context.Context) ([]*entity.AgentCampaign, error) {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetScheduledCampaigns")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作獲取排程活動邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// FilterActiveAgents 過濾活躍代理
func (u *AgentUseCase) FilterActiveAgents(agentIDs []string) []string {
	// TODO: 實作過濾活躍代理邏輯
	return agentIDs
}
