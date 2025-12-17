package repository

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// AgentRepository 代理資料倉儲接口
type AgentRepository interface {
	// 基本 CRUD 操作
	GetByGlobalID(ctx context.Context, globalAgentID string) (*entity.Agent, error)
	GetByAccount(ctx context.Context, merchantID uint64, account string) (*entity.Agent, error)

	// 冪等性操作
	Upsert(ctx context.Context, agent *entity.Agent) error

	// 查詢操作
	FindAgents(ctx context.Context, merchantID uint64, limit, offset int) ([]*entity.Agent, error)
	FindActiveAgents(
		ctx context.Context,
		merchantID uint64,
		activeThreshold time.Time,
		limit, offset int,
	) ([]*entity.Agent, error)

	// 批量操作
	BatchGetOrCreateAgentsByGlobalIDs(
		ctx context.Context,
		globalIDs []string,
		merchantID uint64,
	) (map[string]uint64, error)
	BatchGetAgentsByIDs(ctx context.Context, agentIDs []uint64) ([]*entity.Agent, error)
	BatchGetAgentsByAccounts(
		ctx context.Context,
		merchantID uint64,
		accounts []string,
	) ([]*entity.Agent, error)
	BatchGetActiveAgentsByGlobalIDs(
		ctx context.Context,
		globalIDs []string,
		activeThreshold time.Time,
	) ([]*entity.Agent, error)

	// 關係操作
	QueryAgentsByRelationship(ctx context.Context, parentAgentID string) ([]uint64, error)
	ProcessAgentsByRelationshipInBatches(
		ctx context.Context,
		parentAgentID string,
		batchSize int,
		processor func(ctx context.Context, agentIDs []uint64) (processedCount int, err error),
	) (totalProcessed int, err error)
}

// AgentCampaignRepository 代理訊息活動倉儲接口
type AgentCampaignRepository interface {
	// 基本 CRUD 操作
	Create(ctx context.Context, campaign *entity.AgentCampaign) (*entity.AgentCampaign, error)
	GetByID(ctx context.Context, id uint64) (*entity.AgentCampaign, error)
	Update(ctx context.Context, campaign *entity.AgentCampaign) error
	Delete(ctx context.Context, id uint64) error
	BatchDelete(ctx context.Context, ids []uint64) error
	UpdateFields(
		ctx context.Context,
		id uint64,
		columns map[string]interface{},
	) error
	UpdateFieldsWithCondition(
		ctx context.Context,
		id uint64,
		columns map[string]interface{},
		conditions map[string]interface{},
	) error

	// 查詢操作
	List(
		ctx context.Context,
		query *dto.AgentCampaignsQueryForRepo,
	) ([]*entity.AgentCampaign, int, error)
	GetScheduledCampaigns(
		ctx context.Context,
		currentTime time.Time,
	) ([]*entity.AgentCampaign, error)
	FindSentCampaignsForBackfill(
		ctx context.Context,
		merchantID uint64,
	) ([]*entity.AgentCampaign, error)
	FindSentCampaignsForBackfillPaginated(
		ctx context.Context,
		merchantID uint64,
		targetTypes []string,
		limit, offset int,
	) ([]*entity.AgentCampaign, error)
}

// AgentMessageRepository 代理站內信倉儲接口
type AgentMessageRepository interface {
	// 基本 CRUD 操作
	Create(ctx context.Context, message *entity.AgentMessage) (*entity.AgentMessage, error)
	GetByID(ctx context.Context, id uint64) (*entity.AgentMessage, error)
	Update(ctx context.Context, message *entity.AgentMessage) error
	Delete(ctx context.Context, id uint64) error
	DeleteByCampaignID(ctx context.Context, campaignID uint64) error
	BatchDeleteByCampaignIDs(ctx context.Context, campaignIDs []uint64) error
	BatchHardDeleteByCampaignIDs(ctx context.Context, campaignIDs []uint64) error

	// 批量操作
	CreateBatch(ctx context.Context, messages []*entity.AgentMessage) error
	CheckMessageExistsBatch(
		ctx context.Context,
		agentIDs []uint64,
		campaignID uint64,
	) (map[uint64]bool, error)
	CheckCampaignMessageExistsBatch(
		ctx context.Context,
		agentID uint64,
		campaignIDs []uint64,
	) (map[uint64]bool, error)

	// 查詢操作
	ListByAgent(
		ctx context.Context,
		query *dto.AgentMessagesQuery,
	) ([]*entity.AgentMessage, int, error)
	MarkAsRead(ctx context.Context, messageID uint64, agentID uint64) error

	// 存在性檢查
	ExistsMessage(ctx context.Context, campaignID uint64, agentID uint64) (bool, error)

	// 統計操作
	GetMessageStats(ctx context.Context, agentID uint64) (*dto.AgentMessageStats, error)
}

// AgentRelationshipRepository 代理關係倉儲接口
type AgentRelationshipRepository interface {
	// 批次更新操作 (併發安全)
	BatchUpdate(
		ctx context.Context,
		parentID uint64,
		relationships []*entity.AgentRelationship,
	) error

	// 查詢操作
	GetByParentID(ctx context.Context, parentID uint64) ([]*entity.AgentRelationship, error)
	GetByChildID(ctx context.Context, childID uint64) ([]*entity.AgentRelationship, error)
	FindDescendants(
		ctx context.Context,
		parentID uint64,
		maxDepth int,
	) ([]*entity.AgentRelationship, error)
	FindAncestors(
		ctx context.Context,
		childID uint64,
		maxDepth int,
	) ([]*entity.AgentRelationship, error)

	// 路徑查詢
	FindByPathHash(ctx context.Context, pathHash string) ([]*entity.AgentRelationship, error)
}
