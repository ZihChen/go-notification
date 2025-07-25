package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// MessageCampaignRepository GORM實現的會員訊息活動資料庫
type MessageCampaignRepository struct {
	db *gorm.DB
}

// NewMessageCampaignRepository 創建會員訊息活動資料庫
func NewMessageCampaignRepository(db *gorm.DB) repositoryport.MessageCampaignRepository {
	return &MessageCampaignRepository{db: db}
}

// FindByID 通過ID查找會員訊息活動
func (r *MessageCampaignRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.MessageCampaign, error) {
	var campaign models.MessageCampaign
	result := r.db.WithContext(ctx).First(&campaign, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return &entity.MessageCampaign{}, result.Error
	}

	return mapToDomainMessageCampaign(&campaign), nil
}

// FindAll 查找所有會員訊息活動（支援分頁）
func (r *MessageCampaignRepository) FindAll(
	ctx context.Context,
	page, pageSize int,
) ([]*entity.MessageCampaign, int, error) {
	var campaigns []models.MessageCampaign
	var total int64
	domainCampaigns := make([]*entity.MessageCampaign, len(campaigns))

	// 計算總數
	if err := r.db.WithContext(ctx).Model(&models.MessageCampaign{}).Count(&total).Error; err != nil {
		return domainCampaigns, 0, err
	}

	// 查詢分頁數據
	offset := (page - 1) * pageSize
	result := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&campaigns)

	if result.Error != nil {
		return domainCampaigns, 0, result.Error
	}

	for i, campaign := range campaigns {
		domainCampaigns[i] = mapToDomainMessageCampaign(&campaign)
	}

	return domainCampaigns, int(total), nil
}

// FindActiveByFocus 根據焦點類型查找活躍的會員訊息活動
func (r *MessageCampaignRepository) FindActiveByFocus(
	ctx context.Context,
	focus uint8,
) ([]*entity.MessageCampaign, error) {
	var campaigns []models.MessageCampaign

	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("focus = ? OR focus = ?", focus, entity.FocusAll).
		Where("(send_start_time IS NULL OR send_start_time <= ?) AND (send_end_time IS NULL OR send_end_time >= ?)", now, now).
		Find(&campaigns)

	if result.Error != nil {
		return nil, result.Error
	}

	domainCampaigns := make([]*entity.MessageCampaign, len(campaigns))
	for i, campaign := range campaigns {
		domainCampaigns[i] = mapToDomainMessageCampaign(&campaign)
	}

	return domainCampaigns, nil
}

// Create 創建會員訊息活動
func (r *MessageCampaignRepository) Create(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	campaignModel := mapToDBMessageCampaign(campaign)
	result := r.db.WithContext(ctx).Create(campaignModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	campaign.ID = campaignModel.ID

	return nil
}

// Update 更新會員訊息活動
func (r *MessageCampaignRepository) Update(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	campaignModel := mapToDBMessageCampaign(campaign)
	result := r.db.WithContext(ctx).Save(campaignModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除會員訊息活動
func (r *MessageCampaignRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.MessageCampaign{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// 將DB模型映射到領域模型
func mapToDomainMessageCampaign(campaign *models.MessageCampaign) *entity.MessageCampaign {
	return &entity.MessageCampaign{
		ID:               campaign.ID,
		Category:         campaign.Category,
		Item:             campaign.Item,
		Title:            campaign.Title,
		Content:          campaign.Content,
		Focus:            campaign.Focus,
		AutoSend:         campaign.AutoSend,
		TotalTargetCount: campaign.TotalTargetCount,
		RealSentCount:    campaign.RealSentCount,
		SendStartTime:    campaign.SendStartTime,
		SendEndTime:      campaign.SendEndTime,
		CreatedBy:        campaign.CreatedBy,
		UpdatedBy:        campaign.UpdatedBy,
		CreatedAt:        campaign.CreatedAt,
		UpdatedAt:        campaign.UpdatedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBMessageCampaign(campaign *entity.MessageCampaign) *models.MessageCampaign {
	return &models.MessageCampaign{
		ID:               campaign.ID,
		Category:         campaign.Category,
		Item:             campaign.Item,
		Title:            campaign.Title,
		Content:          campaign.Content,
		Focus:            campaign.Focus,
		AutoSend:         campaign.AutoSend,
		TotalTargetCount: campaign.TotalTargetCount,
		RealSentCount:    campaign.RealSentCount,
		SendStartTime:    campaign.SendStartTime,
		SendEndTime:      campaign.SendEndTime,
		CreatedBy:        campaign.CreatedBy,
		UpdatedBy:        campaign.UpdatedBy,
		CreatedAt:        campaign.CreatedAt,
		UpdatedAt:        campaign.UpdatedAt,
	}
}
