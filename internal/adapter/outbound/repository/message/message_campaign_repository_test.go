package message

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MessageCampaignTestCase struct {
	name              string
	id                uint64
	globalID          string
	setupMock         func(sqlmock.Sqlmock)
	expectedCampaign  *entity.MessageCampaign
	expectedCampaigns []*entity.MessageCampaign
	expectedError     error
	expectedTotal     int
}

func setupMessageCampaignMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	return db, mock, mockDB
}

func createTestMessageCampaign() *entity.MessageCampaign {
	now := time.Now()
	updatedBy := "admin"
	return &entity.MessageCampaign{
		ID:            1,
		Category:      consts.CategoryMember,
		Item:          consts.ItemRegistration,
		TriggerType:   "manual",
		Title:         "Test Campaign",
		MerchantID:    1,
		GlobalID:      "TEST-CAMPAIGN-001",
		Content:       "Test content",
		Target:        consts.TargetAll,
		Status:        consts.MessageCampaignStatusDraft,
		AutoSend:      false,
		RealSentCount: 100,
		SendStartTime: &now,
		SendEndTime:   nil,
		CreatedBy:     "admin",
		UpdatedBy:     &updatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
		DeletedAt:     nil,
	}
}

func TestMessageCampaignRepository_FindByID(t *testing.T) {
	now := time.Now()
	testCases := []MessageCampaignTestCase{
		{
			name: "campaign found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "category", "item", "trigger_type", "title", "merchant_id", "global_id",
					"content", "target", "status", "auto_send", "real_sent_count",
					"send_start_time", "send_end_time", "created_by", "updated_by",
					"created_at", "updated_at", "deleted_at",
				}).AddRow(
					1, consts.CategoryMember, consts.ItemRegistration, "manual", "Test Campaign", 1, "TEST-CAMPAIGN-001",
					"Test content", consts.TargetAll, consts.MessageCampaignStatusDraft, false, 100,
					now, nil, "admin", "admin",
					now, now, nil,
				)

				mock.ExpectQuery("SELECT \\* FROM `message_campaign`").
					WillReturnRows(rows)
			},
			expectedCampaign: createTestMessageCampaign(),
		},
		{
			name: "campaign not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM `message_campaign`").
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			campaign, err := repo.FindByID(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCampaign.ID, campaign.ID)
				assert.Equal(t, tc.expectedCampaign.Title, campaign.Title)
				assert.Equal(t, tc.expectedCampaign.GlobalID, campaign.GlobalID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_FindByGlobalID(t *testing.T) {
	now := time.Now()
	testCases := []MessageCampaignTestCase{
		{
			name:     "campaign found by global id",
			globalID: "TEST-CAMPAIGN-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "category", "item", "trigger_type", "title", "merchant_id", "global_id",
					"content", "target", "status", "auto_send", "real_sent_count",
					"send_start_time", "send_end_time", "created_by", "updated_by",
					"created_at", "updated_at", "deleted_at",
				}).AddRow(
					1, consts.CategoryMember, consts.ItemRegistration, "manual", "Test Campaign", 1, "TEST-CAMPAIGN-001",
					"Test content", consts.TargetAll, consts.MessageCampaignStatusDraft, false, 100,
					now, nil, "admin", "admin",
					now, now, nil,
				)

				mock.ExpectQuery("SELECT \\* FROM `message_campaign`").
					WillReturnRows(rows)
			},
			expectedCampaign: createTestMessageCampaign(),
		},
		{
			name:     "campaign not found by global id",
			globalID: "NON-EXISTENT",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM `message_campaign`").
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			campaign, err := repo.FindByGlobalID(context.Background(), tc.globalID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCampaign.GlobalID, campaign.GlobalID)
				assert.Equal(t, tc.expectedCampaign.Title, campaign.Title)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_Create(t *testing.T) {
	testCases := []MessageCampaignTestCase{
		{
			name: "create campaign success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `message_campaign`").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedCampaign: createTestMessageCampaign(),
		},
		{
			name: "create campaign error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `message_campaign`").
					WillReturnError(errors.New("database error"))
				mock.ExpectRollback()
			},
			expectedCampaign: createTestMessageCampaign(),
			expectedError:    errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			err := repo.Create(context.Background(), tc.expectedCampaign)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "database error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, uint64(1), tc.expectedCampaign.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_Update(t *testing.T) {
	testCases := []MessageCampaignTestCase{
		{
			name: "update campaign success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedCampaign: createTestMessageCampaign(),
		},
		{
			name: "update campaign error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnError(errors.New("update error"))
				mock.ExpectRollback()
			},
			expectedCampaign: createTestMessageCampaign(),
			expectedError:    errors.New("update error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			err := repo.Update(context.Background(), tc.expectedCampaign)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "update error")
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_Delete(t *testing.T) {
	testCases := []MessageCampaignTestCase{
		{
			name: "delete campaign success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "delete campaign not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			err := repo.Delete(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found")
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_UpdateSentCount(t *testing.T) {
	testCases := []MessageCampaignTestCase{
		{
			name: "update sent count success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "update sent count not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			err := repo.UpdateSentCount(context.Background(), tc.id, 150)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found")
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageCampaignRepository_UpdateFields(t *testing.T) {
	testCases := []MessageCampaignTestCase{
		{
			name: "update fields success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "update fields no rows affected",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `message_campaign`").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectedError: errors.New("record not found"),
		},
		{
			name: "update fields empty updates",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database expectations for empty updates
			},
			expectedError: errors.New("no fields to update"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMessageCampaignMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewMessageCampaignRepository(db)

			var updates map[string]interface{}
			if tc.name == "update fields empty updates" {
				updates = map[string]interface{}{}
			} else {
				updates = map[string]interface{}{
					"status": consts.MessageCampaignStatusScheduled,
					"title":  "Updated Title",
				}
			}

			err := repo.UpdateFields(context.Background(), tc.id, updates, false)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.name == "update fields empty updates" {
					assert.Contains(t, err.Error(), "no fields to update")
				} else {
					assert.Contains(t, err.Error(), "record not found")
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
