package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
)

// Merchant 商戶模型
type Merchant struct {
	ID               uint64     `json:"id"`
	GlobalMerchantID string     `json:"global_merchant_id"`
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	APIKey           string     `json:"api_key"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// Player 玩家模型
type Player struct {
	ID             uint64     `json:"id"`
	MerchantID     uint64     `json:"merchant_id"`
	LevelID        uint64     `json:"level_id"`
	GlobalPlayerID string     `json:"global_player_id"`
	APIKey         string     `json:"api_key"`
	Account        string     `json:"account"`
	Email          *string    `json:"email,omitempty"`
	LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// ==== 玩家活躍度管理業務方法 ====

// DetermineFocusType 根據玩家最後活躍時間判斷焦點類型
func (p *Player) DetermineFocusType() string {
	if p.LastActiveAt == nil {
		return consts.TargetNotActivity // 沒有活躍記錄視為不活躍
	}

	now := time.Now()
	daysSinceActive := int(now.Sub(*p.LastActiveAt).Hours() / 24)

	switch {
	case daysSinceActive <= 30:
		return consts.TargetHighActivity // 30天內
	case daysSinceActive <= 100:
		return consts.TargetLowActivity // 31-100天
	default:
		return consts.TargetNotActivity // 100天以上
	}
}

// UpdateLastActive 更新最後活躍時間
func (p *Player) UpdateLastActive() {
	now := time.Now()
	p.LastActiveAt = &now
	p.UpdatedAt = now
}

// ==== API金鑰管理業務方法 ====

// RegenerateAPIKey 重新產生API金鑰
func (p *Player) RegenerateAPIKey() {
	p.APIKey = uuid.New().String()
	p.UpdatedAt = time.Now()
}

// ValidateAPIKey 驗證API金鑰
func (p *Player) ValidateAPIKey(apiKey string) bool {
	return p.APIKey == apiKey
}

// Manager 管理員模型
type Manager struct {
	ID              uint64     `json:"id"`
	MerchantID      uint64     `json:"merchant_id"`
	GlobalManagerID string     `json:"global_manager_id"`
	Account         string     `json:"account"`
	Email           *string    `json:"email,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type Tag struct {
	ID          uint64     `json:"id"`
	MerchantID  uint64     `json:"merchant_id"`
	Name        string     `json:"name"`
	GlobalTagID string     `json:"global_tag_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Level struct {
	ID                  uint64     `json:"id"`
	MerchantID          uint64     `json:"merchant_id"`
	Name                string     `json:"name"`
	GlobalPlayerLevelID string     `json:"global_player_level_id"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
}

type PlayerTag struct {
	PlayerID  uint64    `json:"player_id"`
	TagID     uint64    `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

// LoggerFiled 日誌欄位結構
type LoggerFiled struct {
	Key   string
	Value interface{}
}
