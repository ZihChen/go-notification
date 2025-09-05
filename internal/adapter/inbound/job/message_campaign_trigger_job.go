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
}

func NewMessageCampaignTriggerJob(
	messageUseCase inbound.MessageUseCase,
	logger infrastructure.Logger) *MessageCampaignTriggerJob {
	return &MessageCampaignTriggerJob{
		messageUseCase: messageUseCase,
		logger:         logger,
	}
}

func (j *MessageCampaignTriggerJob) Execute(ctx context.Context) error {
	j.logger.InfoLog("Starting message campaign trigger job execution")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		j.logger.InfoLog("Message campaign trigger job completed",
			j.logger.String("duration", duration.String()))
	}()

	if err := j.messageUseCase.ProcessScheduledCampaigns(ctx); err != nil {
		j.logger.ErrorLog("Failed to process scheduled campaigns", j.logger.Error("err", err))
		return err
	}

	return nil
}

func (j *MessageCampaignTriggerJob) GetName() string {
	return "message-campaign-trigger"
}

func (j *MessageCampaignTriggerJob) GetCron() string {
	return "*/10 * * * * *"
}
