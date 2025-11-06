package dto

import (
	"time"
)

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Stats    PlayerMessageStats `json:"stats"`
	Messages []MessageSummary   `json:"messages"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
}

// MessageSummary 訊息摘要模型（用於列表顯示）
type MessageSummary struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"` // 內容摘要
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// PlayerMessageStats 玩家訊息統計
type PlayerMessageStats struct {
	ReadCount   int `json:"read_count"`
	UnreadCount int `json:"unread_count"`
	TotalCount  int `json:"total_count"`
}

// MessageCampaignsQuery 訊息活動查詢參數
type MessageCampaignsQuery struct {
	Page           int      `json:"page"`
	PageSize       int      `json:"page_size"`
	MerchantID     uint64   `json:"merchant_id"`     // 商戶ID
	Category       string   `json:"category"`        // 類型：member, bonus, others
	Item           string   `json:"item"`            // 項目：registration, identity_verification, bank_card, others, event, all, mission
	Status         []string `json:"status"`          // 狀態：draft, scheduled, sent, cancelled, failed
	IncludeDeleted bool     `json:"include_deleted"` // 是否包含已刪除的活動
	ShowAutoSend   bool     `json:"show_auto_send"`  // 是否顯示站內系統建立
	CreatedBy      string   `json:"created_by"`      // 建立者
	StartAt        string   `json:"start_at"`        // 建立起始時間
	EndAt          string   `json:"end_at"`          // 建立結束時間
}

// ManagerResponse 管理員回應 DTO
type ManagerResponse struct {
	ID       uint64 `json:"id"`
	GlobalID string `json:"global_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// MerchantResponse 商家回應 DTO
type MerchantResponse struct {
	ID            uint64 `json:"id"`
	GlobalID      string `json:"global_id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Configuration string `json:"configuration,omitempty"`
}

// PlayerResponse 玩家回應 DTO
type PlayerResponse struct {
	ID           uint64     `json:"id"`
	GlobalID     string     `json:"global_id"`
	MerchantID   uint64     `json:"merchant_id"`
	Username     string     `json:"username"`
	Level        int        `json:"level"`
	Tags         []string   `json:"tags"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// MessageCampaignResponse 訊息活動回應 DTO
type MessageCampaignResponse struct {
	ID                uint64     `json:"id"`
	GlobalID          string     `json:"global_id"`
	MerchantID        uint64     `json:"merchant_id"`
	Title             string     `json:"title"`
	Content           string     `json:"content"`
	AppContent        *string    `json:"app_content,omitempty"` // App推播內容
	NotificationTypes uint8      `json:"notification_types"`    // 推送類型位元遮罩: 1=站內信, 2=App推播, 4=其他
	TargetType        string     `json:"target_type"`
	TargetDetail      []string   `json:"target_detail,omitempty"`
	ScheduledAt       *time.Time `json:"scheduled_at,omitempty"`
	Status            string     `json:"status"`
	SentCount         int        `json:"sent_count"`
	ReadCount         int        `json:"read_count"`
	IsScheduled       bool       `json:"is_scheduled"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// MessageCampaignListResponse 訊息活動列表回應 DTO
type MessageCampaignListResponse struct {
	Campaigns []MessageCampaignResponse `json:"campaigns"`
	Total     int                       `json:"total"`
	Page      int                       `json:"page"`
	PageSize  int                       `json:"page_size"`
}

// LevelResponse 等級回應 DTO
type LevelResponse struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	GlobalID string `json:"global_id"`
}

// LevelListResponse 等級列表回應 DTO
type LevelListResponse struct {
	Levels []LevelResponse `json:"levels"`
}

// TagResponse 標籤回應 DTO
type TagResponse struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	GlobalID string `json:"global_id"`
}

// TagListResponse 標籤列表回應 DTO
type TagListResponse struct {
	Tags []TagResponse `json:"tags"`
}

// Agent 相關 DTO 定義

