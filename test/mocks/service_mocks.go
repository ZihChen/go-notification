package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/stretchr/testify/mock"
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
