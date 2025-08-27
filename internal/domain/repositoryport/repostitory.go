package repositoryport

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
	FirstOrCreate(ctx context.Context, merchant *entity.Merchant) error
	Create(ctx context.Context, merchant *entity.Merchant) error
	Update(ctx context.Context, merchant *entity.Merchant) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, merchant *entity.Merchant) error
}

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	FindByTargetType(
		ctx context.Context,
		targetType uint8,
		offset, limit int,
	) ([]*entity.Player, error)
	FirstOrCreate(ctx context.Context, player *entity.Player) error
	Create(ctx context.Context, player *entity.Player) error
	Update(ctx context.Context, player *entity.Player) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, player *entity.Player) error
}

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
	FirstOrCreate(ctx context.Context, manager *entity.Manager) error
	Create(ctx context.Context, manager *entity.Manager) error
	Update(ctx context.Context, manager *entity.Manager) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, manager *entity.Manager) error
}

// MessageCampaignRepository 會員訊息活動資料庫接口
type MessageCampaignRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.MessageCampaign, error)
	FindAllWithOptions(
		ctx context.Context,
		query *entity.MessageCampaignsQuery,
	) ([]*entity.MessageCampaign, int, error)
	FindActiveByFocus(ctx context.Context, focus uint8) ([]*entity.MessageCampaign, error)
	FindScheduledCampaigns(ctx context.Context) ([]*entity.MessageCampaign, error)
	Create(ctx context.Context, campaign *entity.MessageCampaign) error
	Update(ctx context.Context, campaign *entity.MessageCampaign) error
	UpdateFields(
		ctx context.Context,
		id uint64,
		updates map[string]interface{},
		includeDeleted bool,
	) error
	UpdateSentCount(ctx context.Context, campaignID uint64, count int64) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerMessageRepository 會員訊息資料庫接口
type PlayerMessageRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.PlayerMessage, error)
	FindByPlayerID(
		ctx context.Context,
		globalPlayerID string,
		page, pageSize int,
	) ([]*entity.PlayerMessage, int, error)
	GetPlayerMessageStats(
		ctx context.Context,
		globalPlayerID string,
	) (*entity.PlayerMessageStats, error)
	Create(ctx context.Context, message *entity.PlayerMessage) error
	CreateBatch(ctx context.Context, messages []*entity.PlayerMessage) error
	CreateBatchOptimized(ctx context.Context, messages []*entity.PlayerMessage, batchSize int) error
	MarkAsRead(ctx context.Context, globalPlayerID string, messageID uint64) error
	CheckMessageExists(ctx context.Context, globalPlayerID string, campaignID uint64) (bool, error)
	CheckMessageExistsBatch(
		ctx context.Context,
		playerIDs []uint64,
		campaignID uint64,
	) (map[uint64]bool, error)
}

type LevelRepository interface {
	Upsert(ctx context.Context, level *entity.Level) error
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Level, error)
}

type TagRepository interface {
	Upsert(ctx context.Context, tag *entity.Tag) error
	BatchUpsert(ctx context.Context, tags []*entity.Tag) error
	FindByGlobalIDs(ctx context.Context, globalIDs []string) ([]*entity.Tag, error)
}

type PlayerTagRepository interface {
	BatchUpdate(ctx context.Context, playerID uint64, tagIDs []uint64) error
}
