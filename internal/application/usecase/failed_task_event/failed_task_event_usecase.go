package usecase

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
)

type FailedTaskEventUseCase struct {
	failedTaskRepo repository.FailedTaskEventRepository
	logger         infrastructure.Logger
}

// NewFailedTaskEventUseCase 創建失敗任務事件UseCase
func NewFailedTaskEventUseCase(
	failedTaskRepo repository.FailedTaskEventRepository,
	logger infrastructure.Logger,
) inbound.FailedTaskEventUseCase {
	return &FailedTaskEventUseCase{
		failedTaskRepo: failedTaskRepo,
		logger:         logger,
	}
}

// CreateFailedTaskEventWithRedisInfo 記錄帶Redis信息的失敗任務事件
func (u *FailedTaskEventUseCase) CreateFailedTaskEventWithRedisInfo(
	ctx context.Context,
	taskID, taskType, queueName, payload, errorMessage, redisKey, redisState string,
	retryCount int,
) error {
	// 創建失敗任務事件實體
	failedTask := entity.NewFailedTaskEventWithRedisInfo(
		taskID,
		taskType,
		queueName,
		payload,
		errorMessage,
		redisKey,
		redisState,
		retryCount,
	)

	// 驗證實體有效性
	if !failedTask.IsValid() {
		u.logger.WarnLog("Invalid failed task event",
			u.logger.String("task_id", taskID),
			u.logger.String("task_type", taskType),
			u.logger.String("queue_name", queueName))
		return nil // 不返回錯誤，避免影響主流程
	}

	// 儲存到資料庫
	if err := u.failedTaskRepo.Create(ctx, failedTask); err != nil {
		u.logger.ErrorLog("Failed to create failed task event",
			u.logger.String("task_id", taskID),
			u.logger.String("task_type", taskType),
			u.logger.String("queue_name", queueName),
			u.logger.Error("err", err))
		return err
	}

	u.logger.InfoLog("Failed task event created successfully",
		u.logger.String("task_id", taskID),
		u.logger.String("task_type", taskType),
		u.logger.String("queue_name", queueName),
		u.logger.Int("retry_count", retryCount))

	return nil
}
