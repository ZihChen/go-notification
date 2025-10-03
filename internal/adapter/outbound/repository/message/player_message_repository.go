package message

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/aggregate"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/constants"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// playerCampaignKey 用於組合 player_id 和 campaign_id 的鍵
type playerCampaignKey struct {
	PlayerID   uint64
	CampaignID uint64
}

// PlayerMessageRepository GORM實現的會員訊息資料庫
type PlayerMessageRepository struct {
	db           *gorm.DB
	redisManager *redis.Manager
}

// NewPlayerMessageRepository 創建會員訊息資料庫
func NewPlayerMessageRepository(
	db *gorm.DB,
	redisManager *redis.Manager,
) repository.PlayerMessageRepository {
	return &PlayerMessageRepository{
		db:           db,
		redisManager: redisManager,
	}
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

	// 計算總數
	if err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ?", globalPlayerID).
		Count(&total).Error; err != nil {
		return nil, 0, err
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
		return nil, 0, result.Error
	}

	// 初始化 domainMessages 並轉換資料
	domainMessages := make([]*entity.PlayerMessage, len(messages))
	for i, message := range messages {
		domainMessages[i] = mapToDomainPlayerMessage(&message)
	}

	return domainMessages, int(total), nil
}

// FindByPlayerIDWithCampaign 通過玩家ID查找訊息列表並JOIN活動資訊（支援分頁）
func (r *PlayerMessageRepository) FindByPlayerIDWithCampaign(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) ([]*aggregate.PlayerMessageAggregate, int, error) {
	var results []aggregate.PlayerMessageQueryResult
	var total int64
	pm := models.PlayerMessage{}

	// 計算總數
	if err := r.db.WithContext(ctx).
		Table(fmt.Sprintf("%s pm", pm.TableName())).
		Joins("LEFT JOIN message_campaign mc ON pm.campaign_id = mc.id").
		Where("pm.global_player_id = ?", globalPlayerID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分頁查詢並JOIN活動資訊
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).
		Table(fmt.Sprintf("%s pm", pm.TableName())).
		Select(`pm.id, pm.global_player_id, pm.player_id, pm.campaign_id, pm.is_read, 
			pm.created_at, pm.updated_at,
			COALESCE(mc.title, '站內信') as campaign_title,
			COALESCE(mc.content, '您有一封新的訊息') as campaign_content`).
		Joins("LEFT JOIN message_campaign mc ON pm.campaign_id = mc.id").
		Where("pm.global_player_id = ?", globalPlayerID).
		Order("pm.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	// 轉換為 Aggregate
	aggregates := make([]*aggregate.PlayerMessageAggregate, len(results))
	for i, result := range results {
		aggregates[i] = result.ToAggregate()
	}

	return aggregates, int(total), nil
}

// GetPlayerMessageStats 獲取玩家訊息統計
func (r *PlayerMessageRepository) GetPlayerMessageStats(
	ctx context.Context,
	globalPlayerID string,
) (*dto.PlayerMessageStats, error) {
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

	stats := &dto.PlayerMessageStats{
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

// CheckAutoMessageExists 檢查自動派發訊息在時間窗口內是否已存在
func (r *PlayerMessageRepository) CheckAutoMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
	withinMinutes int,
) (bool, error) {
	var count int64
	timeThreshold := time.Now().Add(-time.Duration(withinMinutes) * time.Minute)

	err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ? AND campaign_id = ? AND created_at > ?",
			globalPlayerID, campaignID, timeThreshold).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// executeWithDistributedLock 使用 Redsync 分佈式鎖執行操作（參考 player_tag_usecase 的實現）
func (r *PlayerMessageRepository) executeWithDistributedLock(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
	fn func() error,
) error {
	// 如果沒有 redisManager（如 migrate 場景），直接執行函數
	if r.redisManager == nil {
		return fn()
	}

	mutexKey := fmt.Sprintf(constants.AutoNotificationMutexKey, globalPlayerID, campaignID)

	mutex, err := r.redisManager.GetMutexWithOption(mutexKey,
		redsync.WithExpiry(10*time.Second),           // 鎖的過期時間（比 player_tag 更長，因為涉及 DB 操作）
		redsync.WithTries(3),                         // 獲取鎖的重試次數
		redsync.WithRetryDelay(200*time.Millisecond), // 重試間隔
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create mutex for auto notification %s:%d: %w",
			globalPlayerID,
			campaignID,
			err,
		)
	}

	if err = mutex.Lock(); err != nil {
		return fmt.Errorf(
			"failed to acquire lock for auto notification %s:%d: %w",
			globalPlayerID,
			campaignID,
			err,
		)
	}

	defer func() {
		ok, unlockErr := mutex.Unlock()
		if !ok || unlockErr != nil {
			// 這裡應該使用 logger，但為了保持簡潔，暫時使用 fmt
			fmt.Printf(
				"Failed to unlock mutex: key=%s, success=%v, err=%v\n",
				mutexKey,
				ok,
				unlockErr,
			)
		}
	}()

	return fn()
}

// CreateAutoNotification 創建自動派發訊息（使用 Redsync 分佈式鎖保證併發安全）
func (r *PlayerMessageRepository) CreateAutoNotification(
	ctx context.Context,
	message *entity.PlayerMessage,
	timeWindowMinutes int,
) error {
	return r.executeWithDistributedLock(
		ctx,
		message.GlobalPlayerID,
		message.CampaignID,
		func() error {
			// 在分佈式鎖保護下執行原子操作
			return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				// 檢查時間窗口內是否已有相同訊息
				if timeWindowMinutes > 0 {
					exists, err := r.CheckAutoMessageExists(
						ctx,
						message.GlobalPlayerID,
						message.CampaignID,
						timeWindowMinutes,
					)
					if err != nil {
						return fmt.Errorf("check auto message exists: %w", err)
					}
					if exists {
						return fmt.Errorf(
							"duplicate auto notification within %d minutes",
							timeWindowMinutes,
						)
					}
				}

				// 直接創建記錄
				messageModel := &models.PlayerMessage{
					GlobalPlayerID: message.GlobalPlayerID,
					PlayerID:       message.PlayerID,
					CampaignID:     message.CampaignID,
					IsRead:         message.IsRead,
					CreatedAt:      message.CreatedAt,
					UpdatedAt:      message.UpdatedAt,
				}

				if err := tx.Create(messageModel).Error; err != nil {
					return fmt.Errorf("create auto notification: %w", err)
				}

				// 更新ID
				message.ID = messageModel.ID
				return nil
			})
		},
	)
}

