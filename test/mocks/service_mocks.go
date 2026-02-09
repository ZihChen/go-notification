package mocks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// EventProducerMock 統一的 EventProducer Mock
type EventProducerMock struct {
	*BaseMock
}

func NewEventProducerMock(t *testing.T) *EventProducerMock {
	return &EventProducerMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *EventProducerMock) PublishMerchantSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishSSENotification(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) SetupSuccess() {}
func (m *EventProducerMock) SetupError()   {}
func (m *EventProducerMock) SetupEmpty()   {}
func (m *EventProducerMock) Reset() {
	m.Mock = mock.Mock{}
}

// RedisManagerMock 統一的 RedisManager Mock
type RedisManagerMock struct {
	*BaseMock
}

func NewRedisManagerMock(t *testing.T) *RedisManagerMock {
	return &RedisManagerMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *RedisManagerMock) GetMutexWithOption(
	key string,
	options ...interface{},
) (interface{}, error) {
	args := m.Called(key, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *RedisManagerMock) SetupSuccess() {}
func (m *RedisManagerMock) SetupError()   {}
func (m *RedisManagerMock) SetupEmpty()   {}
func (m *RedisManagerMock) Reset() {
	m.Mock = mock.Mock{}
}

// PushNotificationServiceMock 統一的 PushNotificationService Mock
type PushNotificationServiceMock struct {
	*BaseMock
}

// NewPushNotificationServiceMock 創建新的 PushNotificationService Mock
func NewPushNotificationServiceMock(t *testing.T) *PushNotificationServiceMock {
	return &PushNotificationServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PushNotificationServiceMock) SendPushNotification(
	ctx context.Context,
	apiKey string,
	request *service.PushNotificationRequest,
) (*service.PushNotificationResponse, error) {
	args := m.Called(ctx, apiKey, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PushNotificationResponse), args.Error(1)
}

func (m *PushNotificationServiceMock) SetupSuccess() {}
func (m *PushNotificationServiceMock) SetupError()   {}
func (m *PushNotificationServiceMock) SetupEmpty()   {}
func (m *PushNotificationServiceMock) Reset() {
	m.Mock = mock.Mock{}
}

// TracingServiceMock 統一的 TracingService Mock
type TracingServiceMock struct {
	*BaseMock
}

var _ infrastructure.TracingService = (*TracingServiceMock)(nil)

// NewTracingServiceMock 創建新的 TracingService Mock
func NewTracingServiceMock(t *testing.T) *TracingServiceMock {
	return &TracingServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *TracingServiceMock) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	m.Called(ctx, spanName, opts)
	// Return the original context and a nil span for testing
	return ctx, trace.SpanFromContext(ctx)
}

func (m *TracingServiceMock) RecordSpanError(span trace.Span, err error) {
	m.Called(span, err)
}

func (m *TracingServiceMock) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	m.Called(span, attrs)
}

func (m *TracingServiceMock) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	m.Called(span, name, attrs)
}

func (m *TracingServiceMock) SpanEnd(span trace.Span) {
	m.Called(span)
}

func (m *TracingServiceMock) GetTraceparent(ctx context.Context) string {
	args := m.Called(ctx)
	return args.String(0)
}

func (m *TracingServiceMock) InjectTraceparentToJSON(
	ctx context.Context,
	data []byte,
) ([]byte, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *TracingServiceMock) RecordSpanStatus(span trace.Span, code codes.Code, desc string) {
	m.Called(span, code, desc)
}

func (m *TracingServiceMock) TraceWorkerToKDS(
	ctx context.Context,
	eventType, eventID string,
) (context.Context, trace.Span) {
	m.Called(ctx, eventType, eventID)
	return ctx, trace.SpanFromContext(ctx)
}

func (m *TracingServiceMock) ExtractTraceContext(
	ctx context.Context,
	carrier []byte,
) context.Context {
	m.Called(ctx, carrier)
	return ctx
}

func (m *TracingServiceMock) TraceRedisToWorker(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	m.Called(ctx, taskType, taskID)
	return ctx, trace.SpanFromContext(ctx)
}

func (m *TracingServiceMock) TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	m.Called(ctx, taskType, taskID)
	return ctx, trace.SpanFromContext(ctx)
}

func (m *TracingServiceMock) SetupSuccess() {
	// Setup common success scenarios
	m.On("StartSpan", mock.Anything, mock.Anything, mock.Anything).Return(context.Background(), nil)
	m.On("RecordSpanError", mock.Anything, mock.Anything).Return()
	m.On("RecordSpanAttributes", mock.Anything, mock.Anything).Return()
	m.On("TraceEvent", mock.Anything, mock.Anything, mock.Anything).Return()
	m.On("SpanEnd", mock.Anything).Return()
	m.On("GetTraceparent", mock.Anything).Return("")
	m.On("InjectTraceparentToJSON", mock.Anything, mock.Anything).Return([]byte{}, nil)
	m.On("RecordSpanStatus", mock.Anything, mock.Anything, mock.Anything).Return()
	m.On("TraceWorkerToKDS", mock.Anything, mock.Anything, mock.Anything).
		Return(context.Background(), nil)
	m.On("ExtractTraceContext", mock.Anything, mock.Anything).Return(context.Background())
	m.On("TraceRedisToWorker", mock.Anything, mock.Anything, mock.Anything).
		Return(context.Background(), nil)
	m.On("TraceWorkerProcessing", mock.Anything, mock.Anything, mock.Anything).
		Return(context.Background(), nil)
}

