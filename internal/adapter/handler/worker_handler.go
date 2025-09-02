package handler

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
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
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
	logger           infrastructure.Logger
}

// NewWorkerHandler 創建Worker Handler
func NewWorkerHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	levelUseCase inbound.PlayerLevelUseCase,
	playerTagUseCase inbound.PlayerTagUseCase,
	logger infrastructure.Logger,
) *WorkerHandler {
	return &WorkerHandler{
		merchantUseCase:  merchantUseCase,
		playerUseCase:    playerUseCase,
		managerUseCase:   managerUseCase,
		levelUseCase:     levelUseCase,
		playerTagUseCase: playerTagUseCase,
		logger:           logger,
	}
}

func (h *WorkerHandler) RegisterHandlers(mux *asynq.ServeMux) {
	// 使用追蹤包裝器
	mux.Handle(
		queue.TypeMerchantSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleMerchantSync)),
	)
	mux.Handle(
		queue.TypePlayerSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandlePlayerSync)),
	)
	mux.Handle(
		queue.TypeManagerSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleManagerSync)),
	)
	mux.Handle(
		queue.TypePlayerLevelSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandlePlayerLevelSync)),
	)
	mux.Handle(
		queue.TypePlayerTagsSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandlePlayerTagsSync)),
	)
	mux.Handle(
		queue.TypeTagSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleTagSync)),
	)

	h.logger.InfoLog("Registered worker handlers",
		h.logger.String("handler.merchant_sync", queue.TypeMerchantSync),
		h.logger.String("handler.player_sync", queue.TypePlayerSync),
		h.logger.String("handler.manager_sync", queue.TypeManagerSync),
		h.logger.String("handler.player_level_sync", queue.TypePlayerLevelSync),
		h.logger.String("handler.player_tags_sync", queue.TypePlayerTagsSync),
		h.logger.String("handler.tag_sync", queue.TypeTagSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var merchantEvent event.MerchantEvent
	if unmarshalErr := jsoniter.Unmarshal(dataBytes, &merchantEvent); unmarshalErr != nil {
		tracing.RecordSpanError(span, unmarshalErr)
		return fmt.Errorf("unmarshal merchant event: %w", unmarshalErr)
	}

	tracing.TraceEvent(span, "Starting merchant sync processing")

	// 執行實際的同步邏輯
	if syncErr := h.merchantUseCase.SyncMerchant(ctx, &merchantEvent); syncErr != nil {
		h.logger.ErrorLog("Failed to sync merchant",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", syncErr))
		tracing.RecordSpanError(span, syncErr)
		return fmt.Errorf("failed to sync merchant: %w", syncErr)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Merchant sync completed successfully")

	h.logger.InfoLog("Merchant sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandlePlayerSync 處理玩家同步任務
func (h *WorkerHandler) HandlePlayerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting player sync processing")

	// 執行實際的同步邏輯
	if err = h.playerUseCase.SyncPlayer(ctx, &playerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync player",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		// 記錄錯誤
		tracing.RecordSpanError(span, err)

		return fmt.Errorf("failed to sync player: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Player sync completed successfully")

	h.logger.InfoLog("Player sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandleManagerSync 處理管理員同步任務
func (h *WorkerHandler) HandleManagerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing manager sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var managerEvent event.ManagerEvent
	if err = jsoniter.Unmarshal(dataBytes, &managerEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	tracing.TraceEvent(span, "Starting manager sync processing")

	// 執行實際的同步邏輯
	if err = h.managerUseCase.SyncManager(ctx, &managerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync manager",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		// 記錄錯誤
		tracing.RecordSpanError(span, err)

		return fmt.Errorf("failed to sync manager: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Manager sync completed successfully")

	h.logger.InfoLog("Manager sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandlePlayerLevelSync 處理玩家等級同步任務
func (h *WorkerHandler) HandlePlayerLevelSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerLevelSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing player level sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerLevel event.IdentityPlayerLevelSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerLevel); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal player level event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	if err = h.levelUseCase.SyncPlayerLevel(ctx, &playerLevel); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync player level",
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync player level: %w", err)
	}

	tracing.TraceEvent(span, "Player level sync completed successfully")

	h.logger.InfoWithContext(ctx, "Player level sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandlePlayerTagsSync 處理玩家標籤同步任務
func (h *WorkerHandler) HandlePlayerTagsSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerTagsSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing player tags sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	tracing.TraceEvent(span, "Starting player tags sync processing")

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerTag event.IdentityPlayerTagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerTag); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal player tags event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal  player tags event: %w", err)
	}

	if err = h.playerTagUseCase.SyncPlayerTags(ctx, &playerTag); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync player tags",
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync player tags: %w", err)
	}

	tracing.TraceEvent(span, "Player tags sync completed successfully")

	h.logger.InfoWithContext(ctx, "Player tags sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleTagSync 處理標籤同步任務
func (h *WorkerHandler) HandleTagSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeTagSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoWithContext(ctx, "Processing tag sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	tracing.TraceEvent(span, "Starting tag sync processing")

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var tagEvent event.IdentityTagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &tagEvent); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to unmarshal tag event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(dataBytes)))
		return fmt.Errorf("unmarshal tag event: %w", err)
	}

	if err = h.playerTagUseCase.SyncTag(ctx, &tagEvent); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to sync tag",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err),
		)
		return fmt.Errorf("failed to sync tag: %w", err)
	}

	tracing.TraceEvent(span, "Tag sync completed successfully")
	h.logger.InfoWithContext(ctx, "Tag sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

func parseCloudEvent(eventData []byte, span trace.Span) (*event.CloudEvent, error) {
	var cloudEvent event.CloudEvent
	if err := jsoniter.Unmarshal(eventData, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("unmarshal cloud event: %w", err)
	}
	tracing.RecordSpanAttributes(span,
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source))
	return &cloudEvent, nil
}
