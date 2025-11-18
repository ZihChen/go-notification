package inbound

import (
	"context"
)

// FailedTaskEventUseCase 失敗任務事件UseCase接口
type FailedTaskEventUseCase interface {
	// CreateFailedTaskEventWithRedisInfo 記錄帶Redis信息的失敗任務事件
	CreateFailedTaskEventWithRedisInfo(
		ctx context.Context,
		taskID, taskType, queueName, payload, errorMessage, redisKey, redisState string,
		retryCount int,
	) error
}
