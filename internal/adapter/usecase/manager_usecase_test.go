package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 資料庫模擬
type MockManagerRepository struct {
	mock.Mock
}

func (m *MockManagerRepository) FindByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerRepository) FirstOrCreate(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func (m *MockManagerRepository) Create(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func (m *MockManagerRepository) Update(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *MockManagerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockManagerRepository) Upsert(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func createManagerMockDependencies(
	t *testing.T,
) (*MockManagerRepository, *MockMerchantRepository, *MockEventProducer, *helper.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Manager
	var _ repositoryport.ManagerRepository

	managerRepo := new(MockManagerRepository)
	merchantRepo := new(MockMerchantRepository)
	eventProducer := new(MockEventProducer)
	logger := helper.SetupLoggerMock(t)
	return managerRepo, merchantRepo, eventProducer, logger
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

	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

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
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 創建測試事件
	managerEvent := createManagerEvent()

	// 執行測試
	err := useCase.SyncManager(context.Background(), managerEvent)

	// 驗證結果
	assert.NoError(t, err)
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
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
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	managerRepo.On("FindByID", mock.Anything, managerID).Return(existingManager, nil)

	// 創建用例
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	manager, err := useCase.GetManagerByID(context.Background(), managerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, merchantID, manager.MerchantID)
	assert.Equal(t, globalManagerID, manager.GlobalManagerID)
	assert.Equal(t, managerAccount, manager.Account)
	assert.Equal(t, managerEmail, *manager.Email)

	managerRepo.AssertExpectations(t)
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
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	managerRepo.On("FindByGlobalID", mock.Anything, globalManagerID).Return(existingManager, nil)

	// 創建用例
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	manager, err := useCase.GetManagerByGlobalID(context.Background(), globalManagerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, merchantID, manager.MerchantID)
	assert.Equal(t, globalManagerID, manager.GlobalManagerID)
	assert.Equal(t, managerAccount, manager.Account)
	assert.Equal(t, managerEmail, *manager.Email)

	managerRepo.AssertExpectations(t)
}
