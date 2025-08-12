package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"regexp"
	"testing"
	"time"
)

type PlayerTestCase struct {
	name           string
	id             uint64
	setupMock      func(sqlmock.Sqlmock)
	expectedPlayer *entity.Player
	expectedError  error
}

func setupPlayerMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestPlayerRepository_FindByID(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "account", "api_key", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(2, 1, "Test-Player-01", "test-player-01", "abc123", "", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `players`")).
					WithArgs(2, 1).
					WillReturnRows(rows)
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			player, err := repo.FindByID(context.Background(), tc.expectedPlayer.ID)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPlayer.ID, player.ID)
				assert.Equal(t, tc.expectedPlayer.GlobalPlayerID, player.GlobalPlayerID)
				assert.Equal(t, tc.expectedPlayer.Account, player.Account)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_FindByGlobalID(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "account", "api_key", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(2, 1, "Test-Player-01", "test-player-01", "abc123", "", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `players`")).
					WithArgs("Test-Player-01", 1).
					WillReturnRows(rows)
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			player, err := repo.FindByGlobalID(context.Background(), tc.expectedPlayer.GlobalPlayerID)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPlayer.ID, player.ID)
				assert.Equal(t, tc.expectedPlayer.GlobalPlayerID, player.GlobalPlayerID)
				assert.Equal(t, tc.expectedPlayer.Account, player.Account)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_FirstOrCreate(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player first or create",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "account", "api_key", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(2, 1, "Test-Player-01", "test-player-01", "abc123", "", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `players`")).
					WithArgs("Test-Player-01", 2, 1).
					WillReturnRows(rows)
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.FirstOrCreate(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_Create(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player create",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `players`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.Create(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPlayer.ID, uint64(2))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_Update(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player update",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `players`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.Update(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_Delete(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player delete",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `players`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.Delete(context.Background(), tc.id)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_Upsert(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player upsert",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `players`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.Upsert(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
