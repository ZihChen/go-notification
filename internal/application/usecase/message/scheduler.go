package message

import (
	"context"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
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
		if err := u.SendCampaignToPlayersAsync(ctx, campaign.ID); err != nil {
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

	tracing.RecordSpanAttributes(span,
		attribute.Int64("total_sent", totalSent),
	)

	tracing.TraceEvent(span, "Campaign sent to players")

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
		batchSize         = 5000 // 加大批次大小
		dbBatchSize       = 500  // DB操作批次大小
		maxConcurrency    = 10   // 最大並發goroutine數量
		channelBufferSize = 100  // channel緩衝區大小
	)

	var totalSent int64
	offset := 0

	// 創建工作channel和結果channel
	playerBatches := make(chan []*entity.Player, channelBufferSize)
	results := make(chan int64, channelBufferSize)
	errorsCh := make(chan error, channelBufferSize)

	// 啟動worker goroutines
	for i := 0; i < maxConcurrency; i++ {
		go u.processPlayerBatch(ctx, campaignID, playerBatches, results, errorsCh, dbBatchSize)
	}

	// 啟動數據提供者goroutine
	go func() {
		defer close(playerBatches)

		for {
			players, err := u.playerRepo.FindByTargetType(ctx, campaign.Target, offset, batchSize)
			if err != nil {
				select {
				case errorsCh <- fmt.Errorf("find players: %w", err):
				case <-ctx.Done():
				}
				return
			}

			if len(players) == 0 {
				break
			}

			select {
			case playerBatches <- players:
			case <-ctx.Done():
				return
			}

			offset += batchSize

			if len(players) < batchSize {
				break
			}
		}
	}()

	// 收集結果
	completed := 0

	// 等待所有結果
	for completed < maxConcurrency {
		select {
		case count := <-results:
			totalSent += count
		case err := <-errorsCh:
			if err != nil {
				u.logger.ErrorLog("Error in batch processing", u.logger.Error("err", err))
			}
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 檢查是否還有活躍的worker
			if len(playerBatches) == 0 {
				completed++
			}
		}
	}

	// 更新發送統計
	if err := u.campaignRepo.UpdateSentCount(ctx, campaignID, totalSent); err != nil {
		u.logger.WarnLog("Failed to update sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", totalSent),
			u.logger.Error("err", err))
	}

	tracing.RecordSpanAttributes(span,
		attribute.Int64("total_sent", totalSent),
	)

	tracing.TraceEvent(span, "Campaign sent to players asynchronously")

	u.logger.InfoLog("Campaign sent to players asynchronously",
		u.logger.Int64("campaign_id", int64(campaignID)),
		u.logger.String("title", campaign.Title),
		u.logger.Int64("total_sent", totalSent))

	return nil
}

// processPlayerBatch 處理玩家批次的工作函數
func (u *MessageUseCase) processPlayerBatch(
	ctx context.Context,
	campaignID uint64,
	playerBatches <-chan []*entity.Player,
	results chan<- int64,
	errorsCh chan<- error,
	dbBatchSize int,
) {
	defer func() {
		// 發送完成信號
		select {
		case results <- 0:
		case <-ctx.Done():
		}
	}()

	for players := range playerBatches {
		count, err := u.processSingleBatch(ctx, campaignID, players, dbBatchSize)
		if err != nil {
			select {
			case errorsCh <- err:
			case <-ctx.Done():
				return
			}
			continue
		}

		select {
		case results <- count:
		case <-ctx.Done():
			return
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