// CreateBatchOptimized 優化的批次創建玩家訊息（併發安全版本）
func (r *PlayerMessageRepository) CreateBatchOptimized(
	ctx context.Context,
	messages []*entity.PlayerMessage,
	batchSize int,
) error {
	if len(messages) == 0 {
		return nil
	}

	// 分批處理避免單次插入過多記錄
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]

		// 方案1：使用事務 + 預檢查防重複
		if err := r.createBatchWithDuplicateCheck(ctx, batch); err != nil {
			return fmt.Errorf("batch create messages: %w", err)
		}
	}

	return nil
}

// createBatchWithDuplicateCheck 使用分佈式鎖保證併發安全的批次創建
func (r *PlayerMessageRepository) createBatchWithDuplicateCheck(
	ctx context.Context,
	messages []*entity.PlayerMessage,
) error {
	if len(messages) == 0 {
		return nil
	}

	// 按 player_id + campaign_id 分組
	groups := make(map[playerCampaignKey][]*entity.PlayerMessage)
	for _, msg := range messages {
		key := playerCampaignKey{
			PlayerID:   msg.PlayerID,
			CampaignID: msg.CampaignID,
		}
		groups[key] = append(groups[key], msg)
	}

	// 併發處理每個組合，使用分佈式鎖保護
	errChan := make(chan error, len(groups))
	var wg sync.WaitGroup

	for key, groupMessages := range groups {
		wg.Add(1)
		go func(k playerCampaignKey, msgs []*entity.PlayerMessage) {
			defer wg.Done()

			// 使用分佈式鎖保護該組合
			mutexKey := fmt.Sprintf(constants.PlayerMessageBatchMutexKey, k.PlayerID, k.CampaignID)

			err := r.executeWithBatchDistributedLock(ctx, mutexKey, func() error {
				return r.processSingleGroupWithTransaction(ctx, msgs, k)
			})

			errChan <- err
		}(key, groupMessages)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 收集錯誤
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(
			"batch create failed with %d errors, first error: %w",
			len(errors),
			errors[0],
		)
	}

	return nil
}

// executeWithBatchDistributedLock 為批次操作執行分佈式鎖
func (r *PlayerMessageRepository) executeWithBatchDistributedLock(
	ctx context.Context,
	mutexKey string,
	fn func() error,
) error {
	// 如果沒有 redisManager（如 migrate 場景），直接執行
	if r.redisManager == nil {
		return fn()
	}

	mutex, err := r.redisManager.GetMutexWithOption(mutexKey,
		redsync.WithExpiry(15*time.Second),           // 批次操作可能需要更長時間
		redsync.WithTries(5),                         // 批次操作重試次數更多
		redsync.WithRetryDelay(300*time.Millisecond), // 重試間隔稍長
	)
	if err != nil {
		return fmt.Errorf("failed to create batch mutex %s: %w", mutexKey, err)
	}

	if err = mutex.Lock(); err != nil {
		return fmt.Errorf("failed to acquire batch lock %s: %w", mutexKey, err)
	}

	defer func() {
		ok, unlockErr := mutex.Unlock()
		if !ok || unlockErr != nil {
			// 這裡應該使用 logger，但為了保持一致性，暫時使用 fmt
			fmt.Printf(
				"Failed to unlock batch mutex: key=%s, success=%v, err=%v\n",
				mutexKey,
				ok,
				unlockErr,
			)
		}
	}()

	return fn()
}

