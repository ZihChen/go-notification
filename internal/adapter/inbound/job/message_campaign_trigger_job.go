package job

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	jobport "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/job"
)

// 確保 MessageCampaignTriggerJob 實現了 ScheduledJob 介面
var _ jobport.ScheduledJob = (*MessageCampaignTriggerJob)(nil)

type MessageCampaignTriggerJob struct {
	messageUseCase inbound.MessageUseCase
	logger         infrastructure.Logger
	metricsService infrastructure.MetricsService
}

func NewMessageCampaignTriggerJob(
	messageUseCase inbound.MessageUseCase,
	logger infrastructure.Logger,
	metricsService infrastructure.MetricsService) *MessageCampaignTriggerJob {
	return &MessageCampaignTriggerJob{
		messageUseCase: messageUseCase,
		logger:         logger,
		metricsService: metricsService,
	}
}

func (j *MessageCampaignTriggerJob) Execute(ctx context.Context) error {
	j.logger.InfoLog("Starting message campaign trigger job execution")

	start := time.Now()
	success := false
	defer func() {
		duration := time.Since(start)
		j.logger.InfoLog("Message campaign trigger job completed",
			j.logger.String("duration", duration.String()),
			j.logger.Bool("success", success))

		// 記錄 Job metrics
		if j.metricsService != nil && j.metricsService.IsEnabled() {
			j.metricsService.RecordTaskProcessed("message_campaign_trigger", success)
			j.metricsService.RecordTaskDuration("message_campaign_trigger", duration.Seconds())
		}
	}()

	if err := j.messageUseCase.ProcessScheduledCampaigns(ctx); err != nil {
		j.logger.ErrorLog("Failed to process scheduled campaigns", j.logger.Error("err", err))
		return err
	}

	success = true
	return nil
}

func (j *MessageCampaignTriggerJob) GetName() string {
	return "message-campaign-trigger"
}

func (j *MessageCampaignTriggerJob) GetCron() string {
	return "*/10 * * * * *"
}
