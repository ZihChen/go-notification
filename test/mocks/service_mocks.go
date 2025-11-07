package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
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
) ([]string, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *AgentServiceMock) GetAgentLineAncestors(
	ctx context.Context,
	globalAgentID string,
) ([]string, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
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
