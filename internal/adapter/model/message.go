package model

import (
	"time"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID               uint64     `json:"id"`
	Category         uint8      `json:"category"` // 1=member, 2=bonus, 3=others
	Item             uint8      `json:"item"`     // 1=registration, 2=identity_verification, ..., 6=all
	Title            string     `json:"title"`
	Content          string     `json:"content"`            // 可包含 HTML Tag
	Focus            uint8      `json:"focus"`              // 1=_in_thirty, 2=_low_activity, 3=_not_activity, 8=_one, 9=_level, 10=_tag, 11=_all
	AutoSend         bool       `json:"auto_send"`          // 是否為系統自動訊息
	TotalTargetCount int        `json:"total_target_count"` // 預估發送對象總人數
	RealSentCount    int        `json:"real_sent_count"`    // 實際成功發送人數
	SendStartTime    *time.Time `json:"send_start_time,omitempty"`
	SendEndTime      *time.Time `json:"send_end_time,omitempty"`
	CreatedBy        string     `json:"created_by"`           // 建立者帳號或名稱
	UpdatedBy        *string    `json:"updated_by,omitempty"` // 最後更新者帳號或名稱
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// PlayerMessage 會員訊息模型
type PlayerMessage struct {
	ID             uint64    `json:"id"`
	GlobalPlayerID string    `json:"global_player_id"`
	CampaignID     uint64    `json:"campaign_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
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

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Stats    PlayerMessageStats `json:"stats"`
	Messages []MessageSummary   `json:"messages"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
}

// FocusType 焦點類型常數
const (
	FocusInThirty    uint8 = 1  // 30天內活躍
	FocusLowActivity uint8 = 2  // 31-100天活躍
	FocusNotActivity uint8 = 3  // 100天以上不活躍
	FocusOne         uint8 = 8  // 特定玩家
	FocusLevel       uint8 = 9  // 特定等級
	FocusTag         uint8 = 10 // 特定標籤
	FocusAll         uint8 = 11 // 所有玩家
)

// Category 類別常數
const (
	CategoryMember uint8 = 1
	CategoryBonus  uint8 = 2
	CategoryOthers uint8 = 3
)

// Item 項目常數
const (
	ItemRegistration         uint8 = 1
	ItemIdentityVerification uint8 = 2
	ItemDeposit              uint8 = 3
	ItemWithdraw             uint8 = 4
	ItemPromotion            uint8 = 5
	ItemAll                  uint8 = 6
)
