package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentRepository struct {
	db *gorm.DB
}

func NewAgentRepository(db *gorm.DB) repository.AgentRepository {
	return &AgentRepository{db: db}
}

// Create 創建代理
func (r *AgentRepository) Create(ctx context.Context, agent *entity.Agent) (*entity.Agent, error) {
	agentModel := &models.Agent{
		MerchantID:      agent.MerchantID,
		GlobalAgentID:   agent.GlobalAgentID,
		Account:         agent.Account,
		Ancestry:        agent.Ancestry,
		CurrentSignInAt: agent.CurrentSignInAt,
	}

	if err := r.db.WithContext(ctx).Create(agentModel).Error; err != nil {
		return nil, fmt.Errorf("create agent failed: %w", err)
	}

	agent.ID = agentModel.ID
	agent.CreatedAt = agentModel.CreatedAt
	agent.UpdatedAt = agentModel.UpdatedAt
	return agent, nil
}

// GetByID 根據ID獲取代理
func (r *AgentRepository) GetByID(ctx context.Context, id uint64) (*entity.Agent, error) {
	var agentModel models.Agent
	if err := r.db.WithContext(ctx).First(&agentModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent by id failed: %w", err)
	}

	return r.modelToEntity(&agentModel), nil
}

// GetByGlobalID 根據全局代理ID獲取代理
func (r *AgentRepository) GetByGlobalID(
	ctx context.Context,
	globalAgentID string,
) (*entity.Agent, error) {
	var agentModel models.Agent
	if err := r.db.WithContext(ctx).Where("global_agent_id = ?", globalAgentID).First(&agentModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent by global id failed: %w", err)
	}

	return r.modelToEntity(&agentModel), nil
}

// Update 更新代理
func (r *AgentRepository) Update(ctx context.Context, agent *entity.Agent) error {
	agentModel := &models.Agent{
		ID:              agent.ID,
		MerchantID:      agent.MerchantID,
		GlobalAgentID:   agent.GlobalAgentID,
		Account:         agent.Account,
		Ancestry:        agent.Ancestry,
		CurrentSignInAt: agent.CurrentSignInAt,
	}

	if err := r.db.WithContext(ctx).Save(agentModel).Error; err != nil {
		return fmt.Errorf("update agent failed: %w", err)
	}

	agent.UpdatedAt = agentModel.UpdatedAt
	return nil
}

// Delete 刪除代理 (軟刪除)
func (r *AgentRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&models.Agent{}, id).Error; err != nil {
		return fmt.Errorf("delete agent failed: %w", err)
	}
	return nil
}

// Upsert 冪等性創建或更新代理
func (r *AgentRepository) Upsert(ctx context.Context, agent *entity.Agent) error {
	agentModel := &models.Agent{
		MerchantID:      agent.MerchantID,
		GlobalAgentID:   agent.GlobalAgentID,
		Account:         agent.Account,
		Ancestry:        agent.Ancestry,
		CurrentSignInAt: agent.CurrentSignInAt,
	}

	// 使用 GORM 的 ON CONFLICT 處理冪等性
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_agent_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"account", "ancestry", "current_sign_in_at", "updated_at",
		}),
	}).Create(agentModel).Error; err != nil {
		return fmt.Errorf("upsert agent failed: %w", err)
	}

	// 更新 entity 的 ID 和時間戳
	if agent.ID == 0 {
		agent.ID = agentModel.ID
	}
	agent.CreatedAt = agentModel.CreatedAt
	agent.UpdatedAt = agentModel.UpdatedAt

	return nil
}

