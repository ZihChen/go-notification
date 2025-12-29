package player

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/constants"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo     repository.PlayerRepository
	merchantRepo   repository.MerchantRepository
	levelRepo      repository.LevelRepository
	tagRepo        repository.TagRepository
	playerTagRepo  repository.PlayerTagRepository
	eventProducer  service.EventProducer
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
	cacheManager   infrastructure.CacheManager
	lockManager    infrastructure.DistributedLockManager
	batchProcessor *PlayerBatchProcessor // 批次處理器
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	levelRepo repository.LevelRepository,
	tagRepo repository.TagRepository,
	playerTagRepo repository.PlayerTagRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	cacheManager infrastructure.CacheManager,
	lockManager infrastructure.DistributedLockManager,
) inbound.PlayerUseCase {
	useCase := &PlayerUseCase{
		playerRepo:     playerRepo,
		merchantRepo:   merchantRepo,
		levelRepo:      levelRepo,
		tagRepo:        tagRepo,
		playerTagRepo:  playerTagRepo,
		eventProducer:  eventProducer,
		logger:         logger,
		tracingService: tracingService,
		cacheManager:   cacheManager,
		lockManager:    lockManager,
		// 創建批次處理器
		batchProcessor: NewPlayerBatchProcessor(
			playerRepo,
			logger,
			tracingService,
		),
	}

	return useCase
}

// SyncPlayer 同步玩家信息
func (u *PlayerUseCase) SyncPlayer(ctx context.Context, data *event.PlayerEvent) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.SyncPlayer")
	defer u.tracingService.SpanEnd(span)

	// 添加玩家信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("player.global_id", data.GlobalPlayerID),
		attribute.String("player.account", data.Account))

	// 查找對應的商戶 (使用快取優化)
	u.tracingService.TraceEvent(span, "Finding merchant with cache")
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
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	level := &entity.Level{}
	if data.PlayerLevel.GlobalPlayerLevelID != "" {
		// 檢查有無Level，沒有則建立
		level, err = u.findOrCreateLevel(ctx, span, data, merchant.ID)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("find or create level: %w", err)
		}
	}

	var lastActiveAt *time.Time
	if !data.LastActiveAt.IsZero() {
		lastActiveAt = &data.LastActiveAt
	}

	player := entity.Player{
		MerchantID:     merchant.ID,
		LevelID:        level.ID,
		GlobalPlayerID: data.GlobalPlayerID,
		Account:        data.Account,
		Email:          &data.Email,
		LastActiveAt:   lastActiveAt,
		CreatedAt:      data.UpdatedAt,
		UpdatedAt:      data.UpdatedAt,
	}
	// 使用領域方法生成API密鑰
	player.RegenerateAPIKey()

	u.tracingService.TraceEvent(span, "Submit player to batch processor")

	// 提交到批次處理器
	resultChannel := u.batchProcessor.SubmitPlayer(&player, data.GlobalMerchantID)

	// 等待批次處理結果
	select {
	case err = <-resultChannel:
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("batch process player: %w", err)
		}
	case <-ctx.Done():
		u.tracingService.RecordSpanError(span, ctx.Err())
		return fmt.Errorf("context cancelled while waiting for batch processing: %w", ctx.Err())
	}

	// 記錄處理完成
	u.tracingService.TraceEvent(span, "Player sync completed successfully")
	u.logger.InfoWithContext(ctx, "Player processed successfully via batch processor",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("account", player.Account),
		u.logger.Any("event_data", data))
	return nil
}

