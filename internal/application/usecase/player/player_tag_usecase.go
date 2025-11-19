package player

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/constants"
)

type PlayerTagUseCase struct {
	tagRepo        repository.TagRepository
	merchantRepo   repository.MerchantRepository
	playerRepo     repository.PlayerRepository
	playerTagRepo  repository.PlayerTagRepository
	logger         infrastructure.Logger
	lockManager    infrastructure.DistributedLockManager
	tracingService infrastructure.TracingService
}

func NewTagUseCase(
	tagRepo repository.TagRepository,
	merchantRepo repository.MerchantRepository,
	playerRepo repository.PlayerRepository,
	playerTagRepo repository.PlayerTagRepository,
	logger infrastructure.Logger,
	lockManager infrastructure.DistributedLockManager,
	tracingService infrastructure.TracingService,
) inbound.PlayerTagUseCase {
	return &PlayerTagUseCase{
		tagRepo:        tagRepo,
		merchantRepo:   merchantRepo,
		playerRepo:     playerRepo,
		playerTagRepo:  playerTagRepo,
		logger:         logger,
		lockManager:    lockManager,
		tracingService: tracingService,
	}
}

func (u *PlayerTagUseCase) SyncPlayerTags(
	ctx context.Context,
	data *event.IdentityPlayerTagSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerTagUseCase.SyncPlayerTags")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	u.tracingService.TraceEvent(span, "Checking if player exists")
	player, err := u.playerRepo.FindByGlobalID(ctx, data.GlobalPlayerID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
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
	u.tracingService.TraceEvent(span, "Start operation tags batch upsert")
	if err = u.tagRepo.BatchUpsert(ctx, tagsToInsert); err != nil {
		u.tracingService.RecordSpanError(span, err)
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
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find tags by global_ids: %w", err)
	}

	tagIDs := make([]uint64, len(tags))
	for k, tag := range tags {
		tagIDs[k] = tag.ID
	}

	// 建立Player Tags關聯
	u.tracingService.TraceEvent(span, "Start sync player tags relation")
	if err = u.executeLocked(ctx, player.ID, func() error {
		err = u.playerTagRepo.BatchUpdate(ctx, player.ID, tagIDs)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("batch update player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Batch upsert player tags completed",
			u.logger.UInt64("player_id", player.ID),
			u.logger.Int("count", len(tagIDs)),
			u.logger.Any("tag_ids", tagIDs),
		)
		return nil
	}); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert player tags failed: %w", err)
	}

	u.tracingService.TraceEvent(span, "Sync player tags relation completed")
	return nil
}

func (u *PlayerTagUseCase) SyncTag(ctx context.Context, data *event.IdentityTagSyncEvent) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerTagUseCase.SyncTag")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	tagToInsert := &entity.Tag{
		GlobalTagID: data.Tag.GlobalTagID,
		Name:        data.Tag.Name,
		MerchantID:  merchant.ID,
		CreatedAt:   data.Tag.UpdatedAt,
		UpdatedAt:   data.Tag.UpdatedAt,
		DeletedAt: func() *time.Time {
			if data.Tag.DeletedAt == "" {
				return nil
			}
			return &data.Tag.UpdatedAt
		}(),
	}

	if err = u.tagRepo.Upsert(ctx, tagToInsert); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert tag failed: %w", err)
	}
	u.logger.InfoWithContext(
		ctx,
		"Upsert tag completed",
		u.logger.Any("tag", tagToInsert),
	)

	u.tracingService.TraceEvent(span, "Tag sync completed successfully")
	return nil
}

func (u *PlayerTagUseCase) executeLocked(
	ctx context.Context,
	playerID uint64,
	fn func() error,
) error {
	mutexKey := fmt.Sprintf(constants.SyncPlayerTagRedisKey, playerID)

	// 智能重試機制：3次嘗試，每次使用遞增參數
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 智能遞增參數策略
		expiry := time.Duration(10+attempt*5) * time.Second
		tries := 5 + attempt*2
		baseDelay := time.Duration(50*attempt) * time.Millisecond

		lockOptions := infrastructure.LockOptions{
			Expiry:     expiry,
			Tries:      tries,
			RetryDelay: baseDelay,
		}

		mutex, err := u.lockManager.GetLockWithOptions(ctx, mutexKey, lockOptions)
		if err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to create mutex",
				u.logger.String("mutex_key", mutexKey),
				u.logger.UInt64("player_id", playerID),
				u.logger.Int("attempt", attempt),
				u.logger.Error("err", err),
			)
			if attempt < maxAttempts {
				// 指數退避策略: 500ms → 2s → 4.5s
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to create mutex for player %d after %d attempts: %w",
				playerID,
				maxAttempts,
				err,
			)
		}

		if err = mutex.Lock(); err != nil {
			u.logger.WarnWithContext(ctx, "Failed to acquire lock",
				u.logger.String("mutex_key", mutexKey),
				u.logger.UInt64("player_id", playerID),
				u.logger.Int("attempt", attempt),
				u.logger.String("expiry", expiry.String()),
				u.logger.Int("tries", tries),
				u.logger.Error("err", err),
			)

			if attempt < maxAttempts {
				// 指數退避策略
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				u.logger.InfoWithContext(ctx, "Retrying after backoff",
					u.logger.UInt64("player_id", playerID),
					u.logger.String("backoff_time", backoffTime.String()),
					u.logger.Int("next_attempt", attempt+1),
				)
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to acquire lock for player %d after %d attempts: %w",
				playerID,
				maxAttempts,
				err,
			)
		}

		// 成功獲取鎖
		u.logger.InfoWithContext(ctx, "Successfully acquired lock",
			u.logger.UInt64("player_id", playerID),
			u.logger.Int("attempt", attempt),
			u.logger.String("expiry", expiry.String()),
			u.logger.Int("tries", tries),
		)

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

	return fmt.Errorf(
		"failed to acquire lock for player %d after %d attempts",
		playerID,
		maxAttempts,
	)
}

func (u *PlayerTagUseCase) GetTagsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*dto.TagListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerTagUseCase.GetTagsByMerchantID")
	defer u.tracingService.SpanEnd(span)

	tags, err := u.tagRepo.FindByMerchantID(ctx, merchantID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find tags by merchant ID: %w", err)
	}

	tagResponses := make([]dto.TagResponse, len(tags))
	for i, tag := range tags {
		tagResponses[i] = dto.TagResponse{
			ID:       tag.ID,
			Name:     tag.Name,
			GlobalID: tag.GlobalTagID,
		}
	}

	return &dto.TagListResponse{
		Tags: tagResponses,
	}, nil
}
