package message

import (
	"context"
	"errors"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/test/factories"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessScheduledCampaigns_Simple 簡單測試處理排程活動
func TestProcessScheduledCampaigns_Simple(t *testing.T) {
	// 設定測試環境
	factory := factories.NewTestDataFactory()
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	logger := helper.NewMockLogger()

	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		logger:            logger,
	}

	// 使用工廠創建測試數據
	scheduledCampaigns := []*entity.MessageCampaign{
		factory.CreateScheduledCampaign(),
	}

	// 設定Mock期望
	campaignRepo.On("FindScheduledCampaigns", mocks.ContextMatcher()).
		Return(scheduledCampaigns, nil).Once()

	// 為活動設定處理期望（簡化版）
	campaign := scheduledCampaigns[0]
	campaignRepo.On("FindByID", mocks.ContextMatcher(), campaign.ID).
		Return(campaign, nil).Once()

	// 模擬空玩家列表（快速完成）
	playerRepo.On("FindByTargetType", mocks.ContextMatcher(), campaign.Target, 0, 5000).
		Return([]*entity.Player{}, nil).Once()

	// 更新統計和狀態
	campaignRepo.On("UpdateSentCount", mocks.ContextMatcher(), campaign.ID, int64(0)).
		Return(nil).Once()
	campaignRepo.On("UpdateStatus", mocks.ContextMatcher(), campaign.ID, consts.MessageCampaignStatusSent).
		Return(nil).
		Once()

	// 執行測試
	ctx := context.Background()
	err := useCase.ProcessScheduledCampaigns(ctx)

	// 驗證結果
	require.NoError(t, err)
	campaignRepo.AssertExpectations()
	playerRepo.AssertExpectations()

	// 清理
	campaignRepo.Reset()
	playerRepo.Reset()
	playerMessageRepo.Reset()
}

// TestProcessScheduledCampaigns_ErrorCase 測試錯誤情況
func TestProcessScheduledCampaigns_ErrorCase(t *testing.T) {
	// 設定測試環境
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	logger := helper.NewMockLogger()

	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		logger:            logger,
	}

	// 設定Mock返回錯誤
	campaignRepo.On("FindScheduledCampaigns", mocks.ContextMatcher()).
		Return(nil, errors.New("database error")).Once()

	// 執行測試
	ctx := context.Background()
	err := useCase.ProcessScheduledCampaigns(ctx)

	// 驗證結果
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find scheduled campaigns")
	campaignRepo.AssertExpectations()

	// 清理
	campaignRepo.Reset()
	playerRepo.Reset()
	playerMessageRepo.Reset()
}

// TestSendCampaignToPlayersAsync_Simple 簡單測試異步發送
func TestSendCampaignToPlayersAsync_Simple(t *testing.T) {
	// 設定測試環境
	factory := factories.NewTestDataFactory()
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	logger := helper.NewMockLogger()

	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		logger:            logger,
	}

	// 使用工廠創建測試數據
	campaign := factory.CreateScheduledCampaign()

	// 設定Mock期望
	campaignRepo.On("FindByID", mocks.ContextMatcher(), campaign.ID).
		Return(campaign, nil).Once()

	// 模擬空玩家列表
	playerRepo.On("FindByTargetType", mocks.ContextMatcher(), campaign.Target, 0, 5000).
		Return([]*entity.Player{}, nil).Once()

	// 更新統計和狀態
	campaignRepo.On("UpdateSentCount", mocks.ContextMatcher(), campaign.ID, int64(0)).
		Return(nil).Once()
	campaignRepo.On("UpdateStatus", mocks.ContextMatcher(), campaign.ID, consts.MessageCampaignStatusSent).
		Return(nil).
		Once()

	// 執行測試
	ctx := context.Background()
	err := useCase.SendCampaignToPlayersAsync(ctx, campaign.ID)

	// 驗證結果
	require.NoError(t, err)
	campaignRepo.AssertExpectations()
	playerRepo.AssertExpectations()

	// 清理
	campaignRepo.Reset()
	playerRepo.Reset()
	playerMessageRepo.Reset()
}

// TestSendCampaignToPlayersAsync_WithPlayers 測試有玩家的情況
func TestSendCampaignToPlayersAsync_WithPlayers(t *testing.T) {
	// 設定測試環境
	factory := factories.NewTestDataFactory()
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	logger := helper.NewMockLogger()

	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		logger:            logger,
	}

	// 使用工廠創建測試數據
	campaign := factory.CreateScheduledCampaign()
	players := factory.CreateMultiplePlayers(100)

	// 設定Mock期望
	campaignRepo.On("FindByID", mocks.ContextMatcher(), campaign.ID).
		Return(campaign, nil).Once()

	// 模擬分頁查詢 - 第一次查詢返回所有玩家
	playerRepo.On("FindByTargetType", mocks.ContextMatcher(), campaign.Target, 0, 5000).
		Return(players, nil).Once()
	// 第二次查詢返回空（表示沒有更多數據）- 只有當第一批數據達到限制時才會調用
	playerRepo.On("FindByTargetType", mocks.ContextMatcher(), campaign.Target, 100, 5000).
		Return([]*entity.Player{}, nil).Maybe()

	// 模擬批次消息檢查 - 創建結果映射
	expectedResult := make(map[uint64]bool)
	for _, player := range players {
		expectedResult[player.ID] = false // 都不存在，需要創建
	}
	playerMessageRepo.On("CheckMessageExistsBatch", mocks.ContextMatcher(),
		mocks.AnySlice(), campaign.ID).Return(expectedResult, nil).Once()

	playerMessageRepo.On("CreateBatchOptimized", mocks.ContextMatcher(),
		mocks.AnySlice(), mocks.AnyInt()).Return(nil).Once()

	// 更新統計和狀態 - 使用 mocks.AnyInt64 因為實際發送數量可能為0（如果所有消息都已存在）
	campaignRepo.On("UpdateSentCount", mocks.ContextMatcher(), campaign.ID, mocks.AnyInt64()).
		Return(nil).Once()
	campaignRepo.On("UpdateStatus", mocks.ContextMatcher(), campaign.ID, consts.MessageCampaignStatusSent).
		Return(nil).
		Once()

	// 執行測試
	ctx := context.Background()
	err := useCase.SendCampaignToPlayersAsync(ctx, campaign.ID)

	// 驗證結果
	require.NoError(t, err)
	campaignRepo.AssertExpectations()
	playerRepo.AssertExpectations()
	playerMessageRepo.AssertExpectations()

	// 清理
	campaignRepo.Reset()
	playerRepo.Reset()
	playerMessageRepo.Reset()
}
