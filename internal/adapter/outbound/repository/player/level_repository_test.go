package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupMockDB sets up a mock database connection
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

// TestLevelRepository_Upsert tests the Upsert method
func TestLevelRepository_Upsert(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name          string
		level         *entity.Level
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful upsert",
			level: &entity.Level{
				ID:                  1,
				MerchantID:          100,
				GlobalPlayerLevelID: "global-level-1",
				Name:                "VIP",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect the SQL query for upsert
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `level` (`merchant_id`,`name`,`global_player_level_id`,`deleted_at`,`id`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE `name`=CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END,`updated_at`=CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")).
					WithArgs(
						sqlmock.AnyArg(), // MerchantID
						sqlmock.AnyArg(), // Name
						sqlmock.AnyArg(), // GlobalPlayerLevelID
						sqlmock.AnyArg(), // DeletedAt
						sqlmock.AnyArg(), // ID
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name: "upsert error",
			level: &entity.Level{
				ID:                  1,
				MerchantID:          100,
				GlobalPlayerLevelID: "global-level-1",
				Name:                "VIP",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect the SQL query for upsert but return an error
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `level` (`merchant_id`,`name`,`global_player_level_id`,`deleted_at`,`id`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE `name`=CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END,`updated_at`=CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")).
					WithArgs(
						sqlmock.AnyArg(), // MerchantID
						sqlmock.AnyArg(), // Name
						sqlmock.AnyArg(), // GlobalPlayerLevelID
						sqlmock.AnyArg(), // DeletedAt
						sqlmock.AnyArg(), // ID
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnError(errors.New("database error"))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewLevelRepository(db)

			// Call the method
			err := repo.Upsert(context.Background(), tc.level)

			// Assert the result
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestLevelRepository_FindByGlobalID tests the FindByGlobalID method
func TestLevelRepository_FindByGlobalID(t *testing.T) {
	// Setup test cases
	now := time.Now()
	testCases := []struct {
		name          string
		globalID      string
		setupMock     func(sqlmock.Sqlmock)
		expectedLevel *entity.Level
		expectedError error
	}{
		{
			name:     "level found",
			globalID: "global-level-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "name", "global_player_level_id", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, 100, "VIP", "global-level-1", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("global-level-1", 1).
					WillReturnRows(rows)
			},
			expectedLevel: &entity.Level{
				ID:                  1,
				MerchantID:          100,
				GlobalPlayerLevelID: "global-level-1",
				Name:                "VIP",
				CreatedAt:           now,
				UpdatedAt:           now,
				DeletedAt:           nil,
			},
			expectedError: nil,
		},
		{
			name:     "level not found",
			globalID: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("non-existent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedLevel: &entity.Level{},
			expectedError: errmsg.ErrRepoLevelNotFound,
		},
		{
			name:     "database error",
			globalID: "global-level-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("global-level-1", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedLevel: &entity.Level{},
			expectedError: errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewLevelRepository(db)

			// Call the method
			level, err := repo.FindByGlobalID(context.Background(), tc.globalID)

			// Assert the result
			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == errmsg.ErrRepoLevelNotFound {
					assert.ErrorIs(t, err, errmsg.ErrRepoLevelNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLevel.ID, level.ID)
				assert.Equal(t, tc.expectedLevel.MerchantID, level.MerchantID)
				assert.Equal(t, tc.expectedLevel.GlobalPlayerLevelID, level.GlobalPlayerLevelID)
				assert.Equal(t, tc.expectedLevel.Name, level.Name)
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
