package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/model"
	domainModel "github.com/jvdiamondtech/ms-notification-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/repository"
)

// MessageCampaignRepository GORM實現的會員訊息活動資料庫
type MessageCampaignRepository struct {
	db *gorm.DB
}

// NewMessageCampaignRepository 創建會員訊息活動資料庫
func NewMessageCampaignRepository(db *gorm.DB) repository.MessageCampaignRepository {
	return &MessageCampaignRepository{db: db}
}

// FindByID 通過ID查找會員訊息活動
func (r *MessageCampaignRepository) FindByID(ctx context.Context, id uint64) (*domainModel.MessageCampaign, error) {
	var campaign model.MessageCampaign
	result := r.db.WithContext(ctx).First(&campaign, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return nil, result.Error
	}

	return mapToDomainMessageCampaign(&campaign), nil
}

// FindAll 查找所有會員訊息活動（支援分頁）
func (r *MessageCampaignRepository) FindAll(ctx context.Context, page, pageSize int) ([]*domainModel.MessageCampaign, int, error) {
	var campaigns []model.MessageCampaign
	var total int64

	// 計算總數
	if err := r.db.WithContext(ctx).Model(&model.MessageCampaign{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查詢分頁數據
	offset := (page - 1) * pageSize
	result := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&campaigns)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	domainCampaigns := make([]*domainModel.MessageCampaign, len(campaigns))
	for i, campaign := range campaigns {
		domainCampaigns[i] = mapToDomainMessageCampaign(&campaign)
	}

	return domainCampaigns, int(total), nil
}

// FindActiveByFocus 根據焦點類型查找活躍的會員訊息活動
func (r *MessageCampaignRepository) FindActiveByFocus(ctx context.Context, focus uint8) ([]*domainModel.MessageCampaign, error) {
	var campaigns []model.MessageCampaign

	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("focus = ? OR focus = ?", focus, domainModel.FocusAll).
		Where("(send_start_time IS NULL OR send_start_time <= ?) AND (send_end_time IS NULL OR send_end_time >= ?)", now, now).
		Find(&campaigns)

	if result.Error != nil {
		return nil, result.Error
	}

	domainCampaigns := make([]*domainModel.MessageCampaign, len(campaigns))
	for i, campaign := range campaigns {
		domainCampaigns[i] = mapToDomainMessageCampaign(&campaign)
	}

	return domainCampaigns, nil
}

// Create 創建會員訊息活動
func (r *MessageCampaignRepository) Create(ctx context.Context, campaign *domainModel.MessageCampaign) error {
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
func (r *MessageCampaignRepository) Update(ctx context.Context, campaign *domainModel.MessageCampaign) error {
	campaignModel := mapToDBMessageCampaign(campaign)
	result := r.db.WithContext(ctx).Save(campaignModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除會員訊息活動
func (r *MessageCampaignRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.MessageCampaign{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// 將DB模型映射到領域模型
func mapToDomainMessageCampaign(campaign *model.MessageCampaign) *domainModel.MessageCampaign {
	return &domainModel.MessageCampaign{
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
func mapToDBMessageCampaign(campaign *domainModel.MessageCampaign) *model.MessageCampaign {
	return &model.MessageCampaign{
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
