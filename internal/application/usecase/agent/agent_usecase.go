package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// AgentUseCase 代理業務用例實作
type AgentUseCase struct {
	agentRepo          repository.AgentRepository
	agentCampaignRepo  repository.AgentCampaignRepository
	agentMessageRepo   repository.AgentMessageRepository
	agentRelationRepo  repository.AgentRelationshipRepository
	merchantRepo       repository.MerchantRepository
	agentService       service.AgentService
	eventProducer      service.EventProducer
	logger             infrastructure.Logger
	tracingService     infrastructure.TracingService
	distributedLockMgr infrastructure.DistributedLockManager
}

// NewAgentUseCase 創建代理用例
func NewAgentUseCase(
	agentRepo repository.AgentRepository,
	agentCampaignRepo repository.AgentCampaignRepository,
	agentMessageRepo repository.AgentMessageRepository,
	agentRelationRepo repository.AgentRelationshipRepository,
	merchantRepo repository.MerchantRepository,
	agentService service.AgentService,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	distributedLockMgr infrastructure.DistributedLockManager,
) inbound.AgentUseCase {
	return &AgentUseCase{
		agentRepo:          agentRepo,
		agentCampaignRepo:  agentCampaignRepo,
		agentMessageRepo:   agentMessageRepo,
		agentRelationRepo:  agentRelationRepo,
		merchantRepo:       merchantRepo,
		agentService:       agentService,
		eventProducer:      eventProducer,
		logger:             logger,
		tracingService:     tracingService,
		distributedLockMgr: distributedLockMgr,
	}
}

// SyncAgentDataWithRelationships 完整的代理同步邏輯 (資料同步 + 關係建立)
func (u *AgentUseCase) SyncAgentDataWithRelationships(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SyncAgentDataWithRelationships")
	defer u.tracingService.SpanEnd(span)

	// 添加代理信息到 span
	u.tracingService.RecordSpanAttributes(span,
		attribute.String("agent.global_id", agentEvent.GlobalAgentID),
		attribute.String("agent.account", agentEvent.Account),
		attribute.String("merchant.global_id", agentEvent.GlobalMerchantID))

	// 1. 同步當前代理資料
	u.tracingService.TraceEvent(span, "Syncing current agent data")
	if err := u.SyncCurrentAgentUpsert(ctx, agentEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("sync current agent: %w", err)
	}

	// 2. 同步代理關係 (使用Service處理複雜邏輯)
	u.tracingService.TraceEvent(span, "Syncing agent relationships using AgentService")
	if err := u.agentService.SyncAgentRelationshipsUpsert(ctx, agentEvent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("sync agent relationships: %w", err)
	}

	// 記錄完成
	u.tracingService.TraceEvent(span, "Agent sync with relationships completed successfully")
	u.logger.InfoLog("Agent sync with relationships completed successfully",
		u.logger.String("global_agent_id", agentEvent.GlobalAgentID),
		u.logger.String("account", agentEvent.Account))

	return nil
}

// SyncCurrentAgentUpsert 同步當前代理資料
func (u *AgentUseCase) SyncCurrentAgentUpsert(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SyncCurrentAgentUpsert")
	defer u.tracingService.SpanEnd(span)

	// 查找對應的商戶
	u.tracingService.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, agentEvent.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	if merchant == nil {
		u.tracingService.TraceEvent(span, "Merchant not found, skipping agent sync")
		u.logger.WarnLog("Merchant not found for agent sync",
			u.logger.String("global_merchant_id", agentEvent.GlobalMerchantID),
			u.logger.String("global_agent_id", agentEvent.GlobalAgentID))
		return nil
	}

	// 構建代理實體
	agent := entity.Agent{
		MerchantID:      merchant.ID,
		GlobalAgentID:   agentEvent.GlobalAgentID,
		Account:         agentEvent.Account,
		Ancestry:        agentEvent.Ancestry,
		CurrentSignInAt: agentEvent.CurrentSignInAt,
		CreatedAt:       agentEvent.CreatedAt,
		UpdatedAt:       agentEvent.UpdatedAt,
	}

	u.tracingService.TraceEvent(span, "Upsert agent")
	if err = u.agentRepo.Upsert(ctx, &agent); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("upsert agent: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent upserted successfully")
	u.logger.InfoLog("Agent upserted successfully",
		u.logger.String("global_id", agent.GlobalAgentID),
		u.logger.String("account", agent.Account))

	return nil
}

// GetAgentByGlobalID 通過全局ID獲取代理
func (u *AgentUseCase) GetAgentByGlobalID(
	ctx context.Context,
	globalAgentID string,
) (*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentByGlobalID")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.String("agent.global_id", globalAgentID))

	agent, err := u.agentRepo.GetByGlobalID(ctx, globalAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent: %w", err)
	}

	return agent, nil
}

// GetActiveAgents 獲取活躍代理列表
func (u *AgentUseCase) GetActiveAgents(
	ctx context.Context,
	merchantID uint64,
	limit, offset int,
) ([]*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetActiveAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("merchant.id", int64(merchantID)),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset))

	agents, err := u.agentRepo.FindAgents(ctx, merchantID, limit, offset)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find active agents: %w", err)
	}

	return agents, nil
}

