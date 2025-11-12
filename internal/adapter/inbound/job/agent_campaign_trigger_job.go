package job

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	jobport "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/job"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/semaphore"
)

// 確保 AgentCampaignTriggerJob 實現了 ScheduledJob 介面
var _ jobport.ScheduledJob = (*AgentCampaignTriggerJob)(nil)

type AgentCampaignTriggerJob struct {
	agentUseCase       inbound.AgentUseCase
	logger             infrastructure.Logger
	tracingService     infrastructure.TracingService
	distributedLockMgr infrastructure.DistributedLockManager

	// 併發控制
	semaphore *semaphore.Weighted
}

func NewAgentCampaignTriggerJob(
	agentUseCase inbound.AgentUseCase,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	distributedLockMgr infrastructure.DistributedLockManager,
) *AgentCampaignTriggerJob {
	return &AgentCampaignTriggerJob{
		agentUseCase:       agentUseCase,
		logger:             logger,
		tracingService:     tracingService,
		distributedLockMgr: distributedLockMgr,
		semaphore:          semaphore.NewWeighted(10), // 最大併發數 10
	}
}

func (j *AgentCampaignTriggerJob) Execute(ctx context.Context) error {
	ctx, span := j.tracingService.StartSpan(ctx, "AgentCampaignTriggerJob.Execute")
	defer j.tracingService.SpanEnd(span)

	j.logger.InfoLog("Starting agent campaign trigger job execution")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		j.logger.InfoLog("Agent campaign trigger job completed",
			j.logger.String("duration", duration.String()))
	}()

	// 1. 獲取分佈式鎖確保單實例執行
	lockKey := "job:agent_campaigns:trigger"
	mutex, err := j.distributedLockMgr.GetLockWithOptions(ctx, lockKey, infrastructure.LockOptions{
		Expiry:     5 * time.Second,
		Tries:      1, // 不重試，避免重複執行
		RetryDelay: 0,
	})
	if err != nil {
		j.logger.WarnLog("Failed to get distributed lock, continuing without lock",
			j.logger.String("error", err.Error()))
		// 繼續執行，但記錄警告
	} else {
		err := mutex.TryLock()
		if err != nil {
			j.logger.InfoLog("Agent campaign job already running on another instance, skipping")
			return nil
		}
		defer func() {
			if unlocked, unlockErr := mutex.Unlock(); unlockErr != nil || !unlocked {
				j.logger.ErrorLog("Failed to unlock job mutex",
					j.logger.String("error", unlockErr.Error()))
			}
		}()
	}

	// 2. 查詢到期的排程活動
	campaigns, err := j.agentUseCase.GetScheduledCampaigns(ctx, time.Now())
	if err != nil {
		j.tracingService.RecordSpanError(span, err)
		j.logger.ErrorLog("Failed to get scheduled campaigns", j.logger.Error("err", err))
		return err
	}

	if len(campaigns) == 0 {
		j.logger.InfoLog("No scheduled agent campaigns found")
		return nil
	}

	j.logger.InfoLog("Processing scheduled agent campaigns",
		j.logger.Int("campaign_count", len(campaigns)))

	// 3. 併發處理活動
	return j.processCampaignsConcurrently(ctx, campaigns)
}

