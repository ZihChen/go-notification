package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	servicePort "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

// AgentService 代理領域服務實作
type AgentService struct {
	agentRepo      repository.AgentRepository
	relationRepo   repository.AgentRelationshipRepository
	merchantRepo   repository.MerchantRepository
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
	cache          map[string][]uint64 // 簡單的記憶體快取
	cacheMutex     sync.RWMutex
}

// NewAgentService 創建代理服務
func NewAgentService(
	agentRepo repository.AgentRepository,
	relationRepo repository.AgentRelationshipRepository,
	merchantRepo repository.MerchantRepository,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) servicePort.AgentService {
	return &AgentService{
		agentRepo:      agentRepo,
		relationRepo:   relationRepo,
		merchantRepo:   merchantRepo,
		logger:         logger,
		tracingService: tracingService,
		cache:          make(map[string][]uint64),
		cacheMutex:     sync.RWMutex{},
	}
}

// SyncAgentRelationshipsUpsert 複雜的代理關聯同步邏輯 (Service職責)
func (s *AgentService) SyncAgentRelationshipsUpsert(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	ctx, span := s.tracingService.StartSpan(ctx, "AgentService.SyncAgentRelationshipsUpsert")
	defer s.tracingService.SpanEnd(span)

	s.tracingService.RecordSpanAttributes(span,
		attribute.String("global_agent_id", agentEvent.GlobalAgentID),
		attribute.String("agent.ancestry", agentEvent.Ancestry),
		attribute.String("global_merchant_id", agentEvent.GlobalMerchantID))

	// 先透過GlobalMerchantID獲取實際的merchant_id
	merchant, err := s.merchantRepo.FindByGlobalID(ctx, agentEvent.GlobalMerchantID)
	if err != nil {
		s.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("find merchant by global_id %s: %w", agentEvent.GlobalMerchantID, err)
	}
	if merchant == nil {
		s.tracingService.TraceEvent(span, "Merchant not found, skipping agent relationship sync")
		s.logger.WarnLog("Merchant not found for agent relationship sync",
			s.logger.String("global_merchant_id", agentEvent.GlobalMerchantID),
			s.logger.String("global_agent_id", agentEvent.GlobalAgentID))
		return nil
	}
	s.tracingService.RecordSpanAttributes(span,
		attribute.Int64("merchant.id", int64(merchant.ID)))

	// 複雜的ancestry解析邏輯 (領域知識)
	pathParts := s.ParseAgentPath(agentEvent.Ancestry)
	if len(pathParts) == 0 {
		s.tracingService.TraceEvent(span, "Top-level agent, no relationships needed")
		return nil // 頂級代理，無需建立關係
	}

	// 構建完整路徑：ancestry + 當前代理
	fullPath := append(pathParts, agentEvent.GlobalAgentID)

	s.tracingService.RecordSpanAttributes(span,
		attribute.StringSlice("ancestry.path_parts", pathParts),
		attribute.StringSlice("ancestry.full_path", fullPath))

	// 批量獲取或創建所有代理，避免N+1查詢
	agentIDMap, err := s.agentRepo.BatchGetOrCreateAgentsByGlobalIDs(ctx, fullPath, merchant.ID)
	if err != nil {
		s.tracingService.RecordSpanError(span, err)
		return fmt.Errorf("batch get or create agents: %w", err)
	}

	s.tracingService.RecordSpanAttributes(span,
		attribute.Int("agents.batch_processed", len(agentIDMap)))

	// 建立每一層的父子關係 (領域專業邏輯)
	var relationships []*entity.AgentRelationship
	for i := 0; i < len(fullPath)-1; i++ {
		parentGlobalID := fullPath[i]
		childGlobalID := fullPath[i+1]

		// ID轉換邏輯 (領域轉換) - 從批量結果中獲取
		parentID, exists := agentIDMap[parentGlobalID]
		if !exists {
			return fmt.Errorf("parent agent ID not found for %s", parentGlobalID)
		}

		childID, exists := agentIDMap[childGlobalID]
		if !exists {
			return fmt.Errorf("child agent ID not found for %s", childGlobalID)

		}

		// 生成路徑Hash (演算法實作)
		pathHash := s.GeneratePathHash(fullPath[:i+2])

		// 計算層級深度：從0開始，每一層+1
		depthLevel := i + 1

		s.tracingService.RecordSpanAttributes(span,
			attribute.String("parent_global_id", parentGlobalID),
			attribute.String("child_global_id", childGlobalID),
			attribute.Int("depth_level", depthLevel))

		relationships = append(relationships, &entity.AgentRelationship{
			ParentID:   parentID,
			ChildID:    childID,
			DepthLevel: depthLevel,
			PathHash:   pathHash,
		})
	}

	// 找到當前代理的ID用於批量更新 - 從批量結果中獲取
	currentAgentID, exists := agentIDMap[agentEvent.GlobalAgentID]
	if !exists {
		return fmt.Errorf("current agent ID not found for %s", agentEvent.GlobalAgentID)
	}

	// 🔍 詳細追踪：批量更新關係處理
	ctx, updateSpan := s.tracingService.StartSpan(ctx, "AgentService.BatchUpdateRelationships")
	s.tracingService.RecordSpanAttributes(updateSpan,
		attribute.Int64("update.target_agent_id", int64(currentAgentID)),
		attribute.Int("update.relationships_count", len(relationships)))

	if err = s.relationRepo.BatchUpdate(ctx, currentAgentID, relationships); err != nil {
		s.tracingService.RecordSpanError(span, err)
		s.tracingService.RecordSpanError(updateSpan, err)
		s.tracingService.SpanEnd(updateSpan)
		return fmt.Errorf("batch update relationships: %w", err)
	}

	s.tracingService.RecordSpanAttributes(updateSpan,
		attribute.Bool("update.success", true))
	s.tracingService.SpanEnd(updateSpan)

	// 清除相關快取
	s.clearCache(agentEvent.GlobalAgentID)

	s.tracingService.TraceEvent(span, "Agent relationships synced successfully")
	s.logger.InfoLog("Agent relationships synced successfully",
		s.logger.String("global_agent_id", agentEvent.GlobalAgentID),
		s.logger.Int("relationships_count", len(relationships)))

	return nil
}