// CreateAgentCampaign 創建代理訊息活動
func (u *AgentUseCase) CreateAgentCampaign(
	ctx context.Context,
	req *dto.CreateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.CreateAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("campaign.title", req.Title),
		attribute.String("campaign.target_type", req.TargetType))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, req.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}
	u.tracingService.TraceEvent(span, "Creating message campaign")

	// 使用 Entity 工廠方法創建並驗證代理活動
	campaign, err := entity.NewAgentCampaign(req, merchant.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, err
	}

	u.tracingService.TraceEvent(span, "Creating agent campaign")
	createdCampaign, err := u.agentCampaignRepo.Create(ctx, campaign)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("create agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign created successfully")
	u.logger.InfoLog("Agent campaign created successfully",
		u.logger.UInt64("campaign_id", createdCampaign.ID),
		u.logger.String("title", createdCampaign.Title),
		u.logger.String("target_type", createdCampaign.TargetType))

	// 如果是立即發送 (status=scheduled 且 scheduled_at=nil)，異步進行發送
	if u.isImmediateSend(consts.AgentCampaignStatus(req.Status), req.ScheduledAt) {
		u.tracingService.TraceEvent(span, "Starting asynchronous immediate campaign processing")

		// 獲取活動鎖以防止立即發送期間的修改（嚴格模式）
		campaignLockKey := fmt.Sprintf("agent_campaign:lock:%d", createdCampaign.ID)
		campaignMutex, err := u.distributedLockMgr.GetLockWithOptions(ctx, campaignLockKey, infrastructure.LockOptions{
			Expiry:     90 * time.Second, // 90秒過期
			Tries:      1,                // 不重試，快速失敗
			RetryDelay: 0,
		})
		if err != nil {
			u.logger.ErrorLog("Failed to get campaign lock for immediate send",
				u.logger.UInt64("campaign_id", createdCampaign.ID),
				u.logger.String("error", err.Error()))
			// 鎖失敗不影響活動創建，但不進行立即發送
			return createdCampaign, nil
		}

		// 嘗試獲取活動鎖
		err = campaignMutex.TryLock()
		if err != nil {
			u.logger.InfoLog("Campaign lock busy, skipping immediate send",
				u.logger.UInt64("campaign_id", createdCampaign.ID),
				u.logger.String("title", createdCampaign.Title))
			// 鎖忙碌不影響活動創建，但不進行立即發送
			return createdCampaign, nil
		}

		// 使用 goroutine 異步處理，避免阻塞 API 響應
		go func() {
			// 確保在異步處理完成後釋放活動鎖
			defer func() {
				if unlocked, unlockErr := campaignMutex.Unlock(); unlockErr != nil || !unlocked {
					u.logger.ErrorLog("Failed to unlock campaign mutex after async processing",
						u.logger.UInt64("campaign_id", createdCampaign.ID),
						u.logger.String("error", unlockErr.Error()))
				}
			}()

			// 創建新的 context，避免與原始請求的 context 綁定
			asyncCtx := context.Background()

			// 記錄開始處理
			u.logger.InfoLog("Starting asynchronous immediate agent campaign processing",
				u.logger.UInt64("campaign_id", createdCampaign.ID),
				u.logger.String("title", createdCampaign.Title))

			// 執行立即發送處理
			if err = u.processCampaignImmediate(asyncCtx, createdCampaign); err != nil {
				// 異步錯誤記錄，不影響 API 響應
				u.logger.ErrorLog("Asynchronous immediate campaign processing failed",
					u.logger.UInt64("campaign_id", createdCampaign.ID),
					u.logger.String("error", err.Error()))
			} else {
				// 記錄成功完成
				u.logger.InfoLog("Asynchronous immediate agent campaign processing completed successfully",
					u.logger.UInt64("campaign_id", createdCampaign.ID))
			}
		}()

		// API 立即響應，不等待發送完成
		u.tracingService.TraceEvent(span, "Immediate campaign queued for asynchronous processing")
		u.logger.InfoLog("Immediate agent campaign queued for asynchronous processing",
			u.logger.UInt64("campaign_id", createdCampaign.ID),
			u.logger.String("title", createdCampaign.Title))
	}

	return createdCampaign, nil
}

// UpdateAgentCampaign 更新代理訊息活動
func (u *AgentUseCase) UpdateAgentCampaign(
	ctx context.Context,
	req *dto.UpdateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.UpdateAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(req.ID)))

	// 1. 獲取活動鎖以防止併發修改同一活動（嚴格模式）
	campaignLockKey := fmt.Sprintf("agent_campaign:lock:%d", req.ID)
	campaignMutex, err := u.distributedLockMgr.GetLockWithOptions(ctx, campaignLockKey, infrastructure.LockOptions{
		Expiry:     90 * time.Second, // 90秒過期
		Tries:      3,                // 重試3次
		RetryDelay: 100 * time.Millisecond,
	})
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("failed to get campaign lock for %d: %w", req.ID, err)
	}

	// 嘗試獲取活動鎖
	err = campaignMutex.TryLock()
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("failed to acquire campaign lock for %d: campaign is being processed by another operation", req.ID)
	}

	// 2. 先獲取現有的活動
	u.tracingService.TraceEvent(span, "Getting existing campaign")
	existing, err := u.agentCampaignRepo.GetByID(ctx, req.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get existing campaign: %w", err)
	}

	// 轉換為新的 Entity 物件 (參考 message_usecase.go 的做法)
	campaignEntity := &entity.AgentCampaign{}

	// 使用 copier 複製現有資料
	err = copier.Copy(campaignEntity, existing)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("copy campaign failed: %w", err)
	}

	// 檢查活動狀態是否允許更新
	if err = campaignEntity.CanUpdate(); err != nil {
		return nil, err
	}

	// 使用 Entity 方法更新欄位並驗證
	if err = campaignEntity.UpdateFromRequest(req); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, err
	}

	u.tracingService.TraceEvent(span, "Updating agent campaign")
	err = u.agentCampaignRepo.Update(ctx, campaignEntity)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("update agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign updated successfully")
	u.logger.InfoLog("Agent campaign updated successfully",
		u.logger.UInt64("campaign_id", campaignEntity.ID),
		u.logger.String("title", campaignEntity.Title),
		u.logger.String("status", campaignEntity.Status.String()))

	// 如果更新後變成立即發送 (status=scheduled 且 scheduled_at=nil)，異步進行發送
	if u.isImmediateSend(consts.AgentCampaignStatus(req.Status), req.ScheduledAt) {
		u.tracingService.TraceEvent(
			span,
			"Starting asynchronous immediate campaign processing after update",
		)

		// 使用 goroutine 異步處理，避免阻塞 API 響應
		go func() {
			// 確保在異步處理完成後釋放活動鎖
			defer func() {
				if unlocked, unlockErr := campaignMutex.Unlock(); unlockErr != nil || !unlocked {
					u.logger.ErrorLog("Failed to unlock campaign mutex after async processing",
						u.logger.UInt64("campaign_id", campaignEntity.ID),
						u.logger.String("error", unlockErr.Error()))
				}
			}()

			// 創建新的 context，避免與原始請求的 context 綁定
			asyncCtx := context.Background()

			// 記錄開始處理
			u.logger.InfoLog(
				"Starting asynchronous immediate agent campaign processing after update",
				u.logger.UInt64("campaign_id", campaignEntity.ID),
				u.logger.String("title", campaignEntity.Title),
			)

			// 執行立即發送處理
			if err = u.processCampaignImmediate(asyncCtx, campaignEntity); err != nil {
				// 異步錯誤記錄，不影響 API 響應
				u.logger.ErrorLog("Asynchronous immediate campaign processing failed after update",
					u.logger.UInt64("campaign_id", campaignEntity.ID),
					u.logger.String("error", err.Error()))
			} else {
				// 記錄成功完成
				u.logger.InfoLog("Asynchronous immediate agent campaign processing completed successfully after update",
					u.logger.UInt64("campaign_id", campaignEntity.ID))
			}
		}()

		// API 立即響應，不等待發送完成
		u.tracingService.TraceEvent(
			span,
			"Immediate campaign queued for asynchronous processing after update",
		)
		u.logger.InfoLog("Immediate agent campaign queued for asynchronous processing after update",
			u.logger.UInt64("campaign_id", campaignEntity.ID),
			u.logger.String("title", campaignEntity.Title))
	} else {
		// 非立即發送，可以立即釋放活動鎖
		if unlocked, unlockErr := campaignMutex.Unlock(); unlockErr != nil || !unlocked {
			u.logger.ErrorLog("Failed to unlock campaign mutex for non-immediate send",
				u.logger.UInt64("campaign_id", campaignEntity.ID),
				u.logger.String("error", unlockErr.Error()))
		}
	}

	return campaignEntity, nil
}

