package message

import (
	"context"
	"errors"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/test/factories"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MessageTestSuite 訊息測試套件
type MessageTestSuite struct {
	useCase            *MessageUseCase
	campaignRepo       *mocks.MessageCampaignRepositoryMock
	campaignTargetRepo *mocks.CampaignTargetRepositoryMock
	playerRepo         *mocks.PlayerRepositoryMock
	playerMessageRepo  *mocks.PlayerMessageRepositoryMock
	merchantRepo       *mocks.MerchantRepositoryMock
	pushKeyRepo        *mocks.PushKeyRepositoryMock
	pushService        *mocks.PushNotificationServiceMock
	factory            *factories.TestDataFactory
	logger             *helper.MockLogger
}

// setupMessageTestSuite 設定訊息測試套件
func setupMessageTestSuite(t *testing.T) *MessageTestSuite {
	// 創建Mock實例
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	campaignTargetRepo := mocks.NewCampaignTargetRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	tagRepo := mocks.NewTagRepositoryMock(t)
	pushKeyRepo := mocks.NewPushKeyRepositoryMock(t)
	pushService := mocks.NewPushNotificationServiceMock(t)
	logger := helper.NewMockLogger()
	tracingService := mocks.NewTracingServiceMock(t)
	tracingService.SetupSuccess()

	// 創建UseCase實例
	useCase := NewMessageUseCase(
		campaignRepo,
		campaignTargetRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushKeyRepo,
		pushService,
		logger,
		tracingService,
	).(*MessageUseCase)

	// 創建測試數據工廠
	factory := factories.NewTestDataFactory()

	return &MessageTestSuite{
		useCase:            useCase,
		campaignRepo:       campaignRepo,
		campaignTargetRepo: campaignTargetRepo,
		playerRepo:         playerRepo,
		playerMessageRepo:  playerMessageRepo,
		merchantRepo:       merchantRepo,
		pushKeyRepo:        pushKeyRepo,
		pushService:        pushService,
		factory:            factory,
		logger:             logger,
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
	if s.pushKeyRepo != nil {
		s.pushKeyRepo.Reset()
	}
	if s.pushService != nil {
		s.pushService.Reset()
	}
}

// TestCreateMessageCampaign_Success 測試創建訊息活動成功
func TestCreateMessageCampaign_Success(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 使用工廠創建測試數據
	merchant := suite.factory.CreateMerchant()

	notificationTypes := uint8(3) // 站內信+App推播
	createDTO := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID:  merchant.GlobalMerchantID,
		Category:          consts.CategoryMember,
		Item:              consts.ItemRegistration,
		Title:             "測試活動",
		Content:           "測試內容",
		NotificationTypes: &notificationTypes,
		Target:            consts.TargetHighActivity,
		Status:            "draft",
		CreatedBy:         "test@example.com",
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

	notificationTypes := uint8(1) // 僅站內信
	createDTO := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID:  "non-existent-merchant",
		Category:          consts.CategoryMember,
		Item:              consts.ItemRegistration,
		Title:             "測試活動",
		Content:           "測試內容",
		NotificationTypes: &notificationTypes,
		Target:            consts.TargetHighActivity,
		Status:            "draft",
		CreatedBy:         "test@example.com",
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

	// Mock campaign target operations for dual-write
	suite.campaignTargetRepo.On("DeleteByCampaignID", mocks.ContextMatcher(),
		campaign.ID).
		Return(nil).Once()

	// 執行測試
	ctx := context.Background()
	err := suite.useCase.UpdateMessageCampaign(ctx, updateDTO)

	// 驗證結果
	require.NoError(t, err)
	suite.campaignRepo.AssertExpectations()
}

// TestSendAutoNotification_Success 測試發送自動推播成功
func TestSendAutoNotification_Success(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 準備測試數據
	player := suite.factory.CreatePlayer().Build()
	campaign := suite.factory.CreateMessageCampaign().
		WithMerchantID(player.MerchantID).
		Build()
	// 直接設置需要的字段
	campaign.Category = "member"
	campaign.Item = "registration"
	campaign.TriggerType = "success"
	campaign.AutoSend = true
	campaign.NotificationTypes = 3 // 站內信 + App推播
	appContent := "歡迎註冊！"
	campaign.AppContent = &appContent

	pushApiKey := suite.factory.CreatePushKey()
	pushApiKey.MerchantID = player.MerchantID

	// 設定Mock期望
	suite.playerRepo.On("FindByGlobalID", mock.Anything, player.GlobalPlayerID).
		Return(player, nil).Once()

	suite.campaignRepo.On("FindAutoSettingByCategoryItemTrigger",
		mock.Anything, player.MerchantID, "member", "registration", "success").
		Return(campaign, nil).Once()

	suite.playerMessageRepo.On("CreateAutoNotification", mock.Anything, mock.AnythingOfType("*entity.PlayerMessage"), 5).
		Return(nil).
		Once()

	// Mock PushKeyRepository for push notification
	suite.pushKeyRepo.On("FindByMerchantID", mock.Anything, player.MerchantID).
		Return(pushApiKey, nil).Once()

	// Mock PushNotificationService
	suite.pushService.On("SendPushNotification", mock.Anything, pushApiKey.Key, mock.AnythingOfType("*service.PushNotificationRequest")).
		Return(&service.PushNotificationResponse{Success: true}, nil).
		Once()

	// Mock UpdateSentCount
	suite.campaignRepo.On("UpdateSentCount", mock.Anything, campaign.ID, campaign.RealSentCount+1).
		Return(nil).Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: player.GlobalPlayerID,
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "sent", response.Status)
	assert.Equal(t, player.GlobalPlayerID, response.PlayerID)
	assert.Equal(t, "member", response.Category)
	assert.Equal(t, "registration", response.Item)
	assert.Equal(t, "success", response.TriggerType)
	assert.Contains(t, response.SentChannels, "in_app")
	assert.Contains(t, response.SentChannels, "push")

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
	suite.campaignRepo.AssertExpectations()
	suite.playerMessageRepo.AssertExpectations()
	suite.pushKeyRepo.AssertExpectations()
	suite.pushService.AssertExpectations()
}

// TestSendAutoNotification_PlayerNotFound 測試玩家不存在的情況
func TestSendAutoNotification_PlayerNotFound(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 設定Mock期望 - 玩家不存在
	suite.playerRepo.On("FindByGlobalID", mock.Anything, "non-existent-player").
		Return(nil, errors.New("record not found")).Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: "non-existent-player",
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "failed", response.Status)
	assert.Equal(t, "player not found", response.Message)

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
}

