package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/queue"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func getTaskID(task *asynq.Task) string {
	if task == nil {
		return "unknown"
	}
	if w := task.ResultWriter(); w != nil {
		return w.TaskID()
	}

	return fmt.Sprintf("no-writer-%s", uuid.NewString()) // 保證唯一性
}

// WorkerHandler Worker Handler
type WorkerHandler struct {
	merchantUseCase  inbound.MerchantUseCase
	playerUseCase    inbound.PlayerUseCase
	managerUseCase   inbound.ManagerUseCase
	levelUseCase     inbound.PlayerLevelUseCase
	playerTagUseCase inbound.PlayerTagUseCase
	messageUseCase   inbound.MessageUseCase
	agentUseCase     inbound.AgentUseCase
	logger           infrastructure.Logger
	tracingService   infrastructure.TracingService
}

// NewWorkerHandler 創建Worker Handler
func NewWorkerHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	levelUseCase inbound.PlayerLevelUseCase,
	playerTagUseCase inbound.PlayerTagUseCase,
	messageUseCase inbound.MessageUseCase,
	agentUseCase inbound.AgentUseCase,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) *WorkerHandler {
	return &WorkerHandler{
		merchantUseCase:  merchantUseCase,
		playerUseCase:    playerUseCase,
		managerUseCase:   managerUseCase,
		levelUseCase:     levelUseCase,
		playerTagUseCase: playerTagUseCase,
		messageUseCase:   messageUseCase,
		agentUseCase:     agentUseCase,
		logger:           logger,
		tracingService:   tracingService,
	}
}

// StartProcessor 啟動 Worker Handler 及其批次處理器
func (h *WorkerHandler) StartProcessor(ctx context.Context) error {
	// 啟動 PlayerUseCase 的批次處理器
	if err := h.playerUseCase.StartBatchProcessor(ctx); err != nil {
		h.logger.ErrorWithContext(ctx, "Failed to start player batch processor",
			h.logger.Error("error", err))
		return fmt.Errorf("failed to start player batch processor: %w", err)
	}

	h.logger.InfoWithContext(ctx, "Worker handler started successfully with batch processors")
	return nil
}

// ShutdownProcessor 關閉 Worker Handler 及其批次處理器
func (h *WorkerHandler) ShutdownProcessor(ctx context.Context) error {
	h.logger.InfoWithContext(ctx, "Shutting down worker handler...")

	// 停止 PlayerUseCase 的批次處理器
	if err := h.playerUseCase.StopBatchProcessor(ctx); err != nil {
		h.logger.ErrorWithContext(ctx, "Failed to stop player batch processor",
			h.logger.Error("error", err))
		return fmt.Errorf("failed to stop player batch processor: %w", err)
	}

	h.logger.InfoWithContext(ctx, "Worker handler shutdown completed")
	return nil
}

