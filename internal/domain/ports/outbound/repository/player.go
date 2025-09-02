package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

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

// LevelRepository 等級資料庫接口
type LevelRepository interface {
	Upsert(ctx context.Context, level *entity.Level) error
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Level, error)
}

// TagRepository 標籤資料庫接口
type TagRepository interface {
	Upsert(ctx context.Context, tag *entity.Tag) error
	BatchUpsert(ctx context.Context, tags []*entity.Tag) error
	FindByGlobalIDs(ctx context.Context, globalIDs []string) ([]*entity.Tag, error)
}

// PlayerTagRepository 玩家標籤關聯資料庫接口
type PlayerTagRepository interface {
	BatchUpdate(ctx context.Context, playerID uint64, tagIDs []uint64) error
}
