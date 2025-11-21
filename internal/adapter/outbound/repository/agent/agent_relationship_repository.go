package repository

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
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
	// 簡化的批次處理佇列
	batchQueue chan *batchRequest
	processing sync.Map // map[uint64]bool 記錄正在處理的AgentID
}

// batchRequest 批次處理請求
type batchRequest struct {
	targetAgentID uint64
	relationships []*entity.AgentRelationship
	resultChan    chan error
	ctx           context.Context
}

func NewAgentRelationshipRepository(
	db *gorm.DB,
	lockManager infrastructure.DistributedLockManager,
) repository.AgentRelationshipRepository {
	r := &AgentRelationshipRepository{
		db:          db,
		lockManager: lockManager,
		batchQueue:  make(chan *batchRequest, 1000), // 佇列緩衝1000個請求
		processing:  sync.Map{},
	}

	// 啟動批次處理器
	go r.startBatchProcessor()

	return r
}

// BatchUpdate 併發安全的批次更新代理關係（使用佇列機制）
func (r *AgentRelationshipRepository) BatchUpdate(
	ctx context.Context,
	targetAgentID uint64,
	relationships []*entity.AgentRelationship,
) error {
	if len(relationships) == 0 {
		// 如果沒有新關係，則刪除所有相關的舊關係
		return r.clearAgentRelationships(ctx, targetAgentID)
	}

	// 檢查是否已有相同AgentID正在處理
	if _, isProcessing := r.processing.LoadOrStore(targetAgentID, true); isProcessing {
		// 如果正在處理，直接返回成功（去重複）
		log.Printf("Agent %d is already being processed, skipping duplicate request", targetAgentID)
		return nil
	}

	// 確保在所有退出路徑都清理processing標記
	defer r.processing.Delete(targetAgentID)

	// 創建批次請求
	req := &batchRequest{
		targetAgentID: targetAgentID,
		relationships: relationships,
		resultChan:    make(chan error, 1),
		ctx:           ctx,
	}

	// 非阻塞式放入佇列
	select {
	case r.batchQueue <- req:
		// 等待處理結果
		select {
		case result := <-req.resultChan:
			return result
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	default:
		// 佇列滿了，直接執行
		log.Printf("Batch queue is full, processing agent %d directly", targetAgentID)
		return r.optimizedBatchUpdateWithTransaction(ctx, targetAgentID, relationships)
	}
}

// batchUpdateWithDistributedLock 簡化的分佈式鎖批次更新（已廢棄，改用佇列機制）
func (r *AgentRelationshipRepository) batchUpdateWithDistributedLock(
	ctx context.Context,
	targetAgentID uint64,
	relationships []*entity.AgentRelationship,
	lockKey string,
) error {
	// 直接降級到事務模式，不使用分佈式鎖
	log.Printf("Using transaction mode for agent %d (distributed lock bypassed)", targetAgentID)
	return r.optimizedBatchUpdateWithTransaction(ctx, targetAgentID, relationships)
}

// batchUpdateWithTransaction 使用事務的批次更新
// 完全替換指定代理的關係鏈：刪除舊關係，新增新關係
func (r *AgentRelationshipRepository) batchUpdateWithTransaction(
	ctx context.Context,
	targetAgentID uint64,
	relationships []*entity.AgentRelationship,
) error {
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
func (r *AgentRelationshipRepository) GetByParentID(
	ctx context.Context,
	parentID uint64,
) ([]*entity.AgentRelationship, error) {
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
func (r *AgentRelationshipRepository) GetByChildID(
	ctx context.Context,
	childID uint64,
) ([]*entity.AgentRelationship, error) {
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
func (r *AgentRelationshipRepository) FindDescendants(
	ctx context.Context,
	parentID uint64,
	maxDepth int,
) ([]*entity.AgentRelationship, error) {
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
func (r *AgentRelationshipRepository) FindAncestors(
	ctx context.Context,
	childID uint64,
	maxDepth int,
) ([]*entity.AgentRelationship, error) {
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
func (r *AgentRelationshipRepository) FindByPathHash(
	ctx context.Context,
	pathHash string,
) ([]*entity.AgentRelationship, error) {
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
func (r *AgentRelationshipRepository) modelToEntity(
	model *models.AgentRelationship,
) *entity.AgentRelationship {
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
func (r *AgentRelationshipRepository) clearAgentRelationships(
	ctx context.Context,
	targetAgentID uint64,
) error {
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

// clearWithDistributedLock 簡化的清理操作（直接使用事務）
func (r *AgentRelationshipRepository) clearWithDistributedLock(
	ctx context.Context,
	targetAgentID uint64,
	lockKey string,
) error {
	// 直接使用事務模式，避免鎖競爭
	log.Printf("Clearing agent %d relationships using direct transaction", targetAgentID)
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
func (r *AgentRelationshipRepository) clearTargetAgentRelationships(
	tx *gorm.DB,
	targetAgentID uint64,
) error {
	if err := tx.Where("child_id = ?", targetAgentID).
		Delete(&models.AgentRelationship{}).Error; err != nil {
		return fmt.Errorf("delete relationships with target as child failed: %w", err)
	}

	return nil
}

// startBatchProcessor 啟動批次處理器
func (r *AgentRelationshipRepository) startBatchProcessor() {
	log.Println("Starting agent relationship batch processor")

	for req := range r.batchQueue {
		// 使用超時控制，避免長時間阻塞
		processCtx, cancel := context.WithTimeout(req.ctx, 10*time.Second)

		// 直接執行優化的事務更新，不使用分佈式鎖
		err := r.optimizedBatchUpdateWithTransaction(
			processCtx,
			req.targetAgentID,
			req.relationships,
		)

		cancel()

		// 發送結果
		select {
		case req.resultChan <- err:
		default:
			// 如果無法發送結果，記錄錯誤
			log.Printf("Failed to send result for agent %d: %v", req.targetAgentID, err)
		}
	}

	log.Println("Batch processor stopped")
}

// optimizedBatchUpdateWithTransaction 優化的事務批次更新（減少鎖定時間）
func (r *AgentRelationshipRepository) optimizedBatchUpdateWithTransaction(
	ctx context.Context,
	targetAgentID uint64,
	relationships []*entity.AgentRelationship,
) error {
	if len(relationships) == 0 {
		// 快速刪除，不排序
		return r.db.WithContext(ctx).Where("child_id = ?", targetAgentID).
			Delete(&models.AgentRelationship{}).Error
	}

	// 預先構建插入數據，減少事務內的處理時間
	insertModels := make([]*models.AgentRelationship, len(relationships))
	now := time.Now()
	for i, rel := range relationships {
		insertModels[i] = &models.AgentRelationship{
			ParentID:   rel.ParentID,
			ChildID:    rel.ChildID,
			DepthLevel: rel.DepthLevel,
			PathHash:   rel.PathHash,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	// 使用短事務：先刪除再插入
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 快速刪除目標代理的關係
		if err := tx.Where("child_id = ?", targetAgentID).
			Delete(&models.AgentRelationship{}).Error; err != nil {
			return fmt.Errorf("delete old relationships: %w", err)
		}

		// 2. 批次插入新關係，使用UPSERT避免重複鍵衝突
		if len(insertModels) > 0 {
			// 使用ON DUPLICATE KEY UPDATE避免並發插入衝突
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "parent_id"}, {Name: "child_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"depth_level", "path_hash", "updated_at",
				}),
			}).CreateInBatches(insertModels, 100).Error; err != nil {
				return fmt.Errorf("upsert new relationships: %w", err)
			}
		}

		return nil
	})
}