// TestSendAutoNotification_CampaignNotFound 測試找不到對應的自動設定
func TestSendAutoNotification_CampaignNotFound(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 準備測試數據
	player := suite.factory.CreatePlayer().Build()

	// 設定Mock期望
	suite.playerRepo.On("FindByGlobalID", mock.Anything, player.GlobalPlayerID).
		Return(player, nil).Once()

	suite.campaignRepo.On("FindAutoSettingByCategoryItemTrigger",
		mock.Anything, player.MerchantID, "member", "registration", "success").
		Return(nil, errors.New("record not found")).Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: player.GlobalPlayerID,
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "not_found", response.Status)
	assert.Equal(t, "no matching auto notification setting found", response.Message)

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
	suite.campaignRepo.AssertExpectations()
}

// TestSendAutoNotification_InvalidNotificationType 測試無效的通知類型
func TestSendAutoNotification_InvalidNotificationType(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 準備測試數據
	player := suite.factory.CreatePlayer().Build()
	campaign := suite.factory.CreateMessageCampaign().
		WithMerchantID(player.MerchantID).
		Build()
	// 設置無效的 notification type (超出範圍)
	campaign.Category = "member"
	campaign.Item = "registration"
	campaign.TriggerType = "success"
	campaign.AutoSend = true
	campaign.NotificationTypes = 255 // 無效值

	// 設定Mock期望
	suite.playerRepo.On("FindByGlobalID", mock.Anything, player.GlobalPlayerID).
		Return(player, nil).Once()

	suite.campaignRepo.On("FindAutoSettingByCategoryItemTrigger",
		mock.Anything, player.MerchantID, "member", "registration", "success").
		Return(campaign, nil).Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: player.GlobalPlayerID,
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "failed", response.Status)
	assert.Equal(t, "invalid notification type configuration", response.Message)

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
	suite.campaignRepo.AssertExpectations()
}

