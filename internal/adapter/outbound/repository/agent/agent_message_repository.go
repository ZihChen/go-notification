package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
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

// CreateBatch 批量創建代理站內信
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

	// 使用批量插入
	if err := r.db.WithContext(ctx).CreateInBatches(messageModels, 100).Error; err != nil {
		return fmt.Errorf("create batch agent messages failed: %w", err)
	}

	// 更新實體的ID和時間戳
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

// ListByAgent 根據代理查詢訊息列表
func (r *AgentMessageRepository) ListByAgent(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) ([]*entity.AgentMessage, int, error) {
	var messageModels []models.AgentMessage
	var total int64

	db := r.db.WithContext(ctx).Model(&models.AgentMessage{})

	// 添加查詢條件
	if query.AgentID > 0 {
		db = db.Where("agent_id = ?", query.AgentID)
	}

	if query.CampaignID > 0 {
		db = db.Where("agent_campaign_id = ?", query.CampaignID)
	}

	if query.IsRead != nil {
		db = db.Where("is_read = ?", *query.IsRead)
	}

	if !query.StartDate.IsZero() {
		db = db.Where("created_at >= ?", query.StartDate)
	}

	if !query.EndDate.IsZero() {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	// 獲取總數
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agent messages failed: %w", err)
	}

	// 添加排序和分頁
	if query.OrderBy != "" {
		orderDirection := "DESC"
		if query.OrderDirection == "ASC" {
			orderDirection = "ASC"
		}
		db = db.Order(fmt.Sprintf("%s %s", query.OrderBy, orderDirection))
	} else {
		db = db.Order("created_at DESC")
	}

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// 執行查詢
	if err := db.Find(&messageModels).Error; err != nil {
		return nil, 0, fmt.Errorf("list agent messages failed: %w", err)
	}

	messages := make([]*entity.AgentMessage, len(messageModels))
	for i, model := range messageModels {
		messages[i] = r.modelToEntity(&model)
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
