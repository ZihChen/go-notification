package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/valueobject"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createTestAgentCampaign() *entity.AgentCampaign {
	return &entity.AgentCampaign{
		ID:         1,
		Title:      "Test Campaign",
		Content:    "Test Content",
		Status:     "draft",
		MerchantID: 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func createTestAgent() *entity.Agent {
	return &entity.Agent{
		ID:            1,
		GlobalAgentID: "AGENT-001",
		MerchantID:    1,
		Account:       "test-account",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestAgentUseCase_CreateAgentCampaign(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	request := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: "MERCHANT-001",
		Title:            "Test Campaign",
		Content:          "Test Content",
		Status:           "draft",
		TargetType:       "all",
		CreatedBy:        "test-user",
	}

	expectedCampaign := createTestAgentCampaign()
	merchant := &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "MERCHANT-001",
		Name:             "Test Merchant",
	}

	// 設置mock期望
	merchantRepo.On("FindByGlobalID", mock.Anything, request.GlobalMerchantID).
		Return(merchant, nil)
	agentCampaignRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.AgentCampaign")).
		Return(expectedCampaign, nil)

	// 執行測試
	result, err := useCase.CreateAgentCampaign(context.Background(), request)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedCampaign.ID, result.ID)
	assert.Equal(t, expectedCampaign.Title, result.Title)
	assert.Equal(t, expectedCampaign.Content, result.Content)
	assert.Equal(t, expectedCampaign.Status, result.Status)

	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_GetAgentCampaign(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	campaignID := uint64(1)
	expectedCampaign := createTestAgentCampaign()

	// 設置mock期望
	agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
		Return(expectedCampaign, nil)

	// 執行測試
	result, err := useCase.GetAgentCampaign(context.Background(), campaignID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedCampaign.ID, result.ID)
	assert.Equal(t, expectedCampaign.Title, result.Title)
	assert.Equal(t, expectedCampaign.Content, result.Content)

	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_UpdateAgentCampaign(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	newTitle := "Updated Campaign"
	newContent := "Updated Content"
	request := &dto.UpdateAgentCampaignRequest{
		ID:        1,
		Title:     &newTitle,
		Content:   &newContent,
		UpdatedBy: "test-user",
	}

	existingCampaign := createTestAgentCampaign()
	mockMutex := mocks.NewDistributedMutexMock(t)

	// 設置mock期望
	agentCampaignRepo.On("GetByID", mock.Anything, request.ID).
		Return(existingCampaign, nil)
	agentCampaignRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.AgentCampaign")).
		Return(nil)
	distributedLockMgr.On("GetLockWithOptions", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("infrastructure.LockOptions")).
		Return(mockMutex, nil)
	mockMutex.On("TryLock").Return(nil)
	mockMutex.On("Unlock").Return(true, nil)

	// 執行測試
	result, err := useCase.UpdateAgentCampaign(context.Background(), request)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)

	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_DeleteAgentCampaign(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	t.Run("delete draft campaign successfully", func(t *testing.T) {
		// 準備測試數據
		campaignID := uint64(1)
		draftCampaign := createTestAgentCampaign()
		draftCampaign.Status = "draft"

		// 重置mock
		agentCampaignRepo.Mock = mock.Mock{}
		agentMessageRepo.Mock = mock.Mock{}
		// Logger mock 已經在 NewMockLogger() 中設置為 .Maybe()，無需重置

		// 設置mock期望
		agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
			Return(draftCampaign, nil)
		agentCampaignRepo.On("UpdateFields", mock.Anything, campaignID, mock.Anything).
			Return(nil)
		agentMessageRepo.On("DeleteByCampaignID", mock.Anything, campaignID).
			Return(nil)
		agentCampaignRepo.On("Delete", mock.Anything, campaignID, "test_user").
			Return(nil)

		// 執行測試
		err := useCase.DeleteAgentCampaign(context.Background(), campaignID, "test_user")

		// 驗證結果
		assert.NoError(t, err)

		// 等待異步 goroutine 完成
		time.Sleep(100 * time.Millisecond)

		agentCampaignRepo.AssertExpectations()
		agentMessageRepo.AssertExpectations()
	})

	t.Run("delete active campaign successfully", func(t *testing.T) {
		// 準備測試數據
		campaignID := uint64(2)
		activeCampaign := createTestAgentCampaign()
		activeCampaign.Status = "active"

		// 重置mock
		agentCampaignRepo.Mock = mock.Mock{}
		agentMessageRepo.Mock = mock.Mock{}
		// Logger mock 已經在 NewMockLogger() 中設置為 .Maybe()，無需重置

		// 設置mock期望
		agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
			Return(activeCampaign, nil)
		agentCampaignRepo.On("UpdateFields", mock.Anything, campaignID, mock.Anything).
			Return(nil)
		agentMessageRepo.On("DeleteByCampaignID", mock.Anything, campaignID).
			Return(nil)
		agentCampaignRepo.On("Delete", mock.Anything, campaignID, "test_user").
			Return(nil)

		// 執行測試
		err := useCase.DeleteAgentCampaign(context.Background(), campaignID, "test_user")

		// 驗證結果
		assert.NoError(t, err)

		// 等待異步 goroutine 完成
		time.Sleep(100 * time.Millisecond)

		agentCampaignRepo.AssertExpectations()
		agentMessageRepo.AssertExpectations()
	})
}

func TestAgentUseCase_GetAgentCampaigns(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	query := &dto.AgentCampaignsQuery{
		GlobalMerchantID: "MERCHANT-001",
		Page:             1,
		PageSize:         10,
	}

	merchant := &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "MERCHANT-001",
		Name:             "Test Merchant",
	}

	campaigns := []*entity.AgentCampaign{
		createTestAgentCampaign(),
	}
	total := 1

	// 設置mock期望
	merchantRepo.On("FindByGlobalID", mock.Anything, query.GlobalMerchantID).
		Return(merchant, nil)
	agentCampaignRepo.On("List", mock.Anything, mock.AnythingOfType("*valueobject.AgentCampaignsQuery")).
		Return(campaigns, total, nil)

	// 執行測試
	result, err := useCase.GetAgentCampaigns(context.Background(), query)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Campaigns, 1)
	assert.Equal(t, 1, result.Total)

	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_GetAgentMessages(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	query := &dto.AgentMessagesQuery{
		GlobalAgentID: "AGENT-001",
		Page:          1,
		PageSize:      10,
	}

	agent := createTestAgent()
	messages := []*entity.AgentMessage{
		{
			ID:              1,
			AgentCampaignID: 1,
			AgentID:         1,
			IsRead:          false,
			CampaignTitle:   "Test Campaign",
			CampaignContent: "Test Content",
			GlobalAgentID:   "AGENT-001",
		},
	}
	total := 1

	stats := &valueobject.AgentMessageStats{
		TotalCount:  1,
		ReadCount:   0,
		UnreadCount: 1,
	}

	// 設置mock期望
	agentRepo.On("GetByGlobalID", mock.Anything, query.GlobalAgentID).
		Return(agent, nil)
	agentMessageRepo.On("ListByAgent", mock.Anything, mock.AnythingOfType("*valueobject.AgentMessagesQuery")).
		Return(messages, total, nil)
	agentMessageRepo.On("GetMessageStats", mock.Anything, agent.ID).
		Return(stats, nil)

	// 執行測試
	result, err := useCase.GetAgentMessages(context.Background(), query)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "Test Campaign", result.Messages[0].Title)

	agentRepo.AssertExpectations()
	agentMessageRepo.AssertExpectations()
}

