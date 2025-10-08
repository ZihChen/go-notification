package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
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

// FindByGlobalID 通過GlobalID查找會員訊息活動
func (r *MessageCampaignRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	var campaign models.MessageCampaign
	result := r.db.WithContext(ctx).Where("global_id = ?", globalID).First(&campaign)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return &entity.MessageCampaign{}, result.Error
	}

	return mapToDomainMessageCampaign(&campaign), nil
}

// FindAllWithOptions 根據篩選條件查找會員訊息活動
func (r *MessageCampaignRepository) FindAllWithOptions(
	ctx context.Context,
	query *dto.MessageCampaignsQuery,
) ([]*entity.MessageCampaign, int, error) {
	var campaigns []models.MessageCampaign
	var total int64

	builder := r.db.WithContext(ctx).Model(&models.MessageCampaign{})

	// 是否包含已刪除的記錄
	if query.IncludeDeleted {
		builder = builder.Unscoped()
	}

	// 商戶ID篩選
	if query.MerchantID > 0 {
		builder = builder.Where("merchant_id = ?", query.MerchantID)
	}

	// 類別篩選
	if query.Category != "" {
		builder = builder.Where("category = ?", query.Category)
	}

	// 項目篩選
	if query.Item != "" {
		builder = builder.Where("item = ?", query.Item)
	}

	// 狀態篩選
	if len(query.Status) > 0 {
		builder = builder.Where("status IN (?)", query.Status)
	}

	// 是否顯示自動發送
	if !query.ShowAutoSend {
		builder = builder.Where("auto_send = ?", false)
	}

	// 建立者篩選
	if query.CreatedBy != "" {
		builder = builder.Where("created_by = ?", query.CreatedBy)
	}

	// 時間範圍篩選
	if query.StartAt != "" {
		builder = builder.Where("created_at >= ?", query.StartAt)
	}
	if query.EndAt != "" {
		builder = builder.Where("created_at <= ?", query.EndAt)
	}

	// 計算總數
	if err := builder.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count error: %w", err)
	}

	// 查詢分頁數據
	offset := (query.Page - 1) * query.PageSize
	result := builder.
		Order("created_at DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&campaigns)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	// 在查詢到數據後再初始化 slice
	domainCampaigns := make([]*entity.MessageCampaign, len(campaigns))
	for i, campaign := range campaigns {
		domainCampaigns[i] = mapToDomainMessageCampaign(&campaign)
	}

	return domainCampaigns, int(total), nil
}

