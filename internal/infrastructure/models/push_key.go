package models

import (
	"time"
)

// PushKey 商戶推播API金鑰模型
type PushKey struct {
	ID               uint64    `json:"id"                 gorm:"primaryKey;autoIncrement"`
	GlobalMerchantID string    `json:"global_merchant_id" gorm:"column:global_merchant_id;size:100;not null;index"`
	MerchantID       uint64    `json:"merchant_id"        gorm:"column:merchant_id;not null;index"`
	Key              string    `json:"key"                gorm:"column:key;size:100;not null"`
	CreatedAt        time.Time `json:"created_at"         gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `json:"updated_at"         gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (*PushKey) TableName() string {
	return "push_keys"
}
