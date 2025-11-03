package inbound

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// AgentUseCase 代理業務用例接口 (業務流程協調層)
type AgentUseCase interface {
	// 代理同步相關
	SyncAgentDataWithRelationships(ctx context.Context, event *AgentSyncEvent) error
	SyncCurrentAgentUpsert(ctx context.Context, event *AgentSyncEvent) error
	EnsureParentAgentsExistUpsert(ctx context.Context, event *AgentSyncEvent) error

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

// AgentSyncEvent KDS 代理同步事件結構
type AgentSyncEvent struct {
	Agents struct {
		GlobalAgentID   string    `json:"global_agent_id"`
		Account         string    `json:"account"`
		Ancestry        string    `json:"ancestry"`
		CurrentSignInAt time.Time `json:"current_sign_in_at"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       time.Time `json:"updated_at"`
	} `json:"agents"`
	Merchant struct {
		ID               uint64 `json:"id"`
		Name             string `json:"name"`
		GlobalMerchantID string `json:"global_merchant_id"`
	} `json:"merchant"`
}
