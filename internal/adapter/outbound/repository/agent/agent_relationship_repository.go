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
func (r *AgentRelationshipRepository) BatchUpdate(ctx context.Context, parentID uint64, relationships []*entity.AgentRelationship) error {
	if len(relationships) == 0 {
		return nil
	}

	// 智能分組策略：依據 parent_id 進行分組鎖定
	lockKey := fmt.Sprintf("agent_relationship:batch_update:%d", parentID)

	// 如果分佈式鎖管理器可用，使用分佈式鎖
	if r.lockManager != nil && r.lockManager.IsAvailable() {
		return r.batchUpdateWithDistributedLock(ctx, parentID, relationships, lockKey)
	}

	// 分佈式鎖不可用時，降級到單純事務模式
	return r.batchUpdateWithTransaction(ctx, parentID, relationships)
}

// batchUpdateWithDistributedLock 使用分佈式鎖的批次更新
func (r *AgentRelationshipRepository) batchUpdateWithDistributedLock(ctx context.Context, parentID uint64, relationships []*entity.AgentRelationship, lockKey string) error {
	// 設定鎖選項
	lockOptions := infrastructure.LockOptions{
		Expiry:     30 * time.Second,       // 鎖過期時間 30 秒
		Tries:      3,                      // 最多嘗試 3 次
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
	return r.batchUpdateWithTransaction(ctx, parentID, relationships)
}

// batchUpdateWithTransaction 使用事務的批次更新
func (r *AgentRelationshipRepository) batchUpdateWithTransaction(ctx context.Context, parentID uint64, relationships []*entity.AgentRelationship) error {
	// 對關係列表進行排序，確保一致的操作順序
	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].ParentID != relationships[j].ParentID {
			return relationships[i].ParentID < relationships[j].ParentID
		}
		return relationships[i].ChildID < relationships[j].ChildID
	})

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 批次刪除指定父代理的所有關係
		if err := tx.Where("parent_id = ?", parentID).Delete(&models.AgentRelationship{}).Error; err != nil {
			return fmt.Errorf("delete agent relationships failed: %w", err)
		}

		// 2. 批次插入新的關係
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

			// 批次創建，每批100筆
			if err := tx.CreateInBatches(insertModels, 100).Error; err != nil {
				return fmt.Errorf("insert agent relationships failed: %w", err)
			}

			// 更新實體的時間戳
			for i, model := range insertModels {
				relationships[i].CreatedAt = model.CreatedAt
				relationships[i].UpdatedAt = model.UpdatedAt
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
