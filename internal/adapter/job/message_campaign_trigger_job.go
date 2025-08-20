package job

import (
	"context"
	"fmt"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/jobport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"time"
)

type MessageCampaignTriggerJob struct {
	campaignRepo repositoryport.MessageCampaignRepository
}

func NewMessageCampaignTriggerJob(
	campaignRepo repositoryport.MessageCampaignRepository) jobport.ScheduledJob {
	return &MessageCampaignTriggerJob{
		campaignRepo: campaignRepo,
	}
}

func (j *MessageCampaignTriggerJob) Execute(ctx context.Context) error {
	fmt.Printf("[%s] 執行示例任務: %s\n", time.Now().Format("2006-01-02 15:04:05"), j.GetName())
	// 模擬任務執行時間
	time.Sleep(2 * time.Second)
	fmt.Printf("[%s] 任務完成: %s\n", time.Now().Format("2006-01-02 15:04:05"), j.GetName())
	return nil
}

func (j *MessageCampaignTriggerJob) GetName() string {
	return "message-campaign-trigger"
}

func (j *MessageCampaignTriggerJob) GetCron() string {
	return "*/5 * * * * *"
}

func (j *MessageCampaignTriggerJob) GetTimeInterval() time.Duration {
	return 5 * time.Second
}
