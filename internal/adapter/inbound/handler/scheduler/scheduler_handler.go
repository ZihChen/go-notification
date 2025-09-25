package scheduler

import (
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/job"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	jobport "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/job"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/robfig/cron/v3"
)

// Handler 排程處理器
type Handler struct {
	logger         infrastructure.Logger
	redisManager   *redisCache.Manager
	tracingService infrastructure.TracingService
	jobs           []ScheduledJobConfig
}

// ScheduledJobConfig 排程任務配置
type ScheduledJobConfig struct {
	Name     string               // 任務名稱
	Job      jobport.ScheduledJob // 任務執行邏輯
	Schedule string               // Cron表達式 (支援秒級: "* * * * * *")
}

// NewSchedulerHandler 創建排程處理器
func NewSchedulerHandler(
	logger infrastructure.Logger,
	redisManager *redisCache.Manager,
	tracingService infrastructure.TracingService,
	jobRegistry *job.Registry) *Handler {
	h := &Handler{
		logger:         logger,
		redisManager:   redisManager,
		tracingService: tracingService,
		jobs:           make([]ScheduledJobConfig, 0),
	}
	for _, j := range jobRegistry.GetAllJobs() {
		err := h.registerJobWithSchedule(j.GetName(), j, j.GetCron())
		if err != nil {
			h.logger.FatalLog("Failed to register scheduled job")
		}
	}
	return h
}

// RegisterJobs 註冊所有排程任務到cron調度器
func (h *Handler) RegisterJobs(cronManager *cron.Cron) {
	for _, jobConfig := range h.jobs {
		scheduleStr := jobConfig.Schedule
		if jobConfig.Schedule == "" {
			h.logger.FatalLog("Schedule is required for interval job")
		}

		wrapper := &JobWrapper{
			job:            jobConfig.Job,
			logger:         h.logger,
			redisManager:   h.redisManager,
			tracingService: h.tracingService,
		}

		_, err := cronManager.AddFunc(scheduleStr, wrapper.run)

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

// registerJob 註冊排程任務
func (h *Handler) registerJob(config ScheduledJobConfig) error {
	if config.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if config.Job == nil {
		return fmt.Errorf("job implementation is required")
	}
	if config.Schedule == "" {
		return fmt.Errorf("job schedule is required")
	}

	h.jobs = append(h.jobs, config)
	h.logger.InfoLog("Registered scheduled job",
		h.logger.String("job_name", config.Name),
		h.logger.String("schedule", config.Schedule))

	return nil
}

// registerJobWithSchedule 使用Cron表達式註冊任務
func (h *Handler) registerJobWithSchedule(
	name string,
	job jobport.ScheduledJob,
	schedule string,
) error {
	return h.registerJob(ScheduledJobConfig{
		Name:     name,
		Job:      job,
		Schedule: schedule,
	})
}

// GetRegisteredJobs 獲取已註冊的任務列表
func (h *Handler) GetRegisteredJobs() []string {
	jobNames := make([]string, len(h.jobs))
	for i, j := range h.jobs {
		jobNames[i] = j.Name
	}
	return jobNames
}
