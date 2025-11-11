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
		attribute.StringSlice("status", query.Status))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, query.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}
	query.MerchantID = merchant.ID

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

// SendMessageToCampaignTargets 發送訊息給活動目標（批次處理避免 OOM）
func (u *AgentUseCase) SendMessageToCampaignTargets(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SendMessageToCampaignTargets")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("target_type", campaign.TargetType))

	switch campaign.TargetType {
	case consts.AgentTargetTypeAll:
		// target_type = all: 批次處理活躍代理，避免記憶體溪漏
		return u.processBatchAgentsForAll(ctx, campaign)
	case consts.AgentTargetTypeSpecific:
		// target_type = specific: 直接批量處理，不過濾活躍度
		if len(campaign.TargetDetails) == 0 {
			return 0, 0, fmt.Errorf("specific target requires target details")
		}
		return u.processBatchAgentsForSpecific(ctx, campaign, campaign.TargetDetails)
	case consts.AgentTargetTypeLine:
		// target_type = line: 批次處理線下代理，避免記憶體溪漏
		if len(campaign.TargetDetails) == 0 {
			return 0, 0, fmt.Errorf("line target requires target details")
		}
		account := campaign.TargetDetails[0] // 頂層父代理
		return u.processBatchAgentsForLine(ctx, campaign, account)
	default:
		return 0, 0, fmt.Errorf("unsupported target type: %s", campaign.TargetType)
	}
}

// GetScheduledCampaigns 獲取到期的排程活動
func (u *AgentUseCase) GetScheduledCampaigns(
	ctx context.Context,
	currentTime time.Time,
) ([]*entity.AgentCampaign, error) {
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

// FilterActiveAgents 篩選活躍代理（已弃用，保留作為補用函式）
// 新的實作使用 BatchGetActiveAgentsByGlobalIDs 更高效
func (u *AgentUseCase) FilterActiveAgents(
	ctx context.Context,
	globalAgentIDs []string,
) ([]*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.FilterActiveAgents")
	defer u.tracingService.SpanEnd(span)

	if len(globalAgentIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	// 使用新的批量活躍代理查詢方法，直接在資料庫層過濾
	threshold := time.Now().AddDate(0, -1, 0)
	activeAgents, err := u.agentRepo.BatchGetActiveAgentsByGlobalIDs(ctx, globalAgentIDs, threshold)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("batch get active agents: %w", err)
	}

	u.tracingService.TraceEvent(span, "Active agents filtered with database query",
		attribute.Int("requested_agents", len(globalAgentIDs)),
		attribute.Int("active_agents", len(activeAgents)),
		attribute.String("threshold", threshold.Format(time.RFC3339)))

	return activeAgents, nil
}

// UpdateCampaignStatus 更新活動狀態
func (u *AgentUseCase) UpdateCampaignStatus(
	ctx context.Context,
	campaignID uint64,
	status string,
) error {
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
func (u *AgentUseCase) CompleteCampaign(
	ctx context.Context,
	campaignID uint64,
	targetCount, sentCount int,
) error {
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

// resolveLineAgents 解析代理線下的所有代理（使用直接資料庫查詢）
// parentGlobalID: 最上層父代理的全局ID
func (u *AgentUseCase) resolveLineAgents(
	ctx context.Context,
	parentGlobalID string,
) ([]uint64, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.resolveLineAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(
		span,
		attribute.String("parent_global_id", parentGlobalID),
	)

	// 獲取父代理的數值ID
	parentAgent, err := u.agentRepo.GetByGlobalID(ctx, parentGlobalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get parent agent by global id: %w", err)
	}
	if parentAgent == nil {
		return []uint64{}, nil
	}

	// 直接使用 Repository 方法獲取代理線下的所有代理（更高效）
	descendants, err := u.agentRepo.QueryAgentsByRelationship(ctx, parentGlobalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		u.logger.WarnLog("Failed to query agents by relationship",
			u.logger.String("parent_global_id", parentGlobalID),
			u.logger.String("error", err.Error()))
		return nil, fmt.Errorf("query agents by relationship: %w", err)
	}

	// 包含父代理自己和所有後代
	allAgentIDs := make([]uint64, 0, len(descendants)+1)
	allAgentIDs = append(allAgentIDs, parentAgent.ID) // 包含父代理自己
	allAgentIDs = append(allAgentIDs, descendants...) // 添加所有後代

	u.tracingService.TraceEvent(span, "Line agents resolved with repository query",
		attribute.Int("descendants_count", len(descendants)),
		attribute.Int("total_agents", len(allAgentIDs)))

	u.logger.InfoLog("Line agents resolved successfully",
		u.logger.String("parent_global_id", parentGlobalID),
		u.logger.Int("total_agents", len(allAgentIDs)))

	return allAgentIDs, nil
}

// createMessagesForAgents 為代理批量創建訊息
func (u *AgentUseCase) createMessagesForAgents(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	agents []*entity.Agent,
) (int, error) {
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

// processBatchAgentsForAll 批次處理 target_type = all 的活躍代理
func (u *AgentUseCase) processBatchAgentsForAll(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForAll")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int64("merchant.id", int64(campaign.MerchantID)))

	// 定義一個月前的時間點
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	offset := 0
	limit := 5000 // 批次大小，可配置
	totalTargetCount := 0
	totalSentCount := 0

	u.logger.InfoLog("Starting batch processing for all active agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.UInt64("merchant_id", campaign.MerchantID))

	// 分批處理活躍代理
	for {
		// 獲取一批活躍代理
		activeAgents, err := u.agentRepo.FindActiveAgents(
			ctx,
			campaign.MerchantID,
			oneMonthAgo,
			limit,
			offset,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return totalTargetCount, totalSentCount, fmt.Errorf("find active agents batch: %w", err)
		}

		if len(activeAgents) == 0 {
			break // 沒有更多代理了
		}

		totalTargetCount += len(activeAgents)

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, activeAgents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_size", len(activeAgents)),
				u.logger.Int("offset", offset),
				u.logger.String("error", err.Error()))
			// 繼續處理下一批，不因為一批失敗就停止整個流程
		} else {
			totalSentCount += batchSentCount
		}

		u.logger.InfoLog("Processed batch for all active agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(activeAgents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount),
			u.logger.Int("offset", offset))

		// 如果當次查詢結果數量少於 limit，說明已經是最後一批
		if len(activeAgents) < limit {
			break
		}

		offset += limit
	}

	u.tracingService.TraceEvent(span, "Batch processing completed for all active agents",
		attribute.Int("target_count", totalTargetCount),
		attribute.Int("sent_count", totalSentCount))

	u.logger.InfoLog("Batch processing completed for all active agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.Int("target_count", totalTargetCount),
		u.logger.Int("sent_count", totalSentCount))

	return totalTargetCount, totalSentCount, nil
}

// processBatchAgentsForSpecific 批次處理 target_type = specific 的代理
func (u *AgentUseCase) processBatchAgentsForSpecific(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	targetAccounts []string,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForSpecific")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int("target_accounts_count", len(targetAccounts)))

	batchSize := 5000 // 批次大小，可配置
	totalSentCount := 0

	u.logger.InfoLog("Starting batch processing for specific agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.Int("total_targets", len(targetAccounts)))

	// 分批處理指定代理
	for i := 0; i < len(targetAccounts); i += batchSize {
		end := i + batchSize
		if end > len(targetAccounts) {
			end = len(targetAccounts)
		}

		batchAccounts := targetAccounts[i:end]

		// 批量獲取代理詳細資訊
		batchAgents, err := u.agentRepo.BatchGetAgentsByAccounts(ctx, batchAccounts)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to get batch agents",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_start", i),
				u.logger.Int("batch_size", len(batchAccounts)),
				u.logger.String("error", err.Error()))
			continue // 繼續處理下一批
		}

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, batchAgents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for specific batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_agents", len(batchAgents)),
				u.logger.String("error", err.Error()))
		} else {
			totalSentCount += batchSentCount
		}

		u.logger.InfoLog("Processed batch for specific agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(batchAgents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount))
	}

	u.tracingService.TraceEvent(span, "Batch processing completed for specific agents",
		attribute.Int("target_count", len(targetAccounts)),
		attribute.Int("sent_count", totalSentCount))

	return len(targetAccounts), totalSentCount, nil
}

