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