// FindActiveByFocus 根據焦點類型查找活躍的會員訊息活動
func (r *MessageCampaignRepository) FindActiveByFocus(
	ctx context.Context,
	focus string,
) ([]*entity.MessageCampaign, error) {
	var campaigns []models.MessageCampaign

	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("target = ? OR target = ?", focus, consts.TargetAll).
		Where("(send_start_time IS NULL OR send_start_time <= ?) AND (send_end_time IS NULL OR send_end_time >= ?)", now, now).
		Where("auto_send = ?", false).
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

// FindScheduledCampaigns 查找應該立即發送的排程活動
func (r *MessageCampaignRepository) FindScheduledCampaigns(
	ctx context.Context,
) ([]*entity.MessageCampaign, error) {
	var campaigns []models.MessageCampaign

	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("auto_send = ?", false).
		Where("send_start_time IS NOT NULL").
		Where("send_start_time <= ?", now).
		Where("status = ?", consts.MessageCampaignStatusScheduled).
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

// UpdateFields 更新指定字段
func (r *MessageCampaignRepository) UpdateFields(
	ctx context.Context,
	id uint64,
	updates map[string]interface{},
	includeDeleted bool,
) error {
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	builder := r.db.WithContext(ctx).Model(&models.MessageCampaign{})

	// 是否更新已刪除的記錄
	if includeDeleted {
		builder = builder.Unscoped()
	}

	result := builder.Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}
	return nil
}

// UpdateSentCount 更新實際發送人數
func (r *MessageCampaignRepository) UpdateSentCount(
	ctx context.Context,
	campaignID uint64,
	count int64,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.MessageCampaign{}).
		Where("id = ?", campaignID).
		Update("real_sent_count", count)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// UpdateStatus 更新活動狀態
func (r *MessageCampaignRepository) UpdateStatus(
	ctx context.Context,
	campaignID uint64,
	status string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.MessageCampaign{}).
		Where("id = ?", campaignID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// Delete 軟刪除會員訊息活動（設置 deleted_at 時間戳）
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
	var deletedAt *time.Time
	if campaign.DeletedAt.Valid {
		deletedTime := campaign.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &entity.MessageCampaign{
		ID:                campaign.ID,
		Category:          campaign.Category,
		Item:              campaign.Item,
		TriggerType:       campaign.TriggerType,
		Title:             campaign.Title,
		MerchantID:        campaign.MerchantID,
		GlobalID:          campaign.GlobalID,
		LegacyID:          campaign.LegacyID,
		Content:           campaign.Content,
		AppContent:        campaign.AppContent,
		NotificationTypes: campaign.NotificationTypes,
		Target:            campaign.Target,
		TargetDetail:      campaign.TargetDetail,
		Status:            campaign.Status,
		AutoSend:          campaign.AutoSend,
		Active:            campaign.Active,
		RealSentCount:     campaign.RealSentCount,
		SendStartTime:     campaign.SendStartTime,
		SendEndTime:       campaign.SendEndTime,
		CreatedBy:         campaign.CreatedBy,
		UpdatedBy:         campaign.UpdatedBy,
		CreatedAt:         campaign.CreatedAt,
		UpdatedAt:         campaign.UpdatedAt,
		DeletedAt:         deletedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBMessageCampaign(campaign *entity.MessageCampaign) *models.MessageCampaign {
	dbMessageCampaign := &models.MessageCampaign{
		ID:                campaign.ID,
		Category:          campaign.Category,
		Item:              campaign.Item,
		TriggerType:       campaign.TriggerType,
		Title:             campaign.Title,
		Content:           campaign.Content,
		AppContent:        campaign.AppContent,
		NotificationTypes: campaign.NotificationTypes,
		MerchantID:        campaign.MerchantID,
		GlobalID:          campaign.GlobalID,
		LegacyID:          campaign.LegacyID,
		Target:            campaign.Target,
		TargetDetail:      campaign.TargetDetail,
		Status:            campaign.Status,
		AutoSend:          campaign.AutoSend,
		Active:            campaign.Active,
		RealSentCount:     campaign.RealSentCount,
		SendStartTime:     campaign.SendStartTime,
		SendEndTime:       campaign.SendEndTime,
		CreatedBy:         campaign.CreatedBy,
		UpdatedBy:         campaign.UpdatedBy,
		CreatedAt:         campaign.CreatedAt,
		UpdatedAt:         campaign.UpdatedAt,
	}
	if campaign.DeletedAt != nil {
		dbMessageCampaign.DeletedAt = gorm.DeletedAt{
			Time:  *campaign.DeletedAt,
			Valid: true,
		}
	}
	return dbMessageCampaign
}

// FindAutoSettingsByMerchantID 查找商戶的自動設定
func (r *MessageCampaignRepository) FindAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.MessageCampaign, error) {
	var campaigns []models.MessageCampaign
	result := r.db.WithContext(ctx).
		Where("merchant_id = ? AND auto_send = ?", merchantID, true).
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

// FindAutoSettingByCategoryItemTrigger 根據 category、item 和 trigger_type 查找特定自動設定
func (r *MessageCampaignRepository) FindAutoSettingByCategoryItemTrigger(
	ctx context.Context,
	merchantID uint64,
	category string,
	item string,
	triggerType string,
) (*entity.MessageCampaign, error) {
	var campaign models.MessageCampaign
	result := r.db.WithContext(ctx).Where(
		"merchant_id = ? AND category = ? AND item = ? AND trigger_type = ? AND auto_send = ?",
		merchantID, category, item, triggerType, true,
	).First(&campaign)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return nil, result.Error
	}

	return mapToDomainMessageCampaign(&campaign), nil
}

// UpsertAutoSettings 批量新增或更新自動設定
func (r *MessageCampaignRepository) UpsertAutoSettings(
	ctx context.Context,
	campaigns []*entity.MessageCampaign,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, campaign := range campaigns {
			dbCampaign := mapToDBMessageCampaign(campaign)

			// 嘗試查找現有記錄
			var existing models.MessageCampaign
			err := tx.Where(
				"merchant_id = ? AND category = ? AND item = ? AND trigger_type = ? AND auto_send = ?",
				campaign.MerchantID,
				campaign.Category,
				campaign.Item,
				campaign.TriggerType,
				true,
			).First(&existing).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 不存在，建立新記錄
					now := time.Now()
					dbCampaign.CreatedAt = now
					dbCampaign.UpdatedAt = now
					if dbCampaign.SendStartTime == nil {
						dbCampaign.SendStartTime = &now
					}
					if err = tx.Create(dbCampaign).Error; err != nil {
						return fmt.Errorf("tx create error:%w", err)
					}
				} else {
					return err
				}
			} else {
				// 存在，更新記錄
				dbCampaign.ID = existing.ID
				dbCampaign.GlobalID = existing.GlobalID
				dbCampaign.CreatedAt = existing.CreatedAt
				dbCampaign.UpdatedAt = time.Now()
				if err = tx.Save(dbCampaign).Error; err != nil {
					return fmt.Errorf("tx save error:%w", err)
				}
			}
		}
		return nil
	})
}

// DeleteAutoSettingsByMerchantID 刪除商戶的所有自動設定
func (r *MessageCampaignRepository) DeleteAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) error {
	return r.db.WithContext(ctx).
		Where("merchant_id = ? AND auto_send = ?", merchantID, true).
		Delete(&models.MessageCampaign{}).
		Error
}

// GetByGlobalID 通過GlobalID獲取會員訊息活動（用於資料遷移）
func (r *MessageCampaignRepository) GetByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	var campaign models.MessageCampaign
	result := r.db.WithContext(ctx).Where("global_id = ?", globalID).First(&campaign)
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainMessageCampaign(&campaign), nil
}

// GetByLegacyID 通過舊系統ID獲取會員訊息活動（用於資料遷移）
func (r *MessageCampaignRepository) GetByLegacyID(
	ctx context.Context,
	legacyID uint,
) (*entity.MessageCampaign, error) {
	var campaign models.MessageCampaign
	result := r.db.WithContext(ctx).Where("legacy_id = ?", legacyID).First(&campaign)
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainMessageCampaign(&campaign), nil
}

// FindWithLegacyIDPaginated 分頁查詢有 LegacyID 的會員訊息活動（用於資料遷移）
func (r *MessageCampaignRepository) FindWithLegacyIDPaginated(
	ctx context.Context,
	limit, offset int,
) ([]*entity.MessageCampaign, error) {
	var campaigns []models.MessageCampaign
	result := r.db.WithContext(ctx).
		Where("legacy_id IS NOT NULL").
		Order("legacy_id ASC").
		Limit(limit).
		Offset(offset).
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

// CountWithLegacyID 計算有 LegacyID 的會員訊息活動數量（用於資料遷移）
func (r *MessageCampaignRepository) CountWithLegacyID(
	ctx context.Context,
) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.MessageCampaign{}).
		Where("legacy_id IS NOT NULL").
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}
