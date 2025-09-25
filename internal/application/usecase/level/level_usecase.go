package level

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
)

type LevelUseCase struct {
	levelRepo      repository.LevelRepository
	merchantRepo   repository.MerchantRepository
	eventProducer  service.EventProducer
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

func NewLevelUseCase(
	levelRepo repository.LevelRepository,
	merchantRepo repository.MerchantRepository,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) inbound.PlayerLevelUseCase {
	return &LevelUseCase{
		levelRepo:      levelRepo,
		merchantRepo:   merchantRepo,
		logger:         logger,
		tracingService: tracingService,
	}
}

func (u *LevelUseCase) SyncPlayerLevel(
	ctx context.Context,
	data *event.IdentityPlayerLevelSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "LevelUseCase.SyncPlayerLevel")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}
	level := &entity.Level{
		GlobalPlayerLevelID: data.GlobalPlayerLevelID,
		Name:                data.Name,
		MerchantID:          merchant.ID,
		CreatedAt:           data.UpdatedAt,
		UpdatedAt:           data.UpdatedAt,
	}
	err = u.levelRepo.Upsert(ctx, level)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert player level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert player level completed", u.logger.Any("level", data))
	u.tracingService.TraceEvent(span, "Upsert completed")
	return nil
}

func (u *LevelUseCase) GetLevelsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*dto.LevelListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "LevelUseCase.GetLevelsByMerchantID")
	defer u.tracingService.SpanEnd(span)

	levels, err := u.levelRepo.FindByMerchantID(ctx, merchantID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find levels by merchant ID: %w", err)
	}

	levelResponses := make([]dto.LevelResponse, len(levels))
	for i, level := range levels {
		levelResponses[i] = dto.LevelResponse{
			ID:       level.ID,
			Name:     level.Name,
			GlobalID: level.GlobalPlayerLevelID,
		}
	}

	return &dto.LevelListResponse{
		Levels: levelResponses,
	}, nil
}
