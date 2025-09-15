package entity

import (
	"time"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID            uint64     `json:"id"`
	Category      string     `json:"category"`     // member, bonus, others
	Item          string     `json:"item"`         // registration, identity_verification, bank_card, others, event, all, mission
	TriggerType   string     `json:"trigger_type"` // success, failure (for auto-send settings)
	Title         string     `json:"title"`
	MerchantID    uint64     `json:"merchant_id"`
	GlobalID      string     `json:"global_id"`
	Content       string     `json:"content"`         // 可包含 HTML Tag
	Target        string     `json:"target"`          // high_activity, low_activity, not_activity, player, level, tag, all
	Status        string     `json:"status"`          // draft, scheduled, sent, cancelled, failed
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
