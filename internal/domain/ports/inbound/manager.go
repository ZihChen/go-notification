package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

type ManagerUseCase interface {
	SyncManager(ctx context.Context, data *event.ManagerEvent) error
	GetManagerByID(ctx context.Context, id uint64) (*dto.ManagerResponse, error)
	GetManagerByGlobalID(ctx context.Context, globalID string) (*dto.ManagerResponse, error)
}
