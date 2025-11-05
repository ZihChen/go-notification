package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
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
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.CreateAgentCampaign")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作創建代理活動邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// UpdateAgentCampaign 更新代理訊息活動
func (u *AgentUseCase) UpdateAgentCampaign(
	ctx context.Context,
	id uint64,
	req *dto.UpdateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.UpdateAgentCampaign")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作更新代理活動邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// GetAgentCampaign 獲取代理訊息活動
func (u *AgentUseCase) GetAgentCampaign(
	ctx context.Context,
	id uint64,
) (*entity.AgentCampaign, error) {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaign")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作獲取代理活動邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// GetAgentCampaigns 獲取代理訊息活動列表
func (u *AgentUseCase) GetAgentCampaigns(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) (*dto.AgentCampaignListResponse, error) {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaigns")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作獲取代理活動列表邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// DeleteAgentCampaign 刪除代理訊息活動
func (u *AgentUseCase) DeleteAgentCampaign(ctx context.Context, id uint64) error {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.DeleteAgentCampaign")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作刪除代理活動邏輯
	return fmt.Errorf("not implemented yet")
}

// GetAgentMessages 獲取代理站內信列表
func (u *AgentUseCase) GetAgentMessages(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) (*dto.AgentMessageListResponse, error) {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentMessages")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作獲取代理站內信列表邏輯
	return nil, fmt.Errorf("not implemented yet")
}

// MarkMessageAsRead 標記訊息為已讀
func (u *AgentUseCase) MarkMessageAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	//ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.MarkMessageAsRead")
	//defer u.tracingService.SpanEnd(span)

	// TODO: 實作標記訊息已讀邏輯
	return fmt.Errorf("not implemented yet")
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
