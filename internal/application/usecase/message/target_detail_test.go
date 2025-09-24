package message

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMessageCampaign_TargetDetail_Player(t *testing.T) {
	// Setup
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	tagRepo := mocks.NewTagRepositoryMock(t)
	pushKeyRepo := mocks.NewPushKeyRepositoryMock(t)
	pushService := mocks.NewPushNotificationServiceMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()

	useCase := NewMessageUseCase(
		campaignRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushKeyRepo,
		pushService,
		logger,
		tracingService,
	)

	// Test data
	playerAccounts := []string{"player1", "player2", "player3"}
	targetDetailJSON, _ := json.Marshal(playerAccounts)
	targetDetailStr := string(targetDetailJSON)

	campaign := &entity.MessageCampaign{
		ID:           1,
		GlobalID:     "test-campaign",
		Title:        "Test Campaign",
		Target:       consts.TargetPlayer,
		TargetDetail: &targetDetailStr,
	}

	campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), "test-campaign").
		Return(campaign, nil).Once()

	// Execute
	ctx := context.Background()
	response, err := useCase.GetMessageCampaign(ctx, "test-campaign")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, consts.TargetPlayer, response.TargetType)
	assert.Equal(t, playerAccounts, response.TargetDetail)

	campaignRepo.AssertExpectations()
}

func TestGetMessageCampaign_TargetDetail_Level(t *testing.T) {
	// Setup
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	tagRepo := mocks.NewTagRepositoryMock(t)
	pushKeyRepo := mocks.NewPushKeyRepositoryMock(t)
	pushService := mocks.NewPushNotificationServiceMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()

	useCase := NewMessageUseCase(
		campaignRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushKeyRepo,
		pushService,
		logger,
		tracingService,
	)

	// Test data
	levelIDs := []string{"1", "2", "3"}
	targetDetailJSON, _ := json.Marshal(levelIDs)
	targetDetailStr := string(targetDetailJSON)

	campaign := &entity.MessageCampaign{
		ID:           1,
		GlobalID:     "test-campaign",
		Title:        "Test Campaign",
		Target:       consts.TargetLevel,
		TargetDetail: &targetDetailStr,
	}

	levels := []*entity.Level{
		{ID: 1, Name: "Bronze"},
		{ID: 2, Name: "Silver"},
		{ID: 3, Name: "Gold"},
	}

	campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), "test-campaign").
		Return(campaign, nil).Once()
	levelRepo.On("FindByIDs", mocks.ContextMatcher(), []uint64{1, 2, 3}).
		Return(levels, nil).Once()

	// Execute
	ctx := context.Background()
	response, err := useCase.GetMessageCampaign(ctx, "test-campaign")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, consts.TargetLevel, response.TargetType)
	assert.Equal(t, []string{"Bronze", "Silver", "Gold"}, response.TargetDetail)

	campaignRepo.AssertExpectations()
	levelRepo.AssertExpectations()
}

func TestGetMessageCampaign_TargetDetail_Tag(t *testing.T) {
	// Setup
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	tagRepo := mocks.NewTagRepositoryMock(t)
	pushKeyRepo := mocks.NewPushKeyRepositoryMock(t)
	pushService := mocks.NewPushNotificationServiceMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()

	useCase := NewMessageUseCase(
		campaignRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushKeyRepo,
		pushService,
		logger,
		tracingService,
	)

	// Test data
	tagIDs := []string{"10", "20", "30"}
	targetDetailJSON, _ := json.Marshal(tagIDs)
	targetDetailStr := string(targetDetailJSON)

	campaign := &entity.MessageCampaign{
		ID:           1,
		GlobalID:     "test-campaign",
		Title:        "Test Campaign",
		Target:       consts.TargetTag,
		TargetDetail: &targetDetailStr,
	}

	tags := []*entity.Tag{
		{ID: 10, Name: "VIP"},
		{ID: 20, Name: "Premium"},
		{ID: 30, Name: "Regular"},
	}

	campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), "test-campaign").
		Return(campaign, nil).Once()
	tagRepo.On("FindByIDs", mocks.ContextMatcher(), []uint64{10, 20, 30}).
		Return(tags, nil).Once()

	// Execute
	ctx := context.Background()
	response, err := useCase.GetMessageCampaign(ctx, "test-campaign")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, consts.TargetTag, response.TargetType)
	assert.Equal(t, []string{"VIP", "Premium", "Regular"}, response.TargetDetail)

	campaignRepo.AssertExpectations()
	tagRepo.AssertExpectations()
}

func TestGetMessageCampaign_TargetDetail_NoTargetDetail(t *testing.T) {
	// Setup
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	tagRepo := mocks.NewTagRepositoryMock(t)
	pushKeyRepo := mocks.NewPushKeyRepositoryMock(t)
	pushService := mocks.NewPushNotificationServiceMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()

	useCase := NewMessageUseCase(
		campaignRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushKeyRepo,
		pushService,
		logger,
		tracingService,
	)

	// Test data
	campaign := &entity.MessageCampaign{
		ID:           1,
		GlobalID:     "test-campaign",
		Title:        "Test Campaign",
		Target:       consts.TargetAll,
		TargetDetail: nil, // No target detail
	}

	campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), "test-campaign").
		Return(campaign, nil).Once()

	// Execute
	ctx := context.Background()
	response, err := useCase.GetMessageCampaign(ctx, "test-campaign")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, consts.TargetAll, response.TargetType)
	assert.Equal(t, []string{}, response.TargetDetail)

	campaignRepo.AssertExpectations()
}
