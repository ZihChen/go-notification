package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// PlayerMessageRepository GORM實現的會員訊息資料庫
type PlayerMessageRepository struct {
	db *gorm.DB
}

// NewPlayerMessageRepository 創建會員訊息資料庫
func NewPlayerMessageRepository(db *gorm.DB) repositoryport.PlayerMessageRepository {
	return &PlayerMessageRepository{db: db}
}

// FindByID 通過ID查找會員訊息
func (r *PlayerMessageRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.PlayerMessage, error) {
	var message models.PlayerMessage
	result := r.db.WithContext(ctx).First(&message, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return &entity.PlayerMessage{}, result.Error
	}

	return mapToDomainPlayerMessage(&message), nil
}

// FindByPlayerID 通過玩家ID查找訊息列表（支援分頁）
func (r *PlayerMessageRepository) FindByPlayerID(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) ([]*entity.PlayerMessage, int, error) {
	var messages []models.PlayerMessage
	var total int64
	domainMessages := make([]*entity.PlayerMessage, len(messages))

	// 計算總數
	if err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ?", globalPlayerID).
		Count(&total).Error; err != nil {
		return domainMessages, 0, err
	}

	// 查詢分頁數據
	offset := (page - 1) * pageSize
	result := r.db.WithContext(ctx).
		Where("global_player_id = ?", globalPlayerID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages)

	if result.Error != nil {
		return domainMessages, 0, result.Error
	}

	for i, message := range messages {
		domainMessages[i] = mapToDomainPlayerMessage(&message)
	}

	return domainMessages, int(total), nil
}

// GetPlayerMessageStats 獲取玩家訊息統計
func (r *PlayerMessageRepository) GetPlayerMessageStats(
	ctx context.Context,
	globalPlayerID string,
) (*entity.PlayerMessageStats, error) {
	var totalCount int64
	var readCount int64

	// 查詢總數
	if err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ?", globalPlayerID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// 查詢已讀數量
	if err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ? AND is_read = ?", globalPlayerID, true).
		Count(&readCount).Error; err != nil {
		return nil, err
	}

	// 計算未讀數量
	unreadCount := totalCount - readCount

	stats := &entity.PlayerMessageStats{
		TotalCount:  int(totalCount),  // 轉換為 int
		ReadCount:   int(readCount),   // 轉換為 int
		UnreadCount: int(unreadCount), // 轉換為 int
	}

	return stats, nil
}

// Create 創建會員訊息
func (r *PlayerMessageRepository) Create(ctx context.Context, message *entity.PlayerMessage) error {
	messageModel := mapToDBPlayerMessage(message)
	result := r.db.WithContext(ctx).Create(messageModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	message.ID = messageModel.ID

	return nil
}

// CreateBatch 批量創建會員訊息
func (r *PlayerMessageRepository) CreateBatch(
	ctx context.Context,
	messages []*entity.PlayerMessage,
) error {
	if len(messages) == 0 {
		return nil
	}

	messageModels := make([]*models.PlayerMessage, len(messages))
	for i, message := range messages {
		messageModels[i] = mapToDBPlayerMessage(message)
	}

	// 使用批量插入，每次插入1000筆
	batchSize := 1000
	for i := 0; i < len(messageModels); i += batchSize {
		end := i + batchSize
		if end > len(messageModels) {
			end = len(messageModels)
		}

		batch := messageModels[i:end]
		if err := r.db.WithContext(ctx).Create(&batch).Error; err != nil {
			return err
		}
	}

	return nil
}

// MarkAsRead 標記訊息為已讀
func (r *PlayerMessageRepository) MarkAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	result := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("id = ? AND global_player_id = ?", messageID, globalPlayerID).
		Update("is_read", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found or already read")
	}

	return nil
}

// CheckMessageExists 檢查訊息是否已存在
func (r *PlayerMessageRepository) CheckMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ? AND campaign_id = ?", globalPlayerID, campaignID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// 將DB模型映射到領域模型
func mapToDomainPlayerMessage(message *models.PlayerMessage) *entity.PlayerMessage {
	return &entity.PlayerMessage{
		ID:             message.ID,
		GlobalPlayerID: message.GlobalPlayerID,
		PlayerID:       message.PlayerID,
		CampaignID:     message.CampaignID,
		IsRead:         message.IsRead,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBPlayerMessage(message *entity.PlayerMessage) *models.PlayerMessage {
	return &models.PlayerMessage{
		ID:             message.ID,
		GlobalPlayerID: message.GlobalPlayerID,
		PlayerID:       message.PlayerID,
		CampaignID:     message.CampaignID,
		IsRead:         message.IsRead,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}
