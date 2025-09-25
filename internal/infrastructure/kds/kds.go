package kds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	cfg "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// KDSService KDS服務實現
type KDSService struct {
	client         *kinesis.Client
	dynamoClient   *dynamodb.Client
	redisManager   *redisCache.Manager
	consumeStream  string
	tableName      string
	partitionKey   string
	sortKey        string
	config         *cfg.Config
	queueService   service.QueueService
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

// NewKDSService 創建KDS服務
func NewKDSService(
	config *cfg.Config,
	queueService service.QueueService,
	redisManager *redisCache.Manager,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) (*KDSService, error) {
	// 創建AWS配置
	awsConfig, err := config.LoadAWSConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建DynamoDB客戶端
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	logger.InfoLog("Initialized KDS service",
		logger.String("stream_name", config.AWS.KinesisStream),
		logger.String("dynamodb_table", config.AWS.DynamoDBTable))

	return &KDSService{
		client:         kinesisClient,
		dynamoClient:   dynamoClient,
		redisManager:   redisManager,
		consumeStream:  config.AWS.KinesisStream,
		tableName:      config.AWS.DynamoDBTable,
		partitionKey:   config.AWS.PartitionKey,
		sortKey:        config.AWS.SortKey,
		config:         config,
		queueService:   queueService,
		logger:         logger,
		tracingService: tracingService,
	}, nil
}

// Send 發送事件到KDS
func (k *KDSService) Send(ctx context.Context, data []byte, eventType string) error {
	ctx, span := k.tracingService.StartSpan(ctx, "KDS.Send")
	defer k.tracingService.SpanEnd(span)

	// 添加屬性到 span
	k.tracingService.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		traceparent := k.tracingService.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := json.Marshal(jsonData); err == nil {
				data = newData
				k.tracingService.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	k.tracingService.TraceEvent(span, "Sending message to KDS",
		attribute.String("messaging.partition_key", partitionKey),
	)

	_, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(k.consumeStream),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.ErrorLog("Failed to put record to kinesis",
			k.logger.String("event_type", eventType),
			k.logger.Error("err", err))
		k.tracingService.RecordSpanError(span, err)
		k.tracingService.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	k.tracingService.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.InfoLog("Published event to KDS",
		k.logger.String("event_type", eventType),
		k.logger.String("partition_key", partitionKey))

	return nil
}

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracingService.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracingService.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracingService.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracingService.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracingService.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracingService.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// 內部方法：發布事件到KDS
func (k *KDSService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	// 確保 traceparent 在事件中
	event.TraceParent = k.tracingService.GetTraceparent(ctx)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return k.Send(ctx, eventBytes, event.Type)
}