// CreateAgentCampaignRequest 創建代理訊息活動請求 DTO
type CreateAgentCampaignRequest struct {
	GlobalMerchantID string     `json:"global_merchant_id"      swaggerignore:"true"` // 商户全局ID，由中间件自动设置
	Title            string     `json:"title"                    binding:"required,max=255"`
	Content          string     `json:"content"                  binding:"required"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	TargetType       string     `json:"target_type"              binding:"required,oneof=all specific line"`
	TargetDetails    []string   `json:"target_details,omitempty"`
	CreatedBy        string     `json:"created_by"               binding:"required,max=100"` // 创建者，最大100个字符
}

// UpdateAgentCampaignRequest 更新代理訊息活動請求 DTO
type UpdateAgentCampaignRequest struct {
	ID               uint64     `json:"id"`
	GlobalMerchantID string     `json:"global_merchant_id"      swaggerignore:"true"` // 商户全局ID，由中间件自动设置
	Title            *string    `json:"title,omitempty"          binding:"omitempty,max=255"`
	Content          *string    `json:"content,omitempty"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	TargetType       *string    `json:"target_type,omitempty"    binding:"omitempty,oneof=all specific line"`
	TargetDetails    []string   `json:"target_details,omitempty"`
	UpdatedBy        string     `json:"updated_by"               binding:"required,max=100"`
}

// AgentCampaignResponse 代理訊息活動回應 DTO
type AgentCampaignResponse struct {
	ID            uint64     `json:"id"`
	MerchantID    uint64     `json:"merchant_id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	Status        string     `json:"status"`
	TargetType    string     `json:"target_type"`
	TargetDetails []string   `json:"target_details"`
	TargetCount   int64      `json:"target_count"`
	RealSentCount int64      `json:"real_sent_count"`
	CreatedBy     string     `json:"created_by"`
	UpdatedBy     string     `json:"updated_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AgentCampaignsQuery 代理訊息活動查詢參數 DTO
type AgentCampaignsQuery struct {
	Page           int       `json:"page"`
	PageSize       int       `json:"page_size"`
	Limit          int       `json:"limit"`
	Offset         int       `json:"offset"`
	MerchantID     uint64    `json:"merchant_id"`
	Status         string    `json:"status"`
	TargetType     string    `json:"target_type"`
	IncludeDeleted bool      `json:"include_deleted"`
	CreatedBy      string    `json:"created_by"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	OrderBy        string    `json:"order_by"`
	OrderDirection string    `json:"order_direction"`
}

// AgentCampaignListResponse 代理訊息活動列表回應 DTO
type AgentCampaignListResponse struct {
	Campaigns []AgentCampaignResponse `json:"campaigns"`
	Page      int                     `json:"page"`
	PageSize  int                     `json:"page_size"`
	Total     int                     `json:"total"`
}

// AgentMessagesQuery 代理站內信查詢參數 DTO
type AgentMessagesQuery struct {
	Page           int       `json:"page"`
	PageSize       int       `json:"page_size"`
	Limit          int       `json:"limit"`
	Offset         int       `json:"offset"`
	AgentID        uint64    `json:"agent_id"`
	GlobalAgentID  string    `json:"global_agent_id"`
	IsRead         *bool     `json:"is_read,omitempty"`
	CampaignID     uint64    `json:"campaign_id,omitempty"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	OrderBy        string    `json:"order_by"`
	OrderDirection string    `json:"order_direction"`
}

// AgentMessageResponse 代理站內信回應 DTO
type AgentMessageResponse struct {
	ID              uint64     `json:"id"`
	AgentCampaignID uint64     `json:"agent_campaign_id"`
	AgentID         uint64     `json:"agent_id"`
	GlobalAgentID   string     `json:"global_agent_id"` // 通過關聯取得
	Title           string     `json:"title"`           // 通過關聯取得
	Content         string     `json:"content"`         // 通過關聯取得
	IsRead          bool       `json:"is_read"`
	ReadAt          *time.Time `json:"read_at"`
	SentAt          time.Time  `json:"sent_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentMessageListResponse 代理站內信列表回應 DTO
type AgentMessageListResponse struct {
	Stats    AgentMessageStats      `json:"stats"`
	Messages []AgentMessageResponse `json:"messages"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Total    int                    `json:"total"`
}

// AgentMessageStats 代理訊息統計 DTO
type AgentMessageStats struct {
	ReadCount   int64 `json:"read_count"`
	UnreadCount int64 `json:"unread_count"`
	TotalCount  int64 `json:"total_count"`
}
