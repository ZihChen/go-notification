package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAgentCampaignTriggerJob_Execute(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func(mocks *AgentJobMocks)
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "成功處理單個排程活動",
			setupMocks: func(mocks *AgentJobMocks) {
				// Mock tracing service
				mocks.tracingService.SetupSuccess()
				
				// Mock GetScheduledCampaigns
				campaigns := []*entity.AgentCampaign{
					{
						ID:          1,
						MerchantID:  100,
						Title:       "Test Agent Campaign",
						Content:     "Test Agent Content",
						Status:      consts.AgentCampaignStatusScheduled,
						TargetType:  consts.AgentTargetTypeAll,
						ScheduledAt: &time.Time{},
					},
				}
				mocks.agentUseCase.On("GetScheduledCampaigns", mock.Anything, mock.AnythingOfType("time.Time")).
					Return(campaigns, nil)

				// Mock UpdateCampaignStatus to sending
				mocks.agentUseCase.On("UpdateCampaignStatus", mock.Anything, uint64(1), consts.AgentCampaignStatusSending).
					Return(nil)

				// Mock SendMessageToCampaignTargets
				mocks.agentUseCase.On("SendMessageToCampaignTargets", mock.Anything, campaigns[0]).
					Return(10, 8, nil) // 10 targets, 8 sent

				// Mock CompleteCampaign
				mocks.agentUseCase.On("CompleteCampaign", mock.Anything, uint64(1), 10, 8).
					Return(nil)

				// Mock distributed lock
				mocks.distributedLockMgr.On("GetLockWithOptions", mock.Anything, "job:agent_campaigns:trigger", mock.Anything).
					Return(mocks.mutex, nil)
				mocks.mutex.On("TryLock").Return(nil)
				mocks.mutex.On("Unlock").Return(true, nil)
			},
			expectError: false,
		},
		{
			name: "沒有排程活動時正常返回",
			setupMocks: func(mocks *AgentJobMocks) {
				// Mock tracing service
				mocks.tracingService.SetupSuccess()
				
				mocks.agentUseCase.On("GetScheduledCampaigns", mock.Anything, mock.AnythingOfType("time.Time")).
					Return([]*entity.AgentCampaign{}, nil)

				// Mock distributed lock
				mocks.distributedLockMgr.On("GetLockWithOptions", mock.Anything, "job:agent_campaigns:trigger", mock.Anything).
					Return(mocks.mutex, nil)
				mocks.mutex.On("TryLock").Return(nil)
				mocks.mutex.On("Unlock").Return(true, nil)
			},
			expectError: false,
		},
		{
			name: "獲取排程活動失敗",
			setupMocks: func(mocks *AgentJobMocks) {
				// Mock tracing service
				mocks.tracingService.SetupSuccess()
				
				mocks.agentUseCase.On("GetScheduledCampaigns", mock.Anything, mock.AnythingOfType("time.Time")).
					Return(nil, errors.New("database error"))

				// Mock distributed lock
				mocks.distributedLockMgr.On("GetLockWithOptions", mock.Anything, "job:agent_campaigns:trigger", mock.Anything).
					Return(mocks.mutex, nil)
				mocks.mutex.On("TryLock").Return(nil)
				mocks.mutex.On("Unlock").Return(true, nil)
			},
			expectError:      true,
			expectedErrorMsg: "database error",
		},
		{
			name: "鎖已被其他實例占用時跳過執行",
			setupMocks: func(mocks *AgentJobMocks) {
				// Mock tracing service
				mocks.tracingService.SetupSuccess()
				
				mocks.distributedLockMgr.On("GetLockWithOptions", mock.Anything, "job:agent_campaigns:trigger", mock.Anything).
					Return(mocks.mutex, nil)
				mocks.mutex.On("TryLock").Return(errors.New("lock already acquired")) // 鎖已被占用
			},
			expectError: false,
		},
		{
			name: "發送訊息失敗但不中斷整個Job",
			setupMocks: func(mocks *AgentJobMocks) {
				// Mock tracing service
				mocks.tracingService.SetupSuccess()
				
				campaigns := []*entity.AgentCampaign{
					{
						ID:          1,
						MerchantID:  100,
						Title:       "Test Campaign",
						Content:     "Test Content",
						Status:      consts.AgentCampaignStatusScheduled,
						TargetType:  consts.AgentTargetTypeAll,
						ScheduledAt: &time.Time{},
					},
				}
				mocks.agentUseCase.On("GetScheduledCampaigns", mock.Anything, mock.AnythingOfType("time.Time")).
					Return(campaigns, nil)

				// Mock UpdateCampaignStatus to sending
				mocks.agentUseCase.On("UpdateCampaignStatus", mock.Anything, uint64(1), consts.AgentCampaignStatusSending).
					Return(nil)

				// Mock SendMessageToCampaignTargets failure
				mocks.agentUseCase.On("SendMessageToCampaignTargets", mock.Anything, campaigns[0]).
					Return(0, 0, errors.New("send failed"))

				// Mock UpdateCampaignStatus to failed
				mocks.agentUseCase.On("UpdateCampaignStatus", mock.Anything, uint64(1), consts.AgentCampaignStatusFailed).
					Return(nil)

				// Mock distributed lock
				mocks.distributedLockMgr.On("GetLockWithOptions", mock.Anything, "job:agent_campaigns:trigger", mock.Anything).
					Return(mocks.mutex, nil)
				mocks.mutex.On("TryLock").Return(nil)
				mocks.mutex.On("Unlock").Return(true, nil)
			},
			expectError: false, // Job 不會因為單個活動失敗而中斷
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := setupAgentJobMocks(t)
			tt.setupMocks(mocks)

			job := NewAgentCampaignTriggerJob(
				mocks.agentUseCase,
				mocks.logger,
				mocks.tracingService,
				mocks.distributedLockMgr,
			)

			ctx := context.Background()
			err := job.Execute(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// 驗證所有 mock 調用
			mocks.agentUseCase.AssertExpectations()
			mocks.distributedLockMgr.AssertExpectations()
			mocks.mutex.AssertExpectations()
		})
	}
}

func TestAgentCampaignTriggerJob_GetName(t *testing.T) {
	mocks := setupAgentJobMocks(t)
	job := NewAgentCampaignTriggerJob(
		mocks.agentUseCase,
		mocks.logger,
		mocks.tracingService,
		mocks.distributedLockMgr,
	)

	assert.Equal(t, "agent-campaign-trigger", job.GetName())
}

func TestAgentCampaignTriggerJob_GetCron(t *testing.T) {
	mocks := setupAgentJobMocks(t)
	job := NewAgentCampaignTriggerJob(
		mocks.agentUseCase,
		mocks.logger,
		mocks.tracingService,
		mocks.distributedLockMgr,
	)

	assert.Equal(t, "*/30 * * * * *", job.GetCron())
}

// AgentJobMocks 包含代理Job測試所需的所有 mock
type AgentJobMocks struct {
	agentUseCase       *mocks.AgentUseCaseMock
	logger             *helper.MockLogger
	tracingService     *mocks.TracingServiceMock
	distributedLockMgr *mocks.DistributedLockManagerMock
	mutex              *mocks.DistributedMutexMock
}

func setupAgentJobMocks(t *testing.T) *AgentJobMocks {
	return &AgentJobMocks{
		agentUseCase:       mocks.NewAgentUseCaseMock(t),
		logger:             helper.NewMockLogger(),
		tracingService:     mocks.NewTracingServiceMock(t),
		distributedLockMgr: mocks.NewDistributedLockManagerMock(t),
		mutex:              mocks.NewDistributedMutexMock(t),
	}
}