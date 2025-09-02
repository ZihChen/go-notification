package message_campaign

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories
type MockMessageCampaignRepository struct {
	mock.Mock
}

func (m *MockMessageCampaignRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) FindAllWithOptions(
	ctx context.Context,
	query *entity.MessageCampaignsQuery,
) ([]*entity.MessageCampaign, int, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Int(1), args.Error(2)
}

func (m *MockMessageCampaignRepository) FindActiveByFocus(
	ctx context.Context,
	focus uint8,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx, focus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) FindScheduledCampaigns(
	ctx context.Context,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) FindAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) FindAutoSettingByCategoryItemTrigger(
	ctx context.Context,
	merchantID uint64,
	category uint8,
	item uint8,
	triggerType string,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, merchantID, category, item, triggerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MockMessageCampaignRepository) UpsertAutoSettings(
	ctx context.Context,
	campaigns []*entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaigns)
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) DeleteAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) error {
	args := m.Called(ctx, merchantID)
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) Create(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaign)
	campaign.ID = 1 // 設置創建的活動ID
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) Update(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) UpdateFields(
	ctx context.Context,
	id uint64,
	updates map[string]interface{},
	includeDeleted bool,
) error {
	args := m.Called(ctx, id, updates, includeDeleted)
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) UpdateSentCount(
	ctx context.Context,
	campaignID uint64,
	count int64,
) error {
	args := m.Called(ctx, campaignID, count)
	return args.Error(0)
}

func (m *MockMessageCampaignRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

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
	merchant.ID = 1
	return args.Error(0)
}

func (m *MockMerchantRepository) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	merchant.ID = 1
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

type MockPlayerMessageRepository struct {
	mock.Mock
}

func (m *MockPlayerMessageRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.PlayerMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PlayerMessage), args.Error(1)
}

func (m *MockPlayerMessageRepository) FindByPlayerID(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) ([]*entity.PlayerMessage, int, error) {
	args := m.Called(ctx, globalPlayerID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.PlayerMessage), args.Int(1), args.Error(2)
}

func (m *MockPlayerMessageRepository) GetPlayerMessageStats(
	ctx context.Context,
	globalPlayerID string,
) (*entity.PlayerMessageStats, error) {
	args := m.Called(ctx, globalPlayerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PlayerMessageStats), args.Error(1)
}

func (m *MockPlayerMessageRepository) Create(
	ctx context.Context,
	message *entity.PlayerMessage,
) error {
	args := m.Called(ctx, message)
	message.ID = 1
	return args.Error(0)
}

func (m *MockPlayerMessageRepository) CreateBatch(
	ctx context.Context,
	messages []*entity.PlayerMessage,
) error {
	args := m.Called(ctx, messages)
	for i, message := range messages {
		message.ID = uint64(i + 1)
	}
	return args.Error(0)
}

func (m *MockPlayerMessageRepository) CreateBatchOptimized(
	ctx context.Context,
	messages []*entity.PlayerMessage,
	batchSize int,
) error {
	args := m.Called(ctx, messages, batchSize)
	return args.Error(0)
}

func (m *MockPlayerMessageRepository) MarkAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	args := m.Called(ctx, globalPlayerID, messageID)
	return args.Error(0)
}

func (m *MockPlayerMessageRepository) CheckMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
) (bool, error) {
	args := m.Called(ctx, globalPlayerID, campaignID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPlayerMessageRepository) CheckMessageExistsBatch(
	ctx context.Context,
	playerIDs []uint64,
	campaignID uint64,
) (map[uint64]bool, error) {
	args := m.Called(ctx, playerIDs, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uint64]bool), args.Error(1)
}

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
	player.ID = 1
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

// Helper functions
func createMessageMockDependencies(t *testing.T) (
	*MockMessageCampaignRepository,
	*MockMerchantRepository,
	*MockPlayerMessageRepository,
	*MockPlayerRepository,
	*helper.MockLogger,
) {
	campaignRepo := new(MockMessageCampaignRepository)
	merchantRepo := new(MockMerchantRepository)
	playerMessageRepo := new(MockPlayerMessageRepository)
	playerRepo := new(MockPlayerRepository)
	logger := helper.SetupLoggerMock(t)

	return campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger
}

