package player

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo    repository.PlayerRepository
	merchantRepo  repository.MerchantRepository
	levelRepo     repository.LevelRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	levelRepo repository.LevelRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
) inbound.PlayerUseCase {
	return &PlayerUseCase{
		playerRepo:    playerRepo,
		merchantRepo:  merchantRepo,
		levelRepo:     levelRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncPlayer 同步玩家信息
func (u *PlayerUseCase) SyncPlayer(ctx context.Context, data *event.PlayerEvent) error {
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.SyncPlayer")
	defer tracing.SpanEnd(span)

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("player.global_id", data.GlobalPlayerID),
		attribute.String("player.account", data.Account))

	// 查找對應的商戶
	tracing.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	level := &entity.Level{}
	if data.PlayerLevel.GlobalPlayerLevelID != "" {
		// 檢查有無Level，沒有則建立
		level, err = u.findOrCreateLevel(ctx, span, data, merchant.ID)
		if err != nil {
			tracing.RecordSpanError(span, err)
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
		APIKey:         uuid.New().String(), // 生成新的API密鑰
		Account:        data.Account,
		Email:          &data.Email,
		LastActiveAt:   lastActiveAt,
		CreatedAt:      data.UpdatedAt,
		UpdatedAt:      data.UpdatedAt,
	}

	tracing.TraceEvent(span, "Upsert player")
	if err = u.playerRepo.Upsert(ctx, &player); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert player: %w", err)
	}
	// 記錄處理完成
	tracing.TraceEvent(span, "Player sync completed successfully")
	u.logger.InfoLog("Player upserted successfully",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("account", player.Account),
		u.logger.Any("event_data", data))
	return nil
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

	tracing.TraceEvent(span, "Upsert player")
	newLevel := &entity.Level{
		GlobalPlayerLevelID: data.PlayerLevel.GlobalPlayerLevelID,
		MerchantID:          merchantID,
		Name:                data.PlayerLevel.Name,
		CreatedAt:           data.UpdatedAt,
		UpdatedAt:           data.UpdatedAt,
	}

	if err = u.levelRepo.Upsert(ctx, newLevel); err != nil {
		tracing.RecordSpanError(span, err)
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
		tracing.RecordSpanError(span, err)
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
	tracing.TraceEvent(span, "Preparing player sync event for KDS")

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
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	// 添加事件信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	// 發布事件
	if err := u.eventProducer.PublishPlayerSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Player sync event published successfully")

	u.logger.InfoLog("Player sync event published",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetPlayerByID 通過ID獲取玩家
func (u *PlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*dto.PlayerResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
		attribute.Int64("merchant.id", int64(player.MerchantID)))

	// 轉換為 DTO
	response := &dto.PlayerResponse{
		ID:           player.ID,
		GlobalID:     player.GlobalPlayerID,
		MerchantID:   player.MerchantID,
		Username:     player.Account,
		Level:        0, // TODO: 取得玩家等級
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
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByGlobalID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("player.global_id", globalID))

	player, err := u.playerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.Int64("player.id", int64(player.ID)),
		attribute.String("player.account", player.Account),
		attribute.Int64("merchant.id", int64(player.MerchantID)))

	// 轉換為 DTO
	response := &dto.PlayerResponse{
		ID:           player.ID,
		GlobalID:     player.GlobalPlayerID,
		MerchantID:   player.MerchantID,
		Username:     player.Account,
		Level:        0, // TODO: 取得玩家等級
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
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.UpdatePlayerLastActive")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	// 查找玩家
	tracing.TraceEvent(span, "Finding player")
	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account))

	// 更新最後活躍時間
	now := time.Now()
	player.LastActiveAt = &now
	player.UpdatedAt = now

	// 更新玩家
	tracing.TraceEvent(span, "Updating player last active time")
	if err := u.playerRepo.Update(ctx, player); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("update player: %w", err)
	}

	// 記錄更新成功
	tracing.TraceEvent(span, "Player last active time updated successfully",
		attribute.String("last_active_at", now.Format(time.RFC3339)))

	return nil
}
