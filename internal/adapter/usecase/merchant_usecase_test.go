package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 資料庫模擬
type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) FirstOrCreate(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	args := m.Called(ctx, merchant)
	merchant.ID = 1 // 為新創建的商戶設置 ID
	return args.Error(0)
}

func (m *MockMerchantRepository) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	merchant.ID = 1 // 為新創建的商戶設置 ID
	return args.Error(0)
}

func (m *MockMerchantRepository) Update(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMerchantRepository) Upsert(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

// 事件生產者模擬
type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) PublishMerchantSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func createMerchantEvent() *event.MerchantEvent {
	return &event.MerchantEvent{
		ID:               1,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "TestMerchant",
		DisplayName:      "Test Merchant",
	}
}

func TestMerchantUseCase_SyncMerchant(t *testing.T) {
	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("Upsert", mock.Anything, mock.Anything).
		Return(nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := helper.SetupLoggerMock(t)

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 創建測試事件
	merchantEvent := createMerchantEvent()

	// 執行測試
	err := useCase.SyncMerchant(context.Background(), merchantEvent)

	// 驗證結果
	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByID(t *testing.T) {
	// 準備測試數據
	merchantID := uint64(1)
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"
	apiKey := "test-api-key"

	// 準備現有商戶
	existingMerchant := &entity.Merchant{
		ID:               merchantID,
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      displayName,
		APIKey:           apiKey,
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByID", mock.Anything, merchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := helper.SetupLoggerMock(t)

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByID(context.Background(), merchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalMerchantID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, displayName, merchant.DisplayName)
	assert.Equal(t, apiKey, merchant.APIKey)

	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByGlobalID(t *testing.T) {
	// 準備測試數據
	merchantID := uint64(1)
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"
	apiKey := "test-api-key"

	// 準備現有商戶
	existingMerchant := &entity.Merchant{
		ID:               merchantID,
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      displayName,
		APIKey:           apiKey,
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := helper.SetupLoggerMock(t)

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByGlobalID(context.Background(), globalMerchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalMerchantID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, displayName, merchant.DisplayName)
	assert.Equal(t, apiKey, merchant.APIKey)

	merchantRepo.AssertExpectations(t)
}
