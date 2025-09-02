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

// MessageSummary 訊息摘要模型（列表顯示）
type MessageSummary struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// PlayerMessageStats 玩家訊息統計
type PlayerMessageStats struct {
	ReadCount   int64 `json:"read_count"`
	UnreadCount int64 `json:"unread_count"`
	TotalCount  int64 `json:"total_count"`
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
