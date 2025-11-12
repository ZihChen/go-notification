package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

type AgentCampaignRepository struct {
	db *gorm.DB
}

func NewAgentCampaignRepository(db *gorm.DB) repository.AgentCampaignRepository {
	return &AgentCampaignRepository{db: db}
}

// Create 創建代理訊息活動
func (r *AgentCampaignRepository) Create(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (*entity.AgentCampaign, error) {
	model := r.entityToModel(campaign)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, fmt.Errorf("create agent campaign failed: %w", err)
	}

	campaign.ID = model.ID
	campaign.CreatedAt = model.CreatedAt
	campaign.UpdatedAt = model.UpdatedAt
	return campaign, nil
}

// GetByID 根據ID獲取代理訊息活動
func (r *AgentCampaignRepository) GetByID(
	ctx context.Context,
	id uint64,
) (*entity.AgentCampaign, error) {
	var campaignModel models.AgentCampaign
	if err := r.db.WithContext(ctx).First(&campaignModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent campaign by id failed: %w", err)
	}

	return r.modelToEntity(&campaignModel), nil
}

// Update 更新代理訊息活動
func (r *AgentCampaignRepository) Update(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	campaignModel := r.entityToModel(campaign)

	if err := r.db.WithContext(ctx).Save(campaignModel).Error; err != nil {
		return fmt.Errorf("update agent campaign failed: %w", err)
	}

	campaign.UpdatedAt = campaignModel.UpdatedAt
	return nil
}

// Delete 刪除代理訊息活動 (軟刪除)
func (r *AgentCampaignRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&models.AgentCampaign{}, id).Error; err != nil {
		return fmt.Errorf("delete agent campaign failed: %w", err)
	}
	return nil
}

// UpdateFields 更新代理訊息活動特定欄位
func (r *AgentCampaignRepository) UpdateFields(
	ctx context.Context,
	id uint64,
	columns map[string]interface{},
) error {
	if len(columns) == 0 {
		return fmt.Errorf("no fields to update")
	}
	builder := r.db.WithContext(ctx).Model(&models.AgentCampaign{})
	result := builder.Where("id = ?", id).
		Updates(columns)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}
	return nil
}

// List 分頁查詢代理訊息活動列表
func (r *AgentCampaignRepository) List(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) ([]*entity.AgentCampaign, int, error) {
	var campaignModels []models.AgentCampaign
	var total int64

	db := r.db.WithContext(ctx).Model(&models.AgentCampaign{}).
		Where("merchant_id = ?", query.MerchantID)

	// 添加查詢條件
	if len(query.Status) > 0 {
		db = db.Where("status IN ?", query.Status)
	}

	if query.CreatedBy != "" {
		db = db.Where("created_by = ?", query.CreatedBy)
	}

	if query.StartAt != "" {
		db = db.Where("created_at >= ?", query.StartAt)
	}

	if query.EndAt != "" {
		db = db.Where("created_at <= ?", query.EndAt)
	}

	// 獲取總數
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agent campaigns failed: %w", err)
	}

	// 添加排序和分頁
	db = db.Order("created_at DESC")

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// 執行查詢
	if err := db.Find(&campaignModels).Error; err != nil {
		return nil, 0, fmt.Errorf("list agent campaigns failed: %w", err)
	}

	campaigns := make([]*entity.AgentCampaign, len(campaignModels))
	for i, model := range campaignModels {
		campaigns[i] = r.modelToEntity(&model)
	}

	return campaigns, int(total), nil
}

// GetScheduledCampaigns 獲取待發送的排程活動
func (r *AgentCampaignRepository) GetScheduledCampaigns(
	ctx context.Context,
	currentTime time.Time,
) ([]*entity.AgentCampaign, error) {
	var campaignModels []models.AgentCampaign

	if err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", "scheduled", currentTime).
		Order("scheduled_at ASC").
		Find(&campaignModels).Error; err != nil {
		return nil, fmt.Errorf("get scheduled campaigns failed: %w", err)
	}

	campaigns := make([]*entity.AgentCampaign, len(campaignModels))
	for i, model := range campaignModels {
		campaigns[i] = r.modelToEntity(&model)
	}

	return campaigns, nil
}

