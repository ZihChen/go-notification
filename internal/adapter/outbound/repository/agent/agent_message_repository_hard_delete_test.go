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

		// Mock the sequential processing for each campaign
		// Campaign 1: Has 2 messages (< 5000, so loop ends after first batch)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM agent_messages WHERE agent_campaign_id = ? LIMIT ?")).
			WithArgs(1, 5000).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(101).AddRow(102))
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM agent_messages WHERE id IN (?,?)")).
			WithArgs(101, 102).
			WillReturnResult(sqlmock.NewResult(0, 2))

		// Campaign 2: No messages found
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM agent_messages WHERE agent_campaign_id = ? LIMIT ?")).
			WithArgs(2, 5000).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
			// empty result

		// Campaign 3: Has 3 messages (< 5000, so loop ends after first batch)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM agent_messages WHERE agent_campaign_id = ? LIMIT ?")).
			WithArgs(3, 5000).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(201).AddRow(202).AddRow(203))
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM agent_messages WHERE id IN (?,?,?)")).
			WithArgs(201, 202, 203).
			WillReturnResult(sqlmock.NewResult(0, 3))

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

		// Mock the SELECT query that will fail
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM agent_messages WHERE agent_campaign_id = ? LIMIT ?")).
			WithArgs(1, 5000).
			WillReturnError(errors.New("database error"))

		err := repo.BatchHardDeleteByCampaignIDs(context.Background(), campaignIDs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete campaign 1 messages failed")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
