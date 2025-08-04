package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ManagerUseCase 管理員用例
type ManagerUseCase struct {
	managerRepo   repositoryport.ManagerRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
}

// NewManagerUseCase 創建管理員用例
func NewManagerUseCase(
	managerRepo repositoryport.ManagerRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
) usecaseport.ManagerUseCase {
	return &ManagerUseCase{
		managerRepo:   managerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncManager 同步管理員信息
func (u *ManagerUseCase) SyncManager(ctx context.Context, data *event.ManagerEvent) error {
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.SyncManager")
	defer tracing.SpanEnd(span)

	// 添加管理員信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("manager.global_id", data.GlobalManagerID),
		attribute.String("manager.account", data.Account))

	// 查找對應的商戶
	tracing.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 查找管理員是否存在
	tracing.TraceEvent(span, "Checking if manager exists")
	existing, err := u.managerRepo.FindByGlobalID(ctx, data.GlobalManagerID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoManagerNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find manager: %w", err)
	}

	// 設置email
	var email *string
	if data.Email != "" {
		email = &data.Email
	}

	// 創建或更新管理員
	var manager entity.Manager
	if errors.Is(err, errmsg.ErrRepoManagerNotFound) {
		// 創建新管理員
		tracing.TraceEvent(span, "Creating new manager")
		manager = entity.Manager{
			MerchantID:      merchant.ID,
			GlobalManagerID: data.GlobalManagerID,
			Account:         data.Account,
			Email:           email,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err = u.managerRepo.FirstOrCreate(ctx, &manager); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("create manager: %w", err)
		}
		u.logger.InfoLog("Manager created",
			u.logger.String("global_id", manager.GlobalManagerID),
			u.logger.String("account", manager.Account))
	} else {
		// 更新現有管理員
		tracing.TraceEvent(span, "Updating existing manager")

		nowTime := time.Now()

		manager = *existing
		manager.Account = data.Account
		manager.Email = email
		manager.UpdatedAt = nowTime
		manager.DeletedAt = func() *time.Time {
			if data.DeletedAt == "" {
				return nil
			}
			return &nowTime
		}()

		if err = u.managerRepo.Update(ctx, &manager); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("update manager: %w", err)
		}
		u.logger.InfoLog("Manager updated",
			u.logger.String("global_id", manager.GlobalManagerID),
			u.logger.String("account", manager.Account))
	}

	// 記錄資料庫操作完成
	tracing.TraceEvent(span, "Database operation completed")
	tracing.RecordSpanAttributes(span, attribute.Int64("manager.id", int64(manager.ID)))

	// 記錄處理完成
	tracing.TraceEvent(span, "Manager sync completed successfully")

	return nil
}

// 發布管理員同步事件
func (u *ManagerUseCase) publishManagerSyncEvent(
	ctx context.Context,
	manager *entity.Manager,
	globalMerchantID, traceParent string,
) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 記錄發布事件開始
	tracing.TraceEvent(span, "Preparing manager sync event for KDS")

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
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	// 添加事件信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishManagerSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Manager sync event published successfully")

	u.logger.InfoLog("Manager sync event published",
		u.logger.String("global_id", manager.GlobalManagerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetManagerByID 通過ID獲取管理員
func (u *ManagerUseCase) GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("manager.id", int64(id)))

	manager, err := u.managerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account),
		attribute.Int64("merchant.id", int64(manager.MerchantID)),
	)

	return manager, nil
}

// GetManagerByGlobalID 通過全局ID獲取管理員
func (u *ManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByGlobalID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("manager.global_id", globalID))

	manager, err := u.managerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.Int64("manager.id", int64(manager.ID)),
		attribute.String("manager.account", manager.Account),
		attribute.Int64("merchant.id", int64(manager.MerchantID)),
	)

	return manager, nil
}
