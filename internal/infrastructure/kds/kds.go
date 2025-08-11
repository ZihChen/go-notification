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
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/serviceport"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	cfg "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// KDSService KDS服務實現
type KDSService struct {
	client       *kinesis.Client
	dynamoClient *dynamodb.Client
	redisManager *redisCache.Manager
	streamName   string
	tableName    string
	partitionKey string
	sortKey      string
	config       *cfg.Config
	queueService serviceport.QueueService
	logger       infraport.Logger
}

// NewKDSService 創建KDS服務
func NewKDSService(
	config *cfg.Config,
	queueService serviceport.QueueService,
	redisManager *redisCache.Manager,
	logger infraport.Logger,
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
		client:       kinesisClient,
		dynamoClient: dynamoClient,
		redisManager: redisManager,
		streamName:   config.AWS.KinesisStream,
		tableName:    config.AWS.DynamoDBTable,
		partitionKey: config.AWS.PartitionKey,
		sortKey:      config.AWS.SortKey,
		config:       config,
		queueService: queueService,
		logger:       logger,
	}, nil
}

// Send 發送事件到KDS
func (k *KDSService) Send(ctx context.Context, data []byte, eventType string) error {
	ctx, span := tracing.StartSpan(ctx, "KDS.Send")
	defer tracing.SpanEnd(span)

	// 添加屬性到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		traceparent := tracing.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := json.Marshal(jsonData); err == nil {
				data = newData
				tracing.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	tracing.TraceEvent(span, "Sending message to KDS",
		attribute.String("messaging.partition_key", partitionKey),
	)

	_, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(k.streamName),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.ErrorLog("Failed to put record to kinesis",
			k.logger.String("event_type", eventType),
			k.logger.Error("err", err))
		tracing.RecordSpanError(span, err)
		tracing.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.InfoLog("Published event to KDS",
		k.logger.String("event_type", eventType),
		k.logger.String("partition_key", partitionKey))

	return nil
}

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// 內部方法：發布事件到KDS
func (k *KDSService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	// 確保 traceparent 在事件中
	event.TraceParent = tracing.GetTraceparent(ctx)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return k.Send(ctx, eventBytes, event.Type)
}
