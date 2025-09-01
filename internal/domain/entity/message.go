package entity

import (
	"time"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID            uint64     `json:"id"`
	Category      uint8      `json:"category"`     // 1=member, 2=bonus, 3=others
	Item          uint8      `json:"item"`         // 1=registration, 2=identity_verification, ..., 6=all
	TriggerType   string     `json:"trigger_type"` // success, failure (for auto-send settings)
	Title         string     `json:"title"`
	MerchantID    uint64     `json:"merchant_id"`
	GlobalID      string     `json:"global_id"`
	Content       string     `json:"content"`         // 可包含 HTML Tag
	Target        uint8      `json:"target"`          // 1=_in_thirty, 2=_low_activity, 3=_not_activity, 8=_one, 9=_level, 10=_tag, 11=_all
	Status        uint8      `json:"status"`          // 1=draft, 2=scheduled, 3=sent, 4=cancelled, 5=archived
	AutoSend      bool       `json:"auto_send"`       // 是否為系統自動訊息
	RealSentCount int64      `json:"real_sent_count"` // 實際成功發送人數
	SendStartTime *time.Time `json:"send_start_time,omitempty"`
	SendEndTime   *time.Time `json:"send_end_time,omitempty"`
	CreatedBy     string     `json:"created_by"`           // 建立者帳號或名稱
	UpdatedBy     *string    `json:"updated_by,omitempty"` // 最後更新者帳號或名稱
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// PlayerMessage 會員訊息模型
type PlayerMessage struct {
	ID             uint64    `json:"id"`
	GlobalPlayerID string    `json:"global_player_id"`
	PlayerID       uint64    `json:"player_id"`
	CampaignID     uint64    `json:"campaign_id"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Stats    PlayerMessageStats `json:"stats"`
	Messages []MessageSummary   `json:"messages"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
}

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
