package scheduler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/jobport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"time"
)

// JobWrapper 包裝job執行邏輯，添加追蹤和日誌
type JobWrapper struct {
	job    jobport.ScheduledJob
	logger infraport.Logger
}

// run JobWrapper的執行方法
func (w *JobWrapper) run() {
	jobID := uuid.NewString()
	jobName := w.job.GetName()

	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, fmt.Sprintf("ScheduledJob:%s", jobName))
	defer tracing.SpanEnd(span)

	w.logger.InfoWithContext(ctx, "Starting scheduled job execution",
		w.logger.String("job_name", jobName),
		w.logger.String("job_id", jobID))

	tracing.TraceEvent(span, fmt.Sprintf("Executing scheduled job: %s", jobName))

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
