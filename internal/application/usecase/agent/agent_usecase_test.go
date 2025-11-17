package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
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
	agentRepo.On("GetByAccount", ctx, "parent-account").Return(parentAgent, nil)

	// 創建 line 類型的活動
	campaigns := []*entity.AgentCampaign{
		{
			ID:            1,
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
	agentRepo.On("GetByAccount", ctx, "parent-account").Return(parentAgent, nil)

	// 創建 line 類型的活動
	campaigns := []*entity.AgentCampaign{
		{
			ID:            1,
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
