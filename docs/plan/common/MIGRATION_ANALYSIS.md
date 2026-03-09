# PlayerMessage 索引遷移分析

## 代碼修改建議

### 1. 修改 CheckMessageExists 方法
```go
// 原版本：假設唯一約束，查詢 count
func (r *PlayerMessageRepository) CheckMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ? AND campaign_id = ?", globalPlayerID, campaignID).
		Count(&count).Error
	return count > 0, nil
}

// 建議修改：針對自動派發，檢查最近時間內是否已發送
func (r *PlayerMessageRepository) CheckAutoMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
	withinMinutes int, // 例如：5分鐘內不重複發送
) (bool, error) {
	var count int64
	timeThreshold := time.Now().Add(-time.Duration(withinMinutes) * time.Minute)
	
	err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ? AND campaign_id = ? AND created_at > ?", 
			globalPlayerID, campaignID, timeThreshold).
		Count(&count).Error
	return count > 0, err
}
```

### 2. 修改 GetByPlayerAndCampaign 方法
```go
// 原版本：假設唯一記錄
func (r *PlayerMessageRepository) GetByPlayerAndCampaign(
	ctx context.Context,
	playerID uint64,
	campaignID uint64,
) (*entity.PlayerMessage, error) {
	var message models.PlayerMessage
	result := r.db.WithContext(ctx).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		First(&message) // 這裡可能會有問題
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainPlayerMessage(&message), nil
}

// 建議修改：獲取最新的記錄
func (r *PlayerMessageRepository) GetLatestByPlayerAndCampaign(
	ctx context.Context,
	playerID uint64,
	campaignID uint64,
) (*entity.PlayerMessage, error) {
	var message models.PlayerMessage
	result := r.db.WithContext(ctx).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Order("created_at DESC").
		First(&message)
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainPlayerMessage(&message), nil
}

// 新增：獲取所有記錄
func (r *PlayerMessageRepository) GetAllByPlayerAndCampaign(
	ctx context.Context,
	playerID uint64,
	campaignID uint64,
) ([]*entity.PlayerMessage, error) {
	var messages []models.PlayerMessage
	result := r.db.WithContext(ctx).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Order("created_at DESC").
		Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	
	domainMessages := make([]*entity.PlayerMessage, len(messages))
	for i, msg := range messages {
		domainMessages[i] = mapToDomainPlayerMessage(&msg)
	}
	return domainMessages, nil
}
```

### 3. 修改批次處理邏輯
```go
// 移除依賴唯一約束的OnConflict邏輯
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
		messageModels := make([]*models.PlayerMessage, len(batch))

		for j, message := range batch {
			messageModels[j] = &models.PlayerMessage{
				GlobalPlayerID: message.GlobalPlayerID,
				PlayerID:       message.PlayerID,
				CampaignID:     message.CampaignID,
				IsRead:         message.IsRead,
				CreatedAt:      message.CreatedAt,
				UpdatedAt:      message.UpdatedAt,
			}
		}

		// 直接創建，不使用OnConflict
		result := r.db.WithContext(ctx).Create(&messageModels)
		if result.Error != nil {
			return fmt.Errorf("batch create messages: %w", result.Error)
		}
	}

	return nil
}
```

## 數據庫遷移腳本

### Phase 1: 備份和準備
```sql
-- 創建備份表
CREATE TABLE player_message_backup_20251003 AS 
SELECT * FROM player_message;

-- 檢查現有重複數據（應該沒有，因為有唯一約束）
SELECT player_id, campaign_id, COUNT(*) as count 
FROM player_message 
GROUP BY player_id, campaign_id 
HAVING COUNT(*) > 1;
```