func createTestMerchant() *entity.Merchant {
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

func createTestMessageCampaign() *entity.MessageCampaign {
	return &entity.MessageCampaign{
		ID:            1,
		MerchantID:    1,
		GlobalID:      "test-campaign-global-id",
		Category:      1,
		Item:          1,
		TriggerType:   "success",
		Title:         "Test Campaign",
		Content:       "Test campaign content",
		Target:        consts.TargetAll,
		Status:        consts.MessageCampaignStatusDraft,
		AutoSend:      false,
		RealSentCount: 0,
		CreatedBy:     "test-user",
		CreatedAt:     time.Now().Add(-1 * time.Hour),
		UpdatedAt:     time.Now().Add(-1 * time.Hour),
	}
}

func createTestPlayer(lastActive *time.Time) *entity.Player {
	return &entity.Player{
		ID:             1,
		MerchantID:     1,
		GlobalPlayerID: "FATCAT-PLAYER-1",
		APIKey:         "player-api-key",
		Account:        "testplayer",
		Email:          stringPtr("player@example.com"),
		LastActiveAt:   lastActive,
		CreatedAt:      time.Now().Add(-48 * time.Hour),
		UpdatedAt:      time.Now().Add(-48 * time.Hour),
	}
}

func stringPtr(s string) *string {
	return &s
}

// Test CreateMessageCampaign
func TestMessageUseCase_CreateMessageCampaign(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	campaignRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.MessageCampaign")).
		Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Category:         1,
		Item:             1,
		TriggerType:      "success",
		Title:            "Test Campaign",
		Content:          "Test campaign content",
		Target:           consts.TargetAll,
		CreatedBy:        "test-user",
	}

	err := useCase.CreateMessageCampaign(context.Background(), request)

	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
}

// Test CreateMessageCampaign with merchant repository error (not ErrRepoMerchantNotFound)
func TestMessageUseCase_CreateMessageCampaign_MerchantRepositoryError(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("database connection error"))

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Category:         1,
		Item:             1,
		Title:            "Test Campaign",
		Content:          "Test campaign content",
		Target:           consts.TargetAll,
		CreatedBy:        "test-user",
	}

	err := useCase.CreateMessageCampaign(context.Background(), request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations(t)
}

// Test UpdateMessageCampaign
func TestMessageUseCase_UpdateMessageCampaign(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	existingCampaign := createTestMessageCampaign()
	campaignRepo.On("FindByGlobalID", mock.Anything, "test-campaign-global-id").
		Return(existingCampaign, nil)
	campaignRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.MessageCampaign")).
		Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.UpdateMessageCampaignRequest{
		GlobalID:         "test-campaign-global-id",
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Category:         2,
		Item:             2,
		Title:            "Updated Campaign",
		Content:          "Updated campaign content",
		Target:           consts.TargetAll,
		UpdatedBy:        "update-user",
	}

	err := useCase.UpdateMessageCampaign(context.Background(), request)

	assert.NoError(t, err)
	campaignRepo.AssertExpectations(t)
}

// Test DeleteMessageCampaign
func TestMessageUseCase_DeleteMessageCampaign(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	existingCampaign := createTestMessageCampaign()
	campaignRepo.On("FindByGlobalID", mock.Anything, "test-campaign-global-id").
		Return(existingCampaign, nil)
	campaignRepo.On("Delete", mock.Anything, uint64(1)).Return(nil)
	campaignRepo.On("UpdateFields", mock.Anything, uint64(1), mock.Anything, true).Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	err := useCase.DeleteMessageCampaign(context.Background(), "test-campaign-global-id")

	assert.NoError(t, err)
	campaignRepo.AssertExpectations(t)
}