// SyncPlayerFromIdentity 從Identity事件統一同步玩家信息
func (u *PlayerUseCase) SyncPlayerFromIdentity(
	ctx context.Context,
	data *event.IdentityPlayerSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.SyncPlayerFromIdentity")
	defer u.tracingService.SpanEnd(span)

	// 添加玩家信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("player.global_id", data.GlobalPlayerID),
		attribute.String("player.account", data.Account))

	// 查找對應的商戶 (使用快取優化)
	u.tracingService.TraceEvent(span, "Finding merchant with cache")
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
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 解析時間字段
	var lastActiveAt *time.Time
	if data.LastActiveAt != "" {
		if parsedTime, parseErr := time.Parse(time.RFC3339, data.LastActiveAt); parseErr == nil {
			lastActiveAt = &parsedTime
		}
	}

	// 處理玩家等級 - 參考 SyncPlayer 方法的邏輯
	level := &entity.Level{}
	var levelID uint64 = 0

	if data.PlayerLevel != nil && data.PlayerLevel.GlobalPlayerLevelID != "" {
		// 檢查有無Level，沒有則建立
		level, err = u.findOrCreateLevelFromIdentity(ctx, span, data.PlayerLevel, merchant.ID)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("find or create level from identity: %w", err)
		}
		levelID = level.ID
	}

	// 構建玩家實體
	player := &entity.Player{
		MerchantID:     merchant.ID,
		LevelID:        levelID,
		GlobalPlayerID: data.GlobalPlayerID,
		Account:        data.Account,
		Email:          data.Email,
		LastActiveAt:   lastActiveAt,
		APIKey:         uuid.New().String(), // 生成新的API Key
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 更新或創建玩家
	u.tracingService.TraceEvent(span, "Upserting player via batch processor")
	ch := make(chan error, 1)
	playerRequest := PlayerBatchRequest{
		Player:            player,
		GlobalMerchantID:  data.GlobalMerchantID,
		CompletionChannel: ch,
	}

	// 通過批次處理器處理
	select {
	case u.batchProcessor.requestChannel <- &playerRequest:
		// 等待處理結果
		if err = <-ch; err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("batch process player: %w", err)
		}
	case <-ctx.Done():
		u.tracingService.RecordSpanError(span, ctx.Err())
		return fmt.Errorf("context cancelled while waiting for batch processing: %w", ctx.Err())
	}

	// 如果有標籤數據，立即處理（避免競爭條件）
	if len(data.Tags) > 0 {
		u.tracingService.TraceEvent(span, "Processing player tags inline")
		if err = u.syncPlayerTagsInline(ctx, data, merchant); err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to sync player tags inline",
				u.logger.String("global_player_id", data.GlobalPlayerID),
				u.logger.Int("tags_count", len(data.Tags)),
				u.logger.Error("err", err))

			// 標籤同步失敗不應該影響玩家資料同步成功
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnWithContext(ctx, "Player sync completed but tags sync failed - continuing",
				u.logger.String("global_player_id", data.GlobalPlayerID))
		}
	}

	// 記錄處理完成
	u.tracingService.TraceEvent(span, "Player sync from identity completed successfully")
	u.logger.InfoWithContext(ctx, "Player processed successfully from identity event",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("account", player.Account),
		u.logger.Int("tags_processed", len(data.Tags)))
	return nil
}

// syncPlayerTagsInline 在 player 處理完成後立即同步標籤數據，避免競爭條件
func (u *PlayerUseCase) syncPlayerTagsInline(
	ctx context.Context,
	data *event.IdentityPlayerSyncEvent,
	merchant *entity.Merchant,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.syncPlayerTagsInline")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("player.global_id", data.GlobalPlayerID),
		attribute.Int("tags.count", len(data.Tags)))

	// 查詢玩家（剛剛處理完，應該存在）
	u.tracingService.TraceEvent(span, "Finding player for tags sync")
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
	if len(tagItems) == 0 {
		u.logger.InfoWithContext(ctx, "No tags to process")
		return nil
	}

	// 構建標籤實體
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

	// 更新或創建Tags - 使用快取優化
	u.tracingService.TraceEvent(span, "Start filtering tags needing update")
	tagsToUpsert, err := u.filterTagsNeedingUpdate(ctx, tagsToInsert)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("filter tags needing update failed: %w", err)
	}

	if len(tagsToUpsert) == 0 {
		u.logger.InfoWithContext(ctx, "No tags need updating, skipping database operation")
		// 使用原始列表進行後續查詢
	} else {
		// 只對需要更新的標籤執行資料庫操作
		u.tracingService.TraceEvent(span, "Executing batch upsert for filtered tags")
		if err = u.tagRepo.BatchUpsert(ctx, tagsToUpsert); err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("batch upsert tags failed: %w", err)
		}

		// 更新快取 (非同步)
		go u.updateTagCache(context.Background(), tagsToUpsert)
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

