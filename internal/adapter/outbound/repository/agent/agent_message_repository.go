package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/aggregate"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentMessageRepository struct {
	db *gorm.DB
}

func NewAgentMessageRepository(db *gorm.DB) repository.AgentMessageRepository {
	return &AgentMessageRepository{db: db}
}

// Create 創建代理站內信
func (r *AgentMessageRepository) Create(
	ctx context.Context,
	message *entity.AgentMessage,
) (*entity.AgentMessage, error) {
	messageModel := &models.AgentMessage{
		AgentCampaignID: message.AgentCampaignID,
		AgentID:         message.AgentID,
		IsRead:          message.IsRead,
		ReadAt:          message.ReadAt,
	}

	if err := r.db.WithContext(ctx).Create(messageModel).Error; err != nil {
		return nil, fmt.Errorf("create agent message failed: %w", err)
	}

	message.ID = messageModel.ID
	message.CreatedAt = messageModel.CreatedAt
	message.UpdatedAt = messageModel.UpdatedAt
	return message, nil
}

// GetByID 根據ID獲取代理站內信
func (r *AgentMessageRepository) GetByID(
	ctx context.Context,
	id uint64,
) (*entity.AgentMessage, error) {
	var messageModel models.AgentMessage
	if err := r.db.WithContext(ctx).First(&messageModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent message by id failed: %w", err)
	}

	return r.modelToEntity(&messageModel), nil
}

// Update 更新代理站內信
func (r *AgentMessageRepository) Update(ctx context.Context, message *entity.AgentMessage) error {
	messageModel := &models.AgentMessage{
		ID:              message.ID,
		AgentCampaignID: message.AgentCampaignID,
		AgentID:         message.AgentID,
		IsRead:          message.IsRead,
		ReadAt:          message.ReadAt,
	}

	if err := r.db.WithContext(ctx).Save(messageModel).Error; err != nil {
		return fmt.Errorf("update agent message failed: %w", err)
	}

	message.UpdatedAt = messageModel.UpdatedAt
	return nil
}

// Delete 刪除代理站內信 (軟刪除)
func (r *AgentMessageRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&models.AgentMessage{}, id).Error; err != nil {
		return fmt.Errorf("delete agent message failed: %w", err)
	}
	return nil
}

// DeleteByCampaignID 根據活動ID刪除所有關聯的代理站內信 (硬刪除)
func (r *AgentMessageRepository) DeleteByCampaignID(ctx context.Context, campaignID uint64) error {
	// 使用原生SQL DELETE語句進行硬刪除
	deleteSQL := "DELETE FROM agent_messages WHERE agent_campaign_id = ?"
	
	if err := r.db.WithContext(ctx).Exec(deleteSQL, campaignID).Error; err != nil {
		return fmt.Errorf("delete agent messages by campaign ID failed: %w", err)
	}
	return nil
}

// BatchDeleteByCampaignIDs 根據活動ID列表批量刪除所有關聯的代理站內信 (軟刪除)
func (r *AgentMessageRepository) BatchDeleteByCampaignIDs(
	ctx context.Context,
	campaignIDs []uint64,
) error {
	if len(campaignIDs) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Where("agent_campaign_id IN ?", campaignIDs).
		Delete(&models.AgentMessage{})
	if result.Error != nil {
		return fmt.Errorf("batch delete agent messages by campaign IDs failed: %w", result.Error)
	}
	return nil
}

// BatchHardDeleteByCampaignIDs 根據活動ID列表批量硬刪除所有關聯的代理站內信 (永久刪除)
// 使用遞迴處理每個campaignID，分頁批次刪除，最大化性能
func (r *AgentMessageRepository) BatchHardDeleteByCampaignIDs(
	ctx context.Context,
	campaignIDs []uint64,
) error {
	if len(campaignIDs) == 0 {
		return nil
	}

	// 遞迴處理每一個campaignID，避免WHERE IN條件
	for _, campaignID := range campaignIDs {
		if err := r.deleteByCampaignIDWithPagination(ctx, campaignID); err != nil {
			return fmt.Errorf("delete campaign %d messages failed: %w", campaignID, err)
		}
	}

	return nil
}

