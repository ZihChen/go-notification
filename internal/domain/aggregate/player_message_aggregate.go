package aggregate

import "time"

// PlayerMessageAggregate 玩家訊息聚合根
// 包含玩家訊息和相關的活動資訊，用於跨Aggregate查詢結果
type PlayerMessageAggregate struct {
	// PlayerMessage 基本資訊
	ID             uint64    `json:"id"`
	GlobalPlayerID string    `json:"global_player_id"`
	PlayerID       uint64    `json:"player_id"`
	CampaignID     uint64    `json:"campaign_id"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// MessageCampaign 關聯資訊
	CampaignTitle   string `json:"campaign_title"`
	CampaignContent string `json:"campaign_content"`
}

// PlayerMessageQueryResult 玩家訊息查詢結果
// 用於 Repository 層的原始查詢結果映射
type PlayerMessageQueryResult struct {
	ID              uint64    `gorm:"column:id"`
	GlobalPlayerID  string    `gorm:"column:global_player_id"`
	PlayerID        uint64    `gorm:"column:player_id"`
	CampaignID      uint64    `gorm:"column:campaign_id"`
	IsRead          bool      `gorm:"column:is_read"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
	CampaignTitle   string    `gorm:"column:campaign_title"`
	CampaignContent string    `gorm:"column:campaign_content"`
}

// ToAggregate 將查詢結果轉換為聚合根
func (r *PlayerMessageQueryResult) ToAggregate() *PlayerMessageAggregate {
	return &PlayerMessageAggregate{
		ID:              r.ID,
		GlobalPlayerID:  r.GlobalPlayerID,
		PlayerID:        r.PlayerID,
		CampaignID:      r.CampaignID,
		IsRead:          r.IsRead,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		CampaignTitle:   r.CampaignTitle,
		CampaignContent: r.CampaignContent,
	}
}
