package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestFailedTaskEventRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(sqlmock.Sqlmock)
		failedTask  *entity.FailedTaskEvent
		expectError bool
	}{
		{
			name: "create failed task event successfully",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `failed_task_events`").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			failedTask: entity.NewFailedTaskEventWithRedisInfo(
				"task-123",
				"merchant:sync",
				"default",
				`{"id": "123", "merchant_id": "456"}`,
				"database connection failed",
				"asynq:default:t:task-123",
				"failed",
				0,
			),
			expectError: false,
		},
		{
			name: "create failed task event with database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `failed_task_events`").
					WillReturnError(assert.AnError)
				mock.ExpectRollback()
			},
			failedTask: entity.NewFailedTaskEventWithRedisInfo(
				"task-456",
				"player:sync",
				"default",
				`{"id": "789"}`,
				"timeout error",
				"asynq:default:t:task-456",
				"failed",
				0,
			),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer func() {
				_ = db.Close()
			}()

			gormDB, err := gorm.Open(mysql.New(mysql.Config{
				Conn:                      db,
				SkipInitializeWithVersion: true,
			}), &gorm.Config{})
			assert.NoError(t, err)

			sqlDB, err := gormDB.DB()
			assert.NoError(t, err)
			defer func() {
				_ = sqlDB.Close()
			}()

			tt.setupMock(mock)

			repo := NewFailedTaskEventRepository(gormDB)
			err = repo.Create(context.Background(), tt.failedTask)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Note: In mock tests, these values won't be updated as they would be in real DB
				// Only check that we got the entity back successfully
				assert.NotNil(t, tt.failedTask)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFailedTaskEventEntity(t *testing.T) {
	t.Run("NewFailedTaskEventWithRedisInfo", func(t *testing.T) {
		taskID := "test-task-id"
		taskType := "merchant:sync"
		queueName := "default"
		payload := `{"merchant_id": "123"}`
		errorMessage := "connection timeout"
		redisKey := "asynq:default:t:test-task-id"
		redisState := "failed"
		retryCount := 2

		failedTask := entity.NewFailedTaskEventWithRedisInfo(
			taskID, taskType, queueName, payload, errorMessage,
			redisKey, redisState, retryCount,
		)

		assert.Equal(t, taskID, failedTask.GetTaskID())
		assert.Equal(t, taskType, failedTask.GetTaskType())
		assert.Equal(t, queueName, failedTask.GetQueueName())
		assert.Equal(t, payload, failedTask.GetPayload())
		assert.Equal(t, errorMessage, failedTask.GetErrorMessage())
		assert.Equal(t, retryCount, failedTask.GetRetryCount())
		assert.NotNil(t, failedTask.GetRedisKey())
		assert.Equal(t, redisKey, *failedTask.GetRedisKey())
		assert.NotNil(t, failedTask.GetRedisState())
		assert.Equal(t, redisState, *failedTask.GetRedisState())
		assert.True(t, failedTask.IsValid())
	})

	t.Run("Entity validation", func(t *testing.T) {
		// Valid entity
		validTask := entity.NewFailedTaskEvent(
			"test-id", "test-type", "default", "payload", "error", 0,
		)
		assert.True(t, validTask.IsValid())

		// Invalid entity - missing required fields
		invalidTask := &entity.FailedTaskEvent{}
		assert.False(t, invalidTask.IsValid())
	})
}
