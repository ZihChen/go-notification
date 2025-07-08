package repositoryport

import (
	"context"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
	Create(ctx context.Context, merchant *entity.Merchant) error
	Update(ctx context.Context, merchant *entity.Merchant) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	Create(ctx context.Context, player *entity.Player) error
	Update(ctx context.Context, player *entity.Player) error
	Delete(ctx context.Context, id uint64) error
}

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
	Create(ctx context.Context, manager *entity.Manager) error
	Update(ctx context.Context, manager *entity.Manager) error
	Delete(ctx context.Context, id uint64) error
}

// MessageCampaignRepository 會員訊息活動資料庫接口
type MessageCampaignRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.MessageCampaign, error)
	FindAll(ctx context.Context, page, pageSize int) ([]*entity.MessageCampaign, int, error)
	FindActiveByFocus(ctx context.Context, focus uint8) ([]*entity.MessageCampaign, error)
	Create(ctx context.Context, campaign *entity.MessageCampaign) error
	Update(ctx context.Context, campaign *entity.MessageCampaign) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerMessageRepository 會員訊息資料庫接口
type PlayerMessageRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.PlayerMessage, error)
	FindByPlayerID(ctx context.Context, globalPlayerID string, page, pageSize int) ([]*entity.PlayerMessage, int, error)
	GetPlayerMessageStats(ctx context.Context, globalPlayerID string) (*entity.PlayerMessageStats, error)
	Create(ctx context.Context, message *entity.PlayerMessage) error
	CreateBatch(ctx context.Context, messages []*entity.PlayerMessage) error
	MarkAsRead(ctx context.Context, globalPlayerID string, messageID uint64) error
	CheckMessageExists(ctx context.Context, globalPlayerID string, campaignID uint64) (bool, error)
}