// GetAgentLineDescendants 複雜的代理線查詢邏輯 (Service職責)
func (s *AgentService) GetAgentLineDescendants(
	ctx context.Context,
	globalAgentID string,
) ([]uint64, error) {
	ctx, span := s.tracingService.StartSpan(ctx, "AgentService.GetAgentLineDescendants")
	defer s.tracingService.SpanEnd(span)

	s.tracingService.RecordSpanAttributes(span, attribute.String("global_agent_id", globalAgentID))

	// 1. 快取策略 (效能優化邏輯)
	cacheKey := fmt.Sprintf("descendants:%s", globalAgentID)
	if cached := s.getFromCache(cacheKey); cached != nil {
		s.tracingService.TraceEvent(span, "Agent line cache hit")
		s.logger.DebugLog("Agent line cache hit",
			s.logger.String("agent_id", globalAgentID))
		return cached, nil
	}

	// 使用Repository查詢
	descendants, err := s.agentRepo.QueryAgentsByRelationship(ctx, globalAgentID)
	if err != nil {
		s.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("query descendants: %w", err)
	}

	// 快取結果
	s.setCache(cacheKey, descendants)

	s.tracingService.TraceEvent(span, "Agent line descendants retrieved",
		attribute.Int("descendants_count", len(descendants)))

	return descendants, nil
}

// GetAgentLineAncestors 代理祖先查詢邏輯
func (s *AgentService) GetAgentLineAncestors(
	ctx context.Context,
	globalAgentID string,
) ([]uint64, error) {
	ctx, span := s.tracingService.StartSpan(ctx, "AgentService.GetAgentLineAncestors")
	defer s.tracingService.SpanEnd(span)

	s.tracingService.RecordSpanAttributes(span, attribute.String("global_agent_id", globalAgentID))

	// 快取策略
	cacheKey := fmt.Sprintf("ancestors:%s", globalAgentID)
	if cached := s.getFromCache(cacheKey); cached != nil {
		s.tracingService.TraceEvent(span, "Agent ancestors cache hit")
		return cached, nil
	}

	// 使用關係倉儲查詢祖先
	agent, err := s.agentRepo.GetByGlobalID(ctx, globalAgentID)
	if err != nil {
		return nil, fmt.Errorf("get agent by global id: %w", err)
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", globalAgentID)
	}

	relationships, err := s.relationRepo.GetByChildID(ctx, agent.ID)
	if err != nil {
		return nil, fmt.Errorf("get relationships by child id: %w", err)
	}

	ancestors := make([]uint64, len(relationships))
	for i, rel := range relationships {
		ancestors[i] = rel.ParentID
	}

	// 快取結果
	s.setCache(cacheKey, ancestors)

	s.tracingService.TraceEvent(span, "Agent ancestors retrieved",
		attribute.Int("ancestors_count", len(ancestors)))

	return ancestors, nil
}