// TestSendAutoNotification_PlayerMessageCreateFailed 測試站內信創建失敗的情況
func TestSendAutoNotification_PlayerMessageCreateFailed(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 準備測試數據
	player := suite.factory.CreatePlayer().Build()
	campaign := suite.factory.CreateMessageCampaign().
		WithMerchantID(player.MerchantID).
		Build()
	// 設置為僅站內信
	campaign.Category = "member"
	campaign.Item = "registration"
	campaign.TriggerType = "success"
	campaign.AutoSend = true
	campaign.NotificationTypes = 1 // 僅站內信

	// 設定Mock期望
	suite.playerRepo.On("FindByGlobalID", mock.Anything, player.GlobalPlayerID).
		Return(player, nil).Once()

	suite.campaignRepo.On("FindAutoSettingByCategoryItemTrigger",
		mock.Anything, player.MerchantID, "member", "registration", "success").
		Return(campaign, nil).Once()

	// Mock PlayerMessage 創建失敗
	suite.playerMessageRepo.On("CreateAutoNotification", mock.Anything, mock.AnythingOfType("*entity.PlayerMessage"), 5).
		Return(errors.New("database error")).
		Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: player.GlobalPlayerID,
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "failed", response.Status)
	assert.Equal(t, "failed to create in-app message", response.Message)

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
	suite.campaignRepo.AssertExpectations()
	suite.playerMessageRepo.AssertExpectations()
}

// TestSendAutoNotification_TimeWindowDuplicate 測試時間窗口內重複發送的情況
func TestSendAutoNotification_TimeWindowDuplicate(t *testing.T) {
	suite := setupMessageTestSuite(t)
	defer suite.tearDown()

	// 準備測試數據
	player := suite.factory.CreatePlayer().Build()
	campaign := suite.factory.CreateMessageCampaign().
		WithMerchantID(player.MerchantID).
		Build()
	// 設置為僅站內信
	campaign.Category = "member"
	campaign.Item = "registration"
	campaign.TriggerType = "success"
	campaign.AutoSend = true
	campaign.NotificationTypes = 1 // 僅站內信

	// 設定Mock期望
	suite.playerRepo.On("FindByGlobalID", mock.Anything, player.GlobalPlayerID).
		Return(player, nil).Once()

	suite.campaignRepo.On("FindAutoSettingByCategoryItemTrigger",
		mock.Anything, player.MerchantID, "member", "registration", "success").
		Return(campaign, nil).Once()

	// Mock CreateAutoNotification 返回時間窗口重複錯誤
	suite.playerMessageRepo.On("CreateAutoNotification", mock.Anything, mock.AnythingOfType("*entity.PlayerMessage"), 5).
		Return(errors.New("duplicate auto notification within 5 minutes")).
		Once()

	// 準備請求
	req := &dto.SendAutoNotificationRequest{
		GlobalPlayerID: player.GlobalPlayerID,
		Category:       "member",
		Item:           "registration",
		TriggerType:    "success",
	}

	// 執行測試
	ctx := context.Background()
	response, err := suite.useCase.SendAutoNotification(ctx, req)

	// 驗證結果
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "skipped", response.Status)
	assert.Equal(t, "duplicate notification within time window", response.Message)

	// 驗證Mock調用
	suite.playerRepo.AssertExpectations()
	suite.campaignRepo.AssertExpectations()
	suite.playerMessageRepo.AssertExpectations()
}

