package usecase

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFailedTaskEventUseCase_CreateFailedTaskEventWithRedisInfo(t *testing.T) {
	tests := []struct {
		name          string
		taskID        string
		taskType      string
		queueName     string
		payload       string
		errorMessage  string
		redisKey      string
		redisState    string
		retryCount    int
		setupMock     func(*mocks.MockFailedTaskEventRepository)
		expectedError bool
	}{
		{
			name:         "create failed task event successfully",
			taskID:       "task-123",
			taskType:     "merchant:sync",
			queueName:    "default",
			payload:      `{"merchant_id": "456"}`,
			errorMessage: "database connection failed",
			redisKey:     "asynq:default:t:task-123",
			redisState:   "failed",
			retryCount:   0,
			setupMock: func(mockRepo *mocks.MockFailedTaskEventRepository) {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.FailedTaskEvent")).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name:         "create failed task event with invalid data",
			taskID:       "", // invalid - empty taskID
			taskType:     "merchant:sync",
			queueName:    "default",
			payload:      `{"merchant_id": "456"}`,
			errorMessage: "database connection failed",
			redisKey:     "asynq:default:t:task-123",
			redisState:   "failed",
			retryCount:   0,
			setupMock: func(mockRepo *mocks.MockFailedTaskEventRepository) {
				// No expectations since validation should fail
			},
			expectedError: false, // Should not return error, just skip invalid data
		},
		{
			name:         "create failed task event with repository error",
			taskID:       "task-456",
			taskType:     "player:sync",
			queueName:    "default",
			payload:      `{"player_id": "789"}`,
			errorMessage: "timeout error",
			redisKey:     "asynq:default:t:task-456",
			redisState:   "failed",
			retryCount:   2,
			setupMock: func(mockRepo *mocks.MockFailedTaskEventRepository) {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.FailedTaskEvent")).
					Return(assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &mocks.MockFailedTaskEventRepository{}
			mockLogger := helper.NewMockLogger()

			tt.setupMock(mockRepo)

			// Create use case
			useCase := NewFailedTaskEventUseCase(mockRepo, mockLogger)

			// Execute
			err := useCase.CreateFailedTaskEventWithRedisInfo(
				context.Background(),
				tt.taskID,
				tt.taskType,
				tt.queueName,
				tt.payload,
				tt.errorMessage,
				tt.redisKey,
				tt.redisState,
				tt.retryCount,
			)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
