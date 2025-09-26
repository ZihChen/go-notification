package message

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/security"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
)

// ProcessScheduledCampaigns 處理排程的活動
func (u *MessageUseCase) ProcessScheduledCampaigns(ctx context.Context) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.ProcessScheduledCampaigns")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.TraceEvent(span, "Processing scheduled campaigns")

	campaigns, err := u.campaignRepo.FindScheduledCampaigns(ctx)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find scheduled campaigns: %w", err)
	}

	u.tracingService.RecordSpanAttributes(
		span,
		attribute.Int("scheduled_campaigns", len(campaigns)),
	)

	for _, campaign := range campaigns {
		if err = u.SendCampaignToPlayersAsync(ctx, campaign.ID); err != nil {
			u.logger.ErrorLog("Failed to send campaign",
				u.logger.Int64("campaign_id", int64(campaign.ID)),
				u.logger.String("title", campaign.Title),
				u.logger.Error("err", err))
			continue
		}

		u.logger.InfoLog("Successfully processed scheduled campaign",
			u.logger.Int64("campaign_id", int64(campaign.ID)),
			u.logger.String("title", campaign.Title))
	}

	u.tracingService.TraceEvent(span, "Scheduled campaigns processed")
	return nil
}

// SendCampaignToPlayersAsync 高性能異步發送活動給符合條件的玩家
func (u *MessageUseCase) SendCampaignToPlayersAsync(ctx context.Context, campaignID uint64) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.SendCampaignToPlayersAsync")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaignID)))

	campaign, err := u.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.String("campaign.target", campaign.Target),
		attribute.Int("campaign.notification_types", int(campaign.NotificationTypes)),
	)

	// 檢查推播類型並處理App推播邏輯
	notificationType := consts.NotificationType(campaign.NotificationTypes)
	shouldSendPushNotification := notificationType.HasAppPush() // 包含App推播
	shouldCreatePlayerMessage := notificationType.HasInApp()    // 包含站內信

	if shouldSendPushNotification {
		u.logger.InfoLog("Campaign requires push notification",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("notification_types", int(campaign.NotificationTypes)))

		// 發送App推播
		if err := u.sendAppPushNotification(ctx, campaign); err != nil {
			u.logger.ErrorLog("Failed to send app push notification",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Error("error", err))
			// 推播失敗不影響主流程，繼續處理站內信
		}
	}

	// 如果只有App推播(notification_types = 2)，不需要建立player_message
	if !shouldCreatePlayerMessage {
		u.logger.InfoLog("Campaign is push-only, skipping player message creation",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("notification_types", int(campaign.NotificationTypes)))

		// 直接更新狀態為已發送
		if err := u.campaignRepo.UpdateStatus(ctx, campaignID, consts.MessageCampaignStatusSent); err != nil {
			u.logger.WarnLog("Failed to update push-only campaign status",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Error("err", err))
		}

		u.tracingService.TraceEvent(span, "Push-only campaign completed")
		return nil
	}

	// 配置參數
	const (
		operationTimeout = 30 * time.Second // 操作時長為 30 秒
		batchSize        = 5000             // 每批處理的玩家數量
		dbBatchSize      = 500              // DB操作批次大小
		maxConcurrency   = 10               // 最大並發goroutine數量
	)

	// 使用 atomic 操作統計發送數量，避免 mutex 開銷
	var totalSent int64

	// 保存原始 context 用於最終的狀態更新
	originalCtx := ctx

	// 創建 errgroup 統一管理 goroutines 和錯誤處理
	timeoutCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	g, ctx := errgroup.WithContext(timeoutCtx)
	g.SetLimit(maxConcurrency) // 限制並發數量

	// 創建工作 channel，簡化為單一 channel
	playerBatches := make(chan []*entity.Player, maxConcurrency*2)

	// 數據生產者：分頁讀取玩家並發送到 channel
	g.Go(func() error {
		defer close(playerBatches)
		return u.producePlayerBatches(ctx, campaign, batchSize, playerBatches)
	})

	// 數據消費者：並發處理玩家批次
	for i := 0; i < maxConcurrency; i++ {
		g.Go(func() error {
			return u.consumePlayerBatches(ctx, campaignID, playerBatches, dbBatchSize, &totalSent)
		})
	}

	// 等待所有 goroutines 完成，任何錯誤都會導致其他 goroutines 被取消
	if waitErr := g.Wait(); waitErr != nil {
		// 獲取處理過程中已發送的數量（錯誤情況下）
		finalTotalSent := atomic.LoadInt64(&totalSent)
		u.tracingService.RecordSpanError(span, waitErr)

		// 發生錯誤時的回滾策略：保留已處理數據，更新統計並標記為失敗
		u.logger.ErrorLog("Campaign processing failed, performing partial rollback",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("partial_sent_count", finalTotalSent),
			u.logger.Error("error", waitErr))

		// 更新已發送的數量統計（即使失敗也要記錄已處理的數據）
		if updateSentErr := u.campaignRepo.UpdateSentCount(originalCtx, campaignID, finalTotalSent); updateSentErr != nil {
			u.logger.ErrorLog("Failed to update sent count during rollback",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Int64("sent_count", finalTotalSent),
				u.logger.Error("err", updateSentErr))
		}

		// 將活動狀態標記為失敗
		if updateStatusErr := u.campaignRepo.UpdateStatus(originalCtx, campaignID, consts.MessageCampaignStatusFailed); updateStatusErr != nil {
			u.logger.ErrorLog("Failed to update campaign status to failed during rollback",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Error("err", updateStatusErr))
		} else {
			u.logger.InfoLog("Campaign status updated to failed",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Int64("partial_sent_count", finalTotalSent))
		}

		// 記錄部分成功的追蹤信息
		u.tracingService.RecordSpanAttributes(span,
			attribute.Int64("partial_sent", finalTotalSent),
			attribute.String("final_status", consts.MessageCampaignStatusFailed),
			attribute.Bool("partial_success", finalTotalSent > 0),
		)

		return fmt.Errorf(
			"campaign processing failed after sending %d messages: %w",
			finalTotalSent,
			waitErr,
		)
	}

	// 獲取最終已發送的數量（成功情況下）
	finalTotalSent := atomic.LoadInt64(&totalSent)

	// 正常完成：更新發送統計
	u.logger.InfoLog("Updating sent count for campaign",
		u.logger.Int64("campaign_id", int64(campaignID)),
		u.logger.Int64("final_total_sent", finalTotalSent),
		u.logger.String("method", "SendCampaignToPlayersAsync"))

	if updateSentErr := u.campaignRepo.UpdateSentCount(originalCtx, campaignID, finalTotalSent); updateSentErr != nil {
		u.logger.WarnLog("Failed to update sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", finalTotalSent),
			u.logger.Error("err", updateSentErr))
	} else {
		u.logger.InfoLog("Successfully updated sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", finalTotalSent))
	}

	// 正常完成：更新活動狀態為已發送
	if updateStatusErr := u.campaignRepo.UpdateStatus(originalCtx, campaignID, consts.MessageCampaignStatusSent); updateStatusErr != nil {
		u.logger.WarnLog("Failed to update campaign status to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.String("status", consts.MessageCampaignStatusSent),
			u.logger.Error("err", updateStatusErr))
	} else {
		u.logger.InfoLog("Campaign status updated to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.String("status", consts.MessageCampaignStatusSent))
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("total_sent", finalTotalSent),
		attribute.String("final_status", consts.MessageCampaignStatusSent),
	)

	u.tracingService.TraceEvent(span, "Campaign sent to players asynchronously and status updated")

	u.logger.InfoLog("Campaign sent to players asynchronously",
		u.logger.Int64("campaign_id", int64(campaignID)),
		u.logger.String("title", campaign.Title),
		u.logger.Int64("total_sent", finalTotalSent))

	return nil
}