### Phase 2: 移除唯一約束，添加分離索引
```sql
-- 移除唯一約束
ALTER TABLE player_message DROP INDEX idx_player_campaign;

-- 添加分離的索引以優化查詢性能
ALTER TABLE player_message ADD INDEX idx_player_id (player_id);
ALTER TABLE player_message ADD INDEX idx_campaign_id (campaign_id);

-- 添加複合索引用於常見查詢模式（非唯一）
ALTER TABLE player_message ADD INDEX idx_player_campaign_created (player_id, campaign_id, created_at);
```

### Phase 3: 驗證
```sql
-- 檢查索引是否正確創建
SHOW INDEX FROM player_message;

-- 測試插入重複記錄（應該成功）
INSERT INTO player_message (global_player_id, player_id, campaign_id, is_read, created_at, updated_at)
VALUES ('test_player', 1, 1, false, NOW(), NOW());

INSERT INTO player_message (global_player_id, player_id, campaign_id, is_read, created_at, updated_at)
VALUES ('test_player', 1, 1, false, NOW(), NOW());

-- 清理測試數據
DELETE FROM player_message WHERE global_player_id = 'test_player';
```

## 業務邏輯調整建議

### 1. 自動派發防重複機制
```go
// 在 SendAutoNotification 中添加時間窗口檢查
func (u *MessageUseCase) SendAutoNotification(
	ctx context.Context,
	req *dto.SendAutoNotificationRequest,
) (*dto.SendAutoNotificationResponse, error) {
	// ... 現有邏輯 ...

	// 檢查最近5分鐘內是否已發送相同通知
	exists, err := u.playerMessageRepo.CheckAutoMessageExists(
		ctx, req.GlobalPlayerID, campaign.ID, 5)
	if err != nil {
		return nil, err
	}
	if exists {
		return &dto.SendAutoNotificationResponse{
			PlayerID:    req.GlobalPlayerID,
			Category:    req.Category,
			Item:        req.Item,
			TriggerType: req.TriggerType,
			Status:      "skipped",
			Message:     "duplicate notification within time window",
		}, nil
	}

	// ... 繼續原有發送邏輯 ...
}
```

### 2. 統計功能增強
```go
// 更新統計邏輯，考慮可能的重複記錄
func (r *PlayerMessageRepository) GetPlayerMessageStats(
	ctx context.Context,
	globalPlayerID string,
) (*dto.PlayerMessageStats, error) {
	// 統計唯一的campaign數量而非記錄數量
	var uniqueCampaignCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.PlayerMessage{}).
		Select("COUNT(DISTINCT campaign_id)").
		Where("global_player_id = ?", globalPlayerID).
		Count(&uniqueCampaignCount).Error; err != nil {
		return nil, err
	}

	// 總記錄數
	var totalCount int64
	if err := r.db.WithContext(ctx).Model(&models.PlayerMessage{}).
		Where("global_player_id = ?", globalPlayerID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// ... 其他統計邏輯
}
```

## 性能考量

### 索引大小影響
- 原本：1個唯一複合索引
- 修改後：3個索引 (player_id, campaign_id, player_id+campaign_id+created_at)
- 預估增加：約20-30%的索引空間

### 查詢性能
- `player_id`查詢：性能提升
- `campaign_id`查詢：性能提升  
- `player_id + campaign_id`查詢：性能略微下降（需要使用複合索引）

## 建議的實施順序

1. **測試環境驗證**：先在測試環境執行遷移
2. **代碼部署**：部署包含新邏輯的代碼版本
3. **數據庫遷移**：在低峰期執行索引變更
4. **監控觀察**：觀察性能指標和錯誤日誌
5. **回滾準備**：準備快速回滾方案

## 回滾方案
```sql
-- 如果需要回滾
ALTER TABLE player_message DROP INDEX idx_player_id;
ALTER TABLE player_message DROP INDEX idx_campaign_id;
ALTER TABLE player_message DROP INDEX idx_player_campaign_created;

-- 重新創建唯一約束（注意：如果有重複數據會失敗）
ALTER TABLE player_message ADD UNIQUE INDEX idx_player_campaign (player_id, campaign_id);
```