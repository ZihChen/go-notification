package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type PlayerMessageTestCase struct {
	name             string
	id               uint64
	globalPlayerID   string
	setupMock        func(sqlmock.Sqlmock)
	expectedMessage  *entity.PlayerMessage
	expectedMessages []*entity.PlayerMessage
	expectedStats    *entity.PlayerMessageStats
	expectedError    error
	expectedTotal    int
}

func setupPlayerMessageMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func createTestPlayerMessage() *entity.PlayerMessage {
	now := time.Now()
	return &entity.PlayerMessage{
		ID:             1,
		GlobalPlayerID: "TEST-PLAYER-001",
		PlayerID:       1,
		CampaignID:     1,
		IsRead:         false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestPlayerMessageRepository_FindByID(t *testing.T) {
	now := time.Now()
	testCases := []PlayerMessageTestCase{
		{
			name: "message found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "global_player_id", "player_id", "campaign_id", "is_read",
					"created_at", "updated_at",
				}).AddRow(
					1, "TEST-PLAYER-001", 1, 1, false,
					now, now,
				)

				mock.ExpectQuery("SELECT \\* FROM `player_message`").
					WillReturnRows(rows)
			},
			expectedMessage: createTestPlayerMessage(),
		},
		{
			name: "message not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM `player_message`").
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: errors.New("record not found"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			message, err := repo.FindByID(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMessage.ID, message.ID)
				assert.Equal(t, tc.expectedMessage.GlobalPlayerID, message.GlobalPlayerID)
				assert.Equal(t, tc.expectedMessage.PlayerID, message.PlayerID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_FindByPlayerID(t *testing.T) {
	now := time.Now()
	testCases := []PlayerMessageTestCase{
		{
			name:           "messages found for player",
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(countRows)

				// Data query
				rows := sqlmock.NewRows([]string{
					"id", "global_player_id", "player_id", "campaign_id", "is_read",
					"created_at", "updated_at",
				}).
					AddRow(1, "TEST-PLAYER-001", 1, 1, false, now, now).
					AddRow(2, "TEST-PLAYER-001", 1, 2, true, now, now)

				mock.ExpectQuery("SELECT \\* FROM `player_message`").
					WillReturnRows(rows)
			},
			expectedMessages: []*entity.PlayerMessage{
				{
					ID:             1,
					GlobalPlayerID: "TEST-PLAYER-001",
					PlayerID:       1,
					CampaignID:     1,
					IsRead:         false,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					GlobalPlayerID: "TEST-PLAYER-001",
					PlayerID:       1,
					CampaignID:     2,
					IsRead:         true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expectedTotal: 2,
		},
		{
			name:           "no messages found",
			globalPlayerID: "EMPTY-PLAYER",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(countRows)

				// Data query
				rows := sqlmock.NewRows([]string{
					"id", "global_player_id", "player_id", "campaign_id", "is_read",
					"created_at", "updated_at",
				})

				mock.ExpectQuery("SELECT \\* FROM `player_message`").
					WillReturnRows(rows)
			},
			expectedMessages: []*entity.PlayerMessage{},
			expectedTotal:    0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			messages, total, err := repo.FindByPlayerID(
				context.Background(),
				tc.globalPlayerID,
				1,
				10,
			)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedTotal, total)
			assert.Len(t, messages, len(tc.expectedMessages))
			if len(tc.expectedMessages) > 0 {
				assert.Equal(t, tc.expectedMessages[0].ID, messages[0].ID)
				assert.Equal(t, tc.expectedMessages[0].GlobalPlayerID, messages[0].GlobalPlayerID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_GetPlayerMessageStats(t *testing.T) {
	testCases := []PlayerMessageTestCase{
		{
			name:           "get stats success",
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Total count query
				totalRows := sqlmock.NewRows([]string{"count"}).AddRow(10)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(totalRows)

				// Read count query
				readRows := sqlmock.NewRows([]string{"count"}).AddRow(6)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(readRows)
			},
			expectedStats: &entity.PlayerMessageStats{
				TotalCount:  10,
				ReadCount:   6,
				UnreadCount: 4,
			},
		},
		{
			name:           "get stats with zero messages",
			globalPlayerID: "EMPTY-PLAYER",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Total count query
				totalRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(totalRows)

				// Read count query
				readRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(readRows)
			},
			expectedStats: &entity.PlayerMessageStats{
				TotalCount:  0,
				ReadCount:   0,
				UnreadCount: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			stats, err := repo.GetPlayerMessageStats(context.Background(), tc.globalPlayerID)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStats.TotalCount, stats.TotalCount)
			assert.Equal(t, tc.expectedStats.ReadCount, stats.ReadCount)
			assert.Equal(t, tc.expectedStats.UnreadCount, stats.UnreadCount)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_Create(t *testing.T) {
	testCases := []PlayerMessageTestCase{
		{
			name: "create message success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `player_message`").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMessage: createTestPlayerMessage(),
		},
		{
			name: "create message error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `player_message`").
					WillReturnError(errors.New("database error"))
				mock.ExpectRollback()
			},
			expectedMessage: createTestPlayerMessage(),
			expectedError:   errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			err := repo.Create(context.Background(), tc.expectedMessage)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "database error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, uint64(1), tc.expectedMessage.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_CreateBatch(t *testing.T) {
	messages := []*entity.PlayerMessage{
		createTestPlayerMessage(),
		{
			ID:             2,
			GlobalPlayerID: "TEST-PLAYER-002",
			PlayerID:       2,
			CampaignID:     1,
			IsRead:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	testCases := []PlayerMessageTestCase{
		{
			name: "create batch success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `player_message`").
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
			expectedMessages: messages,
		},
		{
			name: "create empty batch",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database expectations for empty batch
			},
			expectedMessages: []*entity.PlayerMessage{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			err := repo.CreateBatch(context.Background(), tc.expectedMessages)

			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_MarkAsRead(t *testing.T) {
	testCases := []PlayerMessageTestCase{
		{
			name:           "mark as read success",
			id:             1,
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `player_message`").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:           "mark as read not found",
			id:             999,
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `player_message`").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectedError: errors.New("record not found or already read"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			err := repo.MarkAsRead(context.Background(), tc.globalPlayerID, tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "record not found or already read")
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_CheckMessageExists(t *testing.T) {
	testCases := []PlayerMessageTestCase{
		{
			name:           "message exists",
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(countRows)
			},
		},
		{
			name:           "message does not exist",
			globalPlayerID: "TEST-PLAYER-001",
			setupMock: func(mock sqlmock.Sqlmock) {
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery("SELECT count\\(\\*\\) FROM `player_message`").
					WillReturnRows(countRows)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			exists, err := repo.CheckMessageExists(context.Background(), tc.globalPlayerID, 1)

			assert.NoError(t, err)
			expectedExists := tc.name == "message exists"
			assert.Equal(t, expectedExists, exists)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_CreateBatchOptimized(t *testing.T) {
	messages := []*entity.PlayerMessage{
		createTestPlayerMessage(),
		{
			ID:             2,
			GlobalPlayerID: "TEST-PLAYER-002",
			PlayerID:       2,
			CampaignID:     1,
			IsRead:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	testCases := []PlayerMessageTestCase{
		{
			name: "create batch optimized success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `player_message`").
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
			expectedMessages: messages,
		},
		{
			name: "create empty batch optimized",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database expectations for empty batch
			},
			expectedMessages: []*entity.PlayerMessage{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			err := repo.CreateBatchOptimized(context.Background(), tc.expectedMessages, 1000)

			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerMessageRepository_CheckMessageExistsBatch(t *testing.T) {
	playerIDs := []uint64{1, 2, 3}

	testCases := []PlayerMessageTestCase{
		{
			name: "check batch exists success",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"player_id"}).
					AddRow(1).
					AddRow(3)

				mock.ExpectQuery("SELECT `player_id` FROM `player_message`").
					WillReturnRows(rows)
			},
		},
		{
			name: "check empty batch",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No database expectations for empty batch
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMessageMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerMessageRepository(db)

			var testPlayerIDs []uint64
			if tc.name != "check empty batch" {
				testPlayerIDs = playerIDs
			}

			existsMap, err := repo.CheckMessageExistsBatch(context.Background(), testPlayerIDs, 1)

			assert.NoError(t, err)
			if tc.name == "check batch exists success" {
				assert.Len(t, existsMap, 3)
				assert.True(t, existsMap[1])  // exists
				assert.False(t, existsMap[2]) // doesn't exist
				assert.True(t, existsMap[3])  // exists
			} else {
				assert.Empty(t, existsMap)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