func (m *TracingServiceMock) SetupError() {}
func (m *TracingServiceMock) SetupEmpty() {}
func (m *TracingServiceMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentServiceMock 統一的 AgentService Mock
type AgentServiceMock struct {
	*BaseMock
}

func NewAgentServiceMock(t *testing.T) *AgentServiceMock {
	return &AgentServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentServiceMock) SyncAgentRelationshipsUpsert(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	args := m.Called(ctx, agentEvent)
	return args.Error(0)
}

func (m *AgentServiceMock) GetAgentLineDescendants(
	ctx context.Context,
	globalAgentID string,
) ([]uint64, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *AgentServiceMock) GetAgentLineAncestors(
	ctx context.Context,
	globalAgentID string,
) ([]uint64, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *AgentServiceMock) GetAgentHierarchy(
	ctx context.Context,
	globalAgentID string,
) (*service.AgentHierarchy, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentHierarchy), args.Error(1)
}

func (m *AgentServiceMock) ParseAgentPath(ancestry string) []string {
	args := m.Called(ancestry)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *AgentServiceMock) GeneratePathHash(pathParts []string) string {
	args := m.Called(pathParts)
	return args.String(0)
}

func (m *AgentServiceMock) ValidateAndFilterAgents(agents []string) []string {
	args := m.Called(agents)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *AgentServiceMock) IsValidAgentID(agentID string) bool {
	args := m.Called(agentID)
	return args.Bool(0)
}

func (m *AgentServiceMock) ExtractAccountFromGlobalID(globalID string) string {
	args := m.Called(globalID)
	return args.String(0)
}

func (m *AgentServiceMock) SetupSuccess() {}
func (m *AgentServiceMock) SetupError()   {}
func (m *AgentServiceMock) SetupEmpty()   {}
func (m *AgentServiceMock) Reset() {
	m.Mock = mock.Mock{}
}

// DistributedLockManagerMock 統一的 DistributedLockManager Mock
type DistributedLockManagerMock struct {
	*BaseMock
}

var _ infrastructure.DistributedLockManager = (*DistributedLockManagerMock)(nil)

func NewDistributedLockManagerMock(t *testing.T) *DistributedLockManagerMock {
	return &DistributedLockManagerMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *DistributedLockManagerMock) GetLock(
	ctx context.Context,
	key string,
) (infrastructure.DistributedMutex, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(infrastructure.DistributedMutex), args.Error(1)
}

func (m *DistributedLockManagerMock) GetLockWithOptions(
	ctx context.Context,
	key string,
	options infrastructure.LockOptions,
) (infrastructure.DistributedMutex, error) {
	args := m.Called(ctx, key, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(infrastructure.DistributedMutex), args.Error(1)
}

func (m *DistributedLockManagerMock) GetMutex(
	key string,
	expireTime time.Duration,
) (infrastructure.DistributedMutex, error) {
	args := m.Called(key, expireTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(infrastructure.DistributedMutex), args.Error(1)
}

func (m *DistributedLockManagerMock) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *DistributedLockManagerMock) SetupSuccess() {}
func (m *DistributedLockManagerMock) SetupError()   {}
func (m *DistributedLockManagerMock) SetupEmpty()   {}
func (m *DistributedLockManagerMock) Reset() {
	m.Mock = mock.Mock{}
}

// DistributedMutexMock 統一的 DistributedMutex Mock
type DistributedMutexMock struct {
	*BaseMock
}

var _ infrastructure.DistributedMutex = (*DistributedMutexMock)(nil)

func NewDistributedMutexMock(t *testing.T) *DistributedMutexMock {
	return &DistributedMutexMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *DistributedMutexMock) Lock() error {
	args := m.Called()
	return args.Error(0)
}

func (m *DistributedMutexMock) TryLock() error {
	args := m.Called()
	return args.Error(0)
}

func (m *DistributedMutexMock) Unlock() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *DistributedMutexMock) Extend() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *DistributedMutexMock) SetupSuccess() {}
func (m *DistributedMutexMock) SetupError()   {}
func (m *DistributedMutexMock) SetupEmpty()   {}
func (m *DistributedMutexMock) Reset() {
	m.Mock = mock.Mock{}
}

// MockCacheManager 統一的 CacheManager Mock
type MockCacheManager struct {
	*BaseMock
}

