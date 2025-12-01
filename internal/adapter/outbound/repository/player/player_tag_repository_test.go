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
		name: "player tag upsert with difference calculation",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			// 優化後的邏輯：先查詢現有標籤
			mock.ExpectQuery(regexp.QuoteMeta("SELECT `tag_id` FROM `player_tags` WHERE player_id = ?")).
				WithArgs(2).
				WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow(1).AddRow(2))

			mock.ExpectBegin()

			// 刪除不需要的標籤 (1, 2 不在新的 [3, 4] 中)
			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ? AND tag_id IN")).
				WithArgs(2, 1, 2).
				WillReturnResult(sqlmock.NewResult(0, 2))

			// 插入新標籤 ([3, 4] 不在現有的 [1, 2] 中)
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

func TestPlayerTagRepository_BatchUpdate_NoChanges(t *testing.T) {
	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	// 模擬現有標籤與新標籤相同的情況
	mock.ExpectQuery(regexp.QuoteMeta("SELECT `tag_id` FROM `player_tags` WHERE player_id = ?")).
		WithArgs(uint64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow(1).AddRow(2))

	repo := NewPlayerTagRepository(db)
	err := repo.BatchUpdate(context.Background(), 123, []uint64{1, 2})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_difference(t *testing.T) {
	repo := &PlayerTagRepository{}

	tests := []struct {
		name     string
		a        []uint64
		b        []uint64
		expected []uint64
	}{
		{
			name:     "simple difference",
			a:        []uint64{1, 2, 3, 4},
			b:        []uint64{2, 3},
			expected: []uint64{1, 4},
		},
		{
			name:     "no difference",
			a:        []uint64{1, 2, 3},
			b:        []uint64{1, 2, 3},
			expected: []uint64{},
		},
		{
			name:     "empty a",
			a:        []uint64{},
			b:        []uint64{1, 2, 3},
			expected: []uint64{},
		},
		{
			name:     "empty b",
			a:        []uint64{1, 2, 3},
			b:        []uint64{},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "both empty",
			a:        []uint64{},
			b:        []uint64{},
			expected: []uint64{},
		},
		{
			name:     "disjoint sets",
			a:        []uint64{1, 3, 5},
			b:        []uint64{2, 4, 6},
			expected: []uint64{1, 3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.difference(tt.a, tt.b)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
