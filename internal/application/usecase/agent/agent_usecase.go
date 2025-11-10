package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	messages, total, err := u.agentMessageRepo.ListByAgent(ctx, query)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent messages: %w", err)
	}

	// 獲取訊息統計
	u.tracingService.TraceEvent(span, "Getting agent message stats")
	stats, err := u.agentMessageRepo.GetMessageStats(ctx, agent.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get message stats: %w", err)
	}

	// 轉換為回應DTO
	messageResponses := make([]dto.AgentMessageResponse, len(messages))
	for i, message := range messages {
		messageResponses[i] = dto.AgentMessageResponse{
			ID:              message.ID,
			AgentCampaignID: message.AgentCampaignID,
			AgentID:         message.AgentID,
			GlobalAgentID:   message.GlobalAgentID,   // 從JOIN查詢獲取
			Title:           message.CampaignTitle,   // 從JOIN查詢獲取
			Content:         message.CampaignContent, // 從JOIN查詢獲取
			IsRead:          message.IsRead,
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
		Total:    total,
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

	// 標記為已讀
	u.tracingService.TraceEvent(span, "Marking message as read")
	if err := u.agentMessageRepo.MarkAsRead(ctx, messageID, agentID); err != nil {
		u.tracingService.RecordSpanError(span, err)
		// 檢查是否是已讀或不存在的錯誤
		if err.Error() == "message not found or already read" {
			u.logger.InfoLog("Message not found or already read",
				u.logger.UInt64("message_id", messageID),
				u.logger.UInt64("agent_id", agentID))
			return fmt.Errorf("record not found or already read")
		}
		return fmt.Errorf("mark message as read: %w", err)
	}

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
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SendMessageToCampaignTargets")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaign.ID)))

	// 1. 解析目標代理
	targetAgents, err := u.resolveTargetAgentsByType(ctx, campaign)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, 0, fmt.Errorf("resolve target agents: %w", err)
	}

	// 2. 篩選活躍代理
	activeAgents, err := u.FilterActiveAgents(ctx, targetAgents)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, 0, fmt.Errorf("filter active agents: %w", err)
	}

	// 3. 批量創建訊息
	sentCount, err = u.createMessagesForAgents(ctx, campaign, activeAgents)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return len(targetAgents), 0, fmt.Errorf("create messages: %w", err)
	}

	u.tracingService.TraceEvent(span, "Messages sent successfully",
		attribute.Int("target_count", len(targetAgents)),
		attribute.Int("sent_count", sentCount))

	return len(targetAgents), sentCount, nil
}

// GetScheduledCampaigns 獲取到期的排程活動
func (u *AgentUseCase) GetScheduledCampaigns(ctx context.Context, currentTime time.Time) ([]*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetScheduledCampaigns")
	defer u.tracingService.SpanEnd(span)

	// 查詢條件：scheduled狀態且到期時間已到
	campaigns, err := u.agentCampaignRepo.GetScheduledCampaigns(ctx, currentTime)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get scheduled campaigns: %w", err)
	}

	u.tracingService.TraceEvent(span, "Found scheduled campaigns",
		attribute.Int("count", len(campaigns)))

	return campaigns, nil
}

// FilterActiveAgents 篩選活躍代理（1個月內登入）
func (u *AgentUseCase) FilterActiveAgents(ctx context.Context, globalAgentIDs []string) ([]*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.FilterActiveAgents")
	defer u.tracingService.SpanEnd(span)

	if len(globalAgentIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	// 批量查詢代理資訊
	agents, err := u.agentRepo.BatchGetAgentsByGlobalIDs(ctx, globalAgentIDs)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("batch get agents: %w", err)
	}

	// 篩選活躍代理（1個月內登入）
	threshold := time.Now().AddDate(0, -1, 0)
	var activeAgents []*entity.Agent

	for _, agent := range agents {
		if agent.CurrentSignInAt != nil && agent.CurrentSignInAt.After(threshold) {
			activeAgents = append(activeAgents, agent)
		}
	}

	u.tracingService.TraceEvent(span, "Active agents filtered",
		attribute.Int("total_agents", len(agents)),
		attribute.Int("active_agents", len(activeAgents)),
		attribute.String("threshold", threshold.Format(time.RFC3339)))

	return activeAgents, nil
}

// UpdateCampaignStatus 更新活動狀態
func (u *AgentUseCase) UpdateCampaignStatus(ctx context.Context, campaignID uint64, status string) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.UpdateCampaignStatus")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaignID)),
		attribute.String("status", status))

	setColumn := map[string]interface{}{
		"status": status,
	}

	if err := u.agentCampaignRepo.UpdateFields(ctx, campaignID, setColumn); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update campaign status: %w", err)
	}

	u.tracingService.TraceEvent(span, "Campaign status updated successfully")
	return nil
}

// CompleteCampaign 完成活動並更新統計
func (u *AgentUseCase) CompleteCampaign(ctx context.Context, campaignID uint64, targetCount, sentCount int) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.CompleteCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaignID)),
		attribute.Int("target_count", targetCount),
		attribute.Int("sent_count", sentCount))

	setColumn := map[string]interface{}{
		"status":          consts.AgentCampaignStatusSent,
		"target_count":    int64(targetCount),
		"real_sent_count": int64(sentCount),
	}

	if err := u.agentCampaignRepo.UpdateFields(ctx, campaignID, setColumn); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("complete campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Campaign completed successfully")
	return nil
}