// GetAgentCampaign 獲取代理訊息活動
func (u *AgentUseCase) GetAgentCampaign(
	ctx context.Context,
	id uint64,
) (*dto.AgentCampaignResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	u.tracingService.TraceEvent(span, "Getting agent campaign")
	campaign, err := u.agentCampaignRepo.GetByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get agent campaign: %w", err)
	}

	// 轉換為回應DTO
	response := &dto.AgentCampaignResponse{
		ID:            campaign.ID,
		MerchantID:    campaign.MerchantID,
		Title:         campaign.Title,
		Content:       campaign.Content,
		ScheduledAt:   campaign.ScheduledAt,
		Status:        campaign.Status.String(),
		ScheduleType:  campaign.ScheduleType,
		TargetType:    campaign.TargetType,
		TargetDetails: campaign.TargetDetails,
		TargetCount:   campaign.TargetCount,
		RealSentCount: campaign.RealSentCount,
		CreatedBy:     campaign.CreatedBy,
		UpdatedBy:     campaign.UpdatedBy,
		CreatedAt:     campaign.CreatedAt,
		UpdatedAt:     campaign.UpdatedAt,
	}

	u.tracingService.TraceEvent(span, "Agent campaign retrieved successfully")
	return response, nil
}

// GetAgentCampaigns 獲取代理訊息活動列表
func (u *AgentUseCase) GetAgentCampaigns(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) (*dto.AgentCampaignListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentCampaigns")
	defer u.tracingService.SpanEnd(span)

	// 解析逗號分隔的status字符串
	var statusList []string
	if query.Status != "" {
		statusList = strings.Split(strings.TrimSpace(query.Status), ",")
		// 清理每個狀態值的空白
		for i := range statusList {
			statusList[i] = strings.TrimSpace(statusList[i])
		}
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("page", query.Page),
		attribute.Int("page_size", query.PageSize),
		attribute.StringSlice("status", statusList))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, query.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 創建用於Repository層的查詢對象，將status字符串轉換為數組
	repoQuery := &dto.AgentCampaignsQueryForRepo{
		GlobalMerchantID: query.GlobalMerchantID,
		Page:             query.Page,
		PageSize:         query.PageSize,
		Limit:            query.Limit,
		Offset:           query.Offset,
		MerchantID:       merchant.ID,
		Status:           statusList,
		CreatedBy:        query.CreatedBy,
		CreatedStartAt:   query.CreatedStartAt,
		CreatedEndAt:     query.CreatedEndAt,
		ScheduledStartAt: query.ScheduledStartAt,
		ScheduledEndAt:   query.ScheduledEndAt,
	}

	// 設定預設值
	if repoQuery.Limit == 0 {
		repoQuery.Limit = repoQuery.PageSize
	}
	if repoQuery.Offset == 0 {
		repoQuery.Offset = (repoQuery.Page - 1) * repoQuery.PageSize
	}

	campaigns, total, err := u.agentCampaignRepo.List(ctx, repoQuery)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent campaigns: %w", err)
	}
	u.tracingService.TraceEvent(span, "Getting agent campaigns list")

	// 轉換為回應DTO
	campaignResponses := make([]dto.AgentCampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		campaignResponses[i] = dto.AgentCampaignResponse{
			ID:            campaign.ID,
			MerchantID:    campaign.MerchantID,
			Title:         campaign.Title,
			Content:       campaign.Content,
			ScheduledAt:   campaign.ScheduledAt,
			Status:        campaign.Status.String(),
			ScheduleType:  campaign.ScheduleType,
			TargetType:    campaign.TargetType,
			TargetDetails: campaign.TargetDetails,
			TargetCount:   campaign.TargetCount,
			RealSentCount: campaign.RealSentCount,
			CreatedBy:     campaign.CreatedBy,
			UpdatedBy:     campaign.UpdatedBy,
			CreatedAt:     campaign.CreatedAt,
			UpdatedAt:     campaign.UpdatedAt,
		}
	}

	response := &dto.AgentCampaignListResponse{
		Campaigns: campaignResponses,
		Page:      query.Page,
		PageSize:  query.PageSize,
		Total:     total,
	}

	u.tracingService.TraceEvent(span, "Agent campaigns list retrieved successfully")
	return response, nil
}

// DeleteAgentCampaign 刪除代理訊息活動
func (u *AgentUseCase) DeleteAgentCampaign(ctx context.Context, id uint64) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.DeleteAgentCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(id)))

	// 先檢查活動是否存在
	u.tracingService.TraceEvent(span, "Checking if campaign exists")
	campaign, err := u.agentCampaignRepo.GetByID(ctx, id)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("get campaign for deletion: %w", err)
	}

	// 先更新狀態為 cancelled
	u.tracingService.TraceEvent(span, "Updating campaign status to cancelled before deletion")
	setColumn := map[string]interface{}{
		"status": consts.MessageCampaignStatusCancelled,
	}
	if err = u.agentCampaignRepo.UpdateFields(ctx, id, setColumn); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("update campaign status to cancelled: %w", err)
	}

	// 使用事務確保刪除操作的原子性
	u.tracingService.TraceEvent(span, "Starting transaction for cascade delete")

	// 先刪除相關的代理站內信
	u.tracingService.TraceEvent(span, "Deleting associated agent messages")
	if err = u.agentMessageRepo.DeleteByCampaignID(ctx, id); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("delete associated agent messages: %w", err)
	}

	// 再刪除代理活動
	u.tracingService.TraceEvent(span, "Deleting agent campaign")
	if err = u.agentCampaignRepo.Delete(ctx, id); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("delete agent campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Agent campaign deleted successfully")
	u.logger.InfoLog("Agent campaign deleted successfully",
		u.logger.UInt64("campaign_id", id),
		u.logger.String("title", campaign.Title))

	return nil
}