func (h *WorkerHandler) RegisterHandlers(mux *asynq.ServeMux) {
	// 使用追蹤包裝器
	mux.Handle(
		queue.TypeMerchantSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandleMerchantSync)),
	)
	mux.Handle(
		queue.TypePlayerSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandlePlayerSync)),
	)
	mux.Handle(
		queue.TypeManagerSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandleManagerSync)),
	)
	mux.Handle(
		queue.TypePlayerLevelSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandlePlayerLevelSync)),
	)
	mux.Handle(
		queue.TypePlayerTagsSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandlePlayerTagsSync)),
	)
	mux.Handle(
		queue.TypeTagSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandleTagSync)),
	)
	mux.Handle(
		queue.TypeAgentSync,
		queue.WrapHandlerWithTracing(h.tracingService, asynq.HandlerFunc(h.HandleAgentSync)),
	)

	h.logger.InfoLog("Registered worker handlers",
		h.logger.String("handler.merchant_sync", queue.TypeMerchantSync),
		h.logger.String("handler.player_sync", queue.TypePlayerSync),
		h.logger.String("handler.manager_sync", queue.TypeManagerSync),
		h.logger.String("handler.player_level_sync", queue.TypePlayerLevelSync),
		h.logger.String("handler.player_tags_sync", queue.TypePlayerTagsSync),
		h.logger.String("handler.tag_sync", queue.TypeTagSync),
		h.logger.String("handler.agent_sync", queue.TypeAgentSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var merchantEvent event.MerchantEvent
	if unmarshalErr := jsoniter.Unmarshal(dataBytes, &merchantEvent); unmarshalErr != nil {
		h.tracingService.RecordSpanError(span, unmarshalErr)
		return fmt.Errorf("unmarshal merchant event: %w", unmarshalErr)
	}

	h.tracingService.TraceEvent(span, "Starting merchant sync processing")

	// 執行實際的同步邏輯
	if syncErr := h.merchantUseCase.SyncMerchant(ctx, &merchantEvent); syncErr != nil {
		h.logger.ErrorLog("Failed to sync merchant",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", syncErr))
		h.tracingService.RecordSpanError(span, syncErr)
		return fmt.Errorf("failed to sync merchant: %w", syncErr)
	}

	// 記錄成功完成任務
	h.tracingService.TraceEvent(span, "Merchant sync completed successfully")

	h.logger.InfoLog("Merchant sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandlePlayerSync 處理玩家同步任務
func (h *WorkerHandler) HandlePlayerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	// 記錄開始處理
	h.tracingService.TraceEvent(span, "Starting player sync processing")

	// 執行實際的同步邏輯
	if err = h.playerUseCase.SyncPlayer(ctx, &playerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync player",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		// 記錄錯誤
		h.tracingService.RecordSpanError(span, err)

		return fmt.Errorf("failed to sync player: %w", err)
	}

	// TODO: 待新版玩家站內信上線再啟用同步玩家的player_message（如果玩家重新上線）
	//if playerEvent.GlobalPlayerID != "" {
	//	if err = h.messageUseCase.ProcessPlayer(ctx, playerEvent.GlobalPlayerID); err != nil {
	//		h.logger.WarnLog("Failed to process player for message sync after player sync",
	//			h.logger.String("global_player_id", playerEvent.GlobalPlayerID),
	//			h.logger.String("task_id", taskID),
	//			h.logger.Error("err", err))
	//		// 即使訊息同步失敗，也不影響玩家資料同步的成功
	//	}
	//}

	// 記錄成功完成任務
	h.tracingService.TraceEvent(span, "Player sync completed successfully")

	h.logger.InfoLog("Player sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandleManagerSync 處理管理員同步任務
func (h *WorkerHandler) HandleManagerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoLog("Processing manager sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var managerEvent event.ManagerEvent
	if err = jsoniter.Unmarshal(dataBytes, &managerEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	h.tracingService.TraceEvent(span, "Starting manager sync processing")

	// 執行實際的同步邏輯
	if err = h.managerUseCase.SyncManager(ctx, &managerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync manager",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		// 記錄錯誤
		h.tracingService.RecordSpanError(span, err)

		return fmt.Errorf("failed to sync manager: %w", err)
	}

	// 記錄成功完成任務
	h.tracingService.TraceEvent(span, "Manager sync completed successfully")

	h.logger.InfoLog("Manager sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandlePlayerLevelSync 處理玩家等級同步任務
func (h *WorkerHandler) HandlePlayerLevelSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypePlayerLevelSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing player level sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerLevel event.IdentityPlayerLevelSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerLevel); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal player level event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	if err = h.levelUseCase.SyncPlayerLevel(ctx, &playerLevel); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync player level",
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync player level: %w", err)
	}

	h.tracingService.TraceEvent(span, "Player level sync completed successfully")

	h.logger.InfoWithContext(ctx, "Player level sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandlePlayerTagsSync 處理玩家標籤同步任務
func (h *WorkerHandler) HandlePlayerTagsSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypePlayerTagsSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing player tags sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	h.tracingService.TraceEvent(span, "Starting player tags sync processing")

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerTag event.IdentityPlayerTagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerTag); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal player tags event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal  player tags event: %w", err)
	}

	if err = h.playerTagUseCase.SyncPlayerTags(ctx, &playerTag); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync player tags",
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync player tags: %w", err)
	}

	h.tracingService.TraceEvent(span, "Player tags sync completed successfully")

	h.logger.InfoWithContext(ctx, "Player tags sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleTagSync 處理標籤同步任務
func (h *WorkerHandler) HandleTagSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypeTagSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing tag sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	h.tracingService.TraceEvent(span, "Starting tag sync processing")

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var tagEvent event.IdentityTagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &tagEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal tag event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal tag event: %w", err)
	}

	if err = h.playerTagUseCase.SyncTag(ctx, &tagEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync tag",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync tag: %w", err)
	}

	h.tracingService.TraceEvent(span, "Tag sync completed successfully")
	h.logger.InfoWithContext(ctx, "Tag sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleAgentSync 處理代理同步任務
func (h *WorkerHandler) HandleAgentSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := h.tracingService.TraceWorkerProcessing(ctx, queue.TypeAgentSync, taskID)
	defer h.tracingService.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing agent sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracingService)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var agentEvent event.AgentSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &agentEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal agent event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal agent event: %w", err)
	}

	h.tracingService.TraceEvent(span, "Starting agent sync processing")

	// 執行完整的代理同步邏輯 (資料同步 + 關係建立)
	if err = h.agentUseCase.SyncAgentDataWithRelationships(ctx, &agentEvent); err != nil {
		h.tracingService.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync agent data and relationships",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("global_agent_id", agentEvent.GlobalAgentID))
		return fmt.Errorf("failed to sync agent data and relationships: %w", err)
	}

	h.tracingService.TraceEvent(span, "Agent sync completed successfully")
	h.logger.InfoWithContext(ctx, "Agent sync task completed successfully",
		h.logger.String("task_id", taskID),
		h.logger.String("global_agent_id", agentEvent.GlobalAgentID))
	return nil
}

func parseCloudEvent(
	eventData []byte,
	span trace.Span,
	tracingService infrastructure.TracingService,
) (*event.CloudEvent, error) {
	var cloudEvent event.CloudEvent
	if err := jsoniter.Unmarshal(eventData, &cloudEvent); err != nil {
		tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("unmarshal cloud event: %w", err)
	}
	tracingService.RecordSpanAttributes(span,
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source))
	return &cloudEvent, nil
}