// FindAgents 分頁查詢代理列表
func (r *AgentRepository) FindAgents(
	ctx context.Context,
	merchantID uint64,
	limit, offset int,
) ([]*entity.Agent, error) {
	var agentModels []models.Agent

	query := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID)
	if err := query.Limit(limit).Offset(offset).Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("find agents failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}

func (r *AgentRepository) GetByAccount(ctx context.Context, account string) (*entity.Agent, error) {
	var agentModel models.Agent
	if err := r.db.WithContext(ctx).Where("account = ?", account).First(&agentModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent by gaccount failed: %w", err)
	}

	return r.modelToEntity(&agentModel), nil
}

// FindActiveAgents 查詢活躍代理列表（指定時間闾值之後登入的代理）
func (r *AgentRepository) FindActiveAgents(
	ctx context.Context,
	merchantID uint64,
	activeThreshold time.Time,
	limit, offset int,
) ([]*entity.Agent, error) {
	var agentModels []models.Agent

	query := r.db.WithContext(ctx).
		Where("merchant_id = ? AND current_sign_in_at IS NOT NULL AND current_sign_in_at > ?",
			merchantID, activeThreshold)
	if err := query.Limit(limit).Offset(offset).Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("find active agents failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}

// QueryAgentsByPath 根據路徑查詢代理ID列表 (legacy 支援)
func (r *AgentRepository) QueryAgentsByPath(
	ctx context.Context,
	targetAgentID string,
) ([]string, error) {
	var agentModels []models.Agent

	// 查詢包含目標代理ID的ancestry路徑
	if err := r.db.WithContext(ctx).
		Where("ancestry LIKE ?", "%"+targetAgentID+"%").
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("query agents by path failed: %w", err)
	}

	agentIDs := make([]string, len(agentModels))
	for i, agent := range agentModels {
		agentIDs[i] = agent.GlobalAgentID
	}

	return agentIDs, nil
}

// GetAgentIDByGlobalID 根據全局ID獲取數值ID
func (r *AgentRepository) GetAgentIDByGlobalID(
	ctx context.Context,
	globalID string,
	merchantID uint64,
) (uint64, error) {
	var agentModel models.Agent
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("global_agent_id = ? AND merchant_id = ?", globalID, merchantID).
		First(&agentModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("get agent id by global id failed: %w", err)
	}

	return agentModel.ID, nil
}

// UpsertRelationship 創建或更新代理關係 (legacy 支援)
func (r *AgentRepository) UpsertRelationship(
	ctx context.Context,
	rel *entity.AgentRelationship,
) error {
	relationModel := &models.AgentRelationship{
		ParentID:   rel.ParentID,
		ChildID:    rel.ChildID,
		DepthLevel: rel.DepthLevel,
		PathHash:   rel.PathHash,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "parent_id"}, {Name: "child_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"depth_level", "path_hash", "updated_at",
		}),
	}).Create(relationModel).Error; err != nil {
		return fmt.Errorf("upsert agent relationship failed: %w", err)
	}

	rel.CreatedAt = relationModel.CreatedAt
	rel.UpdatedAt = relationModel.UpdatedAt
	return nil
}

// QueryAgentsByRelationship 根據關係遞歸查詢所有層級子代理 (無限層級)
func (r *AgentRepository) QueryAgentsByRelationship(
	ctx context.Context,
	parentAgentID string,
) ([]uint64, error) {
	// 首先獲取父代理的數值ID
	var parentAgent models.Agent
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("global_agent_id = ?", parentAgentID).
		First(&parentAgent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []uint64{}, nil
		}
		return nil, fmt.Errorf("get parent agent failed: %w", err)
	}

	// 使用迭代方式實現遞歸查詢，避免 WITH RECURSIVE 兼容性問題
	// 這個方案雖然不是單一SQL，但在大多數情況下性能仍然可接受
	var allChildIDs []uint64
	processedIDs := make(map[uint64]bool)
	currentLevelIDs := []uint64{parentAgent.ID}

	// 限制最大深度避免無限循環，並批量處理提升性能
	maxDepth := 100
	batchSize := 1000

	for depth := 0; depth < maxDepth && len(currentLevelIDs) > 0; depth++ {
		var nextLevelIDs []uint64

		// 分批處理當前層級的代理，避免IN語句過大
		for i := 0; i < len(currentLevelIDs); i += batchSize {
			end := i + batchSize
			if end > len(currentLevelIDs) {
				end = len(currentLevelIDs)
			}
			batchIDs := currentLevelIDs[i:end]

			var relationships []models.AgentRelationship
			if err := r.db.WithContext(ctx).
				Select("child_id").
				Where("parent_id IN ?", batchIDs).
				Find(&relationships).Error; err != nil {
				return nil, fmt.Errorf("query relationships at depth %d failed: %w", depth, err)
			}

			// 收集下一層級的ID
			for _, rel := range relationships {
				if !processedIDs[rel.ChildID] {
					nextLevelIDs = append(nextLevelIDs, rel.ChildID)
					allChildIDs = append(allChildIDs, rel.ChildID)
					processedIDs[rel.ChildID] = true
				}
			}
		}

		currentLevelIDs = nextLevelIDs
	}

	// 直接返回所有子代理的ID列表
	return allChildIDs, nil
}

