package player

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
)

var (
	ErrBatchProcessorFull       = errors.New("batch processor channel is full")
	ErrBatchProcessorStopped    = errors.New("batch processor is stopped")
	ErrBatchProcessorNotStarted = errors.New("batch processor is not started")
)

// PlayerBatchRequest 玩家批次處理請求
type PlayerBatchRequest struct {
	Player            *entity.Player
	GlobalMerchantID  string
	CompletionChannel chan<- error // 異步結果回傳
}

// PlayerBatchProcessor 玩家批次處理器
type PlayerBatchProcessor struct {
	// Repository 和 Service 依賴
	playerRepo repository.PlayerRepository
	logger     infrastructure.Logger
	tracing    infrastructure.TracingService

	// 批次配置
	batchSize    int           // 500
	batchTimeout time.Duration // 3秒
	bufferSize   int           // 1000 Channel緩衝大小

	// Channel 管理
	requestChannel chan *PlayerBatchRequest
	stopChannel    chan struct{}

	// 狀態管理
	mu      sync.RWMutex
	wg      sync.WaitGroup
	started bool
}

// NewPlayerBatchProcessor 創建新的玩家批次處理器
func NewPlayerBatchProcessor(
	playerRepo repository.PlayerRepository,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) *PlayerBatchProcessor {
	bufferSize := 1000 // Channel 緩衝大小

	return &PlayerBatchProcessor{
		playerRepo: playerRepo,
		logger:     logger,
		tracing:    tracing,

		// 批次配置：500筆或3秒超時
		batchSize:    500,
		batchTimeout: 3 * time.Second,
		bufferSize:   bufferSize,

		requestChannel: make(chan *PlayerBatchRequest, bufferSize),
		stopChannel:    make(chan struct{}),
	}
}

// Start 啟動批次處理器
func (p *PlayerBatchProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("batch processor already started")
	}

	p.started = true

	// 啟動批次處理 Goroutine
	p.wg.Add(1)
	go p.processBatches(ctx)

	p.logger.InfoLog("Player batch processor started",
		p.logger.Int("batch_size", p.batchSize),
		p.logger.String("batch_timeout", p.batchTimeout.String()),
		p.logger.Int("buffer_size", p.bufferSize))

	return nil
}

// Stop 停止批次處理器
func (p *PlayerBatchProcessor) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return ErrBatchProcessorNotStarted
	}
	p.mu.Unlock()

	close(p.stopChannel)
	// 等待 goroutine 結束 (不能在持有鎖的情況下等待)
	p.wg.Wait()

	p.mu.Lock()
	p.started = false
	p.mu.Unlock()

	p.logger.InfoWithContext(ctx, "Player batch processor stopped")
	return nil
}

// SubmitPlayer 提交玩家到批次處理
func (p *PlayerBatchProcessor) SubmitPlayer(
	player *entity.Player,
	globalMerchantID string,
) <-chan error {
	completionChannel := make(chan error, 1)

	request := &PlayerBatchRequest{
		Player:            player,
		GlobalMerchantID:  globalMerchantID,
		CompletionChannel: completionChannel,
	}

	// 降級處理機制：channel滿了自動降級
	select {
	case p.requestChannel <- request:
		// 正常批次處理
	default:
		// Channel滿了，直接降級為同步處理
		p.logger.WarnLog("Batch processor channel full, falling back to synchronous processing")
		go p.handleSyncFallback(request)
	}

	return completionChannel
}

// handleSyncFallback 同步降級處理
func (p *PlayerBatchProcessor) handleSyncFallback(request *PlayerBatchRequest) {
	defer close(request.CompletionChannel)

	ctx := context.Background()

	// 單筆資料庫操作
	err := p.playerRepo.Upsert(ctx, request.Player)
	if err != nil {
		p.logger.ErrorLog("Fallback sync upsert failed",
			p.logger.String("global_player_id", request.Player.GlobalPlayerID),
			p.logger.Error("error", err))
		request.CompletionChannel <- err
		return
	}
	request.CompletionChannel <- nil
}

// processBatches 處理批次請求的主要Goroutine
func (p *PlayerBatchProcessor) processBatches(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.batchTimeout)
	defer ticker.Stop()

	var currentBatch []*PlayerBatchRequest

	for {
		select {
		case request := <-p.requestChannel:
			currentBatch = append(currentBatch, request)

			// 檢查是否達到批次大小
			if len(currentBatch) >= p.batchSize {
				p.processBatch(ctx, currentBatch)
				currentBatch = currentBatch[:0] // 清空slice但保留容量
				ticker.Reset(p.batchTimeout)    // 重置計時器
			}

		case <-ticker.C:
			// 超時觸發批次處理
			if len(currentBatch) > 0 {
				p.processBatch(ctx, currentBatch)
				currentBatch = currentBatch[:0] // 清空slice但保留容量
			}

		case <-p.stopChannel:
			// 停止信號，處理剩餘批次後退出
			if len(currentBatch) > 0 {
				p.processBatch(ctx, currentBatch)
			}
			p.logger.InfoWithContext(ctx, "Batch processor goroutine stopped")
			return

		case <-ctx.Done():
			// Context取消，處理剩餘批次後退出
			if len(currentBatch) > 0 {
				p.processBatch(ctx, currentBatch)
			}
			p.logger.InfoWithContext(ctx, "Batch processor goroutine cancelled")
			return
		}
	}
}

// processBatch 處理一個批次的玩家資料
func (p *PlayerBatchProcessor) processBatch(ctx context.Context, batch []*PlayerBatchRequest) {
	ctx, span := p.tracing.StartSpan(ctx, "PlayerBatchProcessor.processBatch")
	defer p.tracing.SpanEnd(span)
	if len(batch) == 0 {
		return
	}

	p.logger.InfoWithContext(ctx, "Processing player batch",
		p.logger.Int("batch_size", len(batch)))

	// 提取玩家實體
	players := make([]*entity.Player, len(batch))

	for i, request := range batch {
		players[i] = request.Player
	}

	// 批次寫入資料庫
	dbErr := p.playerRepo.BatchUpsert(ctx, players)
	if dbErr != nil {
		p.logger.ErrorLog("Batch upsert failed", p.logger.Error("error", dbErr))
		// 所有請求都返回相同的資料庫錯誤
		for _, request := range batch {
			request.CompletionChannel <- dbErr
			close(request.CompletionChannel)
		}
		return
	}

	p.logger.InfoLog("Batch upsert succeeded", p.logger.Int("batch_size", len(batch)))

	// 回傳結果到所有請求
	for _, request := range batch {
		request.CompletionChannel <- nil // 資料庫操作成功
		close(request.CompletionChannel)
	}
}

// SetBatchConfig 設置批次處理配置
func (p *PlayerBatchProcessor) SetBatchConfig(batchSize int, batchTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		p.logger.WarnLog("Cannot change batch config while processor is running")
		return
	}

	p.batchSize = batchSize
	p.batchTimeout = batchTimeout

	p.logger.InfoLog("Batch processor config updated",
		p.logger.Int("batch_size", p.batchSize),
		p.logger.String("batch_timeout", p.batchTimeout.String()))
}

// GetStats 獲取統計資訊
func (p *PlayerBatchProcessor) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"batch_size":       p.batchSize,
		"batch_timeout":    p.batchTimeout,
		"buffer_size":      p.bufferSize,
		"started":          p.started,
		"channel_length":   len(p.requestChannel),
		"channel_capacity": cap(p.requestChannel),
	}
}