func TestAgentUseCase_MarkMessageAsRead(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	messageID := uint64(1)
	agentID := uint64(1)

	// 設置mock期望
	agentMessageRepo.On("MarkAsRead", mock.Anything, messageID, agentID).
		Return(nil)

	// 執行測試
	err := useCase.MarkMessageAsRead(context.Background(), messageID, agentID)

	// 驗證結果
	assert.NoError(t, err)
	agentMessageRepo.AssertExpectations()
}

func TestAgentUseCase_GetAgentByGlobalID(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	globalAgentID := "AGENT-001"
	expectedAgent := createTestAgent()

	// 設置mock期望
	agentRepo.On("GetByGlobalID", mock.Anything, globalAgentID).
		Return(expectedAgent, nil)

	// 執行測試
	result, err := useCase.GetAgentByGlobalID(context.Background(), globalAgentID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedAgent.ID, result.ID)
	assert.Equal(t, expectedAgent.GlobalAgentID, result.GlobalAgentID)
	assert.Equal(t, expectedAgent.Account, result.Account)

	agentRepo.AssertExpectations()
}

func TestAgentUseCase_GetActiveAgents(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	merchantID := uint64(1)
	limit := 10
	offset := 0

	agents := []*entity.Agent{
		createTestAgent(),
	}

	// 設置mock期望
	agentRepo.On("FindAgents", mock.Anything, merchantID, limit, offset).
		Return(agents, nil)

	// 執行測試
	result, err := useCase.GetActiveAgents(context.Background(), merchantID, limit, offset)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)

	agentRepo.AssertExpectations()
}

