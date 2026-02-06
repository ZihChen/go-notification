package merchant

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MerchantUseCase 商戶用例
type MerchantUseCase struct {
	merchantRepo   repository.MerchantRepository
	eventProducer  service.EventProducer
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

// NewMerchantUseCase 創建商戶用例
func NewMerchantUseCase(
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) inbound.MerchantUseCase {
	return &MerchantUseCase{
		merchantRepo:   merchantRepo,
		eventProducer:  eventProducer,
		logger:         logger,
		tracingService: tracingService,
	}
}

// SyncMerchant 同步商戶信息
func (u *MerchantUseCase) SyncMerchant(ctx context.Context, data *event.MerchantEvent) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MerchantUseCase.SyncMerchant")
	defer u.tracingService.SpanEnd(span)

	// 添加商戶信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("global_merchant_id", data.GlobalMerchantID),
		attribute.String("merchant.name", data.Name),
	)

	merchant := &entity.Merchant{
		GlobalMerchantID: data.GlobalMerchantID,
		Name:             data.Name,
		DisplayName:      data.DisplayName,
		APIKey:           uuid.New().String(),
		CreatedAt:        data.UpdatedAt,
		UpdatedAt:        data.UpdatedAt,
	}

	u.tracingService.TraceEvent(span, "Upsert merchant")
	if err := u.merchantRepo.Upsert(ctx, merchant); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert merchant: %w", err)
	}

	// 記錄處理完成
	u.tracingService.TraceEvent(span, "Merchant sync completed successfully")
	u.logger.InfoWithContext(ctx, "Merchant upserted successfully",
		u.logger.String("global_id", merchant.GlobalMerchantID),
		u.logger.String("name", merchant.Name),
		u.logger.Any("event_data", data))
	return nil
}

// 發布商戶同步事件
func (u *MerchantUseCase) publishMerchantSyncEvent(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 記錄發布事件開始
	u.tracingService.TraceEvent(span, "Preparing merchant sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityMerchantSyncEvent{
		GlobalMerchantID: merchant.GlobalMerchantID,
		ID:               merchant.ID,
		Name:             merchant.Name,
		DisplayName:      merchant.DisplayName,
		APIKey:           merchant.APIKey,
		CreatedAt:        merchant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        merchant.UpdatedAt.Format(time.RFC3339),
	}

	if merchant.DeletedAt != nil {
		syncEvent.DeletedAt = merchant.DeletedAt.Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.merchant.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     u.tracingService.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	// 添加事件信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishMerchantSync(ctx, &cloudEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("publish merchant sync: %w", err)
	}

	// 記錄事件發布成功
	u.tracingService.TraceEvent(span, "Merchant sync event published successfully")

	u.logger.InfoLog("Merchant sync event published",
		u.logger.String("global_id", merchant.GlobalMerchantID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetMerchantByID 通過ID獲取商戶
func (u *MerchantUseCase) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*dto.MerchantResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "MerchantUseCase.GetMerchantByID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("merchant.id", int64(id)))

	merchant, err := u.merchantRepo.FindByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 添加商戶信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("global_merchant_id", merchant.GlobalMerchantID),
		attribute.String("merchant.name", merchant.Name),
	)

	// 轉換為 DTO
	response := &dto.MerchantResponse{
		ID:            merchant.ID,
		GlobalID:      merchant.GlobalMerchantID,
		Name:          merchant.Name,
		Status:        "active",
		Configuration: "",
	}

	return response, nil
}

func (u *MerchantUseCase) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.MerchantResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "MerchantUseCase.GetMerchantByGlobalID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("global_merchant_id", globalID))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 添加商戶信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("merchant.id", int64(merchant.ID)),
		attribute.String("merchant.name", merchant.Name),
	)

	// 轉換為 DTO
	response := &dto.MerchantResponse{
		ID:            merchant.ID,
		GlobalID:      merchant.GlobalMerchantID,
		Name:          merchant.Name,
		Status:        "active",
		Configuration: "",
	}

	return response, nil
}
