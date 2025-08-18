package models

import (
	"time"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID            uint64     `json:"id"                        gorm:"primaryKey;autoIncrement"`
	Category      uint8      `json:"category"                  gorm:"column:category;not null"` // 1=member, 2=bonus, 3=others
	Item          uint8      `json:"item"                      gorm:"column:item;not null"`     // 1=registration, 2=identity_verification, ..., 6=all
	Title         string     `json:"title"                     gorm:"size:255;column:title;not null"`
	Content       string     `json:"content"                   gorm:"column:content;type:mediumtext"`          // 可包含 HTML Tag
	Target        uint8      `json:"target"                    gorm:"column:target;not null"`                  // 1=_high_activity, 2=_low_activity, 3=_not_activity, 8=_one, 9=_level, 10=_tag, 11=_all
	AutoSend      bool       `json:"auto_send"                 gorm:"column:auto_send;not null;default:false"` // 是否為系統自動訊息
	RealSentCount int64      `json:"real_sent_count"           gorm:"column:real_sent_count;default:0"`        // 實際成功發送人數
	SendStartTime *time.Time `json:"send_start_time,omitempty" gorm:"type:datetime;column:send_start_time"`
	SendEndTime   *time.Time `json:"send_end_time,omitempty"   gorm:"type:datetime;column:send_end_time"`
	CreatedBy     string     `json:"created_by"                gorm:"size:255;column:created_by;not null"` // 建立者帳號或名稱
	UpdatedBy     *string    `json:"updated_by,omitempty"      gorm:"size:255;column:updated_by"`          // 最後更新者帳號或名稱
	CreatedAt     time.Time  `json:"created_at"                gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `json:"updated_at"                gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (*MessageCampaign) TableName() string {
	return "message_campaign"
}
