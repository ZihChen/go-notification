package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type PlayerTagTestCase struct {
	name               string
	id                 uint64
	setupMock          func(sqlmock.Sqlmock)
	expectedPlayerTags []*entity.PlayerTag
	expectedError      error
}

func setupPlayerTagMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true, // 禁用默認事務
	})
	require.NoError(t, err)

	return db, mock, mockDB
}

func TestPlayerTagRepository_BatchUpdate(t *testing.T) {
	now := time.Now()
	testCase := PlayerTagTestCase{
		name: "player tag upsert with simple delete-insert strategy",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			// 簡化後的邏輯：直接事務操作
			mock.ExpectBegin()

			// 刪除所有現有標籤
			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
				WithArgs(2).
				WillReturnResult(sqlmock.NewResult(0, 2))

			// 插入新標籤
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `player_tags`")).
				WillReturnResult(sqlmock.NewResult(0, 2))

			mock.ExpectCommit()
		},
		expectedPlayerTags: []*entity.PlayerTag{
			{
				PlayerID:  2,
				TagID:     3,
				CreatedAt: now,
			},
			{
				PlayerID:  2,
				TagID:     4,
				CreatedAt: now,
			},
		},
		expectedError: nil,
	}

	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()
	testCase.setupMock(mock)
	repo := NewPlayerTagRepository(db)

	err := repo.BatchUpdate(context.Background(), 2, []uint64{3, 4})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_DeleteByPlayerID(t *testing.T) {
	tests := []struct {
		name      string
		playerID  uint64
		setupMock func(sqlmock.Sqlmock)
		wantError bool
	}{
		{
			name:     "successful delete by player ID",
			playerID: 123,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
					WithArgs(123).
					WillReturnResult(sqlmock.NewResult(0, 2))
			},
			wantError: false,
		},
		{
			name:     "delete by player ID with database error",
			playerID: 456,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
					WithArgs(456).
					WillReturnError(assert.AnError)
			},
			wantError: true,
		},
		{
			name:     "delete by player ID with no records",
			playerID: 789,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
					WithArgs(789).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerTagMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tt.setupMock(mock)
			repo := NewPlayerTagRepository(db)

			err := repo.DeleteByPlayerID(context.Background(), tt.playerID)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "delete player tags failed")
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerTagRepository_BatchUpdate_EmptyTags(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 模擬空標籤情況，應該調用DeleteByPlayerID
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
		WithArgs(uint64(123)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdate(context.Background(), 123, []uint64{})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_BatchUpdate_SameTags(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 簡化邏輯：即使標籤相同，也會執行delete-then-insert操作
	mock.ExpectBegin()

	// 刪除所有現有標籤
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ?")).
		WithArgs(uint64(123)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// 插入標籤
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `player_tags`")).
		WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectCommit()

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdate(context.Background(), 123, []uint64{1, 2})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_BatchUpdateWithDiff(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 測試精確差異更新
	mock.ExpectBegin()

	// 刪除特定標籤 [1, 2]
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ? AND tag_id IN")).
		WithArgs(uint64(123), 1, 2).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// 插入新標籤 [3, 4]
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `player_tags`")).
		WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectCommit()

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdateWithDiff(context.Background(), 123, []uint64{1, 2}, []uint64{3, 4})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_BatchUpdateWithDiff_NoChanges(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 無變化情況下不應該有任何數據庫操作
	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdateWithDiff(context.Background(), 123, []uint64{}, []uint64{})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_BatchUpdateWithDiff_OnlyDelete(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 只刪除，不插入
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ? AND tag_id IN")).
		WithArgs(uint64(123), 1, 2).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdateWithDiff(context.Background(), 123, []uint64{1, 2}, []uint64{})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_BatchUpdateWithDiff_OnlyInsert(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 只插入，不刪除
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `player_tags`")).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdateWithDiff(context.Background(), 123, []uint64{}, []uint64{3, 4})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
