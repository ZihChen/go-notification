package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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
	)

	// 準備測試數據
	request := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: "MERCHANT-001",
		Title:            "Test Campaign",
		Content:          "Test Content",
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 設置mock期望
	agentCampaignRepo.On("GetByID", mock.Anything, request.ID).
		Return(existingCampaign, nil)
	agentCampaignRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.AgentCampaign")).
		Return(nil)

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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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
	)

	t.Run("delete draft campaign successfully", func(t *testing.T) {
		// 準備測試數據
		campaignID := uint64(1)
		draftCampaign := createTestAgentCampaign()
		draftCampaign.Status = "draft"

		// 重置mock
		agentCampaignRepo.Mock = mock.Mock{}

		// 設置mock期望
		agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
			Return(draftCampaign, nil)
		agentCampaignRepo.On("UpdateFields", mock.Anything, campaignID, mock.Anything).
			Return(nil)
		agentCampaignRepo.On("Delete", mock.Anything, campaignID).
			Return(nil)

		// 執行測試
		err := useCase.DeleteAgentCampaign(context.Background(), campaignID)

		// 驗證結果
		assert.NoError(t, err)
		agentCampaignRepo.AssertExpectations()
	})

	t.Run("cannot delete active campaign", func(t *testing.T) {
		// 準備測試數據
		campaignID := uint64(2)
		activeCampaign := createTestAgentCampaign()
		activeCampaign.Status = "active"

		// 重置mock
		agentCampaignRepo.Mock = mock.Mock{}

		// 設置mock期望
		agentCampaignRepo.On("GetByID", mock.Anything, campaignID).
			Return(activeCampaign, nil)

		// 執行測試
		err := useCase.DeleteAgentCampaign(context.Background(), campaignID)

		// 驗證結果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete campaign with status")
		agentCampaignRepo.AssertExpectations()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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
	agentCampaignRepo.On("List", mock.Anything, mock.AnythingOfType("*dto.AgentCampaignsQuery")).
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	stats := &dto.AgentMessageStats{
		TotalCount:  1,
		ReadCount:   0,
		UnreadCount: 1,
	}

	// 設置mock期望
	agentRepo.On("GetByGlobalID", mock.Anything, query.GlobalAgentID).
		Return(agent, nil)
	agentMessageRepo.On("ListByAgent", mock.Anything, mock.AnythingOfType("*dto.AgentMessagesQuery")).
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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

	// 創建記錄器
	logger := helper.NewMockLogger()

	// 創建用例
	tracingService.SetupSuccess()
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
	)

	// 準備測試數據
	request := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: "MERCHANT-001",
		Title:            "Test Campaign",
		Content:          "Test Content",
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