// modelToEntity 將模型轉換為實體
func (r *AgentCampaignRepository) modelToEntity(model *models.AgentCampaign) *entity.AgentCampaign {
	campaign := &entity.AgentCampaign{
		ID:            model.ID,
		MerchantID:    model.MerchantID,
		Title:         model.Title,
		Content:       model.Content,
		ScheduledAt:   model.ScheduledAt,
		Status:        model.Status,
		TargetType:    model.TargetType,
		TargetCount:   model.TargetCount,
		RealSentCount: model.RealSentCount,
		CreatedBy:     model.CreatedBy,
		UpdatedBy:     model.UpdatedBy,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}

	// 處理TargetDetails JSON反序列化
	if model.TargetDetails != "" {
		var targetDetails []string
		if err := json.Unmarshal([]byte(model.TargetDetails), &targetDetails); err != nil {
			// 如果JSON解析失敗，可能是舊的字符串格式，嘗試作為單個字符串處理
			if model.TargetDetails != "" {
				campaign.TargetDetails = []string{model.TargetDetails}
			}
		} else {
			campaign.TargetDetails = targetDetails
		}
	}

	if model.DeletedAt.Valid {
		deletedAt := model.DeletedAt.Time
		campaign.DeletedAt = &deletedAt
	}

	return campaign
}

// entityToModel 將實體轉換為模型
func (r *AgentCampaignRepository) entityToModel(
	campaign *entity.AgentCampaign,
) *models.AgentCampaign {
	model := &models.AgentCampaign{
		ID:            campaign.ID,
		MerchantID:    campaign.MerchantID,
		Title:         campaign.Title,
		Content:       campaign.Content,
		ScheduledAt:   campaign.ScheduledAt,
		Status:        campaign.Status,
		TargetType:    campaign.TargetType,
		TargetCount:   campaign.TargetCount,
		RealSentCount: campaign.RealSentCount,
		CreatedBy:     campaign.CreatedBy,
		UpdatedBy:     campaign.UpdatedBy,
		CreatedAt:     campaign.CreatedAt,
		UpdatedAt:     campaign.UpdatedAt,
	}

	// 處理TargetDetails JSON序列化
	if len(campaign.TargetDetails) > 0 {
		if targetDetailsBytes, err := json.Marshal(campaign.TargetDetails); err == nil {
			model.TargetDetails = string(targetDetailsBytes)
		}
	}

	if campaign.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *campaign.DeletedAt, Valid: true}
	}

	return model
}

// FindSentCampaignsForBackfill 查找用於補派發的已發送活動
func (r *AgentCampaignRepository) FindSentCampaignsForBackfill(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.AgentCampaign, error) {
	var campaignModels []models.AgentCampaign

	// 找出所有符合條件的活動：
	// 1. target_type = 'all' (全員發送)
	// 2. status = 'sent' (已發送)
	// 3. merchant_id = merchantID (該商戶的活動)
	// 4. created_at < now (創建時間早於現在)
	if err := r.db.WithContext(ctx).
		Where("merchant_id = ? AND target_type = ? AND status = ? AND created_at < ?",
			merchantID, "all", "sent", time.Now()).
		Order("created_at DESC").
		Find(&campaignModels).Error; err != nil {
		return nil, fmt.Errorf("find sent campaigns for backfill failed: %w", err)
	}

	// 轉換為 entity
	campaigns := make([]*entity.AgentCampaign, 0, len(campaignModels))
	for _, campaignModel := range campaignModels {
		campaign := r.modelToEntity(&campaignModel)
		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}

// FindSentCampaignsForBackfillPaginated 分頁查找用於補派發的已發送活動
func (r *AgentCampaignRepository) FindSentCampaignsForBackfillPaginated(
	ctx context.Context,
	merchantID uint64,
	limit, offset int,
) ([]*entity.AgentCampaign, error) {
	var campaignModels []models.AgentCampaign

	// 分頁查詢符合條件的活動：
	// 1. target_type = 'all' (全員發送)
	// 2. status = 'sent' (已發送)
	// 3. merchant_id = merchantID (該商戶的活動)
	// 4. created_at < now (創建時間早於現在)
	if err := r.db.WithContext(ctx).
		Where("merchant_id = ? AND target_type = ? AND status = ? AND created_at < ?",
			merchantID, "all", "sent", time.Now()).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&campaignModels).Error; err != nil {
		return nil, fmt.Errorf("find sent campaigns for backfill paginated failed: %w", err)
	}

	// 轉換為 entity
	campaigns := make([]*entity.AgentCampaign, 0, len(campaignModels))
	for _, campaignModel := range campaignModels {
		campaign := r.modelToEntity(&campaignModel)
		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}