// BatchDeleteAgentCampaigns 批量刪除代理訊息活動（同步版本）
func (u *AgentUseCase) BatchDeleteAgentCampaigns(
	ctx context.Context,
	req *dto.BatchDeleteAgentCampaignsRequest,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.BatchDeleteAgentCampaigns")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("campaign_count", len(req.IDs)),
		attribute.String("campaign_ids", fmt.Sprintf("%v", req.IDs)))

	if len(req.IDs) == 0 {
		return fmt.Errorf("no campaign IDs provided")
	}

	// 先批量檢查所有活動是否存在並且可以刪除
	u.tracingService.TraceEvent(span, "Checking campaigns existence and deletion permission")
	var validIDs []uint64
	var invalidCampaigns []uint64

	for _, id := range req.IDs {
		campaign, err := u.agentCampaignRepo.GetByID(ctx, id)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to get campaign for batch deletion validation",
				u.logger.UInt64("campaign_id", id),
				u.logger.Error("error", err))
			invalidCampaigns = append(invalidCampaigns, id)
			continue
		}

		if campaign == nil {
			u.logger.WarnLog("Campaign not found for batch deletion",
				u.logger.UInt64("campaign_id", id))
			invalidCampaigns = append(invalidCampaigns, id)
			continue
		}

		validIDs = append(validIDs, id)
	}

	if len(validIDs) == 0 {
		return fmt.Errorf("no valid campaigns found for deletion")
	}

	// 記錄驗證結果
	u.tracingService.RecordSpanAttributes(span,
		attribute.Int("valid_campaign_count", len(validIDs)),
		attribute.Int("invalid_campaign_count", len(invalidCampaigns)))

	if len(invalidCampaigns) > 0 {
		u.logger.WarnLog("Some campaigns were skipped during batch deletion",
			u.logger.Any("invalid_campaign_ids", invalidCampaigns),
			u.logger.Any("valid_campaign_ids", validIDs))
	}

	// 使用事務確保批量刪除操作的原子性
	u.tracingService.TraceEvent(span, "Starting transaction for cascade batch delete")

	// 異步硬刪除相關的代理站內信 (避免API超時)
	u.tracingService.TraceEvent(span, "Starting async agent messages deletion")
	go func() {
		// 使用新的context避免被原始請求取消
		deleteCtx := context.Background()

		u.logger.InfoLog("Starting async agent messages deletion",
			u.logger.Any("campaign_ids", validIDs),
			u.logger.Int("campaign_count", len(validIDs)))

		if err := u.agentMessageRepo.BatchHardDeleteByCampaignIDs(deleteCtx, validIDs); err != nil {
			u.logger.ErrorLog("Async agent messages deletion failed",
				u.logger.Any("campaign_ids", validIDs),
				u.logger.Error("error", err))
		} else {
			u.logger.InfoLog("Async agent messages deletion completed successfully",
				u.logger.Any("campaign_ids", validIDs),
				u.logger.Int("campaign_count", len(validIDs)))
		}
	}()

	u.logger.InfoLog("Async agent messages deletion task started",
		u.logger.Any("campaign_ids", validIDs))

	// 再執行批量刪除活動
	u.tracingService.TraceEvent(span, "Executing batch delete operation")
	if err := u.agentCampaignRepo.BatchDelete(ctx, validIDs); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("batch delete agent campaigns failed: %w", err)
	}

	u.tracingService.TraceEvent(span, "Batch delete completed successfully")
	u.logger.InfoLog("Batch delete agent campaigns completed successfully",
		u.logger.Any("deleted_campaign_ids", validIDs),
		u.logger.Int("deleted_count", len(validIDs)),
		u.logger.Int("skipped_count", len(invalidCampaigns)))

	return nil
}

// GetAgentMessages 獲取代理站內信列表
func (u *AgentUseCase) GetAgentMessages(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) (*dto.AgentMessageListResponse, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetAgentMessages")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("agent.global_id", query.GlobalAgentID),
		attribute.Int("page", query.Page),
		attribute.Int("page_size", query.PageSize))

	// 根據GlobalAgentID獲取AgentID
	u.tracingService.TraceEvent(span, "Getting agent by global ID")
	agent, err := u.agentRepo.GetByGlobalID(ctx, query.GlobalAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get agent by global ID: %w", err)
	}

	// 設定查詢參數
	query.AgentID = agent.ID
	if query.Limit == 0 {
		query.Limit = query.PageSize
	}
	if query.Offset == 0 {
		query.Offset = (query.Page - 1) * query.PageSize
	}

	// 獲取訊息列表
	u.tracingService.TraceEvent(span, "Getting agent messages")
	messages, total, err := u.agentMessageRepo.ListByAgent(ctx, query)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent messages: %w", err)
	}

	// 獲取訊息統計
	u.tracingService.TraceEvent(span, "Getting agent message stats")
	stats, err := u.agentMessageRepo.GetMessageStats(ctx, agent.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get message stats: %w", err)
	}

	// 轉換為回應DTO
	messageResponses := make([]dto.AgentMessageResponse, len(messages))
	for i, message := range messages {
		messageResponses[i] = dto.AgentMessageResponse{
			ID:              message.ID,
			AgentCampaignID: message.AgentCampaignID,
			AgentID:         message.AgentID,
			GlobalAgentID:   message.GlobalAgentID,   // 從JOIN查詢獲取
			Title:           message.CampaignTitle,   // 從JOIN查詢獲取
			Content:         message.CampaignContent, // 從JOIN查詢獲取
			IsRead:          message.IsRead,
			SentAt:          message.CreatedAt, // 代理信發送時間
		}
	}

	response := &dto.AgentMessageListResponse{
		Stats: dto.AgentMessageStats{
			ReadCount:   stats.ReadCount,
			UnreadCount: stats.UnreadCount,
			TotalCount:  stats.TotalCount,
		},
		Messages: messageResponses,
		Page:     query.Page,
		PageSize: query.PageSize,
		Total:    total,
	}

	u.tracingService.TraceEvent(span, "Agent messages retrieved successfully")
	return response, nil
}

// MarkMessageAsRead 標記訊息為已讀
func (u *AgentUseCase) MarkMessageAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	_, span := u.tracingService.StartSpan(ctx, "AgentUseCase.MarkMessageAsRead")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("message.id", int64(messageID)),
		attribute.Int64("agent.id", int64(agentID)))

	// 標記為已讀
	u.tracingService.TraceEvent(span, "Marking message as read")
	if err := u.agentMessageRepo.MarkAsRead(ctx, messageID, agentID); err != nil {
		u.tracingService.RecordSpanError(span, err)
		// 檢查是否是已讀或不存在的錯誤
		if err.Error() == "message not found or already read" {
			u.logger.InfoLog("Message not found or already read",
				u.logger.UInt64("message_id", messageID),
				u.logger.UInt64("agent_id", agentID))
			return fmt.Errorf("record not found or already read")
		}
		return fmt.Errorf("mark message as read: %w", err)
	}

	u.tracingService.TraceEvent(span, "Message marked as read successfully")
	u.logger.InfoLog("Message marked as read successfully",
		u.logger.UInt64("message_id", messageID),
		u.logger.UInt64("agent_id", agentID))

	return nil
}

