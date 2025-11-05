package entity

import "time"

// Agent 代理實體
type Agent struct {
	ID              uint64     `json:"id"`
	MerchantID      uint64     `json:"merchant_id"`
	GlobalAgentID   string     `json:"global_agent_id"`
	Account         string     `json:"account"`
	Ancestry        string     `json:"ancestry"`           // 父代理層級路徑
	CurrentSignInAt *time.Time `json:"current_sign_in_at"` // 當前登入時間
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// AgentCampaign 代理訊息活動
type AgentCampaign struct {
	ID            uint64     `json:"id"`
	MerchantID    uint64     `json:"merchant_id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	Status        string     `json:"status"`          // draft, scheduled, sending, completed, failed, cancelled
	TargetType    string     `json:"target_type"`     // all, specific, line
	TargetDetails string     `json:"target_details"`  // 目標詳情 (account列表或line路徑)
	TargetCount   int64      `json:"target_count"`    // 目標代理數量
	RealSentCount int64      `json:"real_sent_count"` // 實際發送數量
	CreatedBy     string     `json:"created_by"`      // 建立者
	UpdatedBy     string     `json:"updated_by"`      // 修改者
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// AgentMessage 代理站內信
type AgentMessage struct {
	ID              uint64     `json:"id"`
	AgentCampaignID uint64     `json:"agent_campaign_id"` // 關聯 agent_campaigns.id
	AgentID         uint64     `json:"agent_id"`          // 關聯 agents.id (數值ID優化)
	IsRead          bool       `json:"is_read"`           // 已讀狀態
	ReadAt          *time.Time `json:"read_at"`           // 已讀時間
	CreatedAt       time.Time  `json:"created_at"`        // 創建時間即發送時間
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentRelationship 代理關係 (用於高效查詢代理樹結構) - 雙主鍵設計
type AgentRelationship struct {
	ParentID   uint64    `json:"parent_id"`   // 父代理 ID (複合主鍵1)
	ChildID    uint64    `json:"child_id"`    // 子代理 ID (複合主鍵2)
	DepthLevel int       `json:"depth_level"` // 相對深度
	PathHash   string    `json:"path_hash"`   // 路徑hash，用於快速比對
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AgentHierarchy 代理層級結構 (用於返回完整層級查詢結果)
type AgentHierarchy struct {
	AgentID     string   `json:"agent_id"`
	Ancestors   []string `json:"ancestors"`   // 所有父代理
	Descendants []string `json:"descendants"` // 所有子代理
	TotalCount  int      `json:"total_count"` // 總代理數量 (包含自己)
}
