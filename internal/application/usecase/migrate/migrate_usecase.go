package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"gorm.io/gorm"
)

// migrateUseCase 實作 MigrateUseCase 介面
type migrateUseCase struct {
	legacyDB              *gorm.DB // fatcat_staging 資料庫
	messageRepo           repository.MessageCampaignRepository
	playerRepo            repository.PlayerRepository
	merchantRepo          repository.MerchantRepository
	playerMessageRepo     repository.PlayerMessageRepository
	logger                infrastructure.Logger
}

// NewMigrateUseCase 創建新的 migrate use case
func NewMigrateUseCase(
	legacyDB *gorm.DB,
	messageRepo repository.MessageCampaignRepository,
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	playerMessageRepo repository.PlayerMessageRepository,
	logger infrastructure.Logger,
) inbound.MigrateUseCase {
	return &migrateUseCase{
		legacyDB:              legacyDB,
		messageRepo:           messageRepo,
		playerRepo:            playerRepo,
		merchantRepo:          merchantRepo,
		playerMessageRepo:     playerMessageRepo,
		logger:                logger,
	}
}

// LegacyNotification 舊系統的 notification 資料結構
type LegacyNotification struct {
	ID          uint       `gorm:"primaryKey"`
	Category    int        `gorm:"column:category"`
	Item        int        `gorm:"column:item"`
	Title       string     `gorm:"column:title"`
	Content     string     `gorm:"column:content"`
	Focus       int        `gorm:"column:focus"`
	AutoSend    bool       `gorm:"column:auto_send"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	CreatedBy   string     `gorm:"column:created_by"`
	UpdatedBy   *string    `gorm:"column:updated_by"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

// LegacyUserNotification 舊系統的 user_notification 資料結構
type LegacyUserNotification struct {
	ID             uint      `gorm:"primaryKey"`
	NotificationID uint      `gorm:"column:notification_id"`
	UserID         uint      `gorm:"column:user_id"`
	IsRead         bool      `gorm:"column:is_read"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// LegacyCompany 舊系統的 company 資料結構
type LegacyCompany struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
}

// TableName 設定 LegacyNotification 的表名
func (LegacyNotification) TableName() string {
	return "notifications"
}

// TableName 設定 LegacyUserNotification 的表名
func (LegacyUserNotification) TableName() string {
	return "user_notifications"
}

// TableName 設定 LegacyCompany 的表名
func (LegacyCompany) TableName() string {
	return "companies"
}

// MigrateMessageCampaigns 遷移訊息活動資料
func (uc *migrateUseCase) MigrateMessageCampaigns(ctx context.Context) (*inbound.MigrationStats, error) {
	stats := &inbound.MigrationStats{}
	
	// 計算三個月前的日期
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	
	// 首先獲取 merchant 資訊
	merchant, err := uc.getMerchantInfo(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to get merchant info: %w", err)
	}
	
	// 查詢過去三個月的 notifications
	var legacyNotifications []LegacyNotification
	if err := uc.legacyDB.Where("created_at >= ?", threeMonthsAgo).Find(&legacyNotifications).Error; err != nil {
		return stats, fmt.Errorf("failed to query legacy notifications: %w", err)
	}
	
	uc.logger.InfoLog("Found legacy notifications to migrate", uc.logger.Int("count", len(legacyNotifications)))
	
	// 批次處理
	batchSize := 100
	for i := 0; i < len(legacyNotifications); i += batchSize {
		end := i + batchSize
		if end > len(legacyNotifications) {
			end = len(legacyNotifications)
		}
		
		batch := legacyNotifications[i:end]
		if err := uc.processCampaignBatch(ctx, batch, merchant, stats); err != nil {
			uc.logger.ErrorLog("Failed to process campaign batch", uc.logger.Int("start", i), uc.logger.Int("end", end), uc.logger.Error("error", err))
			stats.AddError(err)
		}
	}
	
	return stats, nil
}

// MigratePlayerMessages 遷移玩家訊息資料
func (uc *migrateUseCase) MigratePlayerMessages(ctx context.Context) (*inbound.MigrationStats, error) {
	stats := &inbound.MigrationStats{}
	
	// 獲取公司資訊用於組成 global_player_id
	var company LegacyCompany
	if err := uc.legacyDB.Where("id = ?", 2).First(&company).Error; err != nil {
		return stats, fmt.Errorf("failed to get company info: %w", err)
	}
	
	// 查詢 user_notifications 和關聯的 notifications
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	var userNotifications []LegacyUserNotification
	query := `
		SELECT un.* FROM user_notifications un
		INNER JOIN notifications n ON un.notification_id = n.id
		WHERE n.created_at >= ?
	`
	if err := uc.legacyDB.Raw(query, threeMonthsAgo).Scan(&userNotifications).Error; err != nil {
		return stats, fmt.Errorf("failed to query legacy user notifications: %w", err)
	}
	
	uc.logger.InfoLog("Found legacy user notifications to migrate", uc.logger.Int("count", len(userNotifications)))
	
	// 批次處理
	batchSize := 100
	for i := 0; i < len(userNotifications); i += batchSize {
		end := i + batchSize
		if end > len(userNotifications) {
			end = len(userNotifications)
		}
		
		batch := userNotifications[i:end]
		if err := uc.processPlayerMessageBatch(ctx, batch, company.Name, stats); err != nil {
			uc.logger.ErrorLog("Failed to process player message batch", uc.logger.Int("start", i), uc.logger.Int("end", end), uc.logger.Error("error", err))
			stats.AddError(err)
		}
	}
	
	return stats, nil
}

// getMerchantInfo 獲取 merchant 資訊
func (uc *migrateUseCase) getMerchantInfo(ctx context.Context) (*entity.Merchant, error) {
	// 從 fatcat_staging.companies 獲取 id=2 的公司名稱
	var company LegacyCompany
	if err := uc.legacyDB.Where("id = ?", 2).First(&company).Error; err != nil {
		return nil, fmt.Errorf("failed to find company with id=2: %w", err)
	}
	
	// 在新系統中查找對應的 merchant
	merchant, err := uc.merchantRepo.GetByName(ctx, company.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to find merchant with name=%s: %w", company.Name, err)
	}
	
	return merchant, nil
}

// processCampaignBatch 批次處理 message campaign
func (uc *migrateUseCase) processCampaignBatch(ctx context.Context, batch []LegacyNotification, merchant *entity.Merchant, stats *inbound.MigrationStats) error {
	for _, legacy := range batch {
		campaign := uc.mapToCampaign(legacy, merchant)
		
		// 使用 UpdateOrCreate 邏輯
		existing, err := uc.messageRepo.GetByGlobalID(ctx, campaign.GlobalID)
		if err != nil && err != gorm.ErrRecordNotFound {
			stats.AddError(fmt.Errorf("failed to check existing campaign: %w", err))
			continue
		}
		
		if existing != nil {
			// 更新現有記錄
			campaign.ID = existing.ID
			if err := uc.messageRepo.Update(ctx, campaign); err != nil {
				stats.AddError(fmt.Errorf("failed to update campaign: %w", err))
				continue
			}
			uc.logger.DebugLog("Updated existing campaign", uc.logger.String("global_id", campaign.GlobalID))
		} else {
			// 創建新記錄
			if err := uc.messageRepo.Create(ctx, campaign); err != nil {
				stats.AddError(fmt.Errorf("failed to create campaign: %w", err))
				continue
			}
			uc.logger.DebugLog("Created new campaign", uc.logger.String("global_id", campaign.GlobalID))
		}
		
		stats.AddSuccess()
	}
	
	return nil
}

// processPlayerMessageBatch 批次處理 player message
func (uc *migrateUseCase) processPlayerMessageBatch(ctx context.Context, batch []LegacyUserNotification, companyName string, stats *inbound.MigrationStats) error {
	for _, legacy := range batch {
		// 組成 global_player_id
		globalPlayerID := fmt.Sprintf("%s-PLAYER-%d", companyName, legacy.UserID)
		
		// 檢查玩家是否存在
		player, err := uc.playerRepo.GetByGlobalPlayerID(ctx, globalPlayerID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				uc.logger.DebugLog("Player not found, skipping", uc.logger.String("global_player_id", globalPlayerID))
				stats.AddSkipped()
				continue
			}
			stats.AddError(fmt.Errorf("failed to check player: %w", err))
			continue
		}
		
		// 根據 notification_id 查找對應的 campaign
		campaign, err := uc.findCampaignByLegacyID(ctx, legacy.NotificationID)
		if err != nil {
			stats.AddError(fmt.Errorf("failed to find campaign for notification_id %d: %w", legacy.NotificationID, err))
			continue
		}
		
		playerMessage := &entity.PlayerMessage{
			GlobalPlayerID: globalPlayerID,
			PlayerID:       player.ID,
			CampaignID:     campaign.ID,
			IsRead:         legacy.IsRead,
			CreatedAt:      legacy.CreatedAt,
			UpdatedAt:      legacy.UpdatedAt,
		}
		
		// 檢查是否已存在
		existing, err := uc.playerMessageRepo.GetByPlayerAndCampaign(ctx, player.ID, campaign.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			stats.AddError(fmt.Errorf("failed to check existing player message: %w", err))
			continue
		}
		
		if existing != nil {
			// 更新現有記錄
			playerMessage.ID = existing.ID
			if err := uc.playerMessageRepo.Update(ctx, playerMessage); err != nil {
				stats.AddError(fmt.Errorf("failed to update player message: %w", err))
				continue
			}
			uc.logger.DebugLog("Updated player message", uc.logger.UInt64("player_id", player.ID), uc.logger.UInt64("campaign_id", campaign.ID))
		} else {
			// 創建新記錄
			if err := uc.playerMessageRepo.Create(ctx, playerMessage); err != nil {
				stats.AddError(fmt.Errorf("failed to create player message: %w", err))
				continue
			}
			uc.logger.DebugLog("Created player message", uc.logger.UInt64("player_id", player.ID), uc.logger.UInt64("campaign_id", campaign.ID))
		}
		
		stats.AddSuccess()
	}
	
	return nil
}

// findCampaignByLegacyID 根據舊系統的 notification_id 查找對應的 campaign
func (uc *migrateUseCase) findCampaignByLegacyID(ctx context.Context, legacyID uint) (*entity.MessageCampaign, error) {
	// 這裡需要一個方法來關聯舊 ID 和新 ID
	// 可以通過一個臨時的映射表或者在 campaign 中存儲舊 ID
	// 暫時使用 global_id 的方式來關聯
	
	// 首先查詢舊記錄
	var legacy LegacyNotification
	if err := uc.legacyDB.Where("id = ?", legacyID).First(&legacy).Error; err != nil {
		return nil, fmt.Errorf("failed to find legacy notification: %w", err)
	}
	
	// 生成相同的 global_id（需要確保與遷移時生成的一致）
	globalID := uc.generateGlobalID(legacy)
	
	campaign, err := uc.messageRepo.GetByGlobalID(ctx, globalID)
	if err != nil {
		return nil, fmt.Errorf("failed to find campaign by global_id: %w", err)
	}
	
	return campaign, nil
}

// generateGlobalID 生成 global_id（需要確保一致性）
func (uc *migrateUseCase) generateGlobalID(legacy LegacyNotification) string {
	// 使用確定性的方法生成 UUID
	// 這裡簡化為使用固定的種子來生成相同的 UUID
	seed := fmt.Sprintf("legacy-%d", legacy.ID)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

// mapToCampaign 將舊系統資料映射為新系統的 MessageCampaign
func (uc *migrateUseCase) mapToCampaign(legacy LegacyNotification, merchant *entity.Merchant) *entity.MessageCampaign {
	campaign := &entity.MessageCampaign{
		Category:      uc.mapCategory(legacy.Category),
		Item:          uc.mapItem(legacy.Item),
		TriggerType:   "success", // 規格中規定統一為 success
		MerchantID:    merchant.ID,
		GlobalID:      uc.generateGlobalID(legacy),
		Title:         legacy.Title,
		Content:       legacy.Content,
		Status:        uc.mapStatus(legacy.DeletedAt),
		Target:        uc.mapTarget(legacy.Focus),
		AutoSend:      legacy.AutoSend,
		SendStartTime: &legacy.CreatedAt,
		SendEndTime:   &legacy.CreatedAt,
		CreatedBy:     legacy.CreatedBy,
		CreatedAt:     legacy.CreatedAt,
		UpdatedAt:     legacy.UpdatedAt,
		DeletedAt:     legacy.DeletedAt,
	}
	
	if legacy.UpdatedBy != nil {
		campaign.UpdatedBy = legacy.UpdatedBy
	}
	
	return campaign
}

// mapCategory 映射 category
func (uc *migrateUseCase) mapCategory(category int) string {
	switch category {
	case 1:
		return "member"
	case 2:
		return "bonus"
	case 3:
		return "others"
	default:
		return "others"
	}
}

// mapItem 映射 item
func (uc *migrateUseCase) mapItem(item int) string {
	switch item {
	case 1:
		return "registration"
	case 2:
		return "identity_verification"
	case 3:
		return "bank_card"
	case 4:
		return "others"
	case 5:
		return "event"
	case 6:
		return "all"
	case 7:
		return "mission"
	default:
		return "others"
	}
}

// mapStatus 映射 status
func (uc *migrateUseCase) mapStatus(deletedAt *time.Time) string {
	if deletedAt != nil {
		return "cancelled"
	}
	return "sent"
}

// mapTarget 映射 target
func (uc *migrateUseCase) mapTarget(focus int) string {
	switch focus {
	case 1:
		return "high_activity"
	case 2:
		return "low_activity"
	case 3:
		return "not_activity"
	case 8:
		return "player"
	case 9:
		return "level"
	case 10:
		return "tag"
	case 11:
		return "all"
	default:
		return "all"
	}
}