func NewMockCacheManager(t *testing.T) *MockCacheManager {
	return &MockCacheManager{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MockCacheManager) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCacheManager) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (string, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockCacheManager) SetNX(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (bool, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheManager) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheManager) Del(ctx context.Context, keys ...string) (int64, error) {
	// 使用可變參數展開來適配mock調用
	callArgs := []interface{}{ctx}
	for _, key := range keys {
		callArgs = append(callArgs, key)
	}
	args := m.Called(callArgs...)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheManager) Exists(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheManager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockCacheManager) Pipeline() (redis.Pipeliner, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(redis.Pipeliner), args.Error(1)
}

func (m *MockCacheManager) GetClient() (*redis.Client, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redis.Client), args.Error(1)
}

func (m *MockCacheManager) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheManager) GetRedsync() (*redsync.Redsync, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redsync.Redsync), args.Error(1)
}

func (m *MockCacheManager) SetupSuccess() {
	// Cache miss scenario - Get returns empty string, error
	// This will trigger the database query in QueryWithCache
	m.On("Get", mock.Anything, mock.Anything).Return("", fmt.Errorf("cache miss"))

	// Async Set call for cache update
	m.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("OK", nil)

	// Cache deletion for invalidation
	m.On("Del", mock.Anything, mock.Anything).Return(int64(1), nil)
}
func (m *MockCacheManager) SetupError() {}
func (m *MockCacheManager) SetupEmpty() {}
func (m *MockCacheManager) Reset() {
	m.Mock = mock.Mock{}
}

// MetricsServiceMock 統一的 MetricsService Mock
type MetricsServiceMock struct {
	*BaseMock
}

var _ infrastructure.MetricsService = (*MetricsServiceMock)(nil)

// NewMetricsServiceMock 創建新的 MetricsService Mock
func NewMetricsServiceMock(t *testing.T) *MetricsServiceMock {
	return &MetricsServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MetricsServiceMock) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MetricsServiceMock) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MetricsServiceMock) RecordAPIRequest(method, path, status string) {
	m.Called(method, path, status)
}

func (m *MetricsServiceMock) RecordAPIDuration(method, path string, duration float64) {
	m.Called(method, path, duration)
}

func (m *MetricsServiceMock) RecordAPIError(method, path, errorType string) {
	m.Called(method, path, errorType)
}

func (m *MetricsServiceMock) RecordCampaignProcessed(merchantID string, campaignType string) {
	m.Called(merchantID, campaignType)
}

func (m *MetricsServiceMock) RecordCampaignDuration(merchantID string, duration float64) {
	m.Called(merchantID, duration)
}

func (m *MetricsServiceMock) RecordCampaignError(merchantID string, errorType string) {
	m.Called(merchantID, errorType)
}

func (m *MetricsServiceMock) RecordMessageSent(merchantID string, notificationType string) {
	m.Called(merchantID, notificationType)
}

func (m *MetricsServiceMock) RecordMessageDelivered(merchantID string) {
	m.Called(merchantID)
}

func (m *MetricsServiceMock) RecordMessageFailed(merchantID string, reason string) {
	m.Called(merchantID, reason)
}

func (m *MetricsServiceMock) RecordEventReceived(eventType string) {
	m.Called(eventType)
}

func (m *MetricsServiceMock) RecordEventProcessed(eventType string, success bool) {
	m.Called(eventType, success)
}

func (m *MetricsServiceMock) RecordEventDuration(eventType string, duration float64) {
	m.Called(eventType, duration)
}

func (m *MetricsServiceMock) RecordTaskProcessed(taskType string, success bool) {
	m.Called(taskType, success)
}

func (m *MetricsServiceMock) RecordTaskDuration(taskType string, duration float64) {
	m.Called(taskType, duration)
}

func (m *MetricsServiceMock) SetupSuccess() {
	m.On("IsEnabled", mock.Anything).Return(true)
	m.On("Shutdown", mock.Anything).Return(nil)
	m.On("RecordAPIRequest", mock.Anything, mock.Anything, mock.Anything).Return()
	m.On("RecordAPIDuration", mock.Anything, mock.Anything, mock.Anything).Return()
	m.On("RecordAPIError", mock.Anything, mock.Anything, mock.Anything).Return()
	m.On("RecordCampaignProcessed", mock.Anything, mock.Anything).Return()
	m.On("RecordCampaignDuration", mock.Anything, mock.Anything).Return()
	m.On("RecordCampaignError", mock.Anything, mock.Anything).Return()
	m.On("RecordMessageSent", mock.Anything, mock.Anything).Return()
	m.On("RecordMessageDelivered", mock.Anything).Return()
	m.On("RecordMessageFailed", mock.Anything, mock.Anything).Return()
	m.On("RecordEventReceived", mock.Anything).Return()
	m.On("RecordEventProcessed", mock.Anything, mock.Anything).Return()
	m.On("RecordEventDuration", mock.Anything, mock.Anything).Return()
	m.On("RecordTaskProcessed", mock.Anything, mock.Anything).Return()
	m.On("RecordTaskDuration", mock.Anything, mock.Anything).Return()
}

func (m *MetricsServiceMock) SetupError() {}
func (m *MetricsServiceMock) SetupEmpty() {}
func (m *MetricsServiceMock) Reset() {
	m.Mock = mock.Mock{}
}