// TestValidateAutoSettingsCompleteness 測試自動設定完整性驗證
func TestValidateAutoSettingsCompleteness(t *testing.T) {
	useCase := &MessageUseCase{}

	tests := []struct {
		name        string
		settings    []dto.AutoSettingItem
		expectError bool
		errorMsg    string
	}{
		{
			name: "完整的六種組合 - 成功",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "registration",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "恭喜，註冊成功",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "success",
					Title:       "實名驗證成功",
					Content:     "恭喜，實名驗證成功",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "failure",
					Title:       "實名驗證失敗",
					Content:     "抱歉，實名驗證失敗",
				},
				{
					Category:    "member",
					Item:        "bank_card",
					TriggerType: "failure",
					Title:       "取款方式綁定失敗",
					Content:     "抱歉，取款方式綁定失敗",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "success",
					Title:       "優惠派發成功",
					Content:     "恭喜，優惠派發成功",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "failure",
					Title:       "優惠活動派發失敗",
					Content:     "抱歉，優惠活動派發失敗",
				},
			},
			expectError: false,
		},
		{
			name: "缺少必需組合 - 失敗",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "registration",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "歡迎加入",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "success",
					Title:       "實名驗證成功",
					Content:     "驗證完成",
				},
			},
			expectError: true,
			errorMsg:    "missing required combinations",
		},
		{
			name: "標題為空 - 失敗",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "registration",
					TriggerType: "success",
					Title:       "",
					Content:     "歡迎加入",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "success",
					Title:       "實名驗證成功",
					Content:     "驗證完成",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "failure",
					Title:       "實名驗證失敗",
					Content:     "驗證失敗",
				},
				{
					Category:    "member",
					Item:        "bank_card",
					TriggerType: "failure",
					Title:       "銀行卡綁定失敗",
					Content:     "綁定失敗",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "success",
					Title:       "優惠派發成功",
					Content:     "優惠已發放",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "failure",
					Title:       "優惠派發失敗",
					Content:     "發放失敗",
				},
			},
			expectError: true,
			errorMsg:    "title is required",
		},
		{
			name: "內容為空 - 失敗",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "registration",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "success",
					Title:       "實名驗證成功",
					Content:     "驗證完成",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "failure",
					Title:       "實名驗證失敗",
					Content:     "驗證失敗",
				},
				{
					Category:    "member",
					Item:        "bank_card",
					TriggerType: "failure",
					Title:       "銀行卡綁定失敗",
					Content:     "綁定失敗",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "success",
					Title:       "優惠派發成功",
					Content:     "優惠已發放",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "failure",
					Title:       "優惠派發失敗",
					Content:     "發放失敗",
				},
			},
			expectError: true,
			errorMsg:    "content is required",
		},
		{
			name: "無效的category - 失敗",
			settings: []dto.AutoSettingItem{
				{
					Category:    "invalid",
					Item:        "registration",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "歡迎加入",
				},
			},
			expectError: true,
			errorMsg:    "invalid category",
		},
		{
			name: "無效的item - 失敗",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "invalid",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "歡迎加入",
				},
			},
			expectError: true,
			errorMsg:    "invalid item",
		},
		{
			name: "使用mission作為item值 - 成功",
			settings: []dto.AutoSettingItem{
				{
					Category:    "member",
					Item:        "registration",
					TriggerType: "success",
					Title:       "註冊成功",
					Content:     "歡迎加入",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "success",
					Title:       "實名驗證成功",
					Content:     "驗證完成",
				},
				{
					Category:    "member",
					Item:        "identity_verification",
					TriggerType: "failure",
					Title:       "實名驗證失敗",
					Content:     "驗證失敗",
				},
				{
					Category:    "member",
					Item:        "bank_card",
					TriggerType: "failure",
					Title:       "銀行卡綁定失敗",
					Content:     "綁定失敗",
				},
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "success",
					Title:       "優惠派發成功",
					Content:     "優惠已發放",
				}, // mission映射到6
				{
					Category:    "bonus",
					Item:        "mission",
					TriggerType: "failure",
					Title:       "優惠派發失敗",
					Content:     "發放失敗",
				}, // mission映射到6
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := useCase.validateAutoSettingsCompleteness(tt.settings)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
