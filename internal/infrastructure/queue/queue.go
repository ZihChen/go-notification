package queue

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// 任務類型常量
const (
	TypeMerchantSync    = "merchant:sync"
	TypePlayerSync      = "player:sync"
	TypeManagerSync     = "manager:sync"
	TypePlayerLevelSync = "player:level:sync"
	TypePlayerTagsSync  = "player:tags:sync"
	TypeTagSync         = "tag:sync"
	TypeAgentSync       = "agent:sync"
)

// QueueService 佇列服務實現
type QueueService struct {
	client         *asynq.Client
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

// NewQueueService 創建佇列服務
func NewQueueService(
	cfg *config.Config,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) (service.QueueService, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.InfoLog("Connecting to Redis",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOpt)

	// 測試Redis連接
	inspector := asynq.NewInspector(redisOpt)
	_, err := inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.InfoLog("Successfully connected to Redis queue")

	return &QueueService{
		client:         client,
		logger:         logger,
		tracingService: tracingService,
	}, nil
}

// EnqueueMerchantSync 將商戶同步任務加入佇列
func (q *QueueService) EnqueueMerchantSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeMerchantSync, data)
}

// EnqueuePlayerSync 將玩家同步任務加入佇列
func (q *QueueService) EnqueuePlayerSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypePlayerSync, data)
}

// EnqueueManagerSync 將管理員同步任務加入佇列
func (q *QueueService) EnqueueManagerSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeManagerSync, data)
}

// EnqueuePlayerLevelSync 將玩家等級同步任務加入佇列
func (q *QueueService) EnqueuePlayerLevelSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypePlayerLevelSync, data)
}

// EnqueuePlayerTagsSync 將玩家標籤同步任務加入佇列
func (q *QueueService) EnqueuePlayerTagsSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypePlayerTagsSync, data)
}

// EnqueueTagSync 將標籤同步任務加入佇列
func (q *QueueService) EnqueueTagSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeTagSync, data)
}

// EnqueueAgentSync 將代理同步任務加入佇列
func (q *QueueService) EnqueueAgentSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeAgentSync, data)
}

// enqueueTask 通用方法，將任務加入佇列並添加追蹤
func (q *QueueService) enqueueTask(ctx context.Context, taskType string, data []byte) error {
	// 從當前上下文中獲取 span
	ctx, span := q.tracingService.StartSpan(ctx, "QueueService.EnqueueTask")
	q.tracingService.RecordSpanAttributes(span,
		attribute.String("messaging.destination", "redis_queue"),
		attribute.String("messaging.task_type", taskType),
	)

	// 提取事件ID並添加到span
	var eventID string
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		if id, ok := jsonData["id"].(string); ok {
			eventID = id
			q.tracingService.RecordSpanAttributes(span, attribute.String("messaging.event_id", id))
		}
	}

	// 將追蹤上下文注入數據中
	tracedData, err := q.tracingService.InjectTraceparentToJSON(ctx, data)
	if err == nil {
		data = tracedData
	}

	// 創建任務
	task := asynq.NewTask(taskType, data)

	// 記錄任務創建事件
	q.tracingService.TraceEvent(span, "Task created for Redis queue")

	// 設置任務選項 - 改進的重試策略
	// 修復：用正確的 asynq.Option 設置
	opts := []asynq.Option{
		// 指數退避重試策略，最多重試5次
		asynq.MaxRetry(5),
		// 使用自訂的重試延遲函數設定
		asynq.ProcessIn(0),              // 立即處理
		asynq.Timeout(30 * time.Second), // 任務超時設置
	}

	// 將任務加入佇列
	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		q.tracingService.RecordSpanError(span, err)
		q.logger.ErrorLog("Failed to enqueue task",
			q.logger.String("task_type", taskType),
			q.logger.String("event_id", eventID),
			q.logger.Error("err", err))
		return fmt.Errorf("failed to enqueue %s task: %w", taskType, err)
	}

	// 記錄成功事件
	q.tracingService.TraceEvent(span, "Task enqueued successfully",
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue))

	// 為任務添加更多屬性
	q.tracingService.RecordSpanAttributes(span,
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue),
	)

	q.logger.InfoLog("Enqueued task successfully",
		q.logger.String("task_type", taskType),
		q.logger.String("task_id", info.ID),
		q.logger.String("queue", info.Queue),
		q.logger.String("event_id", eventID),
		q.logger.String("enqueued_at", time.Now().String()))

	return nil
}

// WrapHandlerWithTracing 包裝處理器以添加追蹤功能 (方法版本)
func (q *QueueService) WrapHandlerWithTracing(h asynq.Handler) asynq.Handler {
	return WrapHandlerWithTracing(q.tracingService, h)
}