// filterTagsNeedingUpdate 使用快取篩選需要更新的標籤
func (u *PlayerUseCase) filterTagsNeedingUpdate(
	ctx context.Context,
	tags []*entity.Tag,
) ([]*entity.Tag, error) {
	if len(tags) == 0 {
		return []*entity.Tag{}, nil
	}

	// 如果快取不可用，回退至所有標籤都需要處理
	if u.cacheManager == nil {
		return tags, nil
	}

	tagsNeedingUpdate := make([]*entity.Tag, 0, len(tags))

	for _, tag := range tags {
		// 從快取查詢現有標籤
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GlobalTagID)

		// 嘗試從快取獲取現有標籤
		cachedData, err := u.cacheManager.Get(ctx, cacheKey)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// 標籤不在快取中，需要處理
				tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
				continue
			}
			// 快取錯誤，為安全起見假設需要處理
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
			continue
		}

		// 解析快取中的標籤
		var cachedTag entity.Tag
		if err = jsoniter.Unmarshal([]byte(cachedData), &cachedTag); err != nil {
			// 快取資料格式錯誤，需要處理
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
			continue
		}

		// 比較標籤是否需要更新
		if u.tagNeedsUpdate(tag, &cachedTag) {
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
		}
	}

	u.logger.InfoWithContext(
		ctx,
		"Filtered tags for update",
		u.logger.Int("original_count", len(tags)),
		u.logger.Int("filtered_count", len(tagsNeedingUpdate)),
		u.logger.Float64(
			"reduction_ratio",
			float64(len(tags)-len(tagsNeedingUpdate))/float64(len(tags))*100,
		),
	)

	return tagsNeedingUpdate, nil
}

// tagNeedsUpdate 比較標籤欄位變動邏輯
func (u *PlayerUseCase) tagNeedsUpdate(newTag, cachedTag *entity.Tag) bool {
	// 比較名稱是否有變動
	if newTag.Name != cachedTag.Name {
		return true
	}

	return false
}

// updateTagCache 使用 Pipeline 批次更新快取
func (u *PlayerUseCase) updateTagCache(ctx context.Context, tags []*entity.Tag) {
	if u.cacheManager == nil || len(tags) == 0 {
		return
	}

	// 獲取 Pipeline 連線
	pipeline, err := u.cacheManager.Pipeline()
	if err != nil {
		u.logger.WarnWithContext(ctx, "Failed to get pipeline, using fallback cache update",
			u.logger.Error("error", err),
		)
		// pipeline 失效回退到逐個更新
		u.updateTagCacheFallback(ctx, tags)
		return
	}

	// 批次準備 SET 命令
	cacheCommands := 0
	for _, tag := range tags {
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GlobalTagID)

		tagData, err := jsoniter.Marshal(tag)
		if err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to marshal tag for cache",
				u.logger.String("global_tag_id", tag.GlobalTagID),
				u.logger.Error("error", err),
			)
			continue
		}

		// 添加 SET 命令到 Pipeline（快取 5 分鐘）
		pipeline.Set(ctx, cacheKey, string(tagData), 5*time.Minute)
		cacheCommands++
	}

	// 執行 Pipeline
	if cacheCommands > 0 {
		_, err = pipeline.Exec(ctx)
		if err != nil {
			u.logger.WarnWithContext(ctx, "Pipeline cache update failed, using fallback",
				u.logger.Int("commands", cacheCommands),
				u.logger.Error("error", err),
			)
			// Pipeline 失敗，回退到逐個更新
			u.updateTagCacheFallback(ctx, tags)
		} else {
			u.logger.InfoWithContext(ctx, "Tag cache batch updated successfully",
				u.logger.Int("updated_tags", cacheCommands),
			)
		}
	}
}

