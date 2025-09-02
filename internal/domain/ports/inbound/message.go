package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
)

type MessageUseCase interface {
	CreateMessageCampaign(ctx context.Context, campaign *dto.CreateMessageCampaignRequest) error
	UpdateMessageCampaign(ctx context.Context, campaign *dto.UpdateMessageCampaignRequest) error
	DeleteMessageCampaign(ctx context.Context, globalID string) error
	GetMessageCampaign(ctx context.Context, globalID string) (*dto.MessageCampaignResponse, error)
	ListMessageCampaigns(
		ctx context.Context,
		req *dto.ListMessageCampaignsRequest,
	) (*dto.MessageCampaignListResponse, error)
	GetMerchantAutoSettings(
		ctx context.Context,
		globalMerchantID string,
	) (*dto.MerchantAutoSettingsResponse, error)
	CreateOrUpdateMerchantAutoSettings(
		ctx context.Context,
		req *dto.MerchantAutoSettingsRequest,
	) (*dto.AutoSettingsOperationResponse, error)
	GetPlayerMessages(
		ctx context.Context,
		globalPlayerID string,
		page, pageSize int,
	) (*dto.MessageListResponse, error)
	MarkMessageAsRead(ctx context.Context, globalPlayerID string, messageID uint64) error
	ProcessScheduledCampaigns(ctx context.Context) error
	SendCampaignToPlayers(ctx context.Context, campaignID uint64) error
	SendCampaignToPlayersAsync(ctx context.Context, campaignID uint64) error
}