// resolveTargetAgentsByType 根據目標類型解析目標代理
func (u *AgentUseCase) resolveTargetAgentsByType(ctx context.Context, campaign *entity.AgentCampaign) ([]string, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.resolveTargetAgentsByType")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("target_type", campaign.TargetType),
		attribute.Int("target_details_count", len(campaign.TargetDetails)))

	switch campaign.TargetType {
	case consts.AgentTargetTypeAll:
		// target_type = all: target_details = "" (空字串)
		// 只找出 current_sign_in_at < 一個月內的代理
		return u.resolveAllAgents(ctx, campaign.MerchantID)
	case consts.AgentTargetTypeSpecific:
		// target_type = specific: target_details = ["account1", "account2"] (特定代理帳號slice)
		if len(campaign.TargetDetails) == 0 {
			return nil, fmt.Errorf("specific target requires target details")
		}
		return campaign.TargetDetails, nil
	case consts.AgentTargetTypeLine:
		// target_type = line: target_details = "account1" (最上層父代理)
		// 找出指定代理線下的所有代理
		if len(campaign.TargetDetails) == 0 {
			return nil, fmt.Errorf("line target requires target details")
		}
		parentAgentID := campaign.TargetDetails[0]
		return u.resolveLineAgents(ctx, parentAgentID)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", campaign.TargetType)
	}
}

// resolveAllAgents 解析所有活躍代理（一個月內登入）
func (u *AgentUseCase) resolveAllAgents(ctx context.Context, merchantID uint64) ([]string, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.resolveAllAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("merchant.id", int64(merchantID)))

	// 定義一個月前的時間點
	oneMonthAgo := time.Now().AddDate(0, -1, 0)

	var activeAgentGlobalIDs []string
	offset := 0
	limit := 1000

	// 分頁查詢所有代理並篩選活躍的
	for {
		agents, err := u.agentRepo.FindAgents(ctx, merchantID, limit, offset)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return nil, fmt.Errorf("find agents: %w", err)
		}

		if len(agents) == 0 {
			break
		}

		// 篩選出一個月內活躍的代理
		for _, agent := range agents {
			// 檢查代理是否在一個月內有登入
			if agent.CurrentSignInAt != nil && agent.CurrentSignInAt.After(oneMonthAgo) {
				activeAgentGlobalIDs = append(activeAgentGlobalIDs, agent.GlobalAgentID)
			}
		}

		if len(agents) < limit {
			break
		}

		offset += limit
	}

	u.tracingService.TraceEvent(span, "Active agents resolved",
		attribute.Int("active_agents", len(activeAgentGlobalIDs)),
		attribute.String("cutoff_date", oneMonthAgo.Format(time.RFC3339)))

	u.logger.InfoLog("Resolved active agents for merchant",
		u.logger.UInt64("merchant_id", merchantID),
		u.logger.Int("active_agents", len(activeAgentGlobalIDs)))

	return activeAgentGlobalIDs, nil
}

// resolveLineAgents 解析代理線下的所有代理
// parentAgentID: 最上層父代理的帳號ID
func (u *AgentUseCase) resolveLineAgents(ctx context.Context, parentAgentID string) ([]string, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.resolveLineAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("parent_agent_id", parentAgentID))

	// 獲取代理線下的所有代理
	descendants, err := u.agentService.GetAgentLineDescendants(ctx, parentAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		u.logger.WarnLog("Failed to get agent line descendants",
			u.logger.String("parent_agent_id", parentAgentID),
			u.logger.String("error", err.Error()))
		return nil, fmt.Errorf("get agent line descendants: %w", err)
	}

	// 包含父代理自己和所有後代
	allAgentIDs := make([]string, 0, len(descendants)+1)
	allAgentIDs = append(allAgentIDs, parentAgentID)  // 包含父代理自己
	allAgentIDs = append(allAgentIDs, descendants...) // 添加所有後代

	u.tracingService.TraceEvent(span, "Line agents resolved",
		attribute.Int("total_agents", len(allAgentIDs)))

	return allAgentIDs, nil
}

// createMessagesForAgents 為代理批量創建訊息
func (u *AgentUseCase) createMessagesForAgents(ctx context.Context, campaign *entity.AgentCampaign, agents []*entity.Agent) (int, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.createMessagesForAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int("agents_count", len(agents)))

	if len(agents) == 0 {
		return 0, nil
	}

	// 建構代理 ID 列表
	agentIDs := make([]uint64, len(agents))
	for i, agent := range agents {
		agentIDs[i] = agent.ID
	}

	// 檢查已存在的訊息，避免重複發送
	existingMessages, err := u.agentMessageRepo.CheckMessageExistsBatch(ctx, agentIDs, campaign.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, fmt.Errorf("check existing messages: %w", err)
	}

	// 構建需要創建的訊息
	var messages []*entity.AgentMessage
	for _, agent := range agents {
		// 跳過已存在的訊息
		if existingMessages[agent.ID] {
			continue
		}

		message := &entity.AgentMessage{
			AgentCampaignID: campaign.ID,
			AgentID:         agent.ID,
			IsRead:          false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		messages = append(messages, message)
	}

	if len(messages) == 0 {
		u.logger.InfoLog("No new messages to create, all agents already have messages",
			u.logger.UInt64("campaign_id", campaign.ID))
		return 0, nil
	}

	// 批量創建訊息
	if err := u.agentMessageRepo.CreateBatch(ctx, messages); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, fmt.Errorf("create messages batch: %w", err)
	}

	u.tracingService.TraceEvent(span, "Messages created successfully",
		attribute.Int("created_count", len(messages)),
		attribute.Int("skipped_count", len(agents)-len(messages)))

	return len(messages), nil
}
