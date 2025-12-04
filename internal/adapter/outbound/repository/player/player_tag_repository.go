package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

type PlayerTagRepository struct {
	db *gorm.DB
}

func NewPlayerTagRepository(db *gorm.DB) repository.PlayerTagRepository {
	return &PlayerTagRepository{db: db}
}

func (r *PlayerTagRepository) BatchUpdate(
	ctx context.Context,
	playerID uint64,
	tagIDs []uint64,
) error {
	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := r.batchUpdateWithoutRetry(ctx, playerID, tagIDs)

		if err == nil {
			return nil
		}

		// 檢查是否為 deadlock 錯誤或其他可重試的錯誤
		if r.isRetriableError(err) && attempt < maxRetries {
			// 指數退避：100ms, 200ms, 400ms, 800ms, 1600ms
			delay := baseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
			continue
		}

		// 非可重試錯誤或達到最大重試次數
		return err
	}

	return fmt.Errorf("batch update failed after %d retries", maxRetries)
}

// batchUpdateWithoutRetry 原始的 BatchUpdate 邏輯，不含重試機制
func (r *PlayerTagRepository) batchUpdateWithoutRetry(
	ctx context.Context,
	playerID uint64,
	tagIDs []uint64,
) error {
	// 處理空標籤情況
	if len(tagIDs) == 0 {
		// 如果tagIDs為空，只需刪除所有關聯
		return r.DeleteByPlayerID(ctx, playerID)
	}

	// 排序以確保一致的鎖定順序，避免死鎖
	sort.Slice(tagIDs, func(i, j int) bool {
		return tagIDs[i] < tagIDs[j]
	})

	// 1. 查詢現有關聯
	var existingTagIDs []uint64
	if err := r.db.WithContext(ctx).Model(&models.PlayerTag{}).
		Where("player_id = ?", playerID).
		Pluck("tag_id", &existingTagIDs).Error; err != nil {
		return fmt.Errorf("query existing player tags failed: %w", err)
	}

	// 2. 計算差異
	toDelete := r.difference(existingTagIDs, tagIDs)
	toInsert := r.difference(tagIDs, existingTagIDs)

	// 3. 無變化直接返回
	if len(toDelete) == 0 && len(toInsert) == 0 {
		return nil
	}

	// 4. 在事務外準備數據
	now := time.Now()
	insertItems := make([]*models.PlayerTag, len(toInsert))
	for i, tagID := range toInsert {
		insertItems[i] = &models.PlayerTag{
			PlayerID:  playerID,
			TagID:     tagID,
			CreatedAt: now,
		}
	}

	// 5. 高效事務操作
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 只刪除需要刪除的
		if len(toDelete) > 0 {
			if err := tx.Where("player_id = ? AND tag_id IN ?", playerID, toDelete).
				Delete(&models.PlayerTag{}).Error; err != nil {
				return fmt.Errorf("delete specific player tags failed: %w", err)
			}
		}

		// 只插入需要插入的
		if len(toInsert) > 0 {
			if err := tx.Create(&insertItems).Error; err != nil {
				return fmt.Errorf("insert new player tags failed: %w", err)
			}
		}

		return nil
	})
}

func (r *PlayerTagRepository) DeleteByPlayerID(ctx context.Context, playerID uint64) error {
	result := r.db.WithContext(ctx).
		Where("player_id = ?", playerID).
		Delete(&models.PlayerTag{})
	if result.Error != nil {
		return fmt.Errorf("delete player tags failed: %w", result.Error)
	}
	return nil
}

// isRetriableError 檢查是否為可重試的錯誤
func (r *PlayerTagRepository) isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// MySQL 死鎖錯誤
	if strings.Contains(errorStr, "Deadlock found") ||
		strings.Contains(errorStr, "1213") ||
		strings.Contains(errorStr, "40001") {
		return true
	}

	// MySQL 鎖等待超時
	if strings.Contains(errorStr, "Lock wait timeout") ||
		strings.Contains(errorStr, "1205") {
		return true
	}

	return false
}

// difference 計算兩個切片的差集：在 a 中但不在 b 中的元素
func (r *PlayerTagRepository) difference(a, b []uint64) []uint64 {
	bMap := make(map[uint64]bool, len(b))
	for _, item := range b {
		bMap[item] = true
	}

	var result []uint64
	for _, item := range a {
		if !bMap[item] {
			result = append(result, item)
		}
	}

	return result
}
