package repository

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentRelationshipRepository struct {
	db          *gorm.DB
	lockManager infrastructure.DistributedLockManager
}

func NewAgentRelationshipRepository(db *gorm.DB, lockManager infrastructure.DistributedLockManager) repository.AgentRelationshipRepository {
	return &AgentRelationshipRepository{
		db:          db,
		lockManager: lockManager,
	}
}

// BatchUpdate 併發安全的批次更新代理關係
func (r *AgentRelationshipRepository) BatchUpdate(ctx context.Context, targetAgentID uint64, relationships []*entity.AgentRelationship) error {
	if len(relationships) == 0 {
		// 如果沒有新關係，則刪除所有相關的舊關係
		return r.clearAgentRelationships(ctx, targetAgentID)
	}

	// 智能分組策略：依據目標代理 ID 進行分組鎖定
	lockKey := fmt.Sprintf("agent_relationship:batch_update:%d", targetAgentID)

	// 如果分佈式鎖管理器可用，使用分佈式鎖
	if r.lockManager != nil && r.lockManager.IsAvailable() {
		return r.batchUpdateWithDistributedLock(ctx, targetAgentID, relationships, lockKey)
	}

	// 分佈式鎖不可用時，降級到單純事務模式
	return r.batchUpdateWithTransaction(ctx, targetAgentID, relationships)
}

// batchUpdateWithDistributedLock 使用分佈式鎖的批次更新
func (r *AgentRelationshipRepository) batchUpdateWithDistributedLock(ctx context.Context, targetAgentID uint64, relationships []*entity.AgentRelationship, lockKey string) error {
	// 設定鎖選項
	lockOptions := infrastructure.LockOptions{
		Expiry:     30 * time.Second,       // 鎖過期時間 30 秒
		Tries:      5,                      // 最多嘗試 3 次
		RetryDelay: 100 * time.Millisecond, // 重試間隔 100ms
	}

	// 取得分佈式鎖
	mutex, err := r.lockManager.GetLockWithOptions(ctx, lockKey, lockOptions)
	if err != nil {
		return fmt.Errorf("failed to get distributed lock: %w", err)
	}

	// 獲取鎖
	if err := mutex.Lock(); err != nil {
		return fmt.Errorf("failed to acquire distributed lock: %w", err)
	}

	defer func() {
		if unlocked, unlockErr := mutex.Unlock(); unlockErr != nil || !unlocked {
			// 記錄解鎖失敗，但不影響主流程
			fmt.Printf("Failed to release distributed lock %s: unlocked=%v, err=%v\n", lockKey, unlocked, unlockErr)
		}
	}()

	// 在鎖保護下執行批次更新
	return r.batchUpdateWithTransaction(ctx, targetAgentID, relationships)
}

// batchUpdateWithTransaction 使用事務的批次更新
// 完全替換指定代理的關係鏈：刪除舊關係，新增新關係
func (r *AgentRelationshipRepository) batchUpdateWithTransaction(ctx context.Context, targetAgentID uint64, relationships []*entity.AgentRelationship) error {
	// 對關係列表進行排序，確保一致的操作順序
	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].ParentID != relationships[j].ParentID {
			return relationships[i].ParentID < relationships[j].ParentID
		}
		return relationships[i].ChildID < relationships[j].ChildID
	})

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 只清除以目標代理為子代理的直接關係
		if err := r.clearTargetAgentRelationships(tx, targetAgentID); err != nil {
			return fmt.Errorf("clear target agent relationships failed: %w", err)
		}

		// 2. 新增新的關係鏈
		if len(relationships) > 0 {
			insertModels := make([]*models.AgentRelationship, len(relationships))
			for i, rel := range relationships {
				insertModels[i] = &models.AgentRelationship{
					ParentID:   rel.ParentID,
					ChildID:    rel.ChildID,
					DepthLevel: rel.DepthLevel,
					PathHash:   rel.PathHash,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
			}

			// 使用UPSERT插入新關係，處理共同父代理關係的重複問題
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "parent_id"}, {Name: "child_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"depth_level", "path_hash", "updated_at",
				}),
			}).CreateInBatches(insertModels, 100).Error; err != nil {
				return fmt.Errorf("upsert agent relationships failed: %w", err)
			}
		}

		return nil
	})
}

// GetByParentID 根據父代理ID獲取所有關係
func (r *AgentRelationshipRepository) GetByParentID(ctx context.Context, parentID uint64) ([]*entity.AgentRelationship, error) {
	var relationModels []models.AgentRelationship
	if err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("depth_level ASC, child_id ASC").
		Find(&relationModels).Error; err != nil {
		return nil, fmt.Errorf("get relationships by parent id failed: %w", err)
	}

	relationships := make([]*entity.AgentRelationship, len(relationModels))
	for i, model := range relationModels {
		relationships[i] = r.modelToEntity(&model)
	}

	return relationships, nil
}

// GetByChildID 根據子代理ID獲取所有關係
func (r *AgentRelationshipRepository) GetByChildID(ctx context.Context, childID uint64) ([]*entity.AgentRelationship, error) {
	var relationModels []models.AgentRelationship
	if err := r.db.WithContext(ctx).
		Where("child_id = ?", childID).
		Order("depth_level ASC, parent_id ASC").
		Find(&relationModels).Error; err != nil {
		return nil, fmt.Errorf("get relationships by child id failed: %w", err)
	}

	relationships := make([]*entity.AgentRelationship, len(relationModels))
	for i, model := range relationModels {
		relationships[i] = r.modelToEntity(&model)
	}

	return relationships, nil
}