// processBatchAgentsForLine 批次處理 target_type = line 的代理
func (u *AgentUseCase) processBatchAgentsForLine(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	account string,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForLine")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("account", account))

	parentAgent, err := u.agentRepo.GetByAccount(ctx, account)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, 0, fmt.Errorf("get agent by account: %w", err)
	}

	batchSize := 1000 // 每批處理的代理數量
	totalTargetCount := 0
	totalSentCount := 0

	u.logger.InfoLog("Starting repository-level batch processing for line agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("parent_agent_id", parentAgent.GlobalAgentID))

	// 創建處理器函數，用於處理每批代理
	processor := func(ctx context.Context, agentIDs []uint64) (processedCount int, err error) {
		// 批量獲取代理詳細資訊
		batchAgents, err := u.agentRepo.BatchGetAgentsByIDs(ctx, agentIDs)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to get batch agents for line processing",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_size", len(agentIDs)),
				u.logger.String("error", err.Error()))
			return 0, err
		}

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, batchAgents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for line batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_agents", len(batchAgents)),
				u.logger.String("error", err.Error()))
			return 0, err
		}

		totalTargetCount += len(batchAgents)
		totalSentCount += batchSentCount

		u.logger.InfoLog("Processed batch for line agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(batchAgents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount))

		return batchSentCount, nil
	}

	// 使用 repository 層面的分頁處理
	totalProcessed, err := u.agentRepo.ProcessAgentsByRelationshipInBatches(
		ctx,
		parentAgent.GlobalAgentID,
		batchSize,
		processor,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return totalTargetCount, totalSentCount, fmt.Errorf(
			"process agents by relationship in batches: %w",
			err,
		)
	}

	u.tracingService.TraceEvent(span, "Repository-level line agents batch processing completed",
		attribute.Int("target_count", totalTargetCount),
		attribute.Int("sent_count", totalProcessed))

	u.logger.InfoLog("Completed repository-level batch processing for line agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("parent_agent_id", parentAgent.GlobalAgentID),
		u.logger.Int("target_count", totalTargetCount),
		u.logger.Int("sent_count", totalProcessed))

	return totalTargetCount, totalProcessed, nil
}