// Error test cases
func TestAgentUseCase_GetAgentCampaign_NotFound(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	campaignID := uint64(999)
	expectedError := errors.New("campaign not found")

	// 設置mock期望
	agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
		Return(nil, expectedError)

	// 執行測試
	result, err := useCase.GetAgentCampaign(context.Background(), campaignID)

	// 驗證結果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get agent campaign")

	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_CreateAgentCampaign_RepositoryError(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據
	request := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: "MERCHANT-001",
		Title:            "Test Campaign",
		Content:          "Test Content",
		Status:           "draft",
		TargetType:       "all",
		CreatedBy:        "test-user",
	}

	merchant := &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "MERCHANT-001",
		Name:             "Test Merchant",
	}

	expectedError := errors.New("database error")

	// 設置mock期望
	merchantRepo.On("FindByGlobalID", mock.Anything, request.GlobalMerchantID).
		Return(merchant, nil)
	agentCampaignRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.AgentCampaign")).
		Return(nil, expectedError)

	// 執行測試
	result, err := useCase.CreateAgentCampaign(context.Background(), request)

	// 驗證結果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create agent campaign")

	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
}

func TestAgentUseCase_BackfillMissedMessages(t *testing.T) {
	// 創建 mocks
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例 (使用具體類型以訪問private方法)
	tracingService.SetupSuccess()
	useCase := &AgentUseCase{
		agentRepo:         agentRepo,
		agentCampaignRepo: agentCampaignRepo,
		agentMessageRepo:  agentMessageRepo,
		agentRelationRepo: agentRelationshipRepo,
		merchantRepo:      merchantRepo,
		agentService:      agentService,
		eventProducer:     eventProducer,
		logger:            logger,
		tracingService:    tracingService,
	}

	ctx := context.Background()
	agentEvent := &event.AgentSyncEvent{
		GlobalAgentID:    "agent123",
		GlobalMerchantID: "merchant123",
		Account:          "test@example.com",
		CurrentSignInAt:  func() *time.Time { t := time.Now().AddDate(0, -2, 0); return &t }(), // 2個月前
	}

	// Mock 代理查詢
	agent := &entity.Agent{
		ID:              1,
		MerchantID:      1,
		GlobalAgentID:   "agent123",
		Account:         "test@example.com",
		CurrentSignInAt: agentEvent.CurrentSignInAt,
	}
	agentRepo.On("GetByGlobalID", ctx, "agent123").Return(agent, nil)

	// Mock 商戶查詢
	merchant := &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "merchant123",
		Name:             "Test Merchant",
	}
	merchantRepo.On("FindByGlobalID", ctx, "merchant123").Return(merchant, nil)

	// Mock 分頁活動查詢
	campaigns := []*entity.AgentCampaign{
		{
			ID:         1,
			MerchantID: 1,
			Title:      "Test Campaign",
			Content:    "Test Content",
			Status:     "sent",
			TargetType: "all",
			CreatedAt:  time.Now().AddDate(0, 0, -1),
		},
	}
	// 第一次查詢返回活動（少於批次大小，會觸發退出）
	targetTypes := []string{"all", "specific", "line"}
	agentCampaignRepo.On("FindSentCampaignsForBackfillPaginated", ctx, uint64(1), targetTypes, 100, 0).
		Return(campaigns, nil)

	// Mock 批次檢查訊息不存在
	existsMap := map[uint64]bool{1: false} // campaign ID 1 不存在
	agentMessageRepo.On("CheckCampaignMessageExistsBatch", ctx, uint64(1), []uint64{1}).
		Return(existsMap, nil)

	// Mock 批次創建訊息
	agentMessageRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*entity.AgentMessage")).
		Return(nil)

	// Mock 更新活動統計
	agentCampaignRepo.On("UpdateFields", ctx, uint64(1), mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// 執行測試
	err := useCase.BackfillMissedMessages(ctx, agentEvent)

	// 驗證結果
	assert.NoError(t, err)
	agentRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
	agentMessageRepo.AssertExpectations()
}