// SendMessageToCampaignTargets 發送訊息給活動目標（批次處理避免 OOM）
func (u *AgentUseCase) SendMessageToCampaignTargets(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.SendMessageToCampaignTargets")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("target_type", campaign.TargetType))

	switch campaign.TargetType {
	case consts.AgentTargetTypeAll:
		// target_type = all: 批次處理活躍代理，避免記憶體溪漏
		return u.processBatchAgentsForAll(ctx, campaign)
	case consts.AgentTargetTypeSpecific:
		// target_type = specific: 直接批量處理，不過濾活躍度
		if len(campaign.TargetDetails) == 0 {
			return 0, 0, fmt.Errorf("specific target requires target details")
		}
		return u.processBatchAgentsForSpecific(ctx, campaign, campaign.TargetDetails)
	case consts.AgentTargetTypeLine:
		// target_type = line: 批次處理線下代理，避免記憶體溪漏
		if len(campaign.TargetDetails) == 0 {
			return 0, 0, fmt.Errorf("line target requires target details")
		}
		account := campaign.TargetDetails[0] // 頂層父代理
		return u.processBatchAgentsForLine(ctx, campaign, account)
	default:
		return 0, 0, fmt.Errorf("unsupported target type: %s", campaign.TargetType)
	}
}

// GetScheduledCampaigns 獲取到期的排程活動
func (u *AgentUseCase) GetScheduledCampaigns(
	ctx context.Context,
	currentTime time.Time,
) ([]*entity.AgentCampaign, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.GetScheduledCampaigns")
	defer u.tracingService.SpanEnd(span)

	// 查詢條件：scheduled狀態且到期時間已到
	campaigns, err := u.agentCampaignRepo.GetScheduledCampaigns(ctx, currentTime)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get scheduled campaigns: %w", err)
	}

	u.tracingService.TraceEvent(span, "Found scheduled campaigns",
		attribute.Int("count", len(campaigns)))

	return campaigns, nil
}

// FilterActiveAgents 篩選活躍代理（已弃用，保留作為補用函式）
// 新的實作使用 BatchGetActiveAgentsByGlobalIDs 更高效
func (u *AgentUseCase) FilterActiveAgents(
	ctx context.Context,
	globalAgentIDs []string,
) ([]*entity.Agent, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.FilterActiveAgents")
	defer u.tracingService.SpanEnd(span)

	if len(globalAgentIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	// 使用新的批量活躍代理查詢方法，直接在資料庫層過濾
	threshold := time.Now().AddDate(0, -1, 0)
	activeAgents, err := u.agentRepo.BatchGetActiveAgentsByGlobalIDs(ctx, globalAgentIDs, threshold)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("batch get active agents: %w", err)
	}

	u.tracingService.TraceEvent(span, "Active agents filtered with database query",
		attribute.Int("requested_agents", len(globalAgentIDs)),
		attribute.Int("active_agents", len(activeAgents)),
		attribute.String("threshold", threshold.Format(time.RFC3339)))

	return activeAgents, nil
}

// UpdateCampaignStatus 更新活動狀態
func (u *AgentUseCase) UpdateCampaignStatus(
	ctx context.Context,
	campaignID uint64,
	status consts.AgentCampaignStatus,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.UpdateCampaignStatus")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaignID)),
		attribute.String("status", status.String()))

	// 1. 驗證新狀態是否有效
	if !status.IsValid() {
		err := fmt.Errorf("invalid campaign status: %s", status.String())
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	// 2. 獲取當前活動狀態以驗證狀態轉換
	campaign, err := u.agentCampaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("get campaign for status validation: %w", err)
	}
	if campaign == nil {
		err := fmt.Errorf("campaign not found: %d", campaignID)
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	// 3. 驗證狀態轉換是否合法
	currentStatus := campaign.Status
	if !currentStatus.CanTransitionTo(status) {
		err := fmt.Errorf(
			"invalid status transition from %s to %s",
			currentStatus.String(),
			status.String(),
		)
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	// 4. 使用樂觀鎖更新狀態 - 通過 repository 實現條件更新
	err = u.updateCampaignStatusWithOptimisticLock(ctx, campaignID, currentStatus, status)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	u.tracingService.TraceEvent(span, "Campaign status updated successfully",
		attribute.String("previous_status", currentStatus.String()),
		attribute.String("new_status", status.String()))

	u.logger.InfoLog("Campaign status updated",
		u.logger.UInt64("campaign_id", campaignID),
		u.logger.String("previous_status", currentStatus.String()),
		u.logger.String("new_status", status.String()))

	return nil
}

// updateCampaignStatusWithOptimisticLock 使用樂觀鎖機制更新活動狀態
func (u *AgentUseCase) updateCampaignStatusWithOptimisticLock(
	ctx context.Context,
	campaignID uint64,
	expectedStatus consts.AgentCampaignStatus,
	newStatus consts.AgentCampaignStatus,
) error {
	// 使用 UpdateFieldsWithCondition 實現真正的樂觀鎖
	setColumns := map[string]interface{}{
		"status": newStatus.String(),
	}

	conditions := map[string]interface{}{
		"status": expectedStatus.String(),
	}

	err := u.agentCampaignRepo.UpdateFieldsWithCondition(ctx, campaignID, setColumns, conditions)
	if err != nil {
		return fmt.Errorf("failed to update campaign status with optimistic lock: %w", err)
	}

	return nil
}

// CompleteCampaign 完成活動並更新統計
func (u *AgentUseCase) CompleteCampaign(
	ctx context.Context,
	campaignID uint64,
	targetCount, sentCount int,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.CompleteCampaign")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaignID)),
		attribute.Int("target_count", targetCount),
		attribute.Int("sent_count", sentCount))

	setColumn := map[string]interface{}{
		"status":          consts.AgentCampaignStatusSent,
		"target_count":    int64(targetCount),
		"real_sent_count": int64(sentCount),
	}

	if err := u.agentCampaignRepo.UpdateFields(ctx, campaignID, setColumn); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("complete campaign: %w", err)
	}

	u.tracingService.TraceEvent(span, "Campaign completed successfully")
	return nil
}

// resolveLineAgents 解析代理線下的所有代理（使用直接資料庫查詢）
// parentGlobalID: 最上層父代理的全局ID
func (u *AgentUseCase) resolveLineAgents(
	ctx context.Context,
	parentGlobalID string,
) ([]uint64, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.resolveLineAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(
		span,
		attribute.String("parent_global_id", parentGlobalID),
	)

	// 獲取父代理的數值ID
	parentAgent, err := u.agentRepo.GetByGlobalID(ctx, parentGlobalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("get parent agent by global id: %w", err)
	}
	if parentAgent == nil {
		return []uint64{}, nil
	}

	// 直接使用 Repository 方法獲取代理線下的所有代理（更高效）
	descendants, err := u.agentRepo.QueryAgentsByRelationship(ctx, parentGlobalID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		u.logger.WarnLog("Failed to query agents by relationship",
			u.logger.String("parent_global_id", parentGlobalID),
			u.logger.String("error", err.Error()))
		return nil, fmt.Errorf("query agents by relationship: %w", err)
	}

	// 包含父代理自己和所有後代
	allAgentIDs := make([]uint64, 0, len(descendants)+1)
	allAgentIDs = append(allAgentIDs, parentAgent.ID) // 包含父代理自己
	allAgentIDs = append(allAgentIDs, descendants...) // 添加所有後代

	u.tracingService.TraceEvent(span, "Line agents resolved with repository query",
		attribute.Int("descendants_count", len(descendants)),
		attribute.Int("total_agents", len(allAgentIDs)))

	u.logger.InfoLog("Line agents resolved successfully",
		u.logger.String("parent_global_id", parentGlobalID),
		u.logger.Int("total_agents", len(allAgentIDs)))

	return allAgentIDs, nil
}

