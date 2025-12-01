package job

import (
	"context"
)

// ScheduledJob 排程任務的介面
type ScheduledJob interface {
	// Execute 執行任務邏輯
	Execute(ctx context.Context) error
	// GetName 任務名稱
	GetName() string
	// GetCron 任務執行時間：Cron表達式
	GetCron() string
}
