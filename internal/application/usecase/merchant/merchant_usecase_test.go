package merchant

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	merchantRepo.On("Upsert", mock.Anything, mock.Anything).
		Return(nil)

	// 創建模擬事件生產者
	eventProducer := mocks.NewEventProducerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 創建測試事件
	merchantEvent := createMerchantEvent()

	// 執行測試
	err := useCase.SyncMerchant(context.Background(), merchantEvent)

	// 驗證結果
	assert.NoError(t, err)
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
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
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	merchantRepo.On("FindByID", mock.Anything, merchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := mocks.NewEventProducerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByID(context.Background(), merchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, "active", merchant.Status)

	merchantRepo.AssertExpectations()
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
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := mocks.NewEventProducerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByGlobalID(context.Background(), globalMerchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, "active", merchant.Status)

	merchantRepo.AssertExpectations()
}
