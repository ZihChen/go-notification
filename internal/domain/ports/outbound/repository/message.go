package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// MessageCampaignRepository 會員訊息活動資料庫接口
type MessageCampaignRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.MessageCampaign, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.MessageCampaign, error)
	FindAllWithOptions(
		ctx context.Context,
		query *dto.MessageCampaignsQuery,
	) ([]*entity.MessageCampaign, int, error)
	FindActiveByFocus(ctx context.Context, focus string) ([]*entity.MessageCampaign, error)
	FindScheduledCampaigns(ctx context.Context) ([]*entity.MessageCampaign, error)
	FindAutoSettingsByMerchantID(
		ctx context.Context,
		merchantID uint64,
	) ([]*entity.MessageCampaign, error)
	FindAutoSettingByCategoryItemTrigger(
		ctx context.Context,
		merchantID uint64,
		category string,
		item string,
		triggerType string,
	) (*entity.MessageCampaign, error)
	UpsertAutoSettings(ctx context.Context, campaigns []*entity.MessageCampaign) error
	DeleteAutoSettingsByMerchantID(ctx context.Context, merchantID uint64) error
	Create(ctx context.Context, campaign *entity.MessageCampaign) error
	Update(ctx context.Context, campaign *entity.MessageCampaign) error
	UpdateFields(
		ctx context.Context,
		id uint64,
		updates map[string]interface{},
		includeDeleted bool,
	) error
	UpdateSentCount(ctx context.Context, campaignID uint64, count int64) error
	UpdateStatus(ctx context.Context, campaignID uint64, status string) error
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
	) (*dto.PlayerMessageStats, error)
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
