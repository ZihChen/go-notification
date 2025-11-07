package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type AgentMessageTestCase struct {
	name            string
	id              uint64
	setupMock       func(sqlmock.Sqlmock)
	expectedMessage *entity.AgentMessage
	expectedError   error
}

func setupAgentMessageMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	// Create a new SQL mock
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create a GORM DB instance using the mock database
	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	return db, mock, mockDB
}

func TestAgentMessageRepository_Create(t *testing.T) {
	now := time.Now()
	testCases := []AgentMessageTestCase{
		{
			name: "create agent message successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `agent_messages`")).
					WithArgs(
						sqlmock.AnyArg(), // agent_campaign_id
						sqlmock.AnyArg(), // agent_id
						sqlmock.AnyArg(), // is_read
						sqlmock.AnyArg(), // read_at
						sqlmock.AnyArg(), // deleted_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMessage: &entity.AgentMessage{
				ID:              0, // ID會在 Create 後被設定
				AgentCampaignID: 1,
				AgentID:         1,
				IsRead:          false,
				ReadAt:          nil,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			result, err := repo.Create(context.Background(), tc.expectedMessage)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, uint64(1), result.ID)
				assert.Equal(t, tc.expectedMessage.AgentCampaignID, result.AgentCampaignID)
				assert.Equal(t, tc.expectedMessage.AgentID, result.AgentID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_GetByID(t *testing.T) {
	now := time.Now()
	testCases := []AgentMessageTestCase{
		{
			name: "get agent message by ID successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "agent_campaign_id", "agent_id", "is_read", "read_at", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					1, 1, 1, false, nil, now, now, nil,
				)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_messages`")).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedMessage: &entity.AgentMessage{
				ID:              1,
				AgentCampaignID: 1,
				AgentID:         1,
				IsRead:          false,
				ReadAt:          nil,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: nil,
		},
		{
			name: "message not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_messages`")).
					WithArgs(999, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedMessage: nil,
			expectedError:   nil, // Repository returns nil for not found
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			result, err := repo.GetByID(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedMessage == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
					assert.Equal(t, tc.expectedMessage.ID, result.ID)
					assert.Equal(t, tc.expectedMessage.AgentCampaignID, result.AgentCampaignID)
					assert.Equal(t, tc.expectedMessage.AgentID, result.AgentID)
					assert.Equal(t, tc.expectedMessage.IsRead, result.IsRead)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_Update(t *testing.T) {
	now := time.Now()
	testCases := []AgentMessageTestCase{
		{
			name: "update agent message successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_messages`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMessage: &entity.AgentMessage{
				ID:              1,
				AgentCampaignID: 1,
				AgentID:         1,
				IsRead:          true,
				ReadAt:          &now,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			err := repo.Update(context.Background(), tc.expectedMessage)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_Delete(t *testing.T) {
	testCases := []struct {
		name          string
		id            uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "delete agent message successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_messages` SET `deleted_at`")).
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			err := repo.Delete(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_CreateBatch(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name          string
		messages      []*entity.AgentMessage
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "create batch messages successfully",
			messages: []*entity.AgentMessage{
				{AgentCampaignID: 1, AgentID: 1, IsRead: false, CreatedAt: now, UpdatedAt: now},
				{AgentCampaignID: 1, AgentID: 2, IsRead: false, CreatedAt: now, UpdatedAt: now},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `agent_messages`")).
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name:     "empty messages slice",
			messages: []*entity.AgentMessage{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database calls expected
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			err := repo.CreateBatch(context.Background(), tc.messages)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_CheckMessageExistsBatch(t *testing.T) {
	testCases := []struct {
		name           string
		agentIDs       []uint64
		campaignID     uint64
		setupMock      func(sqlmock.Sqlmock)
		expectedExists map[uint64]bool
		expectedError  error
	}{
		{
			name:       "check messages exist successfully",
			agentIDs:   []uint64{1, 2, 3},
			campaignID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"agent_id"}).
					AddRow(1).
					AddRow(3)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT `agent_id` FROM `agent_messages`")).
					WithArgs(1, 2, 3, 1).
					WillReturnRows(rows)
			},
			expectedExists: map[uint64]bool{1: true, 2: false, 3: true},
			expectedError:  nil,
		},
		{
			name:       "empty agent IDs",
			agentIDs:   []uint64{},
			campaignID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database calls expected
			},
			expectedExists: map[uint64]bool{},
			expectedError:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			result, err := repo.CheckMessageExistsBatch(
				context.Background(),
				tc.agentIDs,
				tc.campaignID,
			)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedExists, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_ListByAgent(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name          string
		query         *dto.AgentMessagesQuery
		setupMock     func(sqlmock.Sqlmock)
		expectedCount int
		expectedTotal int
		expectedError error
	}{
		{
			name: "list messages by agent successfully",
			query: &dto.AgentMessagesQuery{
				AgentID:  1,
				Page:     1,
				PageSize: 10,
				Limit:    10,
				Offset:   0,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query
				mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*)")).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

				// List query with JOINs
				rows := sqlmock.NewRows([]string{
					"id", "agent_campaign_id", "agent_id", "is_read", "read_at", "created_at", "updated_at",
					"campaign_title", "campaign_content", "global_agent_id",
				}).
					AddRow(1, 1, 1, false, nil, now, now, "Test Campaign", "Test Content", "global-agent-1").
					AddRow(2, 2, 1, true, &now, now, now, "Another Campaign", "Another Content", "global-agent-1")

				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectedTotal: 2,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			messages, total, err := repo.ListByAgent(context.Background(), tc.query)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, messages, tc.expectedCount)
				assert.Equal(t, tc.expectedTotal, total)
				if len(messages) > 0 {
					// 驗證聚合欄位已正確設定
					assert.NotEmpty(t, messages[0].CampaignTitle)
					assert.NotEmpty(t, messages[0].CampaignContent)
					assert.NotEmpty(t, messages[0].GlobalAgentID)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_MarkAsRead(t *testing.T) {
	testCases := []struct {
		name          string
		messageID     uint64
		agentID       uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:      "mark message as read successfully",
			messageID: 1,
			agentID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_messages`")).
					WithArgs(true, sqlmock.AnyArg(), sqlmock.AnyArg(), 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name:      "message not found or already read",
			messageID: 999,
			agentID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_messages`")).
					WithArgs(true, sqlmock.AnyArg(), sqlmock.AnyArg(), 999, 1).
					WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected
				mock.ExpectCommit()
			},
			expectedError: errors.New("message not found or already read"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			err := repo.MarkAsRead(context.Background(), tc.messageID, tc.agentID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentMessageRepository_GetMessageStats(t *testing.T) {
	testCases := []struct {
		name          string
		agentID       uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedStats *dto.AgentMessageStats
		expectedError error
	}{
		{
			name:    "get message stats successfully",
			agentID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"total_count", "read_count", "unread_count"}).
					AddRow(10, 6, 4)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedStats: &dto.AgentMessageStats{
				TotalCount:  10,
				ReadCount:   6,
				UnreadCount: 4,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentMessageRepository(db)

			stats, err := repo.GetMessageStats(context.Background(), tc.agentID)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Equal(t, tc.expectedStats.TotalCount, stats.TotalCount)
				assert.Equal(t, tc.expectedStats.ReadCount, stats.ReadCount)
				assert.Equal(t, tc.expectedStats.UnreadCount, stats.UnreadCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
