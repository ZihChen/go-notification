package message

import (
	"context"
	"errors"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/test/factories"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MessageTestSuite 訊息測試套件
type MessageTestSuite struct {
	useCase           *MessageUseCase
	campaignRepo      *mocks.MessageCampaignRepositoryMock
	playerRepo        *mocks.PlayerRepositoryMock
	playerMessageRepo *mocks.PlayerMessageRepositoryMock
	merchantRepo      *mocks.MerchantRepositoryMock
	factory           *factories.TestDataFactory
	logger            *helper.MockLogger
}

// setupMessageTestSuite 設定訊息測試套件
func setupMessageTestSuite(t *testing.T) *MessageTestSuite {
	// 創建Mock實例
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	logger := helper.NewMockLogger()

	// 創建UseCase實例
	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		merchantRepo:      merchantRepo,
		logger:            logger,
	}

	// 創建測試數據工廠
	factory := factories.NewTestDataFactory()

	return &MessageTestSuite{
		useCase:           useCase,
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		merchantRepo:      merchantRepo,
		factory:           factory,
		logger:            logger,
	}
}

// tearDown 清理測試套件
func (s *MessageTestSuite) tearDown() {
	if s.campaignRepo != nil {
		s.campaignRepo.Reset()
	}
	if s.playerRepo != nil {
		s.playerRepo.Reset()
	}
	if s.playerMessageRepo != nil {
		s.playerMessageRepo.Reset()
	}
	if s.merchantRepo != nil {
		s.merchantRepo.Reset()
	}
}

// TestCreateMessageCampaign_Success 測試創建訊息活動成功
func TestCreateMessageCampaign_Success(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 使用工廠創建測試數據
	merchant := suite.factory.CreateMerchant()
	
	createDTO := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: merchant.GlobalMerchantID,
		Category:         1,
		Item:             1,
		Title:            "測試活動",
		Content:          "測試內容",
		Target:           1,
		CreatedBy:        "test@example.com",
	}

	// 設定Mock期望
	suite.merchantRepo.On("FindByGlobalID", mocks.ContextMatcher(), merchant.GlobalMerchantID).
		Return(merchant, nil).Once()
	
	suite.campaignRepo.On("Create", mocks.ContextMatcher(), 
		mock.AnythingOfType("*entity.MessageCampaign")).
		Return(nil).Once()

	// 執行測試
	ctx := context.Background()
	err := suite.useCase.CreateMessageCampaign(ctx, createDTO)

	// 驗證結果
	require.NoError(t, err)
	suite.campaignRepo.AssertExpectations()
	suite.merchantRepo.AssertExpectations()
}

// TestCreateMessageCampaign_MerchantNotFound 測試商戶不存在
func TestCreateMessageCampaign_MerchantNotFound(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	createDTO := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: "non-existent-merchant",
		Category:         1,
		Item:             1,
		Title:            "測試活動",
		Content:          "測試內容",
		Target:           1,
		CreatedBy:        "test@example.com",
	}

	// 設定Mock期望
	suite.merchantRepo.On("FindByGlobalID", mocks.ContextMatcher(), 
		"non-existent-merchant").
		Return(nil, errors.New("merchant not found")).Once()

	// 執行測試
	ctx := context.Background()
	err := suite.useCase.CreateMessageCampaign(ctx, createDTO)

	// 驗證結果
	require.Error(t, err)
	suite.merchantRepo.AssertExpectations()
}

// TestGetMessageCampaign_Success 測試獲取訊息活動成功
func TestGetMessageCampaign_Success(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 使用工廠創建測試數據
	campaign := suite.factory.CreateMessageCampaign().
		WithStatus(consts.MessageCampaignStatusDraft).
		Build()

	// 設定Mock期望
	suite.campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), 
		campaign.GlobalID).
		Return(campaign, nil).Once()

	// 執行測試
	ctx := context.Background()
	result, err := suite.useCase.GetMessageCampaign(ctx, campaign.GlobalID)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, campaign.Title, result.Title)
	suite.campaignRepo.AssertExpectations()
}

// TestGetMessageCampaign_NotFound 測試獲取不存在的訊息活動
func TestGetMessageCampaign_NotFound(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	nonExistentID := "non-existent-id"

	// 設定Mock期望
	suite.campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), 
		nonExistentID).
		Return(nil, errors.New("campaign not found")).Once()

	// 執行測試
	ctx := context.Background()
	result, err := suite.useCase.GetMessageCampaign(ctx, nonExistentID)

	// 驗證結果
	require.Error(t, err)
	assert.Nil(t, result)
	suite.campaignRepo.AssertExpectations()
}

// TestUpdateMessageCampaign_Success 測試更新訊息活動成功
func TestUpdateMessageCampaign_Success(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 使用工廠創建測試數據
	merchant := suite.factory.CreateMerchant()
	campaign := suite.factory.CreateMessageCampaign().
		WithStatus(consts.MessageCampaignStatusDraft).
		Build()
	
	updateDTO := &dto.UpdateMessageCampaignRequest{
		GlobalID:         campaign.GlobalID,
		GlobalMerchantID: merchant.GlobalMerchantID,
		Category:         campaign.Category,
		Item:             campaign.Item,
		Title:            "更新的標題",
		Content:          "更新的內容",
		Target:           campaign.Target,
		UpdatedBy:        "updater@example.com",
	}

	// 設定Mock期望
	suite.campaignRepo.On("FindByGlobalID", mocks.ContextMatcher(), 
		campaign.GlobalID).
		Return(campaign, nil).Once()
	
	suite.campaignRepo.On("Update", mocks.ContextMatcher(), 
		mock.AnythingOfType("*entity.MessageCampaign")).
		Return(nil).Once()

	// 執行測試
	ctx := context.Background()
	err := suite.useCase.UpdateMessageCampaign(ctx, updateDTO)

	// 驗證結果
	require.NoError(t, err)
	suite.campaignRepo.AssertExpectations()
}