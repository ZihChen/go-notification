package manager

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createManagerMockDependencies(
	t *testing.T,
) (*mocks.ManagerRepositoryMock, *mocks.MerchantRepositoryMock, *mocks.EventProducerMock, *helper.MockLogger, *mocks.TracingServiceMock) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Manager
	var _ repository.ManagerRepository

	managerRepo := mocks.NewManagerRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	return managerRepo, merchantRepo, eventProducer, logger, tracingService
}

func createManagerEvent() *event.ManagerEvent {
	return &event.ManagerEvent{
		ID:               231,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		GlobalManagerID:  "FATCAT-MANAGER-231",
		Account:          "testmanager",
		Email:            "manager@example.com",
	}
}

// 測試 SyncManager 方法 - 創建新管理員
func TestManagerUseCase_SyncManager(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"

	managerRepo, merchantRepo, eventProducer, logger, tracingService := createManagerMockDependencies(
		t,
	)

	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(&entity.Merchant{
		ID:               1,
		GlobalMerchantID: globalMerchantID,
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "merchant-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil)

	managerRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)

	// 創建用例
	tracingService.SetupSuccess()
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger, tracingService)

	// 創建測試事件
	managerEvent := createManagerEvent()

	// 執行測試
	err := useCase.SyncManager(context.Background(), managerEvent)

	// 驗證結果
	assert.NoError(t, err)
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

// 測試 GetManagerByID 方法
func TestManagerUseCase_GetManagerByID(t *testing.T) {
	// 準備測試數據
	managerID := uint64(1)
	merchantID := uint64(2)
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "manager@example.com"

	// 準備現有管理員
	existingManager := &entity.Manager{
		ID:              managerID,
		MerchantID:      merchantID,
		GlobalManagerID: globalManagerID,
		Account:         managerAccount,
		Email:           &managerEmail,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}
	managerRepo, merchantRepo, eventProducer, logger, tracingService := createManagerMockDependencies(
		t,
	)

	managerRepo.On("FindByID", mock.Anything, managerID).Return(existingManager, nil)

	// 創建用例
	tracingService.SetupSuccess()
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger, tracingService)

	// 執行測試
	manager, err := useCase.GetManagerByID(context.Background(), managerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, globalManagerID, manager.GlobalID)
	assert.Equal(t, managerAccount, manager.Name)
	assert.Equal(t, managerEmail, manager.Email)

	managerRepo.AssertExpectations()
}

// 測試 GetManagerByGlobalID 方法
func TestManagerUseCase_GetManagerByGlobalID(t *testing.T) {
	// 準備測試數據
	managerID := uint64(1)
	merchantID := uint64(2)
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "manager@example.com"

	// 準備現有管理員
	existingManager := &entity.Manager{
		ID:              managerID,
		MerchantID:      merchantID,
		GlobalManagerID: globalManagerID,
		Account:         managerAccount,
		Email:           &managerEmail,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}
	managerRepo, merchantRepo, eventProducer, logger, tracingService := createManagerMockDependencies(
		t,
	)

	managerRepo.On("FindByGlobalID", mock.Anything, globalManagerID).Return(existingManager, nil)

	// 創建用例
	tracingService.SetupSuccess()
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger, tracingService)

	// 執行測試
	manager, err := useCase.GetManagerByGlobalID(context.Background(), globalManagerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, globalManagerID, manager.GlobalID)
	assert.Equal(t, managerAccount, manager.Name)
	assert.Equal(t, managerEmail, manager.Email)

	managerRepo.AssertExpectations()
}
