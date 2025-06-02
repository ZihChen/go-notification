package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/model"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Merchant, error)
	Create(ctx context.Context, merchant *model.Merchant) error
	Update(ctx context.Context, merchant *model.Merchant) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Player, error)
	Create(ctx context.Context, player *model.Player) error
	Update(ctx context.Context, player *model.Player) error
	Delete(ctx context.Context, id uint64) error
}

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Manager, error)
	Create(ctx context.Context, manager *model.Manager) error
	Update(ctx context.Context, manager *model.Manager) error
	Delete(ctx context.Context, id uint64) error
}

// MessageCampaignRepository 會員訊息活動資料庫接口
type MessageCampaignRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.MessageCampaign, error)
	FindAll(ctx context.Context, page, pageSize int) ([]*model.MessageCampaign, int, error)
	FindActiveByFocus(ctx context.Context, focus uint8) ([]*model.MessageCampaign, error)
	Create(ctx context.Context, campaign *model.MessageCampaign) error
	Update(ctx context.Context, campaign *model.MessageCampaign) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerMessageRepository 會員訊息資料庫接口
type PlayerMessageRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.PlayerMessage, error)
	FindByPlayerID(ctx context.Context, globalPlayerID string, page, pageSize int) ([]*model.PlayerMessage, int, error)
	GetPlayerMessageStats(ctx context.Context, globalPlayerID string) (*model.PlayerMessageStats, error)
	Create(ctx context.Context, message *model.PlayerMessage) error
	CreateBatch(ctx context.Context, messages []*model.PlayerMessage) error
	MarkAsRead(ctx context.Context, globalPlayerID string, messageID uint64) error
	CheckMessageExists(ctx context.Context, globalPlayerID string, campaignID uint64) (bool, error)
}
