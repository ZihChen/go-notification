package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

// AgentUseCase 代理業務用例接口 (業務流程協調層)
type AgentUseCase interface {
	// 代理同步相關
	SyncAgentDataWithRelationships(ctx context.Context, agentEvent *event.AgentSyncEvent) error
	SyncCurrentAgentUpsert(ctx context.Context, agentEvent *event.AgentSyncEvent) error

	// 代理查詢相關
	GetAgentByGlobalID(ctx context.Context, globalAgentID string) (*entity.Agent, error)
	GetActiveAgents(ctx context.Context, merchantID uint64, limit, offset int) ([]*entity.Agent, error)

	// 代理訊息活動相關
	CreateAgentCampaign(ctx context.Context, req *dto.CreateAgentCampaignRequest) (*entity.AgentCampaign, error)
	UpdateAgentCampaign(ctx context.Context, id uint64, req *dto.UpdateAgentCampaignRequest) (*entity.AgentCampaign, error)
	GetAgentCampaign(ctx context.Context, id uint64) (*entity.AgentCampaign, error)
	GetAgentCampaigns(ctx context.Context, query *dto.AgentCampaignsQuery) (*dto.AgentCampaignListResponse, error)
	DeleteAgentCampaign(ctx context.Context, id uint64) error

	// 代理站內信相關
	GetAgentMessages(ctx context.Context, query *dto.AgentMessagesQuery) (*dto.AgentMessageListResponse, error)
	MarkMessageAsRead(ctx context.Context, messageID uint64, agentID uint64) error

	// 代理訊息發送相關
	SendMessageToCampaignTargets(ctx context.Context, campaign *entity.AgentCampaign) error
	GetScheduledCampaigns(ctx context.Context) ([]*entity.AgentCampaign, error)
	FilterActiveAgents(agentIDs []string) []string
}