// updateTagCacheFallback Pipeline 失敗時的回退機制
func (u *PlayerUseCase) updateTagCacheFallback(ctx context.Context, tags []*entity.Tag) {
	for _, tag := range tags {
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GlobalTagID)

		tagData, err := jsoniter.Marshal(tag)
		if err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to marshal tag for cache (fallback)",
				u.logger.String("global_tag_id", tag.GlobalTagID),
				u.logger.Error("error", err),
			)
			continue
		}

		// 快取 5 分鐘
		if _, err = u.cacheManager.Set(ctx, cacheKey, string(tagData), 5*time.Minute); err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to set tag cache (fallback)",
				u.logger.String("cache_key", cacheKey),
				u.logger.Error("error", err),
			)
		}
	}
}

// findOrCreateLevelFromIdentity 為 Identity 事件處理 level
func (u *PlayerUseCase) findOrCreateLevelFromIdentity(
	ctx context.Context,
	span trace.Span,
	playerLevel *event.PlayerLevel,
	merchantID uint64,
) (*entity.Level, error) {
	level, err := u.findByGlobalID(ctx, span, playerLevel.GlobalPlayerLevelID)
	if err == nil {
		return level, nil
	}

	if !errors.Is(err, errmsg.ErrRepoLevelNotFound) {
		return nil, fmt.Errorf("find level: %w", err)
	}

	u.tracingService.TraceEvent(span, "Creating new level from identity")
	newLevel := &entity.Level{
		GlobalPlayerLevelID: playerLevel.GlobalPlayerLevelID,
		MerchantID:          merchantID,
		Name:                playerLevel.Name,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err = u.levelRepo.Upsert(ctx, newLevel); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("upsert level: %w", err)
	}

	// 取得新的level
	level, err = u.findByGlobalID(ctx, span, playerLevel.GlobalPlayerLevelID)
	if err != nil {
		return level, nil
	}
	return level, nil
}

func (u *PlayerUseCase) findOrCreateLevel(
	ctx context.Context,
	span trace.Span,
	data *event.PlayerEvent,
	merchantID uint64,
) (*entity.Level, error) {
	level, err := u.findByGlobalID(ctx, span, data.PlayerLevel.GlobalPlayerLevelID)
	if err == nil {
		return level, nil
	}

	if !errors.Is(err, errmsg.ErrRepoLevelNotFound) {
		return nil, fmt.Errorf("find level: %w", err)
	}

	u.tracingService.TraceEvent(span, "Upsert player")
	newLevel := &entity.Level{
		GlobalPlayerLevelID: data.PlayerLevel.GlobalPlayerLevelID,
		MerchantID:          merchantID,
		Name:                data.PlayerLevel.Name,
		CreatedAt:           data.UpdatedAt,
		UpdatedAt:           data.UpdatedAt,
	}

	if err = u.levelRepo.Upsert(ctx, newLevel); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("upsert level: %w", err)
	}

	// 取得新的level
	level, err = u.findByGlobalID(ctx, span, data.PlayerLevel.GlobalPlayerLevelID)
	if err != nil {
		return level, nil
	}
	return level, nil
}

func (u *PlayerUseCase) findByGlobalID(
	ctx context.Context,
	span trace.Span,
	globalPlayerLevelID string,
) (*entity.Level, error) {
	level, err := u.levelRepo.FindByGlobalID(ctx, globalPlayerLevelID)
	if err == nil {
		return level, nil
	}

	if !errors.Is(err, errmsg.ErrRepoLevelNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find level: %w", err)
	}
	return nil, errmsg.ErrRepoLevelNotFound
}

