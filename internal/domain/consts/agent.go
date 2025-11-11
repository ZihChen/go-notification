package consts

// 代理活動狀態常數
const (
	// AgentCampaignStatusDraft 草稿狀態
	AgentCampaignStatusDraft = "draft"
	// AgentCampaignStatusScheduled 預約中狀態
	AgentCampaignStatusScheduled = "scheduled"
	// AgentCampaignStatusSent 已完成發送狀態
	AgentCampaignStatusSent = "sent"
	// AgentCampaignStatusFailed 失敗狀態
	AgentCampaignStatusFailed = "failed"
	// AgentCampaignStatusCancelled 已取消狀態
	AgentCampaignStatusCancelled = "cancelled"
)

// 代理目標類型常數
const (
	// AgentTargetTypeAll 全部代理
	AgentTargetTypeAll = "all"
	// AgentTargetTypeSpecific 特定代理
	AgentTargetTypeSpecific = "specific"
	// AgentTargetTypeLine 特定代理線
	AgentTargetTypeLine = "line"
)

// 代理訊息狀態常數
const (
	// AgentMessageStatusPending 待發送狀態
	AgentMessageStatusPending = "pending"
	// AgentMessageStatusSent 已發送狀態
	AgentMessageStatusSent = "sent"
	// AgentMessageStatusFailed 發送失敗狀態
	AgentMessageStatusFailed = "failed"
)

// 代理相關配置常數
const (
	// AgentActiveThresholdDays 代理活躍判斷天數閾值 (30天)
	AgentActiveThresholdDays = 30
	// AgentBatchSize 代理批量處理大小
	AgentBatchSize = 1000
	// AgentWorkerConcurrency 代理訊息發送並發數
	AgentWorkerConcurrency = 50
	// MaxAgentLineSize 最大代理線數量限制
	MaxAgentLineSize = 10000
	// QueryTimeoutSeconds 查詢超時時間(秒)
	QueryTimeoutSeconds = 30
)

// Redis 快取相關常數
const (
	// AgentLineCacheKey 代理線快取鍵模板
	AgentLineCacheKey = "agent:line:%s"
	// AgentAncestorCacheKey 代理祖先快取鍵模板
	AgentAncestorCacheKey = "agent:ancestors:%s"
	// AgentCacheTTLHours 代理快取過期時間(小時)
	AgentCacheTTLHours = 2
)

// 事件類型常數
const (
	// AgentSyncEventType KDS 代理同步事件類型
	AgentSyncEventType = "tw.jvd.fatcat.agent_notification.sync.v1"
)

// 錯誤訊息常數
const (
	// ErrAgentNotFound 代理不存在
	ErrAgentNotFound = "agent not found"
	// ErrAgentCampaignNotFound 代理活動不存在
	ErrAgentCampaignNotFound = "agent campaign not found"
	// ErrAgentMessageNotFound 代理訊息不存在
	ErrAgentMessageNotFound = "agent message not found"
	// ErrInvalidTargetType 無效的目標類型
	ErrInvalidTargetType = "invalid target type"
	// ErrInvalidAgentID 無效的代理ID
	ErrInvalidAgentID = "invalid agent ID"
	// ErrAgentLineQueryTimeout 代理線查詢超時
	ErrAgentLineQueryTimeout = "agent line query timeout"
)

// AgentCampaignStatuses 代理活動狀態列表
var AgentCampaignStatuses = []string{
	AgentCampaignStatusDraft,
	AgentCampaignStatusScheduled,
	AgentCampaignStatusSent,
	AgentCampaignStatusFailed,
	AgentCampaignStatusCancelled,
}

// AgentTargetTypes 代理目標類型列表
var AgentTargetTypes = []string{
	AgentTargetTypeAll,
	AgentTargetTypeSpecific,
	AgentTargetTypeLine,
}

// IsValidAgentCampaignStatus 檢查代理活動狀態是否有效
func IsValidAgentCampaignStatus(status string) bool {
	for _, s := range AgentCampaignStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsValidAgentTargetType 檢查代理目標類型是否有效
func IsValidAgentTargetType(targetType string) bool {
	for _, t := range AgentTargetTypes {
		if t == targetType {
			return true
		}
	}
	return false
}