// createMessagesForAgents 為代理批量創建訊息
func (u *AgentUseCase) createMessagesForAgents(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	agents []*entity.Agent,
) (int, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.createMessagesForAgents")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int("agents_count", len(agents)))

	if len(agents) == 0 {
		return 0, nil
	}

	// 建構代理 ID 列表
	agentIDs := make([]uint64, len(agents))
	for i, agent := range agents {
		agentIDs[i] = agent.ID
	}

	// 檢查已存在的訊息，避免重複發送
	existingMessages, err := u.agentMessageRepo.CheckMessageExistsBatch(ctx, agentIDs, campaign.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, fmt.Errorf("check existing messages: %w", err)
	}

	// 構建需要創建的訊息
	var messages []*entity.AgentMessage
	for _, agent := range agents {
		// 跳過已存在的訊息
		if existingMessages[agent.ID] {
			continue
		}

		message := &entity.AgentMessage{
			AgentCampaignID: campaign.ID,
			AgentID:         agent.ID,
			IsRead:          false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		messages = append(messages, message)
	}

	if len(messages) == 0 {
		u.logger.InfoLog("No new messages to create, all agents already have messages",
			u.logger.UInt64("campaign_id", campaign.ID))
		return 0, nil
	}

	// 批量創建訊息
	if err := u.agentMessageRepo.CreateBatch(ctx, messages); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, fmt.Errorf("create messages batch: %w", err)
	}

	u.tracingService.TraceEvent(span, "Messages created successfully",
		attribute.Int("created_count", len(messages)),
		attribute.Int("skipped_count", len(agents)-len(messages)))

	return len(messages), nil
}

// processBatchAgentsForAll 批次處理 target_type = all 的所有代理
func (u *AgentUseCase) processBatchAgentsForAll(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForAll")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int64("merchant.id", int64(campaign.MerchantID)))

	offset := 0
	limit := 1000 // 批次大小，可配置
	totalTargetCount := 0
	totalSentCount := 0

	u.logger.InfoLog("Starting batch processing for all agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.UInt64("merchant_id", campaign.MerchantID))

	// 分批處理所有代理
	for {
		// 獲取一批代理（移除current_sign_in_at條件篩選）
		agents, err := u.agentRepo.FindAgents(
			ctx,
			campaign.MerchantID,
			limit,
			offset,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return totalTargetCount, totalSentCount, fmt.Errorf("find agents batch: %w", err)
		}

		if len(agents) == 0 {
			break // 沒有更多代理了
		}

		totalTargetCount += len(agents)

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, agents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_size", len(agents)),
				u.logger.Int("offset", offset),
				u.logger.String("error", err.Error()))
			// 繼續處理下一批，不因為一批失敗就停止整個流程
		} else {
			totalSentCount += batchSentCount
		}

		u.logger.InfoLog("Processed batch for all agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(agents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount),
			u.logger.Int("offset", offset))

		// 如果當次查詢結果數量少於 limit，說明已經是最後一批
		if len(agents) < limit {
			break
		}

		offset += limit
	}

	u.tracingService.TraceEvent(span, "Batch processing completed for all agents",
		attribute.Int("target_count", totalTargetCount),
		attribute.Int("sent_count", totalSentCount))

	u.logger.InfoLog("Batch processing completed for all agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.Int("target_count", totalTargetCount),
		u.logger.Int("sent_count", totalSentCount))

	return totalTargetCount, totalSentCount, nil
}

// processBatchAgentsForSpecific 批次處理 target_type = specific 的代理
func (u *AgentUseCase) processBatchAgentsForSpecific(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	targetAccounts []string,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForSpecific")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.Int("target_accounts_count", len(targetAccounts)))

	batchSize := 1000 // 批次大小，可配置
	totalSentCount := 0

	u.logger.InfoLog("Starting batch processing for specific agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.Int("total_targets", len(targetAccounts)))

	// 分批處理指定代理
	for i := 0; i < len(targetAccounts); i += batchSize {
		end := i + batchSize
		if end > len(targetAccounts) {
			end = len(targetAccounts)
		}

		batchAccounts := targetAccounts[i:end]

		// 批量獲取代理詳細資訊
		batchAgents, err := u.agentRepo.BatchGetAgentsByAccounts(
			ctx,
			campaign.MerchantID,
			batchAccounts,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to get batch agents",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_start", i),
				u.logger.Int("batch_size", len(batchAccounts)),
				u.logger.String("error", err.Error()))
			continue // 繼續處理下一批
		}

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, batchAgents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for specific batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_agents", len(batchAgents)),
				u.logger.String("error", err.Error()))
		} else {
			totalSentCount += batchSentCount
		}

		u.logger.InfoLog("Processed batch for specific agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(batchAgents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount))
	}

	u.tracingService.TraceEvent(span, "Batch processing completed for specific agents",
		attribute.Int("target_count", len(targetAccounts)),
		attribute.Int("sent_count", totalSentCount))

	return len(targetAccounts), totalSentCount, nil
}