// FindDescendants 查詢後代關係 (支援深度限制)
func (r *AgentRelationshipRepository) FindDescendants(ctx context.Context, parentID uint64, maxDepth int) ([]*entity.AgentRelationship, error) {
	var relationModels []models.AgentRelationship

	query := r.db.WithContext(ctx).Where("parent_id = ?", parentID)
	if maxDepth > 0 {
		query = query.Where("depth_level <= ?", maxDepth)
	}

	if err := query.Order("depth_level ASC, child_id ASC").Find(&relationModels).Error; err != nil {
		return nil, fmt.Errorf("find descendants failed: %w", err)
	}

	relationships := make([]*entity.AgentRelationship, len(relationModels))
	for i, model := range relationModels {
		relationships[i] = r.modelToEntity(&model)
	}

	return relationships, nil
}

// FindAncestors 查詢祖先關係 (支援深度限制)
func (r *AgentRelationshipRepository) FindAncestors(ctx context.Context, childID uint64, maxDepth int) ([]*entity.AgentRelationship, error) {
	var relationModels []models.AgentRelationship

	query := r.db.WithContext(ctx).Where("child_id = ?", childID)
	if maxDepth > 0 {
		query = query.Where("depth_level <= ?", maxDepth)
	}

	if err := query.Order("depth_level ASC, parent_id ASC").Find(&relationModels).Error; err != nil {
		return nil, fmt.Errorf("find ancestors failed: %w", err)
	}

	relationships := make([]*entity.AgentRelationship, len(relationModels))
	for i, model := range relationModels {
		relationships[i] = r.modelToEntity(&model)
	}

	return relationships, nil
}

// FindByPathHash 根據路徑Hash查詢關係
func (r *AgentRelationshipRepository) FindByPathHash(ctx context.Context, pathHash string) ([]*entity.AgentRelationship, error) {
	var relationModels []models.AgentRelationship
	if err := r.db.WithContext(ctx).
		Where("path_hash = ?", pathHash).
		Order("depth_level ASC, parent_id ASC, child_id ASC").
		Find(&relationModels).Error; err != nil {
		return nil, fmt.Errorf("find relationships by path hash failed: %w", err)
	}

	relationships := make([]*entity.AgentRelationship, len(relationModels))
	for i, model := range relationModels {
		relationships[i] = r.modelToEntity(&model)
	}

	return relationships, nil
}

// modelToEntity 將模型轉換為實體
func (r *AgentRelationshipRepository) modelToEntity(model *models.AgentRelationship) *entity.AgentRelationship {
	return &entity.AgentRelationship{
		ParentID:   model.ParentID,
		ChildID:    model.ChildID,
		DepthLevel: model.DepthLevel,
		PathHash:   model.PathHash,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}

// GenerateLockKey 產生鎖定鍵 (提供給外部使用)
func (r *AgentRelationshipRepository) GenerateLockKey(parentID uint64) string {
	return fmt.Sprintf("agent_relationship:batch_update:%s", strconv.FormatUint(parentID, 10))
}

// clearAgentRelationships 清除指定代理的所有關係
func (r *AgentRelationshipRepository) clearAgentRelationships(ctx context.Context, targetAgentID uint64) error {
	// 使用與BatchUpdate相同的鎖定策略確保併發安全
	lockKey := fmt.Sprintf("agent_relationship:batch_update:%d", targetAgentID)

	// 如果分佈式鎖管理器可用，使用分佈式鎖
	if r.lockManager != nil && r.lockManager.IsAvailable() {
		return r.clearWithDistributedLock(ctx, targetAgentID, lockKey)
	}

	// 分佈式鎖不可用時，降級到單純事務模式
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.clearTargetAgentRelationships(tx, targetAgentID)
	})
}

// clearWithDistributedLock 使用分佈式鎖的清理操作
func (r *AgentRelationshipRepository) clearWithDistributedLock(ctx context.Context, targetAgentID uint64, lockKey string) error {
	// 設定鎖選項
	lockOptions := infrastructure.LockOptions{
		Expiry:     30 * time.Second,       // 鎖過期時間 30 秒
		Tries:      5,                      // 最多嘗試 3 次
		RetryDelay: 100 * time.Millisecond, // 重試間隔 100ms
	}

	// 取得分佈式鎖
	mutex, err := r.lockManager.GetLockWithOptions(ctx, lockKey, lockOptions)
	if err != nil {
		return fmt.Errorf("failed to get distributed lock for clear: %w", err)
	}

	// 獲取鎖
	if err = mutex.Lock(); err != nil {
		return fmt.Errorf("failed to acquire distributed lock for clear: %w", err)
	}

	defer func() {
		if unlocked, unlockErr := mutex.Unlock(); unlockErr != nil || !unlocked {
			// 記錄解鎖失敗，但不影響主流程
			fmt.Printf("Failed to release distributed lock %s: unlocked=%v, err=%v\n", lockKey, unlocked, unlockErr)
		}
	}()

	// 在鎖保護下執行清理操作
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.clearTargetAgentRelationships(tx, targetAgentID)
	})
}

// 清理策略：只清除以目標代理為子代理的直接關係
// 不進行遞歸清理，避免影響其他代理的共同祖先關係
// 例如：
// - H代理：A→B→C→H
// - G代理：A→B→G
// 當清理H時，只刪除C→H關係，保留A→B→C（因為G代理可能還需要A→B）
// clearTargetAgentRelationships 在事務中清除目標代理的相關關係
func (r *AgentRelationshipRepository) clearTargetAgentRelationships(tx *gorm.DB, targetAgentID uint64) error {
	if err := tx.Where("child_id = ?", targetAgentID).
		Delete(&models.AgentRelationship{}).Error; err != nil {
		return fmt.Errorf("delete relationships with target as child failed: %w", err)
	}

	return nil
}