// producePlayerBatches 生產者：分頁讀取玩家數據並發送到 channel
func (u *MessageUseCase) producePlayerBatches(
	ctx context.Context,
	campaign *entity.MessageCampaign,
	batchSize int,
	playerBatches chan<- []*entity.Player,
) error {
	offset := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		players, err := u.playerRepo.FindByTargetType(
			ctx,
			campaign.Target,
			campaign.TargetDetail,
			offset,
			batchSize,
		)
		if err != nil {
			return fmt.Errorf("find players at offset %d: %w", offset, err)
		}

		// 沒有更多數據，正常結束
		if len(players) == 0 {
			return nil
		}

		// 發送數據批次
		select {
		case playerBatches <- players:
		case <-ctx.Done():
			return ctx.Err()
		}

		offset += batchSize

		// 如果這批數據少於預期大小，說明已經到了最後一批
		if len(players) < batchSize {
			return nil
		}
	}
}

// consumePlayerBatches 消費者：處理玩家批次並更新統計
func (u *MessageUseCase) consumePlayerBatches(
	ctx context.Context,
	campaignID uint64,
	playerBatches <-chan []*entity.Player,
	dbBatchSize int,
	totalSent *int64,
) error {
	for {
		select {
		case players, ok := <-playerBatches:
			if !ok {
				// channel 已關閉，正常結束
				return nil
			}

			count, err := u.processSingleBatch(ctx, campaignID, players, dbBatchSize)
			if err != nil {
				return fmt.Errorf("process batch: %w", err)
			}

			// 原子操作更新總數
			oldTotal := atomic.LoadInt64(totalSent)
			atomic.AddInt64(totalSent, count)
			newTotal := atomic.LoadInt64(totalSent)

			u.logger.InfoLog("Updated total sent count",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Int64("batch_count", count),
				u.logger.Int64("old_total", oldTotal),
				u.logger.Int64("new_total", newTotal))

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// processSingleBatch 處理單一玩家批次
func (u *MessageUseCase) processSingleBatch(
	ctx context.Context,
	campaignID uint64,
	players []*entity.Player,
	dbBatchSize int,
) (int64, error) {
	if len(players) == 0 {
		return 0, nil
	}

	// 提取玩家ID列表
	playerIDs := make([]uint64, len(players))
	playerMap := make(map[uint64]*entity.Player)

	for i, player := range players {
		playerIDs[i] = player.ID
		playerMap[player.ID] = player
	}

	// 批次檢查已存在的訊息
	existsMap, err := u.playerMessageRepo.CheckMessageExistsBatch(ctx, playerIDs, campaignID)
	if err != nil {
		return 0, fmt.Errorf("check message exists batch: %w", err)
	}

	// 建立新訊息列表
	var newMessages []*entity.PlayerMessage
	now := time.Now()

	for playerID, exists := range existsMap {
		if !exists {
			if player, ok := playerMap[playerID]; ok {
				message := &entity.PlayerMessage{
					GlobalPlayerID: player.GlobalPlayerID,
					PlayerID:       playerID,
					CampaignID:     campaignID,
					IsRead:         false,
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				newMessages = append(newMessages, message)
			}
		}
	}

	// 批次創建新訊息
	if len(newMessages) > 0 {
		u.logger.InfoLog("Creating optimized batch messages",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("new_messages_count", len(newMessages)),
			u.logger.Int("db_batch_size", dbBatchSize))

		if err := u.playerMessageRepo.CreateBatchOptimized(ctx, newMessages, dbBatchSize); err != nil {
			return 0, fmt.Errorf("create batch messages: %w", err)
		}

		u.logger.InfoLog("Optimized batch messages created successfully",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("created_count", len(newMessages)))
	}

	return int64(len(newMessages)), nil
}

// sendAppPushNotification 發送App推播通知（批次處理所有目標玩家）
func (u *MessageUseCase) sendAppPushNotification(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.sendAppPushNotification")
	defer u.tracingService.SpanEnd(span)

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign.id", int64(campaign.ID)),
		attribute.String("campaign.title", campaign.Title),
	)

	// 查詢商戶的推播API key
	pushApiKey, err := u.pushApiKeyRepo.FindByMerchantID(ctx, campaign.MerchantID)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoPushKeyNotFound) {
			u.logger.WarnLog("Merchant push API key not found, skipping push notification",
				u.logger.Int64("campaign_id", int64(campaign.ID)),
				u.logger.UInt64("merchant_id", campaign.MerchantID))
			return nil // 沒有API key不算錯誤，跳過推播
		}
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant push API key: %w", err)
	}

	// 準備推播內容
	pushContent := campaign.Content // 預設使用一般內容
	if campaign.AppContent != nil && *campaign.AppContent != "" {
		pushContent = *campaign.AppContent // 如果有專用App內容則使用
	}

	// 批次處理所有目標玩家
	totalPushed, err := u.processPushNotificationInBatches(
		ctx,
		campaign,
		pushApiKey.Key,
		pushContent,
	)
	if err != nil {
		u.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("process push notification in batches: %w", err)
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("total_pushed", totalPushed),
		attribute.String("push_content", pushContent),
	)

	u.logger.InfoLog("Push notification completed",
		u.logger.Int64("campaign_id", int64(campaign.ID)),
		u.logger.Int64("total_players_pushed", totalPushed))

	return nil
}

// processPushNotificationInBatches 批次處理推播通知給所有目標玩家
func (u *MessageUseCase) processPushNotificationInBatches(
	ctx context.Context,
	campaign *entity.MessageCampaign,
	apiKey string,
	pushContent string,
) (int64, error) {
	ctx, span := u.tracingService.StartSpan(ctx, "MessageUseCase.processPushNotificationInBatches")
	defer u.tracingService.SpanEnd(span)

	const (
		pushBatchSize  = 1000 // 每次推播的玩家數量限制
		queryBatchSize = 5000 // 每次查詢的玩家數量
	)

	var totalPushed int64
	offset := 0

	for {
		// 查詢一批目標玩家
		players, err := u.playerRepo.FindByTargetType(
			ctx,
			campaign.Target,
			campaign.TargetDetail,
			offset,
			queryBatchSize,
		)
		if err != nil {
			u.tracingService.RecordSpanError(span, err)
			return totalPushed, fmt.Errorf("find players at offset %d: %w", offset, err)
		}

		if len(players) == 0 {
			break // 沒有更多玩家
		}

		u.logger.InfoLog("Processing push batch",
			u.logger.Int64("campaign_id", int64(campaign.ID)),
			u.logger.Int("batch_size", len(players)),
			u.logger.Int("offset", offset),
			u.logger.Int64("total_pushed_so_far", totalPushed))

		// 如果這批玩家超過推播批次大小，需要分割
		for i := 0; i < len(players); i += pushBatchSize {
			end := i + pushBatchSize
			if end > len(players) {
				end = len(players)
			}

			batchPlayers := players[i:end]
			pushedCount, err := u.sendPushNotificationBatch(
				ctx,
				campaign,
				apiKey,
				pushContent,
				batchPlayers,
			)
			if err != nil {
				u.logger.ErrorLog("Failed to send push notification batch",
					u.logger.Int64("campaign_id", int64(campaign.ID)),
					u.logger.Int("batch_start", i),
					u.logger.Int("batch_size", len(batchPlayers)),
					u.logger.Error("error", err))
				// 記錄錯誤但繼續處理下一批
				continue
			}
			totalPushed += pushedCount
		}

		// 如果這批數據少於預期大小，說明已經到了最後一批
		if len(players) < queryBatchSize {
			break
		}

		offset += queryBatchSize
	}

	u.tracingService.RecordSpanAttributes(span,
		attribute.Int64("total_pushed", totalPushed),
		attribute.String("campaign_target", campaign.Target),
	)

	u.logger.InfoLog("Completed push notification batches",
		u.logger.Int64("campaign_id", int64(campaign.ID)),
		u.logger.Int64("total_players_pushed", totalPushed))

	return totalPushed, nil
}

// sendPushNotificationBatch 發送單批推播通知
func (u *MessageUseCase) sendPushNotificationBatch(
	ctx context.Context,
	campaign *entity.MessageCampaign,
	apiKey string,
	pushContent string,
	players []*entity.Player,
) (int64, error) {
	if len(players) == 0 {
		return 0, nil
	}

	// 提取玩家帳號
	playerAccounts := make([]string, len(players))
	for i, player := range players {
		playerAccounts[i] = player.Account
	}

	// 發送推播請求
	pushRequest := &service.PushNotificationRequest{
		Accounts: playerAccounts,
		MsgTitle: campaign.Title,
		MsgBody:  pushContent,
	}

	u.logger.InfoLog("Sending push notification batch",
		u.logger.Int64("campaign_id", int64(campaign.ID)),
		u.logger.Int("batch_size", len(playerAccounts)),
		u.logger.String("api_key", security.SanitizeAPIKey(apiKey)))

	response, err := u.pushService.SendPushNotification(ctx, apiKey, pushRequest)
	if err != nil {
		return 0, fmt.Errorf("send push notification: %w", err)
	}

	if response.Success {
		u.logger.InfoLog("Push notification batch sent successfully",
			u.logger.Int64("campaign_id", int64(campaign.ID)),
			u.logger.Int("batch_size", len(playerAccounts)),
			u.logger.String("response", response.Message))
		return int64(len(playerAccounts)), nil
	} else {
		u.logger.WarnLog("Push notification batch failed",
			u.logger.Int64("campaign_id", int64(campaign.ID)),
			u.logger.Int("batch_size", len(playerAccounts)),
			u.logger.String("error_message", response.Message))
		return 0, fmt.Errorf("push notification failed: %s", response.Message)
	}
}