// ProcessAgentsByRelationshipInBatches 分批查詢並處理代理關係，邊查詢邊處理避免記憶體問題
func (r *AgentRepository) ProcessAgentsByRelationshipInBatches(
	ctx context.Context,
	parentAgentID string,
	batchSize int,
	processor func(ctx context.Context, agentIDs []uint64) (processedCount int, err error),
) (totalProcessed int, err error) {
	// 首先獲取父代理的數值ID
	var parentAgent models.Agent
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("global_agent_id = ?", parentAgentID).
		First(&parentAgent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("get parent agent failed: %w", err)
	}

	// 首先處理父代理自己
	parentProcessed, err := processor(ctx, []uint64{parentAgent.ID})
	if err != nil {
		return 0, fmt.Errorf("process parent agent failed: %w", err)
	}
	totalProcessed += parentProcessed

	// 使用迭代方式實現遞歸查詢，每查詢一批就處理一批
	processedIDs := make(map[uint64]bool)
	processedIDs[parentAgent.ID] = true // 標記父代理已處理
	currentLevelIDs := []uint64{parentAgent.ID}

	maxDepth := 100
	depth := 0

	for depth < maxDepth && len(currentLevelIDs) > 0 {
		var nextLevelIDs []uint64
		var currentBatch []uint64

		// 分批處理當前層級的代理
		for i := 0; i < len(currentLevelIDs); i += batchSize {
			end := i + batchSize
			if end > len(currentLevelIDs) {
				end = len(currentLevelIDs)
			}
			batchIDs := currentLevelIDs[i:end]

			var relationships []models.AgentRelationship
			if err := r.db.WithContext(ctx).
				Select("child_id").
				Where("parent_id IN ?", batchIDs).
				Find(&relationships).Error; err != nil {
				return totalProcessed, fmt.Errorf("query relationships at depth %d failed: %w", depth, err)
			}

			// 收集這批的子代理
			for _, rel := range relationships {
				if !processedIDs[rel.ChildID] {
					nextLevelIDs = append(nextLevelIDs, rel.ChildID)
					currentBatch = append(currentBatch, rel.ChildID)
					processedIDs[rel.ChildID] = true

					// 如果當前批次達到指定大小，就處理這一批
					if len(currentBatch) >= batchSize {
						batchProcessed, err := processor(ctx, currentBatch)
						if err != nil {
							return totalProcessed, fmt.Errorf("process agent batch failed: %w", err)
						}
						totalProcessed += batchProcessed
						currentBatch = currentBatch[:0] // 清空當前批次
					}
				}
			}
		}

		// 處理剩餘的代理（不足一個 batchSize 的部分）
		if len(currentBatch) > 0 {
			batchProcessed, err := processor(ctx, currentBatch)
			if err != nil {
				return totalProcessed, fmt.Errorf("process final agent batch failed: %w", err)
			}
			totalProcessed += batchProcessed
		}

		currentLevelIDs = nextLevelIDs
		depth++
	}

	return totalProcessed, nil
}

