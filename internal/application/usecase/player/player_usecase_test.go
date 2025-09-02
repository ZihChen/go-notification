package player

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/testutil"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 資料庫模擬
type MockPlayerRepository struct {
	mock.Mock
}

func (m *MockPlayerRepository) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerRepository) FindByTargetType(
	ctx context.Context,
	targetType uint8,
	offset, limit int,
) ([]*entity.Player, error) {
	args := m.Called(ctx, targetType, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Player), args.Error(1)
}

func (m *MockPlayerRepository) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockPlayerRepository) Create(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	player.ID = 1 // 為新創建的玩家設置 ID
	return args.Error(0)
}

func (m *MockPlayerRepository) Update(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockPlayerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlayerRepository) Upsert(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

type MockLevelRepository struct {
	mock.Mock
}

func (m *MockLevelRepository) Upsert(ctx context.Context, level *entity.Level) error {
	args := m.Called(ctx, level)
	return args.Error(0)
}

func (m *MockLevelRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Level), args.Error(1)
}

func createPlayerEvent() *event.PlayerEvent {
	return &event.PlayerEvent{
		GlobalMerchantID:    "FATCAT-MERCHANT-1",
		GlobalPlayerLevelID: "FATCAT-PLAYER-7241",
		Account:             "testplayer",
		Email:               "test@example.com",
	}
}

func createMockDependencies(
	t *testing.T,
) (*MockPlayerRepository, *testutil.MockMerchantRepository, *MockLevelRepository, *testutil.MockEventProducer, *helper.MockLogger, *redis.Client) {
	playerRepo := new(MockPlayerRepository)
	merchantRepo := new(testutil.MockMerchantRepository)
	levelRepo := new(MockLevelRepository)
	eventProducer := new(testutil.MockEventProducer)
	logger := helper.SetupLoggerMock(t)
	redisClient, _ := redismock.NewClientMock()
	return playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient
}

// 測試 SyncPlayer 方法 - 創建新玩家
func TestPlayerUseCase_SyncPlayer(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"

	playerRepo, merchantRepo, levelRepo, eventProducer, logger, _ := createMockDependencies(t)

	playerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Player")).Return(nil)

	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(&entity.Merchant{
		ID:               1,
		GlobalMerchantID: globalMerchantID,
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "merchant-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil)

	useCase := NewPlayerUseCase(playerRepo, merchantRepo, levelRepo, eventProducer, logger)

	// 執行測試
	err := useCase.SyncPlayer(context.Background(), createPlayerEvent())

	// 驗證結果
	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

// 測試 GetPlayerByID 方法
func TestPlayerUseCase_GetPlayerByID(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &entity.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	playerRepo, merchantRepo, levelRepo, eventProducer, logger, _ := createMockDependencies(t)

	playerRepo.On("FindByID", mock.Anything, playerID).Return(existingPlayer, nil)

	// 創建用例
	useCase := NewPlayerUseCase(playerRepo, merchantRepo, levelRepo, eventProducer, logger)

	// 執行測試
	player, err := useCase.GetPlayerByID(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, playerID, player.ID)
	assert.Equal(t, merchantID, player.MerchantID)
	assert.Equal(t, globalPlayerID, player.GlobalPlayerID)
	assert.Equal(t, apiKey, player.APIKey)
	assert.Equal(t, playerAccount, player.Account)
	assert.Equal(t, playerEmail, *player.Email)

	playerRepo.AssertExpectations(t)
}

// 測試 GetPlayerByGlobalID 方法
func TestPlayerUseCase_GetPlayerByGlobalID(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &entity.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	playerRepo, merchantRepo, levelRepo, eventProducer, logger, _ := createMockDependencies(t)

	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(existingPlayer, nil)

	// 創建用例
	useCase := NewPlayerUseCase(playerRepo, merchantRepo, levelRepo, eventProducer, logger)

	// 執行測試
	player, err := useCase.GetPlayerByGlobalID(context.Background(), globalPlayerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, playerID, player.ID)
	assert.Equal(t, merchantID, player.MerchantID)
	assert.Equal(t, globalPlayerID, player.GlobalPlayerID)
	assert.Equal(t, apiKey, player.APIKey)
	assert.Equal(t, playerAccount, player.Account)
	assert.Equal(t, playerEmail, *player.Email)

	playerRepo.AssertExpectations(t)
}

// 測試 UpdatePlayerLastActive 方法
func TestPlayerUseCase_UpdatePlayerLastActive(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &entity.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		LastActiveAt:   nil, // 初始為 nil
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	playerRepo, merchantRepo, levelRepo, eventProducer, logger, _ := createMockDependencies(t)

	playerRepo.On("FindByID", mock.Anything, playerID).Return(existingPlayer, nil)
	playerRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Player")).Return(nil)

	// 創建用例
	useCase := NewPlayerUseCase(playerRepo, merchantRepo, levelRepo, eventProducer, logger)

	// 執行測試
	err := useCase.UpdatePlayerLastActive(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證最後活躍時間被更新
	updated := playerRepo.Calls[1].Arguments.Get(1).(*entity.Player)
	assert.NotNil(t, updated.LastActiveAt)

	playerRepo.AssertExpectations(t)
}
