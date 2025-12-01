package repository

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

type FailedTaskEventRepository struct {
	db *gorm.DB
}

// NewFailedTaskEventRepository 創建失敗任務事件Repository
func NewFailedTaskEventRepository(db *gorm.DB) repository.FailedTaskEventRepository {
	return &FailedTaskEventRepository{db: db}
}

// Create 創建失敗任務事件記錄
func (r *FailedTaskEventRepository) Create(
	ctx context.Context,
	failedTask *entity.FailedTaskEvent,
) error {
	model := &models.FailedTaskEvent{
		TaskID:       failedTask.GetTaskID(),
		TaskType:     failedTask.GetTaskType(),
		QueueName:    failedTask.GetQueueName(),
		Payload:      failedTask.GetPayload(),
		ErrorMessage: failedTask.GetErrorMessage(),
		RetryCount:   failedTask.GetRetryCount(),
		FailedAt:     failedTask.GetFailedAt(),
		RedisKey:     failedTask.GetRedisKey(),
		RedisState:   failedTask.GetRedisState(),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create failed task event failed: %w", err)
	}

	// 更新實體的ID和時間戳
	failedTask.SetID(uint(model.ID))
	failedTask.SetCreatedAt(model.CreatedAt)
	failedTask.SetUpdatedAt(model.UpdatedAt)

	return nil
}
