package entity

import (
	"time"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID                uint64     `json:"id"`
	Category          string     `json:"category"`     // member, bonus, others
	Item              string     `json:"item"`         // registration, identity_verification, bank_card, others, event, all, mission
	TriggerType       string     `json:"trigger_type"` // success, failure (for auto-send settings)
	Title             string     `json:"title"`
	MerchantID        uint64     `json:"merchant_id"`
	GlobalID          string     `json:"global_id"`
	LegacyID          *uint      `json:"legacy_id,omitempty"`     // 舊系統的 notification ID，用於資料遷移
	Content           string     `json:"content"`                 // 可包含 HTML Tag
	AppContent        *string    `json:"app_content,omitempty"`   // App推播內容
	NotificationTypes uint8      `json:"notification_types"`      // 推送類型位元遮罩: 1=站內信, 2=App推播, 4=其他
	Target            string     `json:"target"`                  // high_activity, low_activity, not_activity, player, level, tag, all
	TargetDetail      *string    `json:"target_detail,omitempty"` // JSON string containing target-specific details (player accounts, level names, tag names)
	Status            string     `json:"status"`                  // draft, scheduled, sent, cancelled, failed
	AutoSend          bool       `json:"auto_send"`               // 是否為系統自動訊息
	RealSentCount     int64      `json:"real_sent_count"`         // 實際成功發送人數
	SendStartTime     *time.Time `json:"send_start_time,omitempty"`
	SendEndTime       *time.Time `json:"send_end_time,omitempty"`
	CreatedBy         string     `json:"created_by"`           // 建立者帳號或名稱
	UpdatedBy         *string    `json:"updated_by,omitempty"` // 最後更新者帳號或名稱
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

// PushKey 商戶推播API金鑰模型
type PushKey struct {
	ID               uint64    `json:"id"`
	GlobalMerchantID string    `json:"global_merchant_id"`
	MerchantID       uint64    `json:"merchant_id"`
	Key              string    `json:"key"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
