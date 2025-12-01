package models

import "time"

// PlayerMessage 會員訊息模型
type PlayerMessage struct {
	ID             uint64    `json:"id"               gorm:"primaryKey;autoIncrement"`
	GlobalPlayerID string    `json:"global_player_id" gorm:"index;size:255;column:global_player_id;not null"`
	PlayerID       uint64    `json:"player_id"        gorm:"index:idx_player_message_player_id;column:player_id;not null"`
	CampaignID     uint64    `json:"campaign_id"      gorm:"index:idx_player_message_campaign_id;column:campaign_id;not null"`
	IsRead         bool      `json:"is_read"          gorm:"column:is_read;not null;default:false"`
	CreatedAt      time.Time `json:"created_at"       gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time `json:"updated_at"       gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (*PlayerMessage) TableName() string {
	return "player_message"
}