func (j *AgentCampaignTriggerJob) processCampaignsConcurrently(
	ctx context.Context,
	campaigns []*entity.AgentCampaign,
) error {
	ctx, span := j.tracingService.StartSpan(
		ctx,
		"AgentCampaignTriggerJob.processCampaignsConcurrently",
	)
	defer j.tracingService.SpanEnd(span)

	j.tracingService.RecordSpanAttributes(span, attribute.Int("campaigns_count", len(campaigns)))

	errChan := make(chan error, len(campaigns))
	processedCount := 0

	// 併發處理活動，但限制併發數
	for _, campaign := range campaigns {
		// 獲取併發信號量
		if err := j.semaphore.Acquire(ctx, 1); err != nil {
			j.logger.ErrorLog("Failed to acquire semaphore",
				j.logger.UInt64("campaign_id", campaign.ID),
				j.logger.String("error", err.Error()))
			errChan <- err
			continue
		}

		go func(c *entity.AgentCampaign) {
			defer j.semaphore.Release(1)

			if err := j.processCampaign(ctx, c); err != nil {
				j.logger.ErrorLog("Agent campaign processing failed",
					j.logger.UInt64("campaign_id", c.ID),
					j.logger.String("error", err.Error()))
				errChan <- err
			} else {
				errChan <- nil
			}
		}(campaign)
	}

	// 等待所有活動處理完成
	var processingErrors []error
	for i := 0; i < len(campaigns); i++ {
		if err := <-errChan; err != nil {
			processingErrors = append(processingErrors, err)
		} else {
			processedCount++
		}
	}

	j.tracingService.TraceEvent(span, "Agent campaigns processing completed",
		attribute.Int("total_campaigns", len(campaigns)),
		attribute.Int("processed_count", processedCount),
		attribute.Int("error_count", len(processingErrors)))

	j.logger.InfoLog("Agent campaigns processing completed",
		j.logger.Int("total_campaigns", len(campaigns)),
		j.logger.Int("processed_count", processedCount),
		j.logger.Int("error_count", len(processingErrors)))

	// 如果有錯誤，返回第一個錯誤，但不影響其他活動的處理
	if len(processingErrors) > 0 {
		j.logger.WarnLog("Some agent campaigns failed to process",
			j.logger.Int("failed_count", len(processingErrors)))
		// 記錄錯誤但不中斷整個 Job，讓成功的活動能正常處理
	}

	return nil
}

func (j *AgentCampaignTriggerJob) processCampaign(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	ctx, span := j.tracingService.StartSpan(ctx, "AgentCampaignTriggerJob.processCampaign")
	defer j.tracingService.SpanEnd(span)

	j.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaign.ID)))

	j.logger.InfoLog("Processing agent campaign",
		j.logger.UInt64("campaign_id", campaign.ID),
		j.logger.String("title", campaign.Title))

	// 1. 更新狀態為發送中：sending
	if err := j.agentUseCase.UpdateCampaignStatus(ctx, campaign.ID, consts.AgentCampaignStatusSending); err != nil {
		j.logger.ErrorLog("Failed to mark campaign as sending",
			j.logger.UInt64("campaign_id", campaign.ID),
			j.logger.String("update_error", err.Error()))
		j.tracingService.RecordSpanError(span, err)
		return err
	}

	// 2. 委託UseCase執行完整發送流程
	targetCount, sentCount, err := j.agentUseCase.SendMessageToCampaignTargets(ctx, campaign)
	if err != nil {
		// 標記活動失敗
		if updateErr := j.agentUseCase.UpdateCampaignStatus(ctx, campaign.ID, consts.AgentCampaignStatusFailed); updateErr != nil {
			j.logger.ErrorLog("Failed to mark campaign as failed",
				j.logger.UInt64("campaign_id", campaign.ID),
				j.logger.String("update_error", updateErr.Error()))
		}
		j.tracingService.RecordSpanError(span, err)
		return err
	}

	// 3. 更新統計與完成狀態
	if err = j.agentUseCase.CompleteCampaign(ctx, campaign.ID, targetCount, sentCount); err != nil {
		j.tracingService.RecordSpanError(span, err)
		return err
	}

	j.tracingService.TraceEvent(span, "Agent campaign completed successfully",
		attribute.Int("target_count", targetCount),
		attribute.Int("sent_count", sentCount))

	j.logger.InfoLog("Agent campaign completed successfully",
		j.logger.UInt64("campaign_id", campaign.ID),
		j.logger.Int("target_count", targetCount),
		j.logger.Int("sent_count", sentCount))

	return nil
}

func (j *AgentCampaignTriggerJob) GetName() string {
	return "agent-campaign-trigger"
}

func (j *AgentCampaignTriggerJob) GetCron() string {
	return "*/10 * * * * *" // 每30秒執行一次
}
