package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	jobport "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/job"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
)

// JobWrapper 包裝job執行邏輯，添加追蹤、日誌和分布式鎖
type JobWrapper struct {
	job          jobport.ScheduledJob
	logger       infrastructure.Logger
	redisManager *redisCache.Manager
}

// run JobWrapper的執行方法
func (w *JobWrapper) run() {
	jobID := uuid.NewString()
	jobName := w.job.GetName()

	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, fmt.Sprintf("ScheduledJob:%s", jobName))
	defer tracing.SpanEnd(span)

	// 分布式鎖 key，使用 job 名稱
	mutexKey := fmt.Sprintf("scheduler:job:%s", jobName)

	// 如果 RedisManager 不為 nil，則使用分布式鎖
	if w.redisManager != nil {
		// 獲取分布式鎖
		mutex, err := w.redisManager.GetMutexWithOption(mutexKey,
			redsync.WithExpiry(5*time.Second),            // 鎖的過期時間 5 分鐘
			redsync.WithTries(1),                         // 只試一次，不重試
			redsync.WithRetryDelay(100*time.Millisecond), // 重試間隔
		)
		if err != nil {
			tracing.RecordSpanError(span, err)
			w.logger.ErrorWithContext(ctx, "Failed to get distributed lock",
				w.logger.String("job_name", jobName),
				w.logger.String("job_id", jobID),
				w.logger.String("mutex_key", mutexKey),
				w.logger.Error("err", err))
			return
		}

		// 獲取鎖
		if err = mutex.Lock(); err != nil {
			w.logger.InfoWithContext(
				ctx,
				"Failed to acquire distributed lock, another instance may be executing this job",
				w.logger.String("job_name", jobName),
				w.logger.String("job_id", jobID),
				w.logger.String("mutex_key", mutexKey),
				w.logger.Error("err", err),
			)
			return // 不報錯，因為另一個實例可能正在執行
		}
		defer func() {
			unlocked, unlockErr := mutex.Unlock()
			if unlockErr != nil || !unlocked {
				w.logger.WarnWithContext(ctx, "Failed to release distributed lock",
					w.logger.String("job_name", jobName),
					w.logger.String("job_id", jobID),
					w.logger.String("mutex_key", mutexKey),
					w.logger.Bool("unlocked", unlocked),
					w.logger.Error("err", unlockErr))
			}
		}()

		w.logger.InfoWithContext(ctx, "Acquired distributed lock for scheduled job",
			w.logger.String("job_name", jobName),
			w.logger.String("job_id", jobID),
			w.logger.String("mutex_key", mutexKey))
	} else {
		w.logger.InfoWithContext(ctx, "Redis manager not configured, proceeding without distributed lock",
			w.logger.String("job_name", jobName),
			w.logger.String("job_id", jobID))
	}

	w.logger.InfoWithContext(ctx, "Starting scheduled job execution",
		w.logger.String("job_name", jobName),
		w.logger.String("job_id", jobID))

	tracing.TraceEvent(
		span,
		fmt.Sprintf("Executing scheduled job with distributed lock: %s", jobName),
	)

	startTime := time.Now()

	// 執行任務
	if err := w.job.Execute(ctx); err != nil {
		duration := time.Since(startTime)
		tracing.RecordSpanError(span, err)
		w.logger.ErrorWithContext(ctx, "Scheduled job execution failed",
			w.logger.String("job_name", jobName),
			w.logger.String("job_id", jobID),
			w.logger.String("duration", duration.String()),
			w.logger.Error("err", err))
		return
	}

	duration := time.Since(startTime)
	tracing.TraceEvent(span, fmt.Sprintf("Scheduled job completed: %s", jobName))

	w.logger.InfoWithContext(ctx, "Scheduled job execution completed successfully",
		w.logger.String("job_name", jobName),
		w.logger.String("job_id", jobID),
		w.logger.String("duration", duration.String()))
}