// processBatchAgentsForLine 批次處理 target_type = line 的代理
func (u *AgentUseCase) processBatchAgentsForLine(
	ctx context.Context,
	campaign *entity.AgentCampaign,
	account string,
) (targetCount int, sentCount int, err error) {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processBatchAgentsForLine")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("account", account))

	parentAgent, err := u.agentRepo.GetByAccount(ctx, campaign.MerchantID, account)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return 0, 0, fmt.Errorf("get agent by account: %w", err)
	}

	batchSize := 1000 // 每批處理的代理數量
	totalTargetCount := 0
	totalSentCount := 0

	u.logger.InfoLog("Starting repository-level batch processing for line agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("parent_agent_id", parentAgent.GlobalAgentID))

	// 創建處理器函數，用於處理每批代理
	processor := func(ctx context.Context, agentIDs []uint64) (processedCount int, err error) {
		// 批量獲取代理詳細資訊
		batchAgents, err := u.agentRepo.BatchGetAgentsByIDs(ctx, agentIDs)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to get batch agents for line processing",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_size", len(agentIDs)),
				u.logger.String("error", err.Error()))
			return 0, err
		}

		// 為當前批次的代理創建訊息
		batchSentCount, err := u.createMessagesForAgents(ctx, campaign, batchAgents)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.WarnLog("Failed to create messages for line batch",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.Int("batch_agents", len(batchAgents)),
				u.logger.String("error", err.Error()))
			return 0, err
		}

		totalTargetCount += len(batchAgents)
		totalSentCount += batchSentCount

		u.logger.InfoLog("Processed batch for line agents",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.Int("batch_agents", len(batchAgents)),
			u.logger.Int("batch_sent", batchSentCount),
			u.logger.Int("total_sent", totalSentCount))

		return batchSentCount, nil
	}

	// 使用 repository 層面的分頁處理
	totalProcessed, err := u.agentRepo.ProcessAgentsByRelationshipInBatches(
		ctx,
		parentAgent.GlobalAgentID,
		batchSize,
		processor,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return totalTargetCount, totalSentCount, fmt.Errorf(
			"process agents by relationship in batches: %w",
			err,
		)
	}

	u.tracingService.TraceEvent(span, "Repository-level line agents batch processing completed",
		attribute.Int("target_count", totalTargetCount),
		attribute.Int("sent_count", totalProcessed))

	u.logger.InfoLog("Completed repository-level batch processing for line agents",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("parent_agent_id", parentAgent.GlobalAgentID),
		u.logger.Int("target_count", totalTargetCount),
		u.logger.Int("sent_count", totalProcessed))

	return totalTargetCount, totalProcessed, nil
}

// BackfillMissedMessages 為超過一個月沒上線的代理補派發訊息
func (u *AgentUseCase) BackfillMissedMessages(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.BackfillMissedMessages")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("agent.global_id", agentEvent.GlobalAgentID),
		attribute.String("merchant.global_id", agentEvent.GlobalMerchantID))

	// 1. 檢查代理是否需要補派發 (超過一個月沒上線)
	agent, err := u.agentRepo.GetByGlobalID(ctx, agentEvent.GlobalAgentID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("get agent by global id: %w", err)
	}

	// 如果代理不存在，無需補派發（可能是新代理或已被刪除）
	if agent == nil {
		u.tracingService.TraceEvent(span, "Agent not found, skipping backfill")
		u.logger.InfoLog("Agent not found, skipping backfill",
			u.logger.String("global_agent_id", agentEvent.GlobalAgentID))
		return nil
	}

	// 2. 獲取商戶信息
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, agentEvent.GlobalMerchantID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 3. 使用批次分頁查詢找出所有需要補派發的訊息活動
	backfilledCount := 0
	totalCampaignsChecked := 0
	const batchSize = 100 // 批次處理大小
	offset := 0

	// 支援的目標類型：all, specific, line
	targetTypes := []string{"all", "specific", "line"}

	for {
		// 分頁查詢活動
		campaigns, err := u.agentCampaignRepo.FindSentCampaignsForBackfillPaginated(
			ctx,
			merchant.ID,
			targetTypes,
			batchSize,
			offset,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return fmt.Errorf("find sent campaigns for backfill: %w", err)
		}

		if len(campaigns) == 0 {
			break // 沒有更多活動
		}

		u.tracingService.TraceEvent(span, "Processing campaign batch",
			attribute.Int("batch_size", len(campaigns)),
			attribute.Int("offset", offset))

		// 4. 批次檢查是否已經有該代理的訊息記錄
		campaignIDs := make([]uint64, len(campaigns))
		for i, campaign := range campaigns {
			campaignIDs[i] = campaign.ID
		}

		// 使用批次檢查避免 N+1 查詢
		existsMap, err := u.agentMessageRepo.CheckCampaignMessageExistsBatch(
			ctx,
			agent.ID,
			campaignIDs,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			u.logger.ErrorLog("Failed to batch check message existence",
				u.logger.Error("err", err),
				u.logger.UInt64("agent_id", agent.ID))
			offset += len(campaigns)
			continue
		}

		// 5. 準備批次創建的訊息
		var messagesToCreate []*entity.AgentMessage
		now := time.Now()

		for _, campaign := range campaigns {
			// 檢查該活動的訊息是否已存在
			if exists, found := existsMap[campaign.ID]; found && exists {
				continue // 訊息已存在，跳過
			}

			// 根據 target_type 判斷該代理是否應該接收此活動
			shouldReceive := false
			switch campaign.TargetType {
			case "all":
				shouldReceive = true // all 類型的活動所有代理都應該接收
				u.logger.InfoLog("Processing 'all' target type campaign",
					u.logger.UInt64("campaign_id", campaign.ID),
					u.logger.UInt64("agent_id", agent.ID))
			case "specific":
				shouldReceive = u.shouldAgentReceiveSpecificCampaign(agent, campaign)
				u.logger.InfoLog("Processing 'specific' target type campaign",
					u.logger.UInt64("campaign_id", campaign.ID),
					u.logger.UInt64("agent_id", agent.ID),
					u.logger.String("agent_account", agent.Account),
					u.logger.Any("target_details", campaign.TargetDetails),
					u.logger.Bool("should_receive", shouldReceive))
			case "line":
				var err error
				shouldReceive, err = u.shouldAgentReceiveLineCampaign(ctx, agent, campaign)
				if err != nil {
					u.logger.WarnLog("Failed to check line campaign eligibility",
						u.logger.UInt64("campaign_id", campaign.ID),
						u.logger.UInt64("agent_id", agent.ID),
						u.logger.Error("error", err))
					continue // 發生錯誤時跳過此活動
				}
				u.logger.InfoLog("Processing 'line' target type campaign",
					u.logger.UInt64("campaign_id", campaign.ID),
					u.logger.UInt64("agent_id", agent.ID),
					u.logger.String("agent_account", agent.Account),
					u.logger.String("agent_ancestry", agent.Ancestry),
					u.logger.Any("target_details", campaign.TargetDetails),
					u.logger.Bool("should_receive", shouldReceive))
			default:
				u.logger.WarnLog("Unknown target type",
					u.logger.String("target_type", campaign.TargetType),
					u.logger.UInt64("campaign_id", campaign.ID))
				continue
			}

			// 如果該代理應該接收此活動，則創建訊息記錄
			if shouldReceive {
				message := &entity.AgentMessage{
					AgentCampaignID: campaign.ID,
					AgentID:         agent.ID,
					IsRead:          false,
					ReadAt:          nil,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				messagesToCreate = append(messagesToCreate, message)
			}
		}

		// 6. 批次創建缺失的訊息記錄
		if len(messagesToCreate) > 0 {
			err := u.agentMessageRepo.CreateBatch(ctx, messagesToCreate)
			if err != nil {
				u.tracingService.RecordSpanError(span, err)
				u.logger.ErrorLog("Failed to batch create backfill messages",
					u.logger.Error("err", err),
					u.logger.Int("message_count", len(messagesToCreate)),
					u.logger.UInt64("agent_id", agent.ID))
			} else {
				backfilledCount += len(messagesToCreate)
				u.tracingService.TraceEvent(span, "Batch created messages",
					attribute.Int("created_count", len(messagesToCreate)))

				// 7. 更新相關 campaign 的 target_count 和 real_sent_count
				campaignCountMap := make(map[uint64]int)
				for _, message := range messagesToCreate {
					campaignCountMap[message.AgentCampaignID]++
				}

				for campaignID, count := range campaignCountMap {
					err := u.agentCampaignRepo.UpdateFields(ctx, campaignID, map[string]interface{}{
						"target_count":    gorm.Expr("target_count + ?", count),
						"real_sent_count": gorm.Expr("real_sent_count + ?", count),
					})
					if err != nil {
						u.logger.WarnLog("Failed to update campaign counts for backfill",
							u.logger.UInt64("campaign_id", campaignID),
							u.logger.Int("count", count),
							u.logger.Error("error", err))
					} else {
						u.logger.InfoLog("Updated campaign counts for backfill",
							u.logger.UInt64("campaign_id", campaignID),
							u.logger.Int("count", count))
					}
				}
			}
		}

		totalCampaignsChecked += len(campaigns)
		offset += len(campaigns)

		// 如果返回的結果少於批次大小，表示已經處理完所有數據
		if len(campaigns) < batchSize {
			break
		}
	}

	u.tracingService.TraceEvent(span, "Message backfill completed",
		attribute.Int("campaigns_checked", totalCampaignsChecked),
		attribute.Int("messages_backfilled", backfilledCount))

	u.logger.InfoLog("Message backfill completed successfully",
		u.logger.String("global_agent_id", agentEvent.GlobalAgentID),
		u.logger.Int("campaigns_checked", totalCampaignsChecked),
		u.logger.Int("messages_backfilled", backfilledCount))

	return nil
}

