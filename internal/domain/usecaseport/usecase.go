package usecaseport

import (
	"context"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

type MerchantUseCase interface {
	SyncMerchant(ctx context.Context, eventData []byte) error
	GetMerchantByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	GetMerchantByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
}

type ManagerUseCase interface {
	SyncManager(ctx context.Context, eventData []byte) error
	GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error)
	GetManagerByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
}

type PlayerUseCase interface {
	SyncPlayer(ctx context.Context, eventData []byte) error
	GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	UpdatePlayerLastActive(ctx context.Context, id uint64) error
}

type MessageUseCase interface {
	CreateMessageCampaign(ctx context.Context, campaign *entity.MessageCampaign) error
	UpdateMessageCampaign(ctx context.Context, campaign *entity.MessageCampaign) error
	DeleteMessageCampaign(ctx context.Context, id uint64) error
	GetMessageCampaign(ctx context.Context, id uint64) (*entity.MessageCampaign, error)
	ListMessageCampaigns(
		ctx context.Context,
		page, pageSize int,
	) ([]*entity.MessageCampaign, int, error)
	GetPlayerMessages(
		ctx context.Context,
		globalPlayerID string,
		page, pageSize int,
	) (*entity.MessageListResponse, error)
	MarkMessageAsRead(ctx context.Context, globalPlayerID string, messageID uint64) error
}

type PlayerLevelUseCase interface {
	SyncPlayerLevel(
		ctx context.Context,
		data *event.IdentityPlayerLevelSyncEvent,
	) error
}
