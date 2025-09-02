package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

type MerchantUseCase interface {
	SyncMerchant(ctx context.Context, data *event.MerchantEvent) error
	GetMerchantByID(ctx context.Context, id uint64) (*dto.MerchantResponse, error)
	GetMerchantByGlobalID(ctx context.Context, globalID string) (*dto.MerchantResponse, error)
}