// QueryAgentAncestorsByRelationship 根據關係查詢祖先代理 (legacy 支援)
func (r *AgentRepository) QueryAgentAncestorsByRelationship(
	ctx context.Context,
	childAgentID string,
) ([]string, error) {
	// 首先獲取子代理的數值ID
	var childAgent models.Agent
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("global_agent_id = ?", childAgentID).
		First(&childAgent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []string{}, nil
		}
		return nil, fmt.Errorf("get child agent failed: %w", err)
	}

	// 查詢關係表獲取祖先代理ID
	var relationships []models.AgentRelationship
	if err := r.db.WithContext(ctx).
		Where("child_id = ?", childAgent.ID).
		Find(&relationships).Error; err != nil {
		return nil, fmt.Errorf("query relationships failed: %w", err)
	}

	if len(relationships) == 0 {
		return []string{}, nil
	}

	// 獲取祖先代理的全局ID
	ancestorIDs := make([]uint64, len(relationships))
	for i, rel := range relationships {
		ancestorIDs[i] = rel.ParentID
	}

	var ancestorAgents []models.Agent
	if err := r.db.WithContext(ctx).
		Select("global_agent_id").
		Where("id IN ?", ancestorIDs).
		Find(&ancestorAgents).Error; err != nil {
		return nil, fmt.Errorf("get ancestor agents failed: %w", err)
	}

	result := make([]string, len(ancestorAgents))
	for i, agent := range ancestorAgents {
		result[i] = agent.GlobalAgentID
	}

	return result, nil
}

// AgentExists 檢查代理是否存在
func (r *AgentRepository) AgentExists(ctx context.Context, agentID uint64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", agentID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check agent exists failed: %w", err)
	}

	return count > 0, nil
}

// BatchGetOrCreateAgentsByGlobalIDs 批量獲取或創建代理，避免N+1查詢
func (r *AgentRepository) BatchGetOrCreateAgentsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
	merchantID uint64,
) (map[string]uint64, error) {
	if len(globalIDs) == 0 {
		return make(map[string]uint64), nil
	}

	// 去重處理
	uniqueGlobalIDs := make([]string, 0, len(globalIDs))
	seen := make(map[string]bool)
	for _, id := range globalIDs {
		if !seen[id] {
			uniqueGlobalIDs = append(uniqueGlobalIDs, id)
			seen[id] = true
		}
	}

	// 1. 批量查詢已存在的代理
	var existingAgents []models.Agent
	if err := r.db.WithContext(ctx).
		Where("global_agent_id IN ? AND merchant_id = ?", uniqueGlobalIDs, merchantID).
		Find(&existingAgents).Error; err != nil {
		return nil, fmt.Errorf("batch query existing agents failed: %w", err)
	}

	// 建立結果映射
	result := make(map[string]uint64)
	existingGlobalIDs := make(map[string]bool)

	for _, agent := range existingAgents {
		result[agent.GlobalAgentID] = agent.ID
		existingGlobalIDs[agent.GlobalAgentID] = true
	}

	// 2. 找出需要創建的代理
	var toCreateGlobalIDs []string
	for _, globalID := range uniqueGlobalIDs {
		if !existingGlobalIDs[globalID] {
			toCreateGlobalIDs = append(toCreateGlobalIDs, globalID)
		}
	}

	// 3. 批量創建不存在的代理 (使用冪等操作避免併發衝突)
	if len(toCreateGlobalIDs) > 0 {
		var toCreate []models.Agent
		for _, globalID := range toCreateGlobalIDs {
			toCreate = append(toCreate, models.Agent{
				MerchantID:    merchantID,
				GlobalAgentID: globalID,
			})
		}

		// 使用 GORM 的 Clauses 來處理重複鍵衝突
		if err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "global_agent_id"}, {Name: "merchant_id"}},
				DoNothing: true, // 如果已存在則不做任何操作
			}).
			Create(&toCreate).Error; err != nil {
			return nil, fmt.Errorf("batch create agents failed: %w", err)
		}

		// 重新查詢所有需要創建的代理以獲取正確的ID
		var createdAgents []models.Agent
		if err := r.db.WithContext(ctx).
			Where("global_agent_id IN ? AND merchant_id = ?", toCreateGlobalIDs, merchantID).
			Find(&createdAgents).Error; err != nil {
			return nil, fmt.Errorf("query created agents failed: %w", err)
		}

		// 將代理ID添加到結果中
		for _, agent := range createdAgents {
			result[agent.GlobalAgentID] = agent.ID
		}
	}

	// 最後驗證：確保所有請求的globalID都有對應的結果
	for _, globalID := range uniqueGlobalIDs {
		if _, exists := result[globalID]; !exists {
			return nil, fmt.Errorf("failed to get or create agent for global_id: %s", globalID)
		}
	}

	return result, nil
}

