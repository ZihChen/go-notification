package level

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
)

type LevelUseCase struct {
	levelRepo     repository.LevelRepository
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
}

func NewLevelUseCase(
	levelRepo repository.LevelRepository,
	merchantRepo repository.MerchantRepository,
	logger infrastructure.Logger,
) inbound.PlayerLevelUseCase {
	return &LevelUseCase{
		levelRepo:    levelRepo,
		merchantRepo: merchantRepo,
		logger:       logger,
	}
}

func (u *LevelUseCase) SyncPlayerLevel(
	ctx context.Context,
	data *event.IdentityPlayerLevelSyncEvent,
) error {
	ctx, span := tracing.StartSpan(ctx, "LevelUseCase.SyncPlayerLevel")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
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
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert player level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert player level completed", u.logger.Any("level", data))
	tracing.TraceEvent(span, "Upsert completed")
	return nil
}
