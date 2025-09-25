package manager

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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ManagerUseCase 管理員用例
type ManagerUseCase struct {
	managerRepo    repository.ManagerRepository
	merchantRepo   repository.MerchantRepository
	eventProducer  service.EventProducer
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

// NewManagerUseCase 創建管理員用例
func NewManagerUseCase(
	managerRepo repository.ManagerRepository,
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) inbound.ManagerUseCase {
	return &ManagerUseCase{
		managerRepo:    managerRepo,
		merchantRepo:   merchantRepo,
		eventProducer:  eventProducer,
		logger:         logger,
		tracingService: tracingService,
	}
}

// SyncManager 同步管理員信息
func (u *ManagerUseCase) SyncManager(ctx context.Context, data *event.ManagerEvent) error {
	ctx, span := u.tracingService.StartSpan(ctx, "ManagerUseCase.SyncManager")
	defer u.tracingService.SpanEnd(span)

	// 添加管理員信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("manager.global_id", data.GlobalManagerID),
		attribute.String("manager.account", data.Account))

	// 查找對應的商戶
	u.tracingService.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	manager := &entity.Manager{
		MerchantID:      merchant.ID,
		GlobalManagerID: data.GlobalManagerID,
		Account:         data.Account,
		Email:           &data.Email,
		CreatedAt:       data.UpdatedAt,
		UpdatedAt:       data.UpdatedAt,
		DeletedAt: func() *time.Time {
			if data.DeletedAt == "" {
				return nil
			}
			return &data.UpdatedAt
		}(),
	}

	u.tracingService.TraceEvent(span, "Upsert manager")
	if err = u.managerRepo.Upsert(ctx, manager); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert manager: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Manager upserted successfully",
		u.logger.String("global_id", manager.GlobalManagerID),
		u.logger.String("account", manager.Account))
	u.tracingService.TraceEvent(span, "Database operation completed")
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account))
	return nil
}

// 發布管理員同步事件
func (u *ManagerUseCase) publishManagerSyncEvent(
	ctx context.Context,
	manager *entity.Manager,
	globalMerchantID string,
) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 記錄發布事件開始
	u.tracingService.TraceEvent(span, "Preparing manager sync event for KDS")

	// 構建事件數據
	var deletedAt string
	if manager.DeletedAt != nil {
		deletedAt = manager.DeletedAt.Format(time.RFC3339)
	}

	syncEvent := event.IdentityManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalManagerID:  manager.GlobalManagerID,
		ID:               manager.ID,
		MerchantID:       manager.MerchantID,
		Account:          manager.Account,
		Email:            manager.Email,
		CreatedAt:        manager.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        manager.UpdatedAt.Format(time.RFC3339),
		DeletedAt:        deletedAt,
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.manager.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "manager_sync",
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
	if err := u.eventProducer.PublishManagerSync(ctx, &cloudEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync: %w", err)
	}

	// 記錄事件發布成功
	u.tracingService.TraceEvent(span, "Manager sync event published successfully")

	u.logger.InfoLog("Manager sync event published",
		u.logger.String("global_id", manager.GlobalManagerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetManagerByID 通過ID獲取管理員
func (u *ManagerUseCase) GetManagerByID(
	ctx context.Context,
	id uint64,
) (*dto.ManagerResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "ManagerUseCase.GetManagerByID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("manager.id", int64(id)))

	manager, err := u.managerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account),
		attribute.Int64("merchant.id", int64(manager.MerchantID)),
	)

	// 轉換為 DTO
	response := &dto.ManagerResponse{
		ID:       manager.ID,
		GlobalID: manager.GlobalManagerID,
		Name:     manager.Account,
		Email:    "",
		Role:     "manager",
	}
	if manager.Email != nil {
		response.Email = *manager.Email
	}

	return response, nil
}

// GetManagerByGlobalID 通過全局ID獲取管理員
func (u *ManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.ManagerResponse, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracingService.StartSpan(ctx, "ManagerUseCase.GetManagerByGlobalID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("manager.global_id", globalID))

	manager, err := u.managerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("manager.id", int64(manager.ID)),
		attribute.String("manager.account", manager.Account),
		attribute.Int64("merchant.id", int64(manager.MerchantID)),
	)

	// 轉換為 DTO
	response := &dto.ManagerResponse{
		ID:       manager.ID,
		GlobalID: manager.GlobalManagerID,
		Name:     manager.Account,
		Email:    "",
		Role:     "manager",
	}
	if manager.Email != nil {
		response.Email = *manager.Email
	}

	return response, nil
}