// deleteByCampaignIDWithPagination 單一campaignID的分頁批次刪除
func (r *AgentMessageRepository) deleteByCampaignIDWithPagination(
	ctx context.Context,
	campaignID uint64,
) error {
	const batchSize = 5000 // 每批次5000筆

	for {
		// 使用原生SQL獲取要刪除的ID（分頁）
		query := "SELECT id FROM agent_messages WHERE agent_campaign_id = ? LIMIT ?"

		var messageIDs []uint64
		rows, err := r.db.WithContext(ctx).Raw(query, campaignID, batchSize).Rows()
		if err != nil {
			return fmt.Errorf("query message IDs failed: %w", err)
		}

		// 讀取ID列表
		for rows.Next() {
			var id uint64
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan message ID failed: %w", err)
			}
			messageIDs = append(messageIDs, id)
		}
		_ = rows.Close()

		// 如果沒有更多記錄，退出循環
		if len(messageIDs) == 0 {
			break
		}

		// 使用原生SQL批次刪除（最高性能）
		if err = r.deleteMessagesByIDs(ctx, messageIDs); err != nil {
			return fmt.Errorf("batch delete message IDs failed: %w", err)
		}

		// 如果這批記錄數少於批次大小，表示已經處理完成
		if len(messageIDs) < batchSize {
			break
		}
	}

	return nil
}

// deleteMessagesByIDs 使用原生SQL批次刪除指定ID的記錄
func (r *AgentMessageRepository) deleteMessagesByIDs(
	ctx context.Context,
	messageIDs []uint64,
) error {
	if len(messageIDs) == 0 {
		return nil
	}

	// 構建原生SQL DELETE語句
	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	deleteSQL := fmt.Sprintf("DELETE FROM agent_messages WHERE id IN (%s)",
		strings.Join(placeholders, ","))

	if err := r.db.WithContext(ctx).Exec(deleteSQL, args...).Error; err != nil {
		return fmt.Errorf("execute batch delete SQL failed: %w", err)
	}

	return nil
}

// CreateBatch 批量創建代理站內信 (使用UPSERT避免重複寫入)
func (r *AgentMessageRepository) CreateBatch(
	ctx context.Context,
	messages []*entity.AgentMessage,
) error {
	if len(messages) == 0 {
		return nil
	}

	messageModels := make([]*models.AgentMessage, len(messages))
	for i, message := range messages {
		messageModels[i] = &models.AgentMessage{
			AgentCampaignID: message.AgentCampaignID,
			AgentID:         message.AgentID,
			IsRead:          message.IsRead,
			ReadAt:          message.ReadAt,
		}
	}

	// 使用 ON DUPLICATE KEY UPDATE 避免重複插入
	// 當 (agent_campaign_id, agent_id) 組合已存在時，更新 updated_at 時間戳
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_campaign_id"}, {Name: "agent_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"updated_at": gorm.Expr("VALUES(updated_at)"),
		}),
	}).CreateInBatches(messageModels, 100).Error; err != nil {
		return fmt.Errorf("create batch agent messages failed: %w", err)
	}

	// 更新實體的ID和時間戳
	// 注意：對於重複的記錄，ID可能不會更新，需要特殊處理
	for i, model := range messageModels {
		messages[i].ID = model.ID
		messages[i].CreatedAt = model.CreatedAt
		messages[i].UpdatedAt = model.UpdatedAt
	}

	return nil
}

// CheckMessageExistsBatch 批量檢查訊息是否存在
func (r *AgentMessageRepository) CheckMessageExistsBatch(
	ctx context.Context,
	agentIDs []uint64,
	campaignID uint64,
) (map[uint64]bool, error) {
	if len(agentIDs) == 0 {
		return make(map[uint64]bool), nil
	}

	type Result struct {
		AgentID uint64 `json:"agent_id"`
	}

	var results []Result
	if err := r.db.WithContext(ctx).
		Model(&models.AgentMessage{}).
		Select("agent_id").
		Where("agent_id IN ? AND agent_campaign_id = ?", agentIDs, campaignID).
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("check message exists batch failed: %w", err)
	}

	existsMap := make(map[uint64]bool)
	// 初始化所有agentID為false
	for _, agentID := range agentIDs {
		existsMap[agentID] = false
	}
	// 設置存在的為true
	for _, result := range results {
		existsMap[result.AgentID] = true
	}

	return existsMap, nil
}

// CheckCampaignMessageExistsBatch 批次檢查單個代理的多個活動訊息是否存在
func (r *AgentMessageRepository) CheckCampaignMessageExistsBatch(
	ctx context.Context,
	agentID uint64,
	campaignIDs []uint64,
) (map[uint64]bool, error) {
	if len(campaignIDs) == 0 {
		return make(map[uint64]bool), nil
	}

	type Result struct {
		AgentCampaignID uint64 `json:"agent_campaign_id"`
	}

	var results []Result
	if err := r.db.WithContext(ctx).
		Model(&models.AgentMessage{}).
		Select("agent_campaign_id").
		Where("agent_id = ? AND agent_campaign_id IN ?", agentID, campaignIDs).
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("check campaign message exists batch failed: %w", err)
	}

	existsMap := make(map[uint64]bool)
	// 初始化所有campaignID為false
	for _, campaignID := range campaignIDs {
		existsMap[campaignID] = false
	}
	// 設置存在的為true
	for _, result := range results {
		existsMap[result.AgentCampaignID] = true
	}

	return existsMap, nil
}