// 發布玩家同步事件
func (u *PlayerUseCase) publishPlayerSyncEvent(
	ctx context.Context,
	player *entity.Player,
	globalMerchantID string,
) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 記錄發布事件開始
	u.tracingService.TraceEvent(span, "Preparing player sync event for KDS")

	// 構建事件數據
	var lastActiveAt string
	if player.LastActiveAt != nil {
		lastActiveAt = player.LastActiveAt.Format(time.RFC3339)
	}

	var deletedAt string
	if player.DeletedAt != nil {
		deletedAt = player.DeletedAt.Format(time.RFC3339)
	}

	syncEvent := event.IdentityPlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalPlayerID:   player.GlobalPlayerID,
		ID:               player.ID,
		MerchantID:       player.MerchantID,
		APIKey:           player.APIKey,
		Account:          player.Account,
		Email:            player.Email,
		LastActiveAt:     lastActiveAt,
		CreatedAt:        player.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        player.UpdatedAt.Format(time.RFC3339),
		DeletedAt:        deletedAt,
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.player.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     u.tracingService.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	// 添加事件信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	// 發布事件
	if err := u.eventProducer.PublishPlayerSync(ctx, &cloudEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("publish player sync: %w", err)
	}

	// 記錄事件發布成功
	u.tracingService.TraceEvent(span, "Player sync event published successfully")

	u.logger.InfoLog("Player sync event published",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetPlayerByID 通過ID獲取玩家
func (u *PlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*dto.PlayerResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.GetPlayerByID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
		attribute.Int64("merchant.id", int64(player.MerchantID)))

	// 轉換為 DTO
	response := &dto.PlayerResponse{
		ID:           player.ID,
		GlobalID:     player.GlobalPlayerID,
		MerchantID:   player.MerchantID,
		Username:     player.Account,
		Level:        0,          // TODO: 取得玩家等級
		Tags:         []string{}, // TODO: 取得玩家標籤
		LastActiveAt: player.LastActiveAt,
		CreatedAt:    player.CreatedAt,
		UpdatedAt:    player.UpdatedAt,
	}

	return response, nil
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
func (u *PlayerUseCase) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.PlayerResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.GetPlayerByGlobalID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("player.global_id", globalID))

	player, err := u.playerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("player.id", int64(player.ID)),
		attribute.String("player.account", player.Account),
		attribute.Int64("merchant.id", int64(player.MerchantID)))

	// 轉換為 DTO
	response := &dto.PlayerResponse{
		ID:           player.ID,
		GlobalID:     player.GlobalPlayerID,
		MerchantID:   player.MerchantID,
		Username:     player.Account,
		Level:        0,          // TODO: 取得玩家等級
		Tags:         []string{}, // TODO: 取得玩家標籤
		LastActiveAt: player.LastActiveAt,
		CreatedAt:    player.CreatedAt,
		UpdatedAt:    player.UpdatedAt,
	}

	return response, nil
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
func (u *PlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "PlayerUseCase.UpdatePlayerLastActive")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	// 查找玩家
	u.tracingService.TraceEvent(span, "Finding player")
	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account))

	// 使用領域方法更新最後活躍時間
	player.UpdateLastActive()

	// 更新玩家
	u.tracingService.TraceEvent(span, "Updating player last active time")
	if err := u.playerRepo.Update(ctx, player); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update player: %w", err)
	}

	// 記錄更新成功
	u.tracingService.TraceEvent(span, "Player last active time updated successfully",
		attribute.String("last_active_at", player.LastActiveAt.Format(time.RFC3339)))

	return nil
}

// syncPlayerTagsWithDifference 使用快取查詢現有標籤並計算差異
func (u *PlayerUseCase) syncPlayerTagsWithDifference(
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
func (u *PlayerUseCase) difference(a, b []uint64) []uint64 {
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

func (u *PlayerUseCase) executeLocked(
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

// StartBatchProcessor 啟動批次處理器
func (u *PlayerUseCase) StartBatchProcessor(ctx context.Context) error {
	return u.batchProcessor.Start(ctx)
}

// StopBatchProcessor 停止批次處理器
func (u *PlayerUseCase) StopBatchProcessor(ctx context.Context) error {
	return u.batchProcessor.Stop(ctx)
}
