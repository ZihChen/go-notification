package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

type PlayerUseCase interface {
	SyncPlayer(ctx context.Context, data *event.PlayerEvent) error
	GetPlayerByID(ctx context.Context, id uint64) (*dto.PlayerResponse, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*dto.PlayerResponse, error)
	UpdatePlayerLastActive(ctx context.Context, id uint64) error
}

type PlayerLevelUseCase interface {
	SyncPlayerLevel(
		ctx context.Context,
		data *event.IdentityPlayerLevelSyncEvent,
	) error
}

type PlayerTagUseCase interface {
	SyncPlayerTags(
		ctx context.Context,
		data *event.IdentityPlayerTagSyncEvent,
	) error
	SyncTag(
		ctx context.Context,
		data *event.IdentityTagSyncEvent,
	) error
}
