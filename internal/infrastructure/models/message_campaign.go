package models

import (
	"time"

	"gorm.io/gorm"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID                uint64         `json:"id"                        gorm:"primaryKey;autoIncrement"`
	Category          string         `json:"category"                  gorm:"column:category;size:255;not null;index"`                // member, bonus, others
	Item              string         `json:"item"                      gorm:"column:item;size:255;not null"`                          // registration, identity_verification, bank_card, others, event, all, mission
	TriggerType       string         `json:"trigger_type"              gorm:"column:trigger_type;size:20;not null;default:'success'"` // success, failure (for auto-send settings)
	MerchantID        uint64         `json:"merchant_id"               gorm:"index;not null;default:0"`
	GlobalID          string         `json:"global_id"                 gorm:"uniqueIndex;size:100;not null;default:''"`
	LegacyID          *uint          `json:"legacy_id,omitempty"       gorm:"column:legacy_id;index;comment:舊系統notification ID，用於資料遷移"`
	Title             string         `json:"title"                     gorm:"size:255;column:title;not null"`
	Content           string         `json:"content"                   gorm:"column:content;type:mediumtext"`                        // 可包含 HTML Tag
	AppContent        *string        `json:"app_content,omitempty"     gorm:"column:app_content;size:255"`                           // App推播內容
	NotificationTypes uint8          `json:"notification_types"        gorm:"column:notification_types;not null;default:1"`          // 推送類型位元遮罩: 1=站內信, 2=App推播, 3=站內信+App推播
	Status            string         `json:"status"                    gorm:"column:status;size:255;not null;default:'draft';index"` // draft, scheduled, sent, cancelled, failed
	Target            string         `json:"target"                    gorm:"column:target;size:255;not null"`                       // high_activity, low_activity, not_activity, player, level, tag, all
	TargetDetail      *string        `json:"target_detail"             gorm:"column:target_detail;size:512"`                         // JSON string containing target-specific details (player accounts, level names, tag names)
	AutoSend          bool           `json:"auto_send"                 gorm:"column:auto_send;not null;default:false"`               // 是否為系統自動訊息
	Active            bool           `json:"active"                    gorm:"column:active;not null;default:true"`                   // 是否啟用自動發送（僅適用於AutoSend=true的訊息）
	RealSentCount     int64          `json:"real_sent_count"           gorm:"column:real_sent_count;default:0"`                      // 實際成功發送人數
	SendStartTime     *time.Time     `json:"send_start_time,omitempty" gorm:"type:datetime;column:send_start_time"`
	SendEndTime       *time.Time     `json:"send_end_time,omitempty"   gorm:"type:datetime;column:send_end_time"`
	CreatedBy         string         `json:"created_by"                gorm:"size:255;column:created_by;not null;index"` // 建立者帳號或名稱
	UpdatedBy         *string        `json:"updated_by,omitempty"      gorm:"size:255;column:updated_by"`                // 最後更新者帳號或名稱
	CreatedAt         time.Time      `json:"created_at"                gorm:"type:datetime;default:CURRENT_TIMESTAMP;index"`
	UpdatedAt         time.Time      `json:"updated_at"                gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty"      gorm:"index"`
}

func (*MessageCampaign) TableName() string {
	return "message_campaign"
}
