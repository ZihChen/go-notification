package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/usecaseport"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"time"
)

type PlayerTagUseCase struct {
	tagRepo       repositoryport.TagRepository
	merchantRepo  repositoryport.MerchantRepository
	playerRepo    repositoryport.PlayerRepository
	playerTagRepo repositoryport.PlayerTagRepository
	logger        infraport.Logger
	redisManager  *redisCache.Manager
}

func NewTagUseCase(
	tagRepo repositoryport.TagRepository,
	merchantRepo repositoryport.MerchantRepository,
	playerRepo repositoryport.PlayerRepository,
	playerTagRepo repositoryport.PlayerTagRepository,
	logger infraport.Logger,
	redisManager *redisCache.Manager,
) usecaseport.PlayerTagUseCase {
	return &PlayerTagUseCase{
		tagRepo:       tagRepo,
		merchantRepo:  merchantRepo,
		playerRepo:    playerRepo,
		playerTagRepo: playerTagRepo,
		logger:        logger,
		redisManager:  redisManager,
	}
}

func (u *PlayerTagUseCase) SyncPlayerTags(ctx context.Context, data *event.IdentityPlayerTagSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "PlayerTagUseCase.SyncPlayerTags")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	tracing.TraceEvent(span, "Checking if player exists")
	player, err := u.playerRepo.FindByGlobalID(ctx, data.GlobalPlayerID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player by global_id: %w", err)
	}

	tagItems := data.Tags

	tagsToInsert, tagsGlobalIDs := make([]*entity.Tag, len(tagItems)), make([]string, len(tagItems))
	for k, item := range tagItems {
		tagsToInsert[k] = &entity.Tag{
			GlobalTagID: item.GlobalTagID,
			Name:        item.Name,
			MerchantID:  merchant.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		tagsGlobalIDs[k] = item.GlobalTagID
	}

	// 更新或創建Tags
	tracing.TraceEvent(span, "Start operation tags batch upsert")
	if err = u.tagRepo.BatchUpsert(ctx, tagsToInsert); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert tags failed: %w", err)
	}
	u.logger.InfoWithContext(
		ctx,
		"Batch upsert tags completed",
		u.logger.Int("count", len(tagsToInsert)),
		u.logger.Any("tags", tagsToInsert),
	)

	tags, err := u.tagRepo.FindByGlobalIDs(ctx, tagsGlobalIDs)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find tags by global_ids: %w", err)
	}

	tagIDs := make([]uint64, len(tags))
	for k, tag := range tags {
		tagIDs[k] = tag.ID
	}

	// 建立Player Tags關聯
	tracing.TraceEvent(span, "Start sync player tags relation")
	if err = u.executeLocked(ctx, player.ID, func() error {
		err = u.playerTagRepo.BatchUpdate(ctx, player.ID, tagIDs)
		if err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch update player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Batch upsert player tags completed",
			u.logger.UInt64("player_id", player.ID),
			u.logger.Int("count", len(tagIDs)),
			u.logger.Any("tag_ids", tagIDs),
		)
		return nil
	}); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert player tags failed: %w", err)
	}

	tracing.TraceEvent(span, "Sync player tags relation completed")
	return nil
}

func (u *PlayerTagUseCase) SyncTag(ctx context.Context, data *event.IdentityTagSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "PlayerTagUseCase.SyncTag")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	nowTime := time.Now()
	tagToInsert := &entity.Tag{
		GlobalTagID: data.Tag.GlobalTagID,
		Name:        data.Tag.Name,
		MerchantID:  merchant.ID,
		CreatedAt:   nowTime,
		UpdatedAt:   nowTime,
		DeletedAt: func() *time.Time {
			if data.Tag.DeletedAt == "" {
				return nil
			}
			return &nowTime
		}(),
	}

	if err = u.tagRepo.Upsert(ctx, tagToInsert); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert tag failed: %w", err)
	}
	u.logger.InfoWithContext(
		ctx,
		"Upsert tag completed",
		u.logger.Any("tag", tagToInsert),
	)

	tracing.TraceEvent(span, "Tag sync completed successfully")
	return nil
}

func (u *PlayerTagUseCase) executeLocked(ctx context.Context, playerID uint64, fn func() error) error {
	mutexKey := fmt.Sprintf(consts.SyncPlayerTagRedisKey, playerID)

	mutex, err := u.redisManager.GetMutexWithOption(mutexKey,
		redsync.WithExpiry(5*time.Second),            // 鎖的過期時間
		redsync.WithTries(3),                         // 獲取鎖的重試次數
		redsync.WithRetryDelay(200*time.Millisecond), // 重試間隔
	)
	if err != nil {
		return fmt.Errorf("failed to create mutex for player %d: %w", playerID, err)
	}

	if err = mutex.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock for player %d: %w", playerID, err)
	}

	defer func() {
		ok, unlockErr := mutex.Unlock()
		if !ok || unlockErr != nil {
			u.logger.ErrorWithContext(ctx, "Failed to unlock mutex",
				u.logger.String("mutex_key", mutexKey),
				u.logger.UInt64("player_id", playerID),
				u.logger.Error("err", unlockErr),
				u.logger.Bool("unlock_success", ok),
			)
		}
	}()
	return fn()
}
