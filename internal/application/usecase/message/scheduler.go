package message

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
)

// ProcessScheduledCampaigns 處理排程的活動
func (u *MessageUseCase) ProcessScheduledCampaigns(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.ProcessScheduledCampaigns")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Processing scheduled campaigns")

	campaigns, err := u.campaignRepo.FindScheduledCampaigns(ctx)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find scheduled campaigns: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.Int("scheduled_campaigns", len(campaigns)))

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

	tracing.TraceEvent(span, "Scheduled campaigns processed")
	return nil
}

// SendCampaignToPlayers 發送活動給符合條件的玩家
func (u *MessageUseCase) SendCampaignToPlayers(ctx context.Context, campaignID uint64) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.SendCampaignToPlayers")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaignID)))

	campaign, err := u.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.Int("campaign.target", int(campaign.Target)),
	)

	const batchSize = 1000
	var totalSent int64
	offset := 0

	for {
		players, err := u.playerRepo.FindByTargetType(ctx, campaign.Target, offset, batchSize)
		if err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("find players: %w", err)
		}

		if len(players) == 0 {
			break
		}

		var newMessages []*entity.PlayerMessage
		for _, player := range players {
			exists, err := u.playerMessageRepo.CheckMessageExists(
				ctx,
				player.GlobalPlayerID,
				campaignID,
			)
			if err != nil {
				u.logger.WarnLog("Failed to check message existence",
					u.logger.String("global_player_id", player.GlobalPlayerID),
					u.logger.Int64("campaign_id", int64(campaignID)),
					u.logger.Error("err", err))
				continue
			}

			if !exists {
				message := &entity.PlayerMessage{
					GlobalPlayerID: player.GlobalPlayerID,
					PlayerID:       player.ID,
					CampaignID:     campaignID,
					IsRead:         false,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				newMessages = append(newMessages, message)
			}
		}

		if len(newMessages) > 0 {
			if err := u.playerMessageRepo.CreateBatch(ctx, newMessages); err != nil {
				tracing.RecordSpanError(span, err)
				return fmt.Errorf("create batch messages: %w", err)
			}
			totalSent += int64(len(newMessages))
		}

		if len(players) < batchSize {
			break
		}

		offset += batchSize
	}

	if err := u.campaignRepo.UpdateSentCount(ctx, campaignID, totalSent); err != nil {
		u.logger.WarnLog("Failed to update sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", totalSent),
			u.logger.Error("err", err))
	}

	// 更新活動狀態為已發送
	if err := u.campaignRepo.UpdateStatus(ctx, campaignID, consts.MessageCampaignStatusSent); err != nil {
		u.logger.WarnLog("Failed to update campaign status to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("status", int(consts.MessageCampaignStatusSent)),
			u.logger.Error("err", err))
	} else {
		u.logger.InfoLog("Campaign status updated to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("status", int(consts.MessageCampaignStatusSent)))
	}

	tracing.RecordSpanAttributes(span,
		attribute.Int64("total_sent", totalSent),
		attribute.Int("final_status", int(consts.MessageCampaignStatusSent)),
	)

	tracing.TraceEvent(span, "Campaign sent to players and status updated")

	u.logger.InfoLog("Campaign sent to players",
		u.logger.Int64("campaign_id", int64(campaignID)),
		u.logger.String("title", campaign.Title),
		u.logger.Int64("total_sent", totalSent))

	return nil
}

