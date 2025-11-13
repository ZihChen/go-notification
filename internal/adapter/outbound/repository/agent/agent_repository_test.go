package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupAgentMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestNewAgentRepository(t *testing.T) {
	db, _, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	assert.NotNil(t, repo)
}

func TestAgentRepository_GetByGlobalID(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		globalAgentID string
		setupMock     func(sqlmock.Sqlmock)
		expectedAgent *entity.Agent
		expectedError error
	}{
		{
			name:          "get agent by global ID successfully",
			globalAgentID: "AGENT-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at"}).
					AddRow(1, 100, "AGENT-123", "agent123", "parent/child", time.Now(), time.Now(), time.Now())
				mock.ExpectQuery("SELECT .* FROM `agents` WHERE global_agent_id = .* AND .*deleted_at.* IS NULL ORDER BY .*id.* LIMIT .*").
					WithArgs("AGENT-123", 1).
					WillReturnRows(rows)
			},
			expectedAgent: &entity.Agent{
				ID:            1,
				MerchantID:    100,
				GlobalAgentID: "AGENT-123",
				Account:       "agent123",
				Ancestry:      "parent/child",
			},
			expectedError: nil,
		},
		{
			name:          "agent not found",
			globalAgentID: "AGENT-999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .* FROM `agents` WHERE global_agent_id = .* AND .*deleted_at.* IS NULL ORDER BY .*id.* LIMIT .*").
					WithArgs("AGENT-999", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedAgent: nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			agent, err := repo.GetByGlobalID(ctx, tt.globalAgentID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				if tt.expectedAgent != nil {
					assert.Equal(t, tt.expectedAgent.ID, agent.ID)
					assert.Equal(t, tt.expectedAgent.GlobalAgentID, agent.GlobalAgentID)
					assert.Equal(t, tt.expectedAgent.Account, agent.Account)
				} else {
					assert.Nil(t, agent)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentRepository_Upsert(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		agent         *entity.Agent
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "upsert agent successfully",
			agent: &entity.Agent{
				MerchantID:    100,
				GlobalAgentID: "AGENT-123",
				Account:       "agent123",
				Ancestry:      "parent/child",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `agents` \\(.*\\) VALUES \\(.*\\) ON DUPLICATE KEY UPDATE .*").
					WithArgs(100, "AGENT-123", "agent123", "parent/child", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			err := repo.Upsert(ctx, tt.agent)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentRepository_FindAgents(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	ctx := context.Background()

	tests := []struct {
		name           string
		merchantID     uint64
		limit          int
		offset         int
		setupMock      func(sqlmock.Sqlmock)
		expectedAgents []*entity.Agent
		expectedError  error
	}{
		{
			name:       "find agents successfully",
			merchantID: 100,
			limit:      10,
			offset:     0,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at"}).
					AddRow(1, 100, "AGENT-123", "agent123", "", time.Now(), time.Now(), time.Now()).
					AddRow(2, 100, "AGENT-456", "agent456", "", time.Now(), time.Now(), time.Now())
				mock.ExpectQuery("SELECT .* FROM `agents` WHERE merchant_id = .* AND .*deleted_at.* IS NULL LIMIT .*").
					WithArgs(100, 10).
					WillReturnRows(rows)
			},
			expectedAgents: []*entity.Agent{
				{ID: 1, MerchantID: 100, GlobalAgentID: "AGENT-123", Account: "agent123"},
				{ID: 2, MerchantID: 100, GlobalAgentID: "AGENT-456", Account: "agent456"},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			agents, err := repo.FindAgents(ctx, tt.merchantID, tt.limit, tt.offset)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, agents, len(tt.expectedAgents))
				for i, expectedAgent := range tt.expectedAgents {
					assert.Equal(t, expectedAgent.ID, agents[i].ID)
					assert.Equal(t, expectedAgent.GlobalAgentID, agents[i].GlobalAgentID)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentRepository_GetByAccount(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	ctx := context.Background()

	tests := []struct {
		name          string
		account       string
		setupMock     func(sqlmock.Sqlmock)
		expectedAgent *entity.Agent
		expectedError error
	}{
		{
			name:    "get agent by account successfully",
			account: "agent123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at"}).
					AddRow(1, 100, "AGENT-123", "agent123", "", time.Now(), time.Now(), time.Now())
				mock.ExpectQuery("SELECT .* FROM `agents` WHERE account = .* AND .*deleted_at.* IS NULL ORDER BY .*id.* LIMIT .*").
					WithArgs("agent123", 1).
					WillReturnRows(rows)
			},
			expectedAgent: &entity.Agent{
				ID:            1,
				MerchantID:    100,
				GlobalAgentID: "AGENT-123",
				Account:       "agent123",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			agent, err := repo.GetByAccount(ctx, tt.account)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				if tt.expectedAgent != nil {
					assert.Equal(t, tt.expectedAgent.Account, agent.Account)
					assert.Equal(t, tt.expectedAgent.GlobalAgentID, agent.GlobalAgentID)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAgentRepository_BatchGetAgentsByAccounts(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)
	ctx := context.Background()

	tests := []struct {
		name           string
		accounts       []string
		setupMock      func(sqlmock.Sqlmock)
		expectedAgents []*entity.Agent
		expectedError  error
	}{
		{
			name:     "batch get agents by accounts successfully",
			accounts: []string{"agent123", "agent456"},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at"}).
					AddRow(1, 100, "AGENT-123", "agent123", "", time.Now(), time.Now(), time.Now()).
					AddRow(2, 100, "AGENT-456", "agent456", "", time.Now(), time.Now(), time.Now())
				mock.ExpectQuery("SELECT .* FROM `agents` WHERE account IN \\(.+\\) AND .*deleted_at.* IS NULL").
					WithArgs("agent123", "agent456").
					WillReturnRows(rows)
			},
			expectedAgents: []*entity.Agent{
				{ID: 1, Account: "agent123", GlobalAgentID: "AGENT-123"},
				{ID: 2, Account: "agent456", GlobalAgentID: "AGENT-456"},
			},
			expectedError: nil,
		},
		{
			name:           "empty accounts slice",
			accounts:       []string{},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedAgents: []*entity.Agent{},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			agents, err := repo.BatchGetAgentsByAccounts(ctx, tt.accounts)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, agents, len(tt.expectedAgents))
			}

			if len(tt.expectedAgents) > 0 {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}
