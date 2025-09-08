package message

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
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
		batchSize         = 5000 // 加大批次大小
		dbBatchSize       = 500  // DB操作批次大小
		maxConcurrency    = 10   // 最大並發goroutine數量
		channelBufferSize = 200  // 增加channel緩衝區大小防止阻塞
	)

	var (
		totalSent int64
		offset    int
		wg        sync.WaitGroup
		mu        sync.RWMutex
	)

	// 創建工作channel和結果channel，使用更大的緩衝區
	playerBatches := make(chan []*entity.Player, channelBufferSize)
	results := make(chan int64, maxConcurrency*3) // 確保足夠的緩衝空間
	errorsCh := make(chan error, maxConcurrency*3)

	// 創建可取消的子context
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 啟動worker goroutines
	wg.Add(maxConcurrency)
	for i := 0; i < maxConcurrency; i++ {
		go func() {
			defer wg.Done()
			u.processPlayerBatch(workerCtx, campaignID, playerBatches, results, errorsCh, dbBatchSize)
		}()
	}

	// 啟動數據提供者goroutine
	wg.Add(1)
	go func() {
		defer func() {
			close(playerBatches)
			wg.Done()
		}()

		for {
			select {
			case <-workerCtx.Done():
				return
			default:
			}

			players, err := u.playerRepo.FindByTargetType(workerCtx, campaign.Target, offset, batchSize)
			if err != nil {
				select {
				case errorsCh <- fmt.Errorf("find players: %w", err):
				case <-workerCtx.Done():
				}
				return
			}

			if len(players) == 0 {
				break
			}

			select {
			case playerBatches <- players:
			case <-workerCtx.Done():
				return
			}

			offset += batchSize

			if len(players) < batchSize {
				break
			}
		}
	}()

	// 啟動結果收集goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case count := <-results:
				mu.Lock()
				totalSent += count
				mu.Unlock()
			case err := <-errorsCh:
				if err != nil {
					u.logger.ErrorLog("Error in batch processing", u.logger.Error("err", err))
				}
			case <-workerCtx.Done():
				return
			}
		}
	}()

	// 創建一個channel來等待WaitGroup完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待所有goroutines完成或context取消
	select {
	case <-done:
		// 所有goroutines正常完成
	case <-ctx.Done():
		cancel() // 取消所有worker goroutines
		<-done   // 等待所有goroutines清理完成
		return ctx.Err()
	}

	// 確保獲取最終結果
	mu.RLock()
	finalTotalSent := totalSent
	mu.RUnlock()

	// 更新發送統計
	if err := u.campaignRepo.UpdateSentCount(ctx, campaignID, finalTotalSent); err != nil {
		u.logger.WarnLog("Failed to update sent count",
			u.logger.Int64("campaign_id", int64(campaignID)),
			u.logger.Int64("sent_count", finalTotalSent),
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

// processPlayerBatch 處理玩家批次的工作函數
func (u *MessageUseCase) processPlayerBatch(
	ctx context.Context,
	campaignID uint64,
	playerBatches <-chan []*entity.Player,
	results chan<- int64,
	errorsCh chan<- error,
	dbBatchSize int,
) {
	for {
		select {
		case players, ok := <-playerBatches:
			if !ok {
				// channel已關閉，退出
				return
			}
			
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
