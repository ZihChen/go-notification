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

type AgentCampaignTestCase struct {
	name             string
	id               uint64
	setupMock        func(sqlmock.Sqlmock)
	expectedCampaign *entity.AgentCampaign
	expectedError    error
}

func setupAgentCampaignMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestAgentCampaignRepository_Create(t *testing.T) {
	now := time.Now()
	testCases := []AgentCampaignTestCase{
		{
			name: "create agent campaign successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `agent_campaigns`")).
					WithArgs(
						sqlmock.AnyArg(), // merchant_id
						sqlmock.AnyArg(), // title
						sqlmock.AnyArg(), // content
						sqlmock.AnyArg(), // scheduled_at
						sqlmock.AnyArg(), // status
						sqlmock.AnyArg(), // target_type
						sqlmock.AnyArg(), // target_details
						sqlmock.AnyArg(), // target_count
						sqlmock.AnyArg(), // real_sent_count
						sqlmock.AnyArg(), // created_by
						sqlmock.AnyArg(), // updated_by
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						sqlmock.AnyArg(), // deleted_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedCampaign: &entity.AgentCampaign{
				ID:            0, // ID會在 Create 後被設定
				MerchantID:    1,
				Title:         "Test Campaign",
				Content:       "Test Content",
				ScheduledAt:   &now,
				Status:        "draft",
				TargetType:    "all",
				TargetDetails: []string{},
				TargetCount:   0,
				RealSentCount: 0,
				CreatedBy:     "test_user",
				UpdatedBy:     "test_user",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			result, err := repo.Create(context.Background(), tc.expectedCampaign)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, uint64(1), result.ID)
				assert.Equal(t, tc.expectedCampaign.Title, result.Title)
				assert.Equal(t, tc.expectedCampaign.Content, result.Content)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentCampaignRepository_GetByID(t *testing.T) {
	now := time.Now()
	testCases := []AgentCampaignTestCase{
		{
			name: "get agent campaign by ID successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "merchant_id", "title", "content", "scheduled_at",
					"status", "target_type", "target_details", "target_count",
					"real_sent_count", "created_by", "updated_by", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					1, 1, "Test Campaign", "Test Content", &now,
					"draft", "all", "[]", 0,
					0, "test_user", "test_user", now, now, nil,
				)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_campaigns`")).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedCampaign: &entity.AgentCampaign{
				ID:            1,
				MerchantID:    1,
				Title:         "Test Campaign",
				Content:       "Test Content",
				ScheduledAt:   &now,
				Status:        "draft",
				TargetType:    "all",
				TargetDetails: []string{},
				TargetCount:   0,
				RealSentCount: 0,
				CreatedBy:     "test_user",
				UpdatedBy:     "test_user",
				CreatedAt:     now,
				UpdatedAt:     now,
				DeletedAt:     nil,
			},
			expectedError: nil,
		},
		{
			name: "campaign not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_campaigns`")).
					WithArgs(999, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedCampaign: nil,
			expectedError:    nil, // Repository returns nil for not found
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			result, err := repo.GetByID(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedCampaign == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
					assert.Equal(t, tc.expectedCampaign.ID, result.ID)
					assert.Equal(t, tc.expectedCampaign.Title, result.Title)
					assert.Equal(t, tc.expectedCampaign.Content, result.Content)
					assert.Equal(t, tc.expectedCampaign.Status, result.Status)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentCampaignRepository_Update(t *testing.T) {
	now := time.Now()
	testCases := []AgentCampaignTestCase{
		{
			name: "update agent campaign successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_campaigns`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedCampaign: &entity.AgentCampaign{
				ID:            1,
				MerchantID:    1,
				Title:         "Updated Campaign",
				Content:       "Updated Content",
				Status:        "scheduled",
				TargetType:    "specific",
				TargetDetails: []string{"agent1", "agent2"},
				CreatedBy:     "test_user",
				UpdatedBy:     "test_user",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			err := repo.Update(context.Background(), tc.expectedCampaign)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentCampaignRepository_Delete(t *testing.T) {
	testCases := []struct {
		name          string
		id            uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "delete agent campaign successfully",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_campaigns` SET `deleted_at`")).
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

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

func TestAgentCampaignRepository_UpdateFields(t *testing.T) {
	testCases := []struct {
		name          string
		id            uint64
		fields        map[string]interface{}
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "update fields successfully",
			id:   1,
			fields: map[string]interface{}{
				"status":     "cancelled",
				"updated_by": "admin",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_campaigns`")).
					WithArgs("cancelled", "admin", sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name:   "no fields to update",
			id:     1,
			fields: map[string]interface{}{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database calls expected
			},
			expectedError: errors.New("no fields to update"),
		},
		{
			name: "record not found",
			id:   999,
			fields: map[string]interface{}{
				"status": "cancelled",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_campaigns`")).
					WithArgs("cancelled", sqlmock.AnyArg(), 999).
					WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected
				mock.ExpectCommit()
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			err := repo.UpdateFields(context.Background(), tc.id, tc.fields)

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

func TestAgentCampaignRepository_List(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name          string
		query         *dto.AgentCampaignsQuery
		setupMock     func(sqlmock.Sqlmock)
		expectedCount int
		expectedTotal int
		expectedError error
	}{
		{
			name: "list campaigns with pagination",
			query: &dto.AgentCampaignsQuery{
				MerchantID:     1,
				Page:           1,
				PageSize:       10,
				Limit:          10,
				Offset:         0,
				IncludeDeleted: false,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query
				mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*)")).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

				// List query
				rows := sqlmock.NewRows([]string{
					"id", "merchant_id", "title", "content", "scheduled_at",
					"status", "target_type", "target_details", "target_count",
					"real_sent_count", "created_by", "updated_by", "created_at", "updated_at", "deleted_at",
				}).
					AddRow(1, 1, "Campaign 1", "Content 1", &now, "draft", "all", "[]", 0, 0, "user1", "user1", now, now, nil).
					AddRow(2, 1, "Campaign 2", "Content 2", &now, "scheduled", "specific", "[\"agent1\"]", 1, 0, "user2", "user2", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_campaigns`")).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectedTotal: 2,
			expectedError: nil,
		},
		{
			name: "list campaigns with filters",
			query: &dto.AgentCampaignsQuery{
				MerchantID:     1,
				Status:         "draft",
				TargetType:     "all",
				CreatedBy:      "user1",
				Page:           1,
				PageSize:       10,
				Limit:          10,
				Offset:         0,
				IncludeDeleted: false,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query with filters
				mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*)")).
					WithArgs(1, "draft", "all", "user1").
					WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

				// List query with filters
				rows := sqlmock.NewRows([]string{
					"id", "merchant_id", "title", "content", "scheduled_at",
					"status", "target_type", "target_details", "target_count",
					"real_sent_count", "created_by", "updated_by", "created_at", "updated_at", "deleted_at",
				}).
					AddRow(1, 1, "Campaign 1", "Content 1", &now, "draft", "all", "[]", 0, 0, "user1", "user1", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_campaigns`")).
					WithArgs(1, "draft", "all", "user1", 10).
					WillReturnRows(rows)
			},
			expectedCount: 1,
			expectedTotal: 1,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			campaigns, total, err := repo.List(context.Background(), tc.query)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, campaigns, tc.expectedCount)
				assert.Equal(t, tc.expectedTotal, total)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentCampaignRepository_GetScheduledCampaigns(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name          string
		setupMock     func(sqlmock.Sqlmock)
		expectedCount int
		expectedError error
	}{
		{
			name: "get scheduled campaigns successfully",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "merchant_id", "title", "content", "scheduled_at",
					"status", "target_type", "target_details", "target_count",
					"real_sent_count", "created_by", "updated_by", "created_at", "updated_at", "deleted_at",
				}).
					AddRow(1, 1, "Scheduled Campaign", "Content", &now, "scheduled", "all", "[]", 0, 0, "user1", "user1", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_campaigns`")).
					WithArgs("scheduled").
					WillReturnRows(rows)
			},
			expectedCount: 1,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupAgentCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewAgentCampaignRepository(db)

			campaigns, err := repo.GetScheduledCampaigns(context.Background(), time.Now())

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, campaigns, tc.expectedCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
