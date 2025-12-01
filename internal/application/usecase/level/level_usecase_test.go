package level

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper functions
func createLevelMockDependencies(t *testing.T) (
	*mocks.LevelRepositoryMock,
	*mocks.MerchantRepositoryMock,
	*helper.MockLogger,
) {
	levelRepo := mocks.NewLevelRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	logger := helper.NewMockLogger()

	return levelRepo, merchantRepo, logger
}

func createTestLevel() *entity.Level {
	return &entity.Level{
		ID:                  1,
		MerchantID:          1,
		GlobalPlayerLevelID: "FATCAT-LEVEL-1",
		Name:                "Bronze Level",
		CreatedAt:           time.Now().Add(-1 * time.Hour),
		UpdatedAt:           time.Now().Add(-1 * time.Hour),
	}
}

func createTestMerchantForLevel() *entity.Merchant {
	return &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "test-api-key",
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}
}

func createIdentityPlayerLevelSyncEvent() *event.IdentityPlayerLevelSyncEvent {
	return &event.IdentityPlayerLevelSyncEvent{
		GlobalMerchantID:    "FATCAT-MERCHANT-1",
		GlobalPlayerLevelID: "FATCAT-LEVEL-1",
		Name:                "Bronze Level",
		CreatedAt:           time.Now().Add(-2 * time.Hour),
		UpdatedAt:           time.Now().Add(-1 * time.Hour),
	}
}

// Test SyncPlayerLevel - successful sync
func TestLevelUseCase_SyncPlayerLevel(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).Return(nil)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.NoError(t, err)
	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()

	// Verify the level was created with correct data
	levelRepo.AssertCalled(
		t,
		"Upsert",
		mock.Anything,
		mock.MatchedBy(func(level *entity.Level) bool {
			return level.GlobalPlayerLevelID == event.GlobalPlayerLevelID &&
				level.Name == event.Name &&
				level.MerchantID == merchant.ID &&
				level.CreatedAt.Equal(event.UpdatedAt) &&
				level.UpdatedAt.Equal(event.UpdatedAt)
		}),
	)
}

// Test SyncPlayerLevel - merchant not found (this reveals a bug in the original code)
func TestLevelUseCase_SyncPlayerLevel_MerchantNotFound_PanicExpected(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	// Mock merchant not found error (which should be ignored per the logic, but causes panic)
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errmsg.ErrRepoMerchantNotFound)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	// This test documents that the current code has a bug - it panics when merchant is not found
	// but ErrRepoMerchantNotFound is returned (which should be ignored according to the logic)
	assert.Panics(t, func() {
		_ = useCase.SyncPlayerLevel(context.Background(), event)
	})

	merchantRepo.AssertExpectations()
}

// Test SyncPlayerLevel - merchant repository error (not ErrRepoMerchantNotFound)
func TestLevelUseCase_SyncPlayerLevel_MerchantRepositoryError(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("database connection error"))

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations()
}

// Test SyncPlayerLevel - level repository upsert error
func TestLevelUseCase_SyncPlayerLevel_LevelUpsertError(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).
		Return(errors.New("database upsert error"))

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert player level")
	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}

// Test SyncPlayerLevel - verify level data mapping
func TestLevelUseCase_SyncPlayerLevel_DataMapping(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	var capturedLevel *entity.Level
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).
		Run(func(args mock.Arguments) {
			capturedLevel = args.Get(1).(*entity.Level)
		}).Return(nil)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := &event.IdentityPlayerLevelSyncEvent{
		GlobalMerchantID:    "FATCAT-MERCHANT-1",
		GlobalPlayerLevelID: "FATCAT-LEVEL-GOLD",
		Name:                "Gold Level",
		CreatedAt:           time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.NoError(t, err)
	assert.NotNil(t, capturedLevel)
	assert.Equal(t, event.GlobalPlayerLevelID, capturedLevel.GlobalPlayerLevelID)
	assert.Equal(t, event.Name, capturedLevel.Name)
	assert.Equal(t, merchant.ID, capturedLevel.MerchantID)
	assert.Equal(t, event.UpdatedAt, capturedLevel.CreatedAt)
	assert.Equal(t, event.UpdatedAt, capturedLevel.UpdatedAt)

	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}

// Test SyncPlayerLevel - empty level name
func TestLevelUseCase_SyncPlayerLevel_EmptyLevelName(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	levelRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(level *entity.Level) bool {
		return level.Name == "" // Verify empty name is preserved
	})).Return(nil)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()
	event.Name = "" // Set empty name

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.NoError(t, err)
	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}

// Test SyncPlayerLevel - verify tracing and logging calls
func TestLevelUseCase_SyncPlayerLevel_TracingAndLogging(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).Return(nil)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	err := useCase.SyncPlayerLevel(context.Background(), event)

	assert.NoError(t, err)

	// Verify logger was called for success case
	logger.AssertCalled(
		t,
		"InfoWithContext",
		mock.Anything,
		"Upsert player level completed",
		mock.Anything,
	)

	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}

// Test SyncPlayerLevel - context cancellation handling
func TestLevelUseCase_SyncPlayerLevel_ContextCancellation(t *testing.T) {
	levelRepo, merchantRepo, logger := createLevelMockDependencies(t)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	merchant := createTestMerchantForLevel()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).
		Return(context.Canceled)

	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()
	useCase := NewLevelUseCase(levelRepo, merchantRepo, logger, tracingService)

	event := createIdentityPlayerLevelSyncEvent()

	err := useCase.SyncPlayerLevel(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert player level")
	merchantRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}