// WrapHandlerWithTracing 包裝處理器以添加追蹤功能 (函數版本)
func WrapHandlerWithTracing(
	tracingService infrastructure.TracingService,
	h asynq.Handler,
) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		if task == nil || len(task.Payload()) == 0 || task.Type() == "" {
			return asynq.SkipRetry
		}

		// 從任務中提取 traceparent
		data := task.Payload()
		ctxWithTrace := tracingService.ExtractTraceContext(ctx, data)

		// 創建處理任務的 span
		ctxWithTrace, span := tracingService.TraceRedisToWorker(
			ctxWithTrace,
			task.Type(),
			task.ResultWriter().TaskID(),
		)
		defer tracingService.SpanEnd(span)

		// 記錄任務開始處理
		tracingService.TraceEvent(span, "Starting worker task processing")
		tracingService.RecordSpanAttributes(span,
			attribute.Int("task.payload_size_bytes", len(data)),
			// 修復：移除 Retried 方法的調用，因為它不存在
		)

		// 提取事件ID記錄在span中
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				tracingService.RecordSpanAttributes(
					span,
					attribute.String("messaging.event_id", id),
				)
			}
		}

		// 處理任務
		err := h.ProcessTask(ctxWithTrace, task)

		// 處理錯誤情況
		if err != nil {
			// 記錄錯誤
			tracingService.RecordSpanError(span, err)
			tracingService.RecordSpanStatus(
				span,
				codes.Error,
				fmt.Sprintf("task processing failed: %v", err),
			)
			// 檢查錯誤類型，決定是否需要重試
			if strings.Contains(err.Error(), "(will retry)") {
				// 可重試錯誤，例如暫時性的資源不可用
				tracingService.TraceEvent(span, "Task processing failed, will retry",
					attribute.String("error", err.Error()))
				return fmt.Errorf("retriable error: %w", err)
			} else {
				// 無法重試的錯誤
				tracingService.TraceEvent(span, "Task processing failed, will not retry",
					attribute.String("error", err.Error()))
				return err
			}
		}

		// 記錄成功處理
		tracingService.TraceEvent(span, "Task processed successfully")
		return nil
	})
}

// Close 關閉佇列連接
func (q *QueueService) Close() error {
	return q.client.Close()
}

// generateTaskIDFromPayload 從payload生成或提取taskID
func generateTaskIDFromPayload(taskType string, payload []byte) string {
	// 首先嘗試從JSON payload中提取事件ID
	var jsonData map[string]interface{}
	if err := json.Unmarshal(payload, &jsonData); err == nil {
		// 嘗試多個可能的ID字段名稱
		idFields := []string{"id", "event_id", "task_id", "messageId", "ID", "global_id"}
		for _, field := range idFields {
			if id, exists := jsonData[field]; exists {
				if idStr, ok := id.(string); ok && idStr != "" {
					return idStr
				}
			}
		}
	}

	// 如果無法提取ID，則生成一個基於payload內容的唯一標識符
	hash := md5.Sum(payload)
	hashStr := hex.EncodeToString(hash[:])

	// 結合任務類型和時間戳，確保唯一性
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d_%s", taskType, timestamp, hashStr[:12])
}

// NewWorkerServer 創建Worker服務器
func NewWorkerServer(
	cfg *config.Config,
	logger infrastructure.Logger,
	failedTaskUseCase inbound.FailedTaskEventUseCase,
) (*asynq.Server, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.InfoLog("Creating worker server",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 設置服務器配置
	concurrency := 10
	queues := map[string]int{
		"default":  5,  // 默認優先級
		"critical": 10, // 高優先級
	}

	logger.InfoLog("Worker server configuration",
		logger.Int("concurrency", concurrency),
		logger.Any("queues", queues))

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorLog("Panic in RetryDelayFunc",
							logger.Any("recover", r),
							logger.Int("retry_count", n))
					}
				}()

				if task != nil {
					logger.WarnLog("Task retry scheduled",
						logger.Int("retry_count", n),
						logger.Error("err", err),
						logger.String("task_type", task.Type()),
						logger.String("payload", string(task.Payload())))
				}

				// 使用指數退避策略，但設置上限
				delay := time.Duration(n*n) * time.Second
				maxDelay := 5 * time.Minute
				if delay > maxDelay {
					delay = maxDelay
				}
				return delay
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					taskID := "unknown"

					// 方法1: 通過ResultWriter (首選方法)
					if w := task.ResultWriter(); w != nil {
						taskID = w.TaskID()
						logger.DebugLog(
							"TaskID obtained from ResultWriter",
							logger.String("task_id", taskID),
						)
					}

					// 方法2: 如果ResultWriter不可用，從payload中提取或生成唯一ID
					if taskID == "unknown" || taskID == "" {
						logger.WarnLog("ResultWriter unavailable, generating taskID from payload")
						taskID = generateTaskIDFromPayload(task.Type(), task.Payload())
						logger.DebugLog(
							"TaskID generated from payload",
							logger.String("task_id", taskID),
						)
					}

					logger.ErrorLog("Task processing failed - storing to DB",
						logger.String("task_id", taskID),
						logger.String("type", task.Type()),
						logger.Error("err", err))

					// 異步處理：存儲錯誤事件到DB
					go func() {
						bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()

						// 記錄失敗任務事件到資料庫
						redisKey := fmt.Sprintf("asynq:default:t:%s", taskID)
						if createErr := failedTaskUseCase.CreateFailedTaskEventWithRedisInfo(
							bgCtx,
							taskID,
							task.Type(),
							"default", // 默認queue
							string(task.Payload()),
							err.Error(),
							redisKey,
							"failed", // 設置為失敗狀態
							0,        // ErrorHandler中的重試次數為0（不會重試）
						); createErr != nil {
							logger.ErrorLog("Failed to record failed task event",
								logger.Error("err", createErr),
								logger.String("task_id", taskID),
								logger.String("task_type", task.Type()))
						} else {
							logger.InfoLog("Failed task event recorded to DB",
								logger.String("task_id", taskID),
								logger.String("task_type", task.Type()))
						}
					}()
				},
			),
		},
	)

	// 測試Redis連接
	inspector := asynq.NewInspector(redisOpt)
	_, err := inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.InfoLog("Worker server created successfully")
	return server, nil
}