// processSingleGroupWithTransaction 在分佈式鎖保護下處理單個組合的批次插入
func (r *PlayerMessageRepository) processSingleGroupWithTransaction(
	ctx context.Context,
	messages []*entity.PlayerMessage,
	key playerCampaignKey,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 檢查是否已存在
		var existingCount int64
		err := tx.Model(&models.PlayerMessage{}).
			Where("player_id = ? AND campaign_id = ?", key.PlayerID, key.CampaignID).
			Count(&existingCount).Error

		if err != nil {
			return fmt.Errorf("check existence for player %d campaign %d: %w",
				key.PlayerID, key.CampaignID, err)
		}

		// 如果已存在，跳過
		if existingCount > 0 {
			return nil
		}

		// 創建該組的所有訊息
		for _, msg := range messages {
			messageModel := &models.PlayerMessage{
				GlobalPlayerID: msg.GlobalPlayerID,
				PlayerID:       msg.PlayerID,
				CampaignID:     msg.CampaignID,
				IsRead:         msg.IsRead,
				CreatedAt:      msg.CreatedAt,
				UpdatedAt:      msg.UpdatedAt,
			}

			if err := tx.Create(messageModel).Error; err != nil {
				return fmt.Errorf("create message for player %d campaign %d: %w",
					msg.PlayerID, msg.CampaignID, err)
			}

			// 更新原始 entity 的 ID
			msg.ID = messageModel.ID
		}

		return nil
	})
}

// CheckMessageExistsBatch 批次檢查訊息是否存在
func (r *PlayerMessageRepository) CheckMessageExistsBatch(
	ctx context.Context,
	playerIDs []uint64,
	campaignID uint64,
) (map[uint64]bool, error) {
	if len(playerIDs) == 0 {
		return make(map[uint64]bool), nil
	}

	var existingRecords []struct {
		PlayerID uint64 `gorm:"column:player_id"`
	}

	result := r.db.WithContext(ctx).
		Model(&models.PlayerMessage{}).
		Select("player_id").
		Where("player_id IN ? AND campaign_id = ?", playerIDs, campaignID).
		Find(&existingRecords)

	if result.Error != nil {
		return nil, fmt.Errorf("check message exists batch: %w", result.Error)
	}

	existsMap := make(map[uint64]bool)

	// 初始化所有玩家為false
	for _, playerID := range playerIDs {
		existsMap[playerID] = false
	}

	// 標記已存在的玩家為true
	for _, record := range existingRecords {
		existsMap[record.PlayerID] = true
	}

	return existsMap, nil
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

// GetByPlayerAndCampaign 通過玩家ID和活動ID獲取訊息（用於資料遷移）
func (r *PlayerMessageRepository) GetByPlayerAndCampaign(
	ctx context.Context,
	playerID uint64,
	campaignID uint64,
) (*entity.PlayerMessage, error) {
	var message models.PlayerMessage
	result := r.db.WithContext(ctx).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		First(&message)
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainPlayerMessage(&message), nil
}

// Update 更新玩家訊息（用於資料遷移）
func (r *PlayerMessageRepository) Update(
	ctx context.Context,
	message *entity.PlayerMessage,
) error {
	dbMessage := mapToDBPlayerMessage(message)
	result := r.db.WithContext(ctx).Save(dbMessage)
	if result.Error != nil {
		return fmt.Errorf("failed to update player message: %w", result.Error)
	}
	return nil
}

// FindExistingCampaignIDs 查找指定玩家已存在的活動ID列表
func (r *PlayerMessageRepository) FindExistingCampaignIDs(
	ctx context.Context,
	globalPlayerID string,
	campaignIDs []uint64,
) ([]uint64, error) {
	if len(campaignIDs) == 0 {
		return nil, nil
	}

	var existingCampaignIDs []uint64
	result := r.db.WithContext(ctx).
		Model(&models.PlayerMessage{}).
		Select("campaign_id").
		Where("global_player_id = ? AND campaign_id IN ?", globalPlayerID, campaignIDs).
		Find(&existingCampaignIDs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find existing campaign IDs: %w", result.Error)
	}

	return existingCampaignIDs, nil
}
