package player

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"sort"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/constants"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils"
)

type PlayerTagUseCase struct {
	tagRepo        repository.TagRepository
	merchantRepo   repository.MerchantRepository
	playerRepo     repository.PlayerRepository
	playerTagRepo  repository.PlayerTagRepository
	logger         infrastructure.Logger
	lockManager    infrastructure.DistributedLockManager
	tracingService infrastructure.TracingService
	cacheManager   infrastructure.CacheManager
}

func NewTagUseCase(
	tagRepo repository.TagRepository,
	merchantRepo repository.MerchantRepository,
	playerRepo repository.PlayerRepository,
	playerTagRepo repository.PlayerTagRepository,
	logger infrastructure.Logger,
	lockManager infrastructure.DistributedLockManager,
	tracingService infrastructure.TracingService,
	cacheManager infrastructure.CacheManager,
) inbound.PlayerTagUseCase {
	return &PlayerTagUseCase{
		tagRepo:        tagRepo,
		merchantRepo:   merchantRepo,
		playerRepo:     playerRepo,
		playerTagRepo:  playerTagRepo,
		logger:         logger,
		lockManager:    lockManager,
		tracingService: tracingService,
		cacheManager:   cacheManager,
	}
}

func (u *PlayerTagUseCase) SyncPlayerTags(
	ctx context.Context,
	data *event.IdentityPlayerTagSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerTagUseCase.SyncPlayerTags")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.TraceEvent(span, "Checking if merchant exists")
	cacheKey := fmt.Sprintf(consts.RedisMerchantGlobalIDKey, data.GlobalMerchantID)
	merchant, err := utils.QueryWithCache(
		ctx,
		u.cacheManager,
		cacheKey,
		10*time.Minute, // 商戶資訊快取10分鐘
		"merchant",
		func(ctx context.Context) (*entity.Merchant, error) {
			return u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
		},
	)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	u.tracingService.TraceEvent(span, "Checking if player exists")
	playerCacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, data.GlobalPlayerID)
	player, err := utils.QueryWithCache(
		ctx,
		u.cacheManager,
		playerCacheKey,
		5*time.Minute,
		"player",
		func(ctx context.Context) (*entity.Player, error) {
			return u.playerRepo.FindByGlobalID(ctx, data.GlobalPlayerID)
		},
	)
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
		return u.syncPlayerTagsWithDifference(ctx, player.ID, tagIDs)
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
	cacheKey := fmt.Sprintf(consts.RedisMerchantGlobalIDKey, data.GlobalMerchantID)
	merchant, err := utils.QueryWithCache(
		ctx,
		u.cacheManager,
		cacheKey,
		10*time.Minute, // 商戶資訊快取10分鐘
		"merchant",
		func(ctx context.Context) (*entity.Merchant, error) {
			return u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
		},
	)
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

// syncPlayerTagsWithDifference 使用快取查詢現有標籤並計算差異
func (u *PlayerTagUseCase) syncPlayerTagsWithDifference(
	ctx context.Context,
	playerID uint64,
	newTagIDs []uint64,
) error {
	// 處理空標籤情況
	if len(newTagIDs) == 0 {
		if err := u.playerTagRepo.DeleteByPlayerID(ctx, playerID); err != nil {
			return fmt.Errorf("delete all player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Deleted all player tags",
			u.logger.UInt64("player_id", playerID),
		)
		return nil
	}

	// 排序以確保一致的鎖定順序，避免死鎖
	sort.Slice(newTagIDs, func(i, j int) bool {
		return newTagIDs[i] < newTagIDs[j]
	})

	// 1. 使用快取查詢現有關聯
	cacheKey := fmt.Sprintf(consts.RedisPlayerTagsKey, playerID)
	existingTagIDs, err := utils.QueryWithCache(
		ctx,
		u.cacheManager,
		cacheKey,
		5*time.Minute, // 快取5分鐘
		"player_tags",
		func(ctx context.Context) ([]uint64, error) {
			return u.playerRepo.GetPlayerTagIDs(ctx, playerID)
		},
	)
	if err != nil {
		return fmt.Errorf("query existing player tags failed: %w", err)
	}

	// 2. 計算差異
	toDelete := u.difference(existingTagIDs, newTagIDs)
	toInsert := u.difference(newTagIDs, existingTagIDs)

	// 3. 無變化直接返回
	if len(toDelete) == 0 && len(toInsert) == 0 {
		u.logger.InfoWithContext(ctx, "No changes needed for player tags",
			u.logger.UInt64("player_id", playerID),
		)
		return nil
	}

	// 4. 執行精確的批量更新（只操作真正需要變化的部分）
	if err = u.playerTagRepo.BatchUpdateWithDiff(ctx, playerID, toDelete, toInsert); err != nil {
		return fmt.Errorf("batch update player tags with diff failed: %w", err)
	}

	// 5. 清除快取
	if _, err := u.cacheManager.Del(ctx, cacheKey); err != nil {
		u.logger.WarnWithContext(ctx, "Failed to clear cache after update",
			u.logger.String("cache_key", cacheKey),
			u.logger.Error("err", err),
		)
	}

	u.logger.InfoWithContext(ctx, "Player tags sync completed with precise diff update",
		u.logger.UInt64("player_id", playerID),
		u.logger.Int("total_new_tags", len(newTagIDs)),
		u.logger.Int("existing_tags", len(existingTagIDs)),
		u.logger.Int("to_delete", len(toDelete)),
		u.logger.Int("to_insert", len(toInsert)),
		u.logger.String("optimization", "only_changed_tags_updated"),
	)

	return nil
}

// difference 計算兩個切片的差集：在 a 中但不在 b 中的元素
func (u *PlayerTagUseCase) difference(a, b []uint64) []uint64 {
	bMap := make(map[uint64]bool, len(b))
	for _, item := range b {
		bMap[item] = true
	}

	var result []uint64
	for _, item := range a {
		if !bMap[item] {
			result = append(result, item)
		}
	}

	return result
}