// shouldAgentReceiveSpecificCampaign 判斷代理是否應該接收 specific target_type 的活動
func (u *AgentUseCase) shouldAgentReceiveSpecificCampaign(
	agent *entity.Agent,
	campaign *entity.AgentCampaign,
) bool {
	if campaign.TargetType != "specific" {
		return false
	}

	// 檢查代理帳號是否在 target_details 中
	for _, targetAccount := range campaign.TargetDetails {
		if agent.Account == targetAccount {
			return true
		}
	}

	return false
}

// shouldAgentReceiveLineCampaign 判斷代理是否應該接收 line target_type 的活動
func (u *AgentUseCase) shouldAgentReceiveLineCampaign(
	ctx context.Context,
	agent *entity.Agent,
	campaign *entity.AgentCampaign,
) (bool, error) {
	if campaign.TargetType != "line" {
		return false, nil
	}

	// line target_type 的 target_details 應該只包含一個上級代理帳號
	if len(campaign.TargetDetails) != 1 {
		return false, fmt.Errorf("line target_type should have exactly one parent agent account")
	}

	parentAccount := campaign.TargetDetails[0]

	// 獲取父代理信息以取得其 GlobalAgentID
	parentAgent, err := u.agentRepo.GetByAccount(ctx, campaign.MerchantID, parentAccount)
	if err != nil {
		return false, fmt.Errorf("get parent agent by account: %w", err)
	}

	// 如果父代理不存在，該 line 活動無效
	if parentAgent == nil {
		return false, fmt.Errorf("parent agent not found with account: %s", parentAccount)
	}

	// 高效能字串比對：檢查代理的 Ancestry 是否包含指定的父代理
	// Ancestry 格式: "FATCAT-AGENT-01/FATCAT-AGENT-02/FATCAT-AGENT-03"
	// 只需檢查父代理的 global_agent_id 是否在 ancestry 路徑中
	if agent.Ancestry == "" {
		return false, nil // 頂級代理，無父代理關係
	}

	// 檢查父代理 GlobalAgentID 是否在 Ancestry 路徑中
	return strings.Contains(agent.Ancestry, parentAgent.GlobalAgentID), nil
}

// processCampaignImmediate 處理立即發送活動
func (u *AgentUseCase) processCampaignImmediate(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "AgentUseCase.processCampaignImmediate")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaign.ID)))

	u.logger.InfoLog("Processing immediate agent campaign",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.String("title", campaign.Title))

	// 1. 冪等性檢查 - 重新獲取活動狀態確保只有 scheduled 狀態才能處理
	u.tracingService.TraceEvent(span, "Checking campaign status for idempotency")
	currentCampaign, err := u.agentCampaignRepo.GetByID(ctx, campaign.ID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("failed to get current campaign status: %w", err)
	}

	// 檢查活動是否還是 scheduled 狀態，如果不是則表示已被處理
	if currentCampaign.Status != consts.AgentCampaignStatusScheduled {
		u.logger.InfoLog("Campaign is no longer in scheduled status, skipping processing",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.String("current_status", currentCampaign.Status.String()),
			u.logger.String("title", campaign.Title))
		return nil // 不是錯誤，只是已經被其他進程處理過了
	}

	// 2. 更新狀態為發送中：sending
	if err := u.UpdateCampaignStatus(ctx, campaign.ID, consts.AgentCampaignStatusSending); err != nil {
		u.logger.ErrorLog("Failed to mark campaign as sending",
			u.logger.UInt64("campaign_id", campaign.ID),
			u.logger.String("update_error", err.Error()))
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	// 3. 委託UseCase執行完整發送流程
	targetCount, sentCount, err := u.SendMessageToCampaignTargets(ctx, campaign)
	if err != nil {
		// 標記活動失敗
		if updateErr := u.UpdateCampaignStatus(ctx, campaign.ID, consts.AgentCampaignStatusFailed); updateErr != nil {
			u.logger.ErrorLog("Failed to mark campaign as failed",
				u.logger.UInt64("campaign_id", campaign.ID),
				u.logger.String("update_error", updateErr.Error()))
		}
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	// 4. 更新統計與完成狀態
	if err = u.CompleteCampaign(ctx, campaign.ID, targetCount, sentCount); err != nil {
		u.tracingService.RecordSpanError(span, err)
		return err
	}

	u.tracingService.TraceEvent(span, "Immediate agent campaign completed successfully",
		attribute.Int("target_count", targetCount),
		attribute.Int("sent_count", sentCount))

	u.logger.InfoLog("Immediate agent campaign completed successfully",
		u.logger.UInt64("campaign_id", campaign.ID),
		u.logger.Int("target_count", targetCount),
		u.logger.Int("sent_count", sentCount))

	return nil
}

// isImmediateSend 檢查是否為立即發送 (status=scheduled 且 scheduled_at=nil)
func (u *AgentUseCase) isImmediateSend(
	status consts.AgentCampaignStatus,
	scheduledAt *time.Time,
) bool {
	return status == consts.AgentCampaignStatusScheduled && scheduledAt == nil
}
