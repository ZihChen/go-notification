package player

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
) (*mocks.PlayerRepositoryMock, *mocks.MerchantRepositoryMock, *mocks.LevelRepositoryMock, *mocks.EventProducerMock, *helper.MockLogger, *redis.Client) {
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := helper.NewMockLogger()
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
	playerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
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
	assert.Equal(t, globalPlayerID, player.GlobalID)
	assert.Equal(t, playerAccount, player.Username)

	playerRepo.AssertExpectations()
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
	assert.Equal(t, globalPlayerID, player.GlobalID)
	assert.Equal(t, playerAccount, player.Username)

	playerRepo.AssertExpectations()
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

	playerRepo.AssertExpectations()
}
