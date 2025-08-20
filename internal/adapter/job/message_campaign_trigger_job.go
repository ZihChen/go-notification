package job

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/jobport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/usecaseport"
)

type MessageCampaignTriggerJob struct {
	messageUseCase usecaseport.MessageUseCase
	logger         infraport.Logger
}

func NewMessageCampaignTriggerJob(
	messageUseCase usecaseport.MessageUseCase,
	logger infraport.Logger) jobport.ScheduledJob {
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
	return "0 */1 * * * *"
}

func (j *MessageCampaignTriggerJob) GetTimeInterval() time.Duration {
	return 1 * time.Minute
}