// SendCampaignToPlayersAsync 高性能異步發送活動給符合條件的玩家
func (u *MessageUseCase) SendCampaignToPlayersAsync(ctx context.Context, campaignID uint64) error {
	ctx, span := tracing.StartSpan(ctx, "MessageUseCase.SendCampaignToPlayersAsync")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("campaign.id", int64(campaignID)))

	campaign, err := u.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find campaign: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("campaign.title", campaign.Title),
		attribute.Int("campaign.target", int(campaign.Target)),
	)

	// 配置參數
	const (
		operationTimeout = 30 * time.Second // 操作時長為 30 秒
		batchSize        = 5000             // 每批處理的玩家數量
		dbBatchSize      = 500              // DB操作批次大小
		maxConcurrency   = 10               // 最大並發goroutine數量
	)

	// 使用 atomic 操作統計發送數量，避免 mutex 開銷
	var totalSent int64

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

	// 獲取處理過程中已發送的數量
	finalTotalSent := atomic.LoadInt64(&totalSent)

	// 等待所有 goroutines 完成，任何錯誤都會導致其他 goroutines 被取消
	if waitErr := g.Wait(); waitErr != nil {
		tracing.RecordSpanError(span, waitErr)

		// 發生錯誤時的回滾策略：保留已處理數據，更新統計並標記為失敗
		u.logger.ErrorLog("Campaign processing failed, performing partial rollback",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("partial_sent_count", finalTotalSent),
			u.logger.Error("error", waitErr))

		// 更新已發送的數量統計（即使失敗也要記錄已處理的數據）
		if updateSentErr := u.campaignRepo.UpdateSentCount(ctx, campaignID, finalTotalSent); updateSentErr != nil {
			u.logger.ErrorLog("Failed to update sent count during rollback",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Int64("sent_count", finalTotalSent),
				u.logger.Error("err", updateSentErr))
		}

		// 將活動狀態標記為失敗
		if updateStatusErr := u.campaignRepo.UpdateStatus(ctx, campaignID, consts.MessageCampaignStatusFailed); updateStatusErr != nil {
			u.logger.ErrorLog("Failed to update campaign status to failed during rollback",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Error("err", updateStatusErr))
		} else {
			u.logger.InfoLog("Campaign status updated to failed",
				u.logger.Int64("campaign_id", int64(campaignID)),
				u.logger.Int64("partial_sent_count", finalTotalSent))
		}

		// 記錄部分成功的追蹤信息
		tracing.RecordSpanAttributes(span,
			attribute.Int64("partial_sent", finalTotalSent),
			attribute.Int("final_status", int(consts.MessageCampaignStatusFailed)),
			attribute.Bool("partial_success", finalTotalSent > 0),
		)

		return fmt.Errorf(
			"campaign processing failed after sending %d messages: %w",
			finalTotalSent,
			waitErr,
		)
	}

	// 正常完成：更新發送統計
	if updateSentErr := u.campaignRepo.UpdateSentCount(ctx, campaignID, finalTotalSent); updateSentErr != nil {
		u.logger.WarnLog("Failed to update sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", finalTotalSent),
			u.logger.Error("err", updateSentErr))
	}

	// 正常完成：更新活動狀態為已發送
	if updateStatusErr := u.campaignRepo.UpdateStatus(ctx, campaignID, consts.MessageCampaignStatusSent); updateStatusErr != nil {
		u.logger.WarnLog("Failed to update campaign status to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("status", int(consts.MessageCampaignStatusSent)),
			u.logger.Error("err", updateStatusErr))
	} else {
		u.logger.InfoLog("Campaign status updated to sent",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int("status", int(consts.MessageCampaignStatusSent)))
	}

	tracing.RecordSpanAttributes(span,
		attribute.Int64("total_sent", finalTotalSent),
		attribute.Int("final_status", int(consts.MessageCampaignStatusSent)),
	)

	tracing.TraceEvent(span, "Campaign sent to players asynchronously and status updated")

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

		players, err := u.playerRepo.FindByTargetType(ctx, campaign.Target, offset, batchSize)
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
			atomic.AddInt64(totalSent, count)

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
		if err := u.playerMessageRepo.CreateBatchOptimized(ctx, newMessages, dbBatchSize); err != nil {
			return 0, fmt.Errorf("create batch messages: %w", err)
		}
	}

	return int64(len(newMessages)), nil
}
