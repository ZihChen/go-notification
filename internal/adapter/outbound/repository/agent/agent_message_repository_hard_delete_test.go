package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestAgentMessageRepository_BatchHardDeleteByCampaignIDs(t *testing.T) {
	db, mock, mockDB := setupAgentMessageMockDB(t)
	defer func() {
		_ = mockDB.Close()
	}()

	repo := &AgentMessageRepository{db: db}

	t.Run("successful batch hard delete", func(t *testing.T) {
		campaignIDs := []uint64{1, 2, 3}

		// Mock DELETE query with Unscoped()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `agent_messages` WHERE agent_campaign_id IN (?,?,?)")).
			WithArgs(1, 2, 3).
			WillReturnResult(sqlmock.NewResult(0, 5))

		// 假設刪除了5條記錄
		mock.ExpectCommit()

		err := repo.BatchHardDeleteByCampaignIDs(context.Background(), campaignIDs)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty campaign IDs", func(t *testing.T) {
		err := repo.BatchHardDeleteByCampaignIDs(context.Background(), []uint64{})
		assert.NoError(t, err)
		// 沒有SQL執行，所以不需要mock期望
	})

	t.Run("database error", func(t *testing.T) {
		campaignIDs := []uint64{1}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `agent_messages` WHERE agent_campaign_id IN (?)")).
			WithArgs(1).
			WillReturnError(errors.New("database error"))
		mock.ExpectRollback()

		err := repo.BatchHardDeleteByCampaignIDs(context.Background(), campaignIDs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch hard delete agent messages by campaign IDs failed")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