// Test GetMessageCampaign
func TestMessageUseCase_GetMessageCampaign(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	expectedCampaign := createTestMessageCampaign()
	campaignRepo.On("FindByGlobalID", mock.Anything, "test-campaign-global-id").
		Return(expectedCampaign, nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	campaign, err := useCase.GetMessageCampaign(context.Background(), "test-campaign-global-id")

	assert.NoError(t, err)
	assert.NotNil(t, campaign)
	assert.Equal(t, expectedCampaign.ID, campaign.ID)
	assert.Equal(t, expectedCampaign.Title, campaign.Title)
	assert.Equal(t, expectedCampaign.Content, campaign.Content)
	campaignRepo.AssertExpectations(t)
}

// Test ListMessageCampaigns
func TestMessageUseCase_ListMessageCampaigns(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	expectedCampaigns := []*entity.MessageCampaign{createTestMessageCampaign()}
	expectedTotal := 1

	campaignRepo.On("FindAllWithOptions", mock.Anything, mock.AnythingOfType("*entity.MessageCampaignsQuery")).
		Return(expectedCampaigns, expectedTotal, nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.ListMessageCampaignsRequest{
		Page:     1,
		PageSize: 10,
		Category: 1,
	}

	campaigns, total, err := useCase.ListMessageCampaigns(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, expectedTotal, total)
	assert.Len(t, campaigns, 1)
	assert.Equal(t, expectedCampaigns[0].ID, campaigns[0].ID)
	campaignRepo.AssertExpectations(t)
}

// Test GetPlayerMessages
func TestMessageUseCase_GetPlayerMessages(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalPlayerID := "FATCAT-PLAYER-1"
	lastActive := time.Now().Add(-15 * 24 * time.Hour) // 15 days ago - high activity
	testPlayer := createTestPlayer(&lastActive)

	// Mock player lookup and campaign matching
	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(testPlayer, nil)
	campaignRepo.On("FindActiveByFocus", mock.Anything, consts.TargetHighActivity).
		Return([]*entity.MessageCampaign{}, nil)
	playerMessageRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entity.PlayerMessage")).
		Return(nil).
		Maybe()

	// Mock stats and messages
	stats := &entity.PlayerMessageStats{
		TotalCount:  5,
		ReadCount:   2,
		UnreadCount: 3,
	}
	messages := []*entity.PlayerMessage{
		{
			ID:             1,
			GlobalPlayerID: globalPlayerID,
			CampaignID:     1,
			IsRead:         false,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             2,
			GlobalPlayerID: globalPlayerID,
			CampaignID:     2,
			IsRead:         true,
			CreatedAt:      time.Now().Add(-2 * time.Hour),
		},
	}

	playerMessageRepo.On("GetPlayerMessageStats", mock.Anything, globalPlayerID).Return(stats, nil)
	playerMessageRepo.On("FindByPlayerID", mock.Anything, globalPlayerID, 1, 10).
		Return(messages, 2, nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	response, err := useCase.GetPlayerMessages(context.Background(), globalPlayerID, 1, 10)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, stats.TotalCount, response.Stats.TotalCount)
	assert.Equal(t, stats.ReadCount, response.Stats.ReadCount)
	assert.Equal(t, stats.UnreadCount, response.Stats.UnreadCount)
	assert.Len(t, response.Messages, 2)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.Equal(t, 2, response.Total)

	playerRepo.AssertExpectations(t)
	playerMessageRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
}

// Test MarkMessageAsRead
func TestMessageUseCase_MarkMessageAsRead(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalPlayerID := "FATCAT-PLAYER-1"
	messageID := uint64(1)

	playerMessageRepo.On("MarkAsRead", mock.Anything, globalPlayerID, messageID).Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	err := useCase.MarkMessageAsRead(context.Background(), globalPlayerID, messageID)

	assert.NoError(t, err)
	playerMessageRepo.AssertExpectations(t)
}

// Test GetMerchantAutoSettings
func TestMessageUseCase_GetMerchantAutoSettings(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalMerchantID := "FATCAT-MERCHANT-1"
	merchant := createTestMerchant()
	autoSettings := []*entity.MessageCampaign{createTestMessageCampaign()}

	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(merchant, nil)
	campaignRepo.On("FindAutoSettingsByMerchantID", mock.Anything, merchant.ID).
		Return(autoSettings, nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	response, err := useCase.GetMerchantAutoSettings(context.Background(), globalMerchantID)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, globalMerchantID, response.GlobalMerchantID)
	assert.Len(t, response.Settings, 1)
	assert.Equal(t, autoSettings[0].ID, response.Settings[0].ID)

	merchantRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
}

// Test CreateOrUpdateMerchantAutoSettings
func TestMessageUseCase_CreateOrUpdateMerchantAutoSettings(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalMerchantID := "FATCAT-MERCHANT-1"
	merchant := createTestMerchant()

	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(merchant, nil)
	campaignRepo.On("UpsertAutoSettings", mock.Anything, mock.AnythingOfType("[]*entity.MessageCampaign")).
		Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.MerchantAutoSettingsRequest{
		GlobalMerchantID: globalMerchantID,
		Settings: []dto.AutoSettingItem{
			{
				Category:    1,
				Item:        1,
				TriggerType: "success",
				Title:       "Auto Setting 1",
				Content:     "Auto setting content 1",
			},
			{
				Category:    2,
				Item:        2,
				TriggerType: "failure",
				Title:       "Auto Setting 2",
				Content:     "Auto setting content 2",
			},
		},
	}

	response, err := useCase.CreateOrUpdateMerchantAutoSettings(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, globalMerchantID, response.GlobalMerchantID)
	assert.Equal(t, 2, response.CreatedCount)
	assert.Equal(t, 0, response.UpdatedCount)
	assert.Equal(t, "upsert", response.Operation)

	merchantRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
}

// Test processPlayerMessages with high activity player
func TestMessageUseCase_processPlayerMessages_HighActivity(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalPlayerID := "FATCAT-PLAYER-1"
	lastActive := time.Now().Add(-15 * 24 * time.Hour) // 15 days ago - high activity
	testPlayer := createTestPlayer(&lastActive)

	activeCampaigns := []*entity.MessageCampaign{createTestMessageCampaign()}

	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(testPlayer, nil)
	campaignRepo.On("FindActiveByFocus", mock.Anything, consts.TargetHighActivity).
		Return(activeCampaigns, nil)
	playerMessageRepo.On("CheckMessageExists", mock.Anything, globalPlayerID, uint64(1)).
		Return(false, nil)
	playerMessageRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entity.PlayerMessage")).
		Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	// Call the private method through reflection or create a test wrapper
	// For now, we test through GetPlayerMessages which calls processPlayerMessages
	playerMessageRepo.On("GetPlayerMessageStats", mock.Anything, globalPlayerID).
		Return(&entity.PlayerMessageStats{TotalCount: 1}, nil)
	playerMessageRepo.On("FindByPlayerID", mock.Anything, globalPlayerID, 1, 10).
		Return([]*entity.PlayerMessage{}, 0, nil)

	_, err := useCase.GetPlayerMessages(context.Background(), globalPlayerID, 1, 10)

	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
	playerMessageRepo.AssertExpectations(t)
}

// Test processPlayerMessages with no activity player
func TestMessageUseCase_processPlayerMessages_NoActivity(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalPlayerID := "FATCAT-PLAYER-1"
	testPlayer := createTestPlayer(nil) // No last active time

	activeCampaigns := []*entity.MessageCampaign{createTestMessageCampaign()}

	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(testPlayer, nil)
	campaignRepo.On("FindActiveByFocus", mock.Anything, consts.TargetNotActivity).
		Return(activeCampaigns, nil)
	playerMessageRepo.On("CheckMessageExists", mock.Anything, globalPlayerID, uint64(1)).
		Return(false, nil)
	playerMessageRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*entity.PlayerMessage")).
		Return(nil)

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	playerMessageRepo.On("GetPlayerMessageStats", mock.Anything, globalPlayerID).
		Return(&entity.PlayerMessageStats{TotalCount: 1}, nil)
	playerMessageRepo.On("FindByPlayerID", mock.Anything, globalPlayerID, 1, 10).
		Return([]*entity.PlayerMessage{}, 0, nil)

	_, err := useCase.GetPlayerMessages(context.Background(), globalPlayerID, 1, 10)

	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
	playerMessageRepo.AssertExpectations(t)
}

// Test error cases
func TestMessageUseCase_CreateMessageCampaign_RepositoryError(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)
	campaignRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.MessageCampaign")).
		Return(errors.New("database error"))

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	request := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Category:         1,
		Item:             1,
		Title:            "Test Campaign",
		Content:          "Test campaign content",
		Target:           consts.TargetAll,
		CreatedBy:        "test-user",
	}

	err := useCase.CreateMessageCampaign(context.Background(), request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create campaign")
	merchantRepo.AssertExpectations(t)
	campaignRepo.AssertExpectations(t)
}

func TestMessageUseCase_GetMessageCampaign_NotFound(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	campaignRepo.On("FindByGlobalID", mock.Anything, "non-existent-id").
		Return(nil, errors.New("campaign not found"))

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	campaign, err := useCase.GetMessageCampaign(context.Background(), "non-existent-id")

	assert.Error(t, err)
	assert.Nil(t, campaign)
	campaignRepo.AssertExpectations(t)
}

func TestMessageUseCase_MarkMessageAsRead_Error(t *testing.T) {
	campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger := createMessageMockDependencies(
		t,
	)

	globalPlayerID := "FATCAT-PLAYER-1"
	messageID := uint64(1)

	playerMessageRepo.On("MarkAsRead", mock.Anything, globalPlayerID, messageID).
		Return(errors.New("database error"))

	useCase := NewMessageUseCase(campaignRepo, merchantRepo, playerMessageRepo, playerRepo, logger)

	err := useCase.MarkMessageAsRead(context.Background(), globalPlayerID, messageID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mark message as read")
	playerMessageRepo.AssertExpectations(t)
}