// ListByAgent 根據代理查詢訊息列表（包含活動資訊）
func (r *AgentMessageRepository) ListByAgent(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) ([]*entity.AgentMessage, int, error) {
	var aggregates []aggregate.AgentMessageAggregate
	var total int64

	// 構建基礎查詢，加入 JOIN
	db := r.db.WithContext(ctx).
		Table("agent_messages").
		Select(`
			agent_messages.id,
			agent_messages.agent_campaign_id,
			agent_messages.agent_id,
			agent_messages.is_read,
			agent_messages.read_at,
			agent_messages.created_at,
			agent_messages.updated_at,
			agent_campaigns.title as campaign_title,
			agent_campaigns.content as campaign_content,
			agents.global_agent_id
		`).
		Joins("LEFT JOIN agent_campaigns ON agent_messages.agent_campaign_id = agent_campaigns.id").
		Joins("LEFT JOIN agents ON agent_messages.agent_id = agents.id").
		Where("agent_messages.deleted_at IS NULL")

	// 只保留 agent_id 查詢條件
	if query.AgentID > 0 {
		db = db.Where("agent_messages.agent_id = ?", query.AgentID)
	}

	// 獲取總數
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agent messages failed: %w", err)
	}

	// 固定排序：按創建日期由近到遠
	db = db.Order("agent_messages.created_at DESC")

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// 執行查詢
	if err := db.Find(&aggregates).Error; err != nil {
		return nil, 0, fmt.Errorf("list agent messages failed: %w", err)
	}

	messages := make([]*entity.AgentMessage, len(aggregates))
	for i, ag := range aggregates {
		messages[i] = r.aggregateToEntity(&ag)
	}

	return messages, int(total), nil
}

// MarkAsRead 標記訊息為已讀
func (r *AgentMessageRepository) MarkAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&models.AgentMessage{}).
		Where("id = ? AND agent_id = ? AND is_read = false", messageID, agentID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"read_at":    &now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("mark message as read failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found or already read")
	}

	return nil
}

// ExistsMessage 檢查特定代理是否已有特定活動的訊息
func (r *AgentMessageRepository) ExistsMessage(
	ctx context.Context,
	campaignID uint64,
	agentID uint64,
) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AgentMessage{}).
		Where("agent_campaign_id = ? AND agent_id = ?", campaignID, agentID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check message exists failed: %w", err)
	}
	return count > 0, nil
}

// GetMessageStats 獲取代理訊息統計
func (r *AgentMessageRepository) GetMessageStats(
	ctx context.Context,
	agentID uint64,
) (*dto.AgentMessageStats, error) {
	type statsResult struct {
		TotalCount  int64 `json:"total_count"`
		ReadCount   int64 `json:"read_count"`
		UnreadCount int64 `json:"unread_count"`
	}

	var result statsResult

	if err := r.db.WithContext(ctx).
		Model(&models.AgentMessage{}).
		Select(`
			COUNT(*) as total_count,
			SUM(CASE WHEN is_read = true THEN 1 ELSE 0 END) as read_count,
			SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) as unread_count
		`).
		Where("agent_id = ?", agentID).
		Where("deleted_at IS NULL").
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("get message stats failed: %w", err)
	}

	return &dto.AgentMessageStats{
		TotalCount:  result.TotalCount,
		ReadCount:   result.ReadCount,
		UnreadCount: result.UnreadCount,
	}, nil
}

// modelToEntity 將模型轉換為實體
func (r *AgentMessageRepository) modelToEntity(model *models.AgentMessage) *entity.AgentMessage {
	return &entity.AgentMessage{
		ID:              model.ID,
		AgentCampaignID: model.AgentCampaignID,
		AgentID:         model.AgentID,
		IsRead:          model.IsRead,
		ReadAt:          model.ReadAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

// aggregateToEntity 將聚合根轉換為實體
func (r *AgentMessageRepository) aggregateToEntity(
	aggregate *aggregate.AgentMessageAggregate,
) *entity.AgentMessage {
	return &entity.AgentMessage{
		ID:              aggregate.ID,
		AgentCampaignID: aggregate.AgentCampaignID,
		AgentID:         aggregate.AgentID,
		IsRead:          aggregate.IsRead,
		ReadAt:          aggregate.ReadAt,
		CreatedAt:       aggregate.CreatedAt,
		UpdatedAt:       aggregate.UpdatedAt,
		// 聚合的活動信息欄位
		CampaignTitle:   aggregate.CampaignTitle,
		CampaignContent: aggregate.CampaignContent,
		GlobalAgentID:   aggregate.GlobalAgentID,
	}
}