// GetAgentHierarchy 完整代理層級查詢 (Service職責：並發優化)
func (s *AgentService) GetAgentHierarchy(
	ctx context.Context,
	globalAgentID string,
) (*servicePort.AgentHierarchy, error) {
	ctx, span := s.tracingService.StartSpan(ctx, "AgentService.GetAgentHierarchy")
	defer s.tracingService.SpanEnd(span)

	s.tracingService.RecordSpanAttributes(span, attribute.String("global_agent_id", globalAgentID))

	// Service職責：複雜的並發邏輯與效能優化
	var ancestors, descendants []uint64
	var err1, err2 error

	done := make(chan struct{}, 2)

	// 並發查詢祖先 (效能優化邏輯)
	go func() {
		defer func() { done <- struct{}{} }()
		ancestors, err1 = s.GetAgentLineAncestors(ctx, globalAgentID)
	}()

	// 並發查詢後代 (效能優化邏輯)
	go func() {
		defer func() { done <- struct{}{} }()
		descendants, err2 = s.GetAgentLineDescendants(ctx, globalAgentID)
	}()

	// 等待兩個查詢完成
	<-done
	<-done

	// 檢查錯誤
	if err1 != nil {
		s.tracingService.RecordSpanError(span, err1)
		return nil, fmt.Errorf("get ancestors: %w", err1)
	}
	if err2 != nil {
		s.tracingService.RecordSpanError(span, err2)
		return nil, fmt.Errorf("get descendants: %w", err2)
	}

	hierarchy := &servicePort.AgentHierarchy{
		AgentID:     globalAgentID,
		Ancestors:   ancestors,
		Descendants: descendants,
	}

	s.tracingService.TraceEvent(span, "Agent hierarchy retrieved successfully",
		attribute.Int("ancestors_count", len(ancestors)),
		attribute.Int("descendants_count", len(descendants)))

	return hierarchy, nil
}

// ParseAgentPath 解析ancestry路徑 (領域邏輯)
func (s *AgentService) ParseAgentPath(ancestry string) []string {
	if ancestry == "" {
		return nil
	}

	// 支援多種分隔符格式
	var pathParts []string
	if strings.Contains(ancestry, "/") {
		pathParts = strings.Split(ancestry, "/")
	} else if strings.Contains(ancestry, ",") {
		pathParts = strings.Split(ancestry, ",")
	} else {
		// 單一值，可能是直接父代理
		pathParts = []string{ancestry}
	}

	// 清理和驗證
	var validParts []string
	for _, part := range pathParts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && s.IsValidAgentID(trimmed) {
			validParts = append(validParts, trimmed)
		}
	}

	return validParts
}

// GeneratePathHash 複雜的領域計算邏輯 (Service職責)
func (s *AgentService) GeneratePathHash(pathParts []string) string {
	// Service職責：演算法實作與領域計算
	pathStr := strings.Join(pathParts, "/")
	hash := sha256.Sum256([]byte(pathStr))
	return hex.EncodeToString(hash[:])[:16] // 取前16字符
}

// ValidateAndFilterAgents 領域驗證邏輯 (Service職責)
func (s *AgentService) ValidateAndFilterAgents(agents []string) []string {
	// Service職責：領域規則驗證
	var validAgents []string

	for _, agentID := range agents {
		// 領域驗證：ID格式檢查
		if s.IsValidAgentID(agentID) {
			validAgents = append(validAgents, agentID)
		}
	}

	return validAgents
}

// IsValidAgentID 領域知識：代理ID格式驗證（已移除格式限制）
func (s *AgentService) IsValidAgentID(agentID string) bool {
	// 移除格式限制，只檢查非空字串
	return agentID != ""
}

// ExtractAccountFromGlobalID 從 global_id 提取 account
func (s *AgentService) ExtractAccountFromGlobalID(globalID string) string {
	// 範例: "FATCAT-AGENT-123" -> "agent123"
	parts := strings.Split(globalID, "-")
	if len(parts) >= 3 {
		return strings.ToLower(parts[1]) + parts[2] // "agent123"
	}
	return globalID // fallback
}

// 快取相關方法
func (s *AgentService) getFromCache(key string) []uint64 {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return s.cache[key]
}

func (s *AgentService) setCache(key string, value []uint64) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache[key] = value
}

func (s *AgentService) clearCache(agentID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	// 清除相關的快取項目
	delete(s.cache, fmt.Sprintf("descendants:%s", agentID))
	delete(s.cache, fmt.Sprintf("ancestors:%s", agentID))
}
