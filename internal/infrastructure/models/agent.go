package models

import (
	"time"

	"gorm.io/gorm"
)

// Agent 代理數據模型
type Agent struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"                                                    json:"id"`
	MerchantID      uint64         `gorm:"index;not null"                                                              json:"merchant_id"`
	GlobalAgentID   string         `gorm:"uniqueIndex:unique_global_agent_id;size:255;not null;column:global_agent_id" json:"global_agent_id"`
	Account         string         `gorm:"size:255;not null;index"                                                     json:"account"`
	Ancestry        string         `gorm:"type:text"                                                                   json:"ancestry"`           // 父代理層級路徑
	CurrentSignInAt *time.Time     `gorm:"type:datetime;index;column:current_sign_in_at"                               json:"current_sign_in_at"` // 當前登入時間，添加索引
	CreatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                                     json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;index"   json:"updated_at"` // 添加索引用於時間戳比較
	DeletedAt       gorm.DeletedAt `gorm:"index"                                                                       json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (*Agent) TableName() string {
	return "agents"
}

// AgentCampaign 代理訊息活動數據模型
type AgentCampaign struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement"                                                                            json:"id"`
	MerchantID    uint64         `gorm:"index;not null"                                                                                      json:"merchant_id"`
	Title         string         `gorm:"size:255;not null"                                                                                   json:"title"`
	Content       string         `gorm:"type:mediumtext;not null"                                                                                  json:"content"`
	ScheduledAt   *time.Time     `gorm:"type:datetime;index"                                                                                 json:"scheduled_at,omitempty"`
	Status        string         `gorm:"type:enum('draft','scheduled','sending','sent','failed','cancelled');not null;default:'draft';index" json:"status"`
	TargetType    string         `gorm:"type:enum('all','specific','line');not null"                                                         json:"target_type"`
	TargetDetails string         `gorm:"type:text"                                                                                           json:"target_details"`  // JSON格式的目標詳情
	TargetCount   int64          `gorm:"default:0"                                                                                           json:"target_count"`    // 目標代理數量
	RealSentCount int64          `gorm:"default:0"                                                                                           json:"real_sent_count"` // 實際發送數量
	CreatedBy     string         `gorm:"size:255;not null;index"                                                                             json:"created_by"`      // 建立者，添加索引
	UpdatedBy     string         `gorm:"size:255"                                                                                            json:"updated_by"`      // 更新者
	CreatedAt     time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                                                             json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"                                 json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index"                                                                                               json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (*AgentCampaign) TableName() string {
	return "agent_campaigns"
}

// AgentMessage 代理站內信數據模型
type AgentMessage struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"                                            json:"id"`
	AgentCampaignID uint64         `gorm:"uniqueIndex:unique_agent_campaign_message;index;not null"            json:"agent_campaign_id"` // 關聯到代理活動
	AgentID         uint64         `gorm:"uniqueIndex:unique_agent_campaign_message;index;not null"            json:"agent_id"`          // 關聯到代理
	IsRead          bool           `gorm:"default:false;index"                                                 json:"is_read"`           // 是否已讀
	ReadAt          *time.Time     `gorm:"type:datetime;index"                                                 json:"read_at"`           // 讀取時間，添加索引
	CreatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP;index"                       json:"created_at"`        // 創建時間即發送時間
	UpdatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index"                                                               json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (*AgentMessage) TableName() string {
	return "agent_messages"
}

// AgentRelationship 代理關係數據模型 - 雙主鍵設計
type AgentRelationship struct {
	ParentID   uint64    `gorm:"primaryKey;not null"                                                 json:"parent_id"`   // 父代理 agents.id (複合主鍵1)
	ChildID    uint64    `gorm:"primaryKey;not null"                                                 json:"child_id"`    // 子代理 agents.id (複合主鍵2)
	DepthLevel int       `gorm:"not null;index"                                                      json:"depth_level"` // 層級深度
	PathHash   string    `gorm:"size:64;index"                                                       json:"path_hash"`   // 路徑hash，用於快速比對
	CreatedAt  time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                             json:"created_at"`  // 創建時間
	UpdatedAt  time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`  // 更新時間
}

// TableName 指定表名
func (*AgentRelationship) TableName() string {
	return "agent_relationships"
}
