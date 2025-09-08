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
	Page           int     `json:"page"`
	PageSize       int     `json:"page_size"`
	Category       uint8   `json:"category"`        // 類型：1=member, 2=bonus, 3=others
	Item           uint8   `json:"item"`            // 項目：1=registration, 2=identity_verification, ..., 6=all
	Status         []uint8 `json:"status"`          // 狀態：1=草稿 2=已排程 3=已發送 4=已取消 5=已歸檔
	IncludeDeleted bool    `json:"include_deleted"` // 是否包含已刪除的活動
	ShowAutoSend   bool    `json:"show_auto_send"`  // 是否顯示站內系統建立
	CreatedBy      string  `json:"created_by"`      // 建立者
	StartAt        string  `json:"start_at"`        // 建立起始時間
	EndAt          string  `json:"end_at"`          // 建立結束時間
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
	ID             uint64     `json:"id"`
	GlobalID       string     `json:"global_id"`
	MerchantID     uint64     `json:"merchant_id"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	TargetType     string     `json:"target_type"`
	TargetCriteria string     `json:"target_criteria,omitempty"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	Status         string     `json:"status"`
	SentCount      int        `json:"sent_count"`
	ReadCount      int        `json:"read_count"`
	IsScheduled    bool       `json:"is_scheduled"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MessageCampaignListResponse 訊息活動列表回應 DTO
type MessageCampaignListResponse struct {
	Campaigns []MessageCampaignResponse `json:"campaigns"`
	Total     int                       `json:"total"`
	Page      int                       `json:"page"`
	PageSize  int                       `json:"page_size"`
}
