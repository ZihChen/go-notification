package aggregate

import "time"

// AgentMessageAggregate 代理訊息聚合根
// 包含代理訊息和相關的活動資訊，用於跨Aggregate查詢結果和Repository映射
type AgentMessageAggregate struct {
	// AgentMessage 基本資訊
	ID              uint64     `json:"id" gorm:"column:id"`
	AgentCampaignID uint64     `json:"agent_campaign_id" gorm:"column:agent_campaign_id"`
	AgentID         uint64     `json:"agent_id" gorm:"column:agent_id"`
	IsRead          bool       `json:"is_read" gorm:"column:is_read"`
	ReadAt          *time.Time `json:"read_at" gorm:"column:read_at"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"column:updated_at"`

	// AgentCampaign 關聯資訊
	CampaignTitle   string `json:"campaign_title" gorm:"column:campaign_title"`
	CampaignContent string `json:"campaign_content" gorm:"column:campaign_content"`
	
	// Agent 關聯資訊
	GlobalAgentID   string `json:"global_agent_id" gorm:"column:global_agent_id"`
}