func TestAgentUseCase_BackfillMissedMessages_SpecificTargetType(t *testing.T) {
	// 創建 mocks
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	useCase := &AgentUseCase{
		agentRepo:         agentRepo,
		agentCampaignRepo: agentCampaignRepo,
		agentMessageRepo:  agentMessageRepo,
		agentRelationRepo: agentRelationshipRepo,
		merchantRepo:      merchantRepo,
		agentService:      agentService,
		eventProducer:     eventProducer,
		logger:            logger,
		tracingService:    tracingService,
	}

	ctx := context.Background()
	agentEvent := &event.AgentSyncEvent{
		GlobalAgentID:    "test-agent-123",
		GlobalMerchantID: "test-merchant-456",
		Account:          "test-account",
	}

	// Mock 代理信息
	agent := &entity.Agent{
		ID:      1,
		Account: "test-account",
	}
	agentRepo.On("GetByGlobalID", ctx, agentEvent.GlobalAgentID).Return(agent, nil)

	// Mock 商戶信息
	merchant := &entity.Merchant{ID: 1}
	merchantRepo.On("FindByGlobalID", ctx, agentEvent.GlobalMerchantID).Return(merchant, nil)

	// 創建 specific 類型的活動，包含目標代理帳號
	campaigns := []*entity.AgentCampaign{
		{
			ID:            1,
			TargetType:    "specific",
			TargetDetails: []string{"test-account", "other-account"}, // 包含目標代理
		},
		{
			ID:            2,
			TargetType:    "specific",
			TargetDetails: []string{"other-account"}, // 不包含目標代理
		},
	}

	targetTypes := []string{"all", "specific", "line"}
	agentCampaignRepo.On("FindSentCampaignsForBackfillPaginated", ctx, uint64(1), targetTypes, 100, 0).
		Return(campaigns, nil)

	// Mock 批次檢查訊息不存在
	existsMap := map[uint64]bool{1: false, 2: false}
	agentMessageRepo.On("CheckCampaignMessageExistsBatch", ctx, uint64(1), []uint64{1, 2}).
		Return(existsMap, nil)

	// 預期只為第一個活動創建訊息（因為代理在目標列表中）
	expectedMessages := []*entity.AgentMessage{
		{
			AgentCampaignID: 1,
			AgentID:         1,
			IsRead:          false,
		},
	}
	agentMessageRepo.On("CreateBatch", ctx, mock.MatchedBy(func(messages []*entity.AgentMessage) bool {
		if len(messages) != 1 {
			return false
		}
		return messages[0].AgentCampaignID == expectedMessages[0].AgentCampaignID &&
			messages[0].AgentID == expectedMessages[0].AgentID &&
			messages[0].IsRead == expectedMessages[0].IsRead
	})).
		Return(nil)

	// Mock 更新活動統計
	agentCampaignRepo.On("UpdateFields", ctx, uint64(1), mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// 執行測試
	err := useCase.BackfillMissedMessages(ctx, agentEvent)

	// 驗證結果
	assert.NoError(t, err)
	agentRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
	agentMessageRepo.AssertExpectations()
}

func TestAgentUseCase_BackfillMissedMessages_LineTargetType(t *testing.T) {
	// 創建 mocks
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	useCase := &AgentUseCase{
		agentRepo:         agentRepo,
		agentCampaignRepo: agentCampaignRepo,
		agentMessageRepo:  agentMessageRepo,
		agentRelationRepo: agentRelationshipRepo,
		merchantRepo:      merchantRepo,
		agentService:      agentService,
		eventProducer:     eventProducer,
		logger:            logger,
		tracingService:    tracingService,
	}

	ctx := context.Background()
	agentEvent := &event.AgentSyncEvent{
		GlobalAgentID:    "test-agent-123",
		GlobalMerchantID: "test-merchant-456",
		Account:          "child-account",
	}

	// Mock 代理信息 (下級代理，包含 Ancestry)
	childAgent := &entity.Agent{
		ID:       2,
		Account:  "child-account",
		Ancestry: "top-agent/parent-account", // 包含父代理的 Ancestry
	}
	agentRepo.On("GetByGlobalID", ctx, agentEvent.GlobalAgentID).Return(childAgent, nil)

	// Mock 商戶信息
	merchant := &entity.Merchant{ID: 1}
	merchantRepo.On("FindByGlobalID", ctx, agentEvent.GlobalMerchantID).Return(merchant, nil)

	// Mock 父代理信息
	parentAgent := &entity.Agent{
		ID:            1,
		Account:       "parent-account",
		GlobalAgentID: "parent-account", // 需要 GlobalAgentID 進行字串比對
	}
	agentRepo.On("GetByAccount", ctx, uint64(1), "parent-account").Return(parentAgent, nil)

	// 創建 line 類型的活動
	campaigns := []*entity.AgentCampaign{
		{
			ID:            1,
			MerchantID:    1,
			TargetType:    "line",
			TargetDetails: []string{"parent-account"}, // 父代理帳號
		},
	}

	targetTypes := []string{"all", "specific", "line"}
	agentCampaignRepo.On("FindSentCampaignsForBackfillPaginated", ctx, uint64(1), targetTypes, 100, 0).
		Return(campaigns, nil)

	// 不再需要 Mock 代理關係查詢，改為使用 Ancestry 字串比對

	// Mock 批次檢查訊息不存在
	existsMap := map[uint64]bool{1: false}
	agentMessageRepo.On("CheckCampaignMessageExistsBatch", ctx, uint64(2), []uint64{1}).
		Return(existsMap, nil)

	// 預期為 line 活動創建訊息（因為代理是指定父代理的下級）
	expectedMessages := []*entity.AgentMessage{
		{
			AgentCampaignID: 1,
			AgentID:         2,
			IsRead:          false,
		},
	}
	agentMessageRepo.On("CreateBatch", ctx, mock.MatchedBy(func(messages []*entity.AgentMessage) bool {
		if len(messages) != 1 {
			return false
		}
		return messages[0].AgentCampaignID == expectedMessages[0].AgentCampaignID &&
			messages[0].AgentID == expectedMessages[0].AgentID &&
			messages[0].IsRead == expectedMessages[0].IsRead
	})).
		Return(nil)

	// Mock 更新活動統計
	agentCampaignRepo.On("UpdateFields", ctx, uint64(1), mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// 執行測試
	err := useCase.BackfillMissedMessages(ctx, agentEvent)

	// 驗證結果
	assert.NoError(t, err)
	agentRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
	agentMessageRepo.AssertExpectations()
	// agentRelationshipRepo 不再需要驗證，因為改用 Ancestry 字串比對
}

func TestAgentUseCase_BatchDeleteAgentCampaigns(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// Note: BatchHardDeleteByCampaignIDs expectations are set per-test due to async goroutines

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	testCases := []struct {
		name        string
		request     *dto.BatchDeleteAgentCampaignsRequest
		setupMock   func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful batch delete",
			request: &dto.BatchDeleteAgentCampaignsRequest{
				IDs:       []uint64{1, 2, 3},
				UpdatedBy: "test_user",
			},
			setupMock: func() {
				// Mock campaigns that can be deleted
				for i, id := range []uint64{1, 2, 3} {
					campaign := createTestAgentCampaign()
					campaign.ID = id
					campaign.Status = consts.AgentCampaignStatusDraft // Can be deleted
					campaign.Title = fmt.Sprintf("Campaign %d", i+1)

					agentCampaignRepo.On("GetByID", mock.Anything, id).
						Return(campaign, nil)
				}

				// async call - use specific expectation for this test
				agentMessageRepo.On("BatchHardDeleteByCampaignIDs", mock.Anything, []uint64{1, 2, 3}).
					Return(nil)
				agentCampaignRepo.On("BatchDelete", mock.Anything, []uint64{1, 2, 3}, "test_user").
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "empty IDs array",
			request: &dto.BatchDeleteAgentCampaignsRequest{
				IDs:       []uint64{},
				UpdatedBy: "test_user",
			},
			setupMock: func() {
				// No mock setup needed
			},
			expectError: true,
			errorMsg:    "no campaign IDs provided",
		},
		{
			name: "some campaigns not found",
			request: &dto.BatchDeleteAgentCampaignsRequest{
				IDs:       []uint64{1, 999, 3},
				UpdatedBy: "test_user",
			},
			setupMock: func() {
				// Campaign 1 exists and can be deleted
				campaign1 := createTestAgentCampaign()
				campaign1.ID = 1
				campaign1.Status = consts.AgentCampaignStatusDraft
				agentCampaignRepo.On("GetByID", mock.Anything, uint64(1)).
					Return(campaign1, nil)

				// Campaign 999 not found
				agentCampaignRepo.On("GetByID", mock.Anything, uint64(999)).
					Return(nil, errors.New("not found"))

				// Campaign 3 exists and can be deleted
				campaign3 := createTestAgentCampaign()
				campaign3.ID = 3
				campaign3.Status = consts.AgentCampaignStatusDraft
				agentCampaignRepo.On("GetByID", mock.Anything, uint64(3)).
					Return(campaign3, nil)

				// Only valid campaigns are deleted
				agentMessageRepo.On("BatchHardDeleteByCampaignIDs", mock.Anything, []uint64{1, 3}).
					Return(nil)
				agentCampaignRepo.On("BatchDelete", mock.Anything, []uint64{1, 3}, "test_user").
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "delete sent campaigns successfully",
			request: &dto.BatchDeleteAgentCampaignsRequest{
				IDs:       []uint64{1, 2},
				UpdatedBy: "test_user",
			},
			setupMock: func() {
				// Both campaigns exist and can be deleted (even sent status)
				for _, id := range []uint64{1, 2} {
					campaign := createTestAgentCampaign()
					campaign.ID = id
					campaign.Status = consts.AgentCampaignStatusSent // Can now be deleted

					agentCampaignRepo.On("GetByID", mock.Anything, id).
						Return(campaign, nil)
				}
				// BatchDelete call expected for all valid campaigns
				// async call - use specific expectation for this test
				agentMessageRepo.On("BatchHardDeleteByCampaignIDs", mock.Anything, []uint64{1, 2}).
					Return(nil)
				agentCampaignRepo.On("BatchDelete", mock.Anything, []uint64{1, 2}, "test_user").
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "repository batch delete fails",
			request: &dto.BatchDeleteAgentCampaignsRequest{
				IDs:       []uint64{1},
				UpdatedBy: "test_user",
			},
			setupMock: func() {
				campaign := createTestAgentCampaign()
				campaign.ID = 1
				campaign.Status = consts.AgentCampaignStatusDraft

				agentCampaignRepo.On("GetByID", mock.Anything, uint64(1)).
					Return(campaign, nil)
				// async call - use specific expectation for this test
				agentMessageRepo.On("BatchHardDeleteByCampaignIDs", mock.Anything, []uint64{1}).
					Return(nil)
				agentCampaignRepo.On("BatchDelete", mock.Anything, []uint64{1}, "test_user").
					Return(errors.New("database error"))
			},
			expectError: true,
			errorMsg:    "batch delete agent campaigns failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 重置mock以避免測試間干擾 - 但保留agentMessageRepo的expectations因為async goroutines可能仍在執行
			agentCampaignRepo.ExpectedCalls = agentCampaignRepo.ExpectedCalls[:0]
			agentCampaignRepo.Calls = agentCampaignRepo.Calls[:0]
			// 不重置agentMessageRepo期望，因為異步goroutines可能仍在執行BatchHardDeleteByCampaignIDs

			tc.setupMock()

			err := useCase.BatchDeleteAgentCampaigns(context.Background(), tc.request)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// 驗證所有mock期望
			agentCampaignRepo.AssertExpectations()
		})
	}
}

func TestAgentUseCase_BackfillMissedMessages_LineTargetType_NotInAncestry(t *testing.T) {
	// 創建 mocks
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	useCase := &AgentUseCase{
		agentRepo:         agentRepo,
		agentCampaignRepo: agentCampaignRepo,
		agentMessageRepo:  agentMessageRepo,
		agentRelationRepo: agentRelationshipRepo,
		merchantRepo:      merchantRepo,
		agentService:      agentService,
		eventProducer:     eventProducer,
		logger:            logger,
		tracingService:    tracingService,
	}

	ctx := context.Background()
	agentEvent := &event.AgentSyncEvent{
		GlobalAgentID:    "test-agent-123",
		GlobalMerchantID: "test-merchant-456",
		Account:          "child-account",
	}

	// Mock 代理信息 (Ancestry 中不包含目標父代理)
	childAgent := &entity.Agent{
		ID:       2,
		Account:  "child-account",
		Ancestry: "top-agent/other-parent", // 不包含 "parent-account"
	}
	agentRepo.On("GetByGlobalID", ctx, agentEvent.GlobalAgentID).Return(childAgent, nil)

	// Mock 商戶信息
	merchant := &entity.Merchant{ID: 1}
	merchantRepo.On("FindByGlobalID", ctx, agentEvent.GlobalMerchantID).Return(merchant, nil)

	// Mock 父代理信息
	parentAgent := &entity.Agent{
		ID:            1,
		Account:       "parent-account",
		GlobalAgentID: "parent-account",
	}
	agentRepo.On("GetByAccount", ctx, uint64(1), "parent-account").Return(parentAgent, nil)

	// 創建 line 類型的活動
	campaigns := []*entity.AgentCampaign{
		{
			ID:            1,
			MerchantID:    1,
			TargetType:    "line",
			TargetDetails: []string{"parent-account"},
		},
	}

	targetTypes := []string{"all", "specific", "line"}
	agentCampaignRepo.On("FindSentCampaignsForBackfillPaginated", ctx, uint64(1), targetTypes, 100, 0).
		Return(campaigns, nil)

	// Mock 批次檢查訊息不存在
	existsMap := map[uint64]bool{1: false}
	agentMessageRepo.On("CheckCampaignMessageExistsBatch", ctx, uint64(2), []uint64{1}).
		Return(existsMap, nil)

	// 預期不會創建訊息（因為代理不在指定父代理的線下）
	// 因此不需要 mock CreateBatch

	// 執行測試
	err := useCase.BackfillMissedMessages(ctx, agentEvent)

	// 驗證結果
	assert.NoError(t, err)
	agentRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
	agentMessageRepo.AssertExpectations()
}

// 這個測試驗證創建活動的基本流程，使用draft狀態避免觸發異步處理
func TestAgentUseCase_CreateAgentCampaign_DraftStatus(t *testing.T) {
	// 創建模擬repositories
	agentRepo := mocks.NewAgentRepositoryMock(t)
	agentCampaignRepo := mocks.NewAgentCampaignRepositoryMock(t)
	agentMessageRepo := mocks.NewAgentMessageRepositoryMock(t)
	agentRelationshipRepo := mocks.NewAgentRelationshipRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)

	// 創建模擬服務
	agentService := mocks.NewAgentServiceMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	tracingService := mocks.NewTracingServiceMock(t)
	distributedLockMgr := mocks.NewDistributedLockManagerMock(t)

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
	distributedLockMgr.SetupSuccess()
	useCase := NewAgentUseCase(
		agentRepo,
		agentCampaignRepo,
		agentMessageRepo,
		agentRelationshipRepo,
		merchantRepo,
		agentService,
		eventProducer,
		logger,
		tracingService,
		distributedLockMgr,
	)

	// 準備測試數據 - 使用draft狀態避免觸發立即發送
	req := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: "test-merchant",
		Title:            "Draft Test Campaign",
		Content:          "Test Content",
		Status:           "draft", // 使用draft狀態，不觸發立即發送
		ScheduledAt:      nil,
		TargetType:       "all",
		CreatedBy:        "test-user",
	}

	// Mock 商戶
	merchant := &entity.Merchant{ID: 1}
	merchantRepo.On("FindByGlobalID", mock.Anything, "test-merchant").Return(merchant, nil)

	// Mock 活動創建
	expectedCampaign := &entity.AgentCampaign{
		ID:          1,
		Title:       "Draft Test Campaign",
		Content:     "Test Content",
		Status:      consts.AgentCampaignStatusDraft,
		ScheduledAt: nil,
		TargetType:  "all",
		MerchantID:  1,
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	agentCampaignRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.AgentCampaign")).
		Return(expectedCampaign, nil)

	// 記錄API響應時間
	start := time.Now()

	// 執行測試
	result, err := useCase.CreateAgentCampaign(context.Background(), req)

	// 計算響應時間
	responseTime := time.Since(start)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Draft Test Campaign", result.Title)
	assert.Equal(t, consts.AgentCampaignStatusDraft, result.Status)
	assert.Nil(t, result.ScheduledAt)

	// 驗證響應時間很快
	assert.Less(t, responseTime, 100*time.Millisecond, "API response should be fast")

	t.Logf("API response time: %v", responseTime)

	// 驗證基本 mock 調用
	merchantRepo.AssertExpectations()
	agentCampaignRepo.AssertExpectations()
}

// TestAgentCampaign_NewAgentCampaign_ImmediateSending 測試創建時的立即發送邏輯
func TestAgentCampaign_NewAgentCampaign_ImmediateSending(t *testing.T) {
	// 使用Value Object創建數據
	data := &valueobject.AgentCampaignCreationData{
		Title:       "Test Campaign",
		Content:     "Test content",
		Status:      "scheduled", // scheduled 狀態
		ScheduledAt: nil,         // nil 表示立即發送
		TargetType:  "all",
		CreatedBy:   "test-user",
	}

	campaign, err := entity.NewAgentCampaign(data, 1)

	assert.NoError(t, err)
	assert.NotNil(t, campaign)
	assert.Equal(t, consts.AgentCampaignStatusScheduled, campaign.Status)
	assert.NotNil(
		t,
		campaign.ScheduledAt,
		"ScheduledAt should be set to current time for immediate sending",
	)

	// 驗證 ScheduledAt 是當下時間（允許 1 秒誤差）
	timeDiff := time.Since(*campaign.ScheduledAt)
	assert.Less(t, timeDiff, time.Second, "ScheduledAt should be set to current time")
}

// TestAgentCampaign_UpdateFromData_ImmediateSending 測試更新時的立即發送邏輯
func TestAgentCampaign_UpdateFromData_ImmediateSending(t *testing.T) {
	// 創建一個現有的 draft 活動
	campaign := &entity.AgentCampaign{
		ID:          1,
		MerchantID:  1,
		Title:       "Existing Campaign",
		Content:     "Existing content",
		Status:      consts.AgentCampaignStatusDraft,
		ScheduledAt: nil, // draft 狀態沒有排程時間
		TargetType:  "all",
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
	}

	// 使用Value Object進行更新：改為 scheduled 且不提供 ScheduledAt（立即發送）
	updateData := &valueobject.AgentCampaignUpdateData{
		Status:    "scheduled",
		UpdatedBy: "test-user",
	}

	err := campaign.UpdateFromData(updateData)

	assert.NoError(t, err)
	assert.Equal(t, consts.AgentCampaignStatusScheduled, campaign.Status)
	assert.NotNil(
		t,
		campaign.ScheduledAt,
		"ScheduledAt should be set to current time for immediate sending",
	)

	// 驗證 ScheduledAt 是當下時間（允許 1 秒誤差）
	timeDiff := time.Since(*campaign.ScheduledAt)
	assert.Less(t, timeDiff, time.Second, "ScheduledAt should be set to current time")
}

// TestAgentCampaign_UpdateFromData_ScheduledWithTime 測試更新時提供具體排程時間
func TestAgentCampaign_UpdateFromData_ScheduledWithTime(t *testing.T) {
	// 創建一個現有的 draft 活動
	campaign := &entity.AgentCampaign{
		ID:          1,
		MerchantID:  1,
		Title:       "Existing Campaign",
		Content:     "Existing content",
		Status:      consts.AgentCampaignStatusDraft,
		ScheduledAt: nil,
		TargetType:  "all",
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
	}

	// 指定的排程時間（未來時間）
	futureTime := time.Now().Add(time.Hour)

	// 使用Value Object進行更新：改為 scheduled 並提供具體的 ScheduledAt
	updateData := &valueobject.AgentCampaignUpdateData{
		Status:      "scheduled",
		ScheduledAt: &futureTime,
		UpdatedBy:   "test-user",
	}

	err := campaign.UpdateFromData(updateData)

	assert.NoError(t, err)
	assert.Equal(t, consts.AgentCampaignStatusScheduled, campaign.Status)
	assert.Equal(
		t,
		futureTime,
		*campaign.ScheduledAt,
		"ScheduledAt should be set to the provided time",
	)
}
