package models

import (
	"time"
)

// CampaignTarget 訊息活動目標關聯表模型
type CampaignTarget struct {
	CampaignID uint64    `json:"campaign_id" gorm:"not null;index:idx_campaign_id"`
	TargetType string    `json:"target_type" gorm:"not null;size:20;index:idx_target_type_id"` // player, level, tag, all
	TargetID   uint64    `json:"target_id"   gorm:"index:idx_target_type_id"`                  // 目標ID：player_id/level_id/tag_id，all類型為NULL
	CreatedAt  time.Time `json:"created_at"  gorm:"type:datetime;default:CURRENT_TIMESTAMP;autoCreateTime"`
}

func (*CampaignTarget) TableName() string {
	return "campaign_targets"
}
