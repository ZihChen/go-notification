package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/robfig/cron/v3"
)

// ScheduledJob 代表一個排程任務的介面
type ScheduledJob interface {
	Execute(ctx context.Context) error
	GetName() string
}

// JobWrapper 包裝job執行邏輯，添加追蹤和日誌
type JobWrapper struct {
	job    ScheduledJob
	logger infraport.Logger
}

// SchedulerHandler 排程處理器
type SchedulerHandler struct {
	logger infraport.Logger
	jobs   []ScheduledJobConfig
}

// ScheduledJobConfig 排程任務配置
type ScheduledJobConfig struct {
	Name     string        // 任務名稱
	Job      ScheduledJob  // 任務執行邏輯
	Schedule string        // Cron表達式 (支援秒級: "* * * * * *")
	Interval time.Duration // 固定間隔(可選，與Schedule二選一)
}

// NewSchedulerHandler 創建排程處理器
func NewSchedulerHandler(logger infraport.Logger) *SchedulerHandler {
	return &SchedulerHandler{
		logger: logger,
		jobs:   make([]ScheduledJobConfig, 0),
	}
}

// RegisterJob 註冊排程任務
func (h *SchedulerHandler) RegisterJob(config ScheduledJobConfig) error {
	if config.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if config.Job == nil {
		return fmt.Errorf("job implementation is required")
	}

	// 驗證schedule或interval至少有一個
	if config.Schedule == "" && config.Interval == 0 {
		return fmt.Errorf("either schedule or interval must be specified")
	}

	// 如果同時指定了schedule和interval，優先使用schedule
	if config.Schedule != "" && config.Interval != 0 {
		h.logger.WarnLog("Both schedule and interval specified, using schedule",
			h.logger.String("job_name", config.Name),
			h.logger.String("schedule", config.Schedule),
			h.logger.String("interval", config.Interval.String()))
		config.Interval = 0 // 清除interval
	}

	h.jobs = append(h.jobs, config)
	h.logger.InfoLog("Registered scheduled job",
		h.logger.String("job_name", config.Name),
		h.logger.String("schedule", config.Schedule),
		h.logger.String("interval", config.Interval.String()))

	return nil
}

// RegisterJobs 註冊所有排程任務到cron調度器
func (h *SchedulerHandler) RegisterJobs(cronManager *cron.Cron) {
	for _, jobConfig := range h.jobs {
		wrapper := &JobWrapper{
			job:    jobConfig.Job,
			logger: h.logger,
		}

		var scheduleStr string
		var err error

		if jobConfig.Schedule != "" {
			// 使用Cron表達式
			scheduleStr = jobConfig.Schedule
			_, err = cronManager.AddFunc(scheduleStr, wrapper.run)
		} else if jobConfig.Interval > 0 {
			// 使用固定間隔，轉換為Cron表達式
			scheduleStr = h.intervalToCron(jobConfig.Interval)
			_, err = cronManager.AddFunc(scheduleStr, wrapper.run)
		}

		if err != nil {
			h.logger.ErrorLog("Failed to register scheduled job",
				h.logger.String("job_name", jobConfig.Name),
				h.logger.String("schedule", scheduleStr),
				h.logger.Error("err", err))
		} else {
			h.logger.InfoLog("Successfully registered scheduled job",
				h.logger.String("job_name", jobConfig.Name),
				h.logger.String("schedule", scheduleStr))
		}
	}
}

// RegisterJobWithInterval 便利方法：使用固定間隔註冊任務
func (h *SchedulerHandler) RegisterJobWithInterval(
	name string,
	job ScheduledJob,
	interval time.Duration,
) error {
	return h.RegisterJob(ScheduledJobConfig{
		Name:     name,
		Job:      job,
		Interval: interval,
	})
}

// RegisterJobWithSchedule 便利方法：使用Cron表達式註冊任務
func (h *SchedulerHandler) RegisterJobWithSchedule(
	name string,
	job ScheduledJob,
	schedule string,
) error {
	return h.RegisterJob(ScheduledJobConfig{
		Name:     name,
		Job:      job,
		Schedule: schedule,
	})
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

// intervalToCron 將時間間隔轉換為Cron表達式
func (h *SchedulerHandler) intervalToCron(interval time.Duration) string {
	switch {
	case interval < time.Minute:
		// 小於1分鐘的間隔，每N秒執行一次
		seconds := int(interval.Seconds())
		if seconds <= 0 {
			seconds = 1
		}
		// 格式: "*/N * * * * *" (每N秒)
		return fmt.Sprintf("*/%d * * * * *", seconds)

	case interval < time.Hour:
		// 小於1小時的間隔，每N分鐘執行一次
		minutes := int(interval.Minutes())
		if minutes <= 0 {
			minutes = 1
		}
		// 格式: "0 */N * * * *" (每N分鐘)
		return fmt.Sprintf("0 */%d * * * *", minutes)

	case interval < 24*time.Hour:
		// 小於24小時的間隔，每N小時執行一次
		hours := int(interval.Hours())
		if hours <= 0 {
			hours = 1
		}
		// 格式: "0 0 */N * * *" (每N小時)
		return fmt.Sprintf("0 0 */%d * * *", hours)

	default:
		// 大於等於24小時的間隔，每N天執行一次
		days := int(interval.Hours() / 24)
		if days <= 0 {
			days = 1
		}
		// 如果超過30天，設為每月執行一次
		if days > 30 {
			return "0 0 0 1 * *" // 每月1號執行
		}
		// 由於cron不直接支援"每N天"，這裡簡化為每天執行
		// 更複雜的情況建議直接使用cron表達式
		return "0 0 0 * * *" // 每天執行
	}
}

// GetRegisteredJobs 獲取已註冊的任務列表
func (h *SchedulerHandler) GetRegisteredJobs() []string {
	jobNames := make([]string, len(h.jobs))
	for i, job := range h.jobs {
		jobNames[i] = job.Name
	}
	return jobNames
}
