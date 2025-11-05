package service

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
)

// AgentHierarchy 代理層級結構
type AgentHierarchy struct {
	AgentID     string   `json:"agent_id"`
	Ancestors   []string `json:"ancestors"`   // 往上的祖先代理
	Descendants []string `json:"descendants"` // 往下的後代代理
}

// AgentService 代理領域服務接口
type AgentService interface {
	// 複雜的代理關聯同步邏輯 (Service職責)
	SyncAgentRelationshipsUpsert(ctx context.Context, agentEvent *event.AgentSyncEvent) error

	// 代理線查詢邏輯 (含快取策略)
	GetAgentLineDescendants(ctx context.Context, globalAgentID string) ([]string, error)
	GetAgentLineAncestors(ctx context.Context, globalAgentID string) ([]string, error)

	// 完整代理層級查詢 (並發優化)
	GetAgentHierarchy(ctx context.Context, globalAgentID string) (*AgentHierarchy, error)

	// 路徑解析和轉換邏輯
	ParseAgentPath(ancestry string) []string
	GeneratePathHash(pathParts []string) string

	// 代理驗證邏輯
	ValidateAndFilterAgents(agents []string) []string
	IsValidAgentID(agentID string) bool

	// ID轉換邏輯
	ExtractAccountFromGlobalID(globalID string) string
}