// modelToEntity 將模型轉換為實體
func (r *AgentRepository) modelToEntity(model *models.Agent) *entity.Agent {
	agent := &entity.Agent{
		ID:              model.ID,
		MerchantID:      model.MerchantID,
		GlobalAgentID:   model.GlobalAgentID,
		Account:         model.Account,
		Ancestry:        model.Ancestry,
		CurrentSignInAt: model.CurrentSignInAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}

	if model.DeletedAt.Valid {
		deletedAt := model.DeletedAt.Time
		agent.DeletedAt = &deletedAt
	}

	return agent
}

// BatchGetAgentsByGlobalIDs 批量根據全局ID獲取代理
func (r *AgentRepository) BatchGetAgentsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Agent, error) {
	if len(globalIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	var agentModels []models.Agent
	if err := r.db.WithContext(ctx).
		Where("global_agent_id IN ?", globalIDs).
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("batch get agents by global ids failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}

// BatchGetAgentsByIDs 批量根據數值ID獲取代理
func (r *AgentRepository) BatchGetAgentsByIDs(
	ctx context.Context,
	agentIDs []uint64,
) ([]*entity.Agent, error) {
	if len(agentIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	var agentModels []models.Agent
	if err := r.db.WithContext(ctx).
		Where("id IN ?", agentIDs).
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("batch get agents by IDs failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}

// BatchGetAgentsByAccounts 批量根據賬號獲取代理
func (r *AgentRepository) BatchGetAgentsByAccounts(
	ctx context.Context,
	accounts []string,
) ([]*entity.Agent, error) {
	if len(accounts) == 0 {
		return []*entity.Agent{}, nil
	}

	var agentModels []models.Agent
	if err := r.db.WithContext(ctx).
		Where("account IN ?", accounts).
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("batch get agents by accounts failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}

// BatchConvertGlobalIDsToAccounts 批量將全局ID轉換為帳號名
func (r *AgentRepository) BatchConvertGlobalIDsToAccounts(
	ctx context.Context,
	globalIDs []string,
) ([]string, error) {
	if len(globalIDs) == 0 {
		return []string{}, nil
	}

	var agentModels []models.Agent
	if err := r.db.WithContext(ctx).
		Select("account").
		Where("global_agent_id IN ?", globalIDs).
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("batch convert global ids to accounts failed: %w", err)
	}

	accounts := make([]string, len(agentModels))
	for i, model := range agentModels {
		accounts[i] = model.Account
	}

	return accounts, nil
}

// BatchGetActiveAgentsByGlobalIDs 批量根據全局ID獲取活躍代理
func (r *AgentRepository) BatchGetActiveAgentsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
	activeThreshold time.Time,
) ([]*entity.Agent, error) {
	if len(globalIDs) == 0 {
		return []*entity.Agent{}, nil
	}

	var agentModels []models.Agent
	if err := r.db.WithContext(ctx).
		Where("global_agent_id IN ? AND current_sign_in_at IS NOT NULL AND current_sign_in_at > ?",
			globalIDs, activeThreshold).
		Find(&agentModels).Error; err != nil {
		return nil, fmt.Errorf("batch get active agents by global ids failed: %w", err)
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = r.modelToEntity(&model)
	}

	return agents, nil
}
