package sse_notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
)

// MockSSEManager SSE Manager Mock
type MockSSEManager struct {
	mock.Mock
}

func (m *MockSSEManager) RegisterConnection(ctx context.Context, playerID string, writer inbound.SSEWriter) error {
	args := m.Called(ctx, playerID, writer)
	return args.Error(0)
}

func (m *MockSSEManager) UnregisterConnection(ctx context.Context, playerID string) error {
	args := m.Called(ctx, playerID)
	return args.Error(0)
}

func (m *MockSSEManager) IsPlayerOnline(ctx context.Context, playerID string) (bool, error) {
	args := m.Called(ctx, playerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSSEManager) GetOnlinePlayerCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockSSEManager) GetOnlinePlayerIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSSEManager) BroadcastToAll(ctx context.Context, notification *entity.SSENotification) (int, error) {
	args := m.Called(ctx, notification)
	return args.Int(0), args.Error(1)
}

func (m *MockSSEManager) SendToPlayer(ctx context.Context, playerID string, notification *entity.SSENotification) error {
	args := m.Called(ctx, playerID, notification)
	return args.Error(0)
}

func (m *MockSSEManager) SendToPlayers(ctx context.Context, playerIDs []string, notification *entity.SSENotification) (int, error) {
	args := m.Called(ctx, playerIDs, notification)
	return args.Int(0), args.Error(1)
}

func (m *MockSSEManager) EnqueueOfflineMessage(ctx context.Context, playerID string, notification *entity.SSENotification) error {
	args := m.Called(ctx, playerID, notification)
	return args.Error(0)
}

func (m *MockSSEManager) GetOfflineMessages(ctx context.Context, playerID string) ([]*entity.SSENotification, error) {
	args := m.Called(ctx, playerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.SSENotification), args.Error(1)
}

func (m *MockSSEManager) ClearOfflineMessages(ctx context.Context, playerID string) error {
	args := m.Called(ctx, playerID)
	return args.Error(0)
}

// Verify interface implementation
var _ service.SSEManager = (*MockSSEManager)(nil)

// MockLogger Logger Mock (simplified)
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) ErrorWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (m *MockLogger) WarnWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)  {}
func (m *MockLogger) InfoWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)  {}
func (m *MockLogger) DebugWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (m *MockLogger) FatalWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {}
func (m *MockLogger) DebugLog(msg string, fields ...*entity.LoggerFiled)                              {}
func (m *MockLogger) InfoLog(msg string, fields ...*entity.LoggerFiled)                               {}
func (m *MockLogger) ErrorLog(msg string, fields ...*entity.LoggerFiled)                              {}
func (m *MockLogger) WarnLog(msg string, fields ...*entity.LoggerFiled)                               {}
func (m *MockLogger) FatalLog(msg string, fields ...*entity.LoggerFiled)                              {}
func (m *MockLogger) Error(key string, value error) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (m *MockLogger) String(key string, value string) *entity.LoggerFiled { return &entity.LoggerFiled{} }
func (m *MockLogger) Int(key string, value int) *entity.LoggerFiled       { return &entity.LoggerFiled{} }
func (m *MockLogger) Int64(key string, value int64) *entity.LoggerFiled   { return &entity.LoggerFiled{} }
func (m *MockLogger) UInt64(key string, value uint64) *entity.LoggerFiled { return &entity.LoggerFiled{} }
func (m *MockLogger) Float64(key string, value float64) *entity.LoggerFiled {
	return &entity.LoggerFiled{}
}
func (m *MockLogger) Bool(key string, value bool) *entity.LoggerFiled     { return &entity.LoggerFiled{} }
func (m *MockLogger) Any(key string, value interface{}) *entity.LoggerFiled { return &entity.LoggerFiled{} }
func (m *MockLogger) Close()                                                {}

// Verify interface implementation
var _ infrastructure.Logger = (*MockLogger)(nil)

// TestBroadcastNotification 測試廣播推送
func TestBroadcastNotification(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockSSEManager := new(MockSSEManager)
	mockLogger := new(MockLogger)

	useCase := NewSSENotificationUseCase(mockSSEManager, mockLogger)

	req := &dto.BroadcastRequest{
		Title:    "測試通知",
		Message:  "這是一個測試訊息",
		Type:     "system",
		Priority: "high",
	}

	// Mock 期望
	mockSSEManager.On("GetOnlinePlayerCount", ctx).Return(100, nil)
	mockSSEManager.On("BroadcastToAll", ctx, mock.AnythingOfType("*entity.SSENotification")).Return(95, nil)

	// Act
	response, err := useCase.BroadcastNotification(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 100, response.TotalOnline)
	assert.Equal(t, 95, response.SuccessfulSends)
	assert.Equal(t, 5, response.FailedSends)
	assert.NotEmpty(t, response.NotificationID)

	mockSSEManager.AssertExpectations(t)
}

// TestSendNotification 測試個別推送
func TestSendNotification(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockSSEManager := new(MockSSEManager)
	mockLogger := new(MockLogger)

	useCase := NewSSENotificationUseCase(mockSSEManager, mockLogger)

	playerIDs := []string{"player1", "player2", "player3"}
	req := &dto.SendNotificationRequest{
		PlayerIDs: playerIDs,
		Title:     "測試通知",
		Message:   "這是一個測試訊息",
		Type:      "activity",
		Priority:  "medium",
	}

	// Mock 期望
	mockSSEManager.On("SendToPlayers", ctx, playerIDs, mock.AnythingOfType("*entity.SSENotification")).Return(3, nil)

	// Act
	response, err := useCase.SendNotification(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 3, response.TotalTargets)
	assert.Equal(t, 3, response.SuccessfulSends)
	assert.Equal(t, 0, response.FailedSends)
	assert.NotEmpty(t, response.NotificationID)

	mockSSEManager.AssertExpectations(t)
}

// TestSendNotification_ExceedBatchSize 測試超過批次大小限制
func TestSendNotification_ExceedBatchSize(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockSSEManager := new(MockSSEManager)
	mockLogger := new(MockLogger)

	useCase := NewSSENotificationUseCase(mockSSEManager, mockLogger)

	// 創建超過 1000 個玩家 ID
	playerIDs := make([]string, 1001)
	for i := 0; i < 1001; i++ {
		playerIDs[i] = "player"
	}

	req := &dto.SendNotificationRequest{
		PlayerIDs: playerIDs,
		Title:     "測試通知",
		Message:   "這是一個測試訊息",
		Type:      "activity",
	}

	// Act
	response, err := useCase.SendNotification(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "exceeds maximum batch size")
}
