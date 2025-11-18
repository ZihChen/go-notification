package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// FailedTaskEventRepository 失敗任務事件Repository接口
type FailedTaskEventRepository interface {
	// Create 創建失敗任務事件記錄
	Create(ctx context.Context, failedTask *entity.FailedTaskEvent) error
}
