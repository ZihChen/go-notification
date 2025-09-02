package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

type PlayerUseCase interface {
	SyncPlayer(ctx context.Context, data *event.PlayerEvent) error
	GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
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
