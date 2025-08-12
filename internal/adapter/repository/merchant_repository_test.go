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

type MerchantTestCase struct {
	name             string
	id               uint64
	setupMock        func(sqlmock.Sqlmock)
	expectedMerchant *entity.Merchant
	expectedError    error
}

func setupMerchantMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestMerchantRepository_FindByID(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "global_merchant_id", "name", "display_name", "api_key", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, "Test-Merchant-01", "Merchant-01", "Merchant-Nickname", "123", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `merchants`")).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			merchant, err := repo.FindByID(context.Background(), tc.expectedMerchant.ID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMerchant.ID, merchant.ID)
				assert.Equal(t, tc.expectedMerchant.GlobalMerchantID, merchant.GlobalMerchantID)
				assert.Equal(t, tc.expectedMerchant.Name, merchant.Name)
				assert.Equal(t, tc.expectedMerchant.DisplayName, merchant.DisplayName)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMerchantRepository_FindByGlobalID(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "global_merchant_id", "name", "display_name", "api_key", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, "Test-Merchant-01", "Merchant-01", "Merchant-Nickname", "123", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `merchants`")).
					WithArgs("Test-Merchant-01", 1).
					WillReturnRows(rows)
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			merchant, err := repo.FindByGlobalID(
				context.Background(),
				tc.expectedMerchant.GlobalMerchantID,
			)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMerchant.ID, merchant.ID)
				assert.Equal(t, tc.expectedMerchant.GlobalMerchantID, merchant.GlobalMerchantID)
				assert.Equal(t, tc.expectedMerchant.Name, merchant.Name)
				assert.Equal(t, tc.expectedMerchant.DisplayName, merchant.DisplayName)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMerchantRepository_Create(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant create",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `merchants`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			err := repo.Create(context.Background(), tc.expectedMerchant)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMerchant.ID, uint64(1))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMerchantRepository_Update(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant update",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `merchants`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			err := repo.Update(context.Background(), tc.expectedMerchant)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMerchantRepository_Delete(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant delete",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `merchants`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			err := repo.Delete(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMerchantRepository_Upsert(t *testing.T) {
	now := time.Now()
	testCases := []MerchantTestCase{
		{
			name: "merchant update",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `merchants`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedMerchant: &entity.Merchant{
				ID:               1,
				GlobalMerchantID: "Test-Merchant-01",
				Name:             "Merchant-01",
				DisplayName:      "Merchant-Nickname",
				CreatedAt:        now,
				UpdatedAt:        now,
				DeletedAt:        nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMerchantMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewMerchantRepository(db)

			err := repo.Upsert(context.Background(), tc.expectedMerchant)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoMerchantNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
