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

// setupManagerMockDB sets up a mock database connection for manager tests
func setupManagerMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

// TestManagerRepository_FindByID tests the FindByID method
func TestManagerRepository_FindByID(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []struct {
		name            string
		id              uint64
		setupMock       func(sqlmock.Sqlmock)
		expectedManager *entity.Manager
		expectedError   error
	}{
		{
			name: "manager found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, 100, "global-manager-1", "testaccount", email, now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE `managers`.`id` = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedManager: &entity.Manager{
				ID:              1,
				MerchantID:      100,
				GlobalManagerID: "global-manager-1",
				Account:         "testaccount",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
				DeletedAt:       nil,
			},
			expectedError: nil,
		},
		{
			name: "manager not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE `managers`.`id` = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs(999, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedManager: nil,
			expectedError:   errmsg.ErrRepoManagerNotFound,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE `managers`.`id` = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedManager: &entity.Manager{},
			expectedError:   errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			manager, err := repo.FindByID(context.Background(), tc.id)

			// Assert the result
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoManagerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoManagerNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedManager.ID, manager.ID)
				assert.Equal(t, tc.expectedManager.MerchantID, manager.MerchantID)
				assert.Equal(t, tc.expectedManager.GlobalManagerID, manager.GlobalManagerID)
				assert.Equal(t, tc.expectedManager.Account, manager.Account)
				assert.Equal(t, tc.expectedManager.Email, manager.Email)
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestManagerRepository_FirstOrCreate tests the FirstOrCreate method
func TestManagerRepository_FirstOrCreate(t *testing.T) {
	// Setup test data
	now := time.Now()
	email := "test@example.com"

	testCases := []struct {
		name          string
		setupMock     func(sqlmock.Sqlmock)
		manager       *entity.Manager
		expectedError error
		expectedID    uint64
	}{
		{
			name: "create new manager - record not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GORM FirstOrCreate: First SELECT query returns empty result (no error)
				emptyRows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"})
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers`")).
					WithArgs("FATCAT-MANAGER-001", 1).
					WillReturnRows(emptyRows)
				
				// Then create with transaction
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `managers`")).
					WithArgs(
						sqlmock.AnyArg(), // merchant_id
						"FATCAT-MANAGER-001", // global_manager_id
						"test@example.com",  // account
						&email,              // email
						sqlmock.AnyArg(),    // created_at
						sqlmock.AnyArg(),    // updated_at
						sqlmock.AnyArg(),    // deleted_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			manager: &entity.Manager{
				MerchantID:      1,
				GlobalManagerID: "FATCAT-MANAGER-001",
				Account:         "test@example.com",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: nil,
			expectedID:    1,
		},
		{
			name: "find existing manager - record found",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Record found, no insert needed
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(2, 1, "FATCAT-MANAGER-002", "existing@example.com", &email, now, now, nil)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers`")).
					WithArgs("FATCAT-MANAGER-002", 1).
					WillReturnRows(rows)
			},
			manager: &entity.Manager{
				MerchantID:      1,
				GlobalManagerID: "FATCAT-MANAGER-002",
				Account:         "existing@example.com",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: nil,
			expectedID:    2,
		},
		{
			name: "database error during create",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GORM FirstOrCreate: First SELECT query returns empty result
				emptyRows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"})
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers`")).
					WithArgs("FATCAT-MANAGER-003", 1).
					WillReturnRows(emptyRows)
				
				// Create fails
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `managers`")).
					WillReturnError(errors.New("database connection error"))
				mock.ExpectRollback()
			},
			manager: &entity.Manager{
				MerchantID:      1,
				GlobalManagerID: "FATCAT-MANAGER-003",
				Account:         "error@example.com",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			expectedError: errors.New("database connection error"),
			expectedID:    0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			db, mock, sqlDB := setupManagerMockDB(t)
			defer sqlDB.Close()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Execute
			err := repo.FirstOrCreate(context.Background(), tc.manager)

			// Verify
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, tc.manager.ID)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestManagerRepository_Create tests the Create method
func TestManagerRepository_Create(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []struct {
		name          string
		manager       *entity.Manager
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful create",
			manager: &entity.Manager{
				MerchantID:      100,
				GlobalManagerID: "global-manager-1",
				Account:         "testaccount",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect insert
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `managers` (`merchant_id`,`global_manager_id`,`account`,`email`,`deleted_at`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?)")).
					WithArgs(
						sqlmock.AnyArg(), // MerchantID
						sqlmock.AnyArg(), // GlobalManagerID
						sqlmock.AnyArg(), // Account
						sqlmock.AnyArg(), // Email
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
						sqlmock.AnyArg(), // DeletedAt
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			err := repo.Create(context.Background(), tc.manager)

			// Assert the result
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, uint64(1), tc.manager.ID) // ID should be set after creation
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestManagerRepository_Update tests the Update method
func TestManagerRepository_Update(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []struct {
		name          string
		manager       *entity.Manager
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful update",
			manager: &entity.Manager{
				ID:              1,
				MerchantID:      100,
				GlobalManagerID: "global-manager-1",
				Account:         "testaccount",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect update
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			err := repo.Update(context.Background(), tc.manager)

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

// TestManagerRepository_Delete tests the Delete method
func TestManagerRepository_Delete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name          string
		id            uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "successful delete",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect delete
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE").
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name: "manager not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect delete but no rows affected
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE").
					WithArgs(sqlmock.AnyArg(), 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectedError: errmsg.ErrRepoDeleteManagerNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			err := repo.Delete(context.Background(), tc.id)

			// Assert the result
			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == errmsg.ErrRepoDeleteManagerNotFound {
					assert.ErrorIs(t, err, errmsg.ErrRepoDeleteManagerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestManagerRepository_Upsert tests the Upsert method
func TestManagerRepository_Upsert(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []struct {
		name          string
		manager       *entity.Manager
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful upsert",
			manager: &entity.Manager{
				MerchantID:      100,
				GlobalManagerID: "global-manager-1",
				Account:         "testaccount",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect upsert
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `managers`").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			err := repo.Upsert(context.Background(), tc.manager)

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

// TestManagerRepository_FindByGlobalID tests the FindByGlobalID method
func TestManagerRepository_FindByGlobalID(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []struct {
		name            string
		globalID        string
		setupMock       func(sqlmock.Sqlmock)
		expectedManager *entity.Manager
		expectedError   error
	}{
		{
			name:     "manager found",
			globalID: "global-manager-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, 100, "global-manager-1", "testaccount", email, now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE global_manager_id = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs("global-manager-1", 1).
					WillReturnRows(rows)
			},
			expectedManager: &entity.Manager{
				ID:              1,
				MerchantID:      100,
				GlobalManagerID: "global-manager-1",
				Account:         "testaccount",
				Email:           &email,
				CreatedAt:       now,
				UpdatedAt:       now,
				DeletedAt:       nil,
			},
			expectedError: nil,
		},
		{
			name:     "manager not found",
			globalID: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE global_manager_id = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs("non-existent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedManager: nil,
			expectedError:   errmsg.ErrRepoManagerNotFound,
		},
		{
			name:     "database error",
			globalID: "global-manager-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers` WHERE global_manager_id = ? AND `managers`.`deleted_at` IS NULL ORDER BY `managers`.`id` LIMIT ?")).
					WithArgs("global-manager-1", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedManager: &entity.Manager{},
			expectedError:   errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			// Setup mock expectations
			tc.setupMock(mock)

			// Create repository
			repo := NewManagerRepository(db)

			// Call the method
			manager, err := repo.FindByGlobalID(context.Background(), tc.globalID)

			// Assert the result
			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == errmsg.ErrRepoManagerNotFound {
					assert.ErrorIs(t, err, errmsg.ErrRepoManagerNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedManager.ID, manager.ID)
				assert.Equal(t, tc.expectedManager.MerchantID, manager.MerchantID)
				assert.Equal(t, tc.expectedManager.GlobalManagerID, manager.GlobalManagerID)
				assert.Equal(t, tc.expectedManager.Account, manager.Account)
				assert.Equal(t, tc.expectedManager.Email, manager.Email)
			}

			// Verify that all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
