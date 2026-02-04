package event

import (
	"time"
)

// CloudEvent 通用事件模型
type CloudEvent struct {
	SpecVersion     string      `json:"specversion"`
	Type            string      `json:"type"`
	Source          string      `json:"source"`
	Subject         string      `json:"subject"`
	ID              string      `json:"id"`
	Time            time.Time   `json:"time"`
	DataContentType string      `json:"datacontenttype"`
	TraceParent     string      `json:"traceparent"`
	Data            interface{} `json:"data"`
}

// MerchantSyncEvent 商戶同步事件數據
type MerchantSyncEvent struct {
	GlobalMerchantID string       `json:"global_merchant_id"`
	Merchant         MerchantData `json:"merchant"`
}

type MerchantEvent struct {
	GlobalMerchantID string    `json:"global_merchant_id"`
	ID               uint64    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	APIKey           string    `json:"api_key"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// MerchantData 從 KDS 接收的商戶數據
type MerchantData struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	GlobalMerchantID string `json:"global_merchant_id"`
	// 其他字段省略，因為不需要處理
}

// PlayerSyncEvent 玩家同步事件數據
type PlayerSyncEvent struct {
	GlobalMerchantID string     `json:"global_merchant_id"`
	Player           PlayerData `json:"player"`
	PlayerLevel      LevelData  `json:"player_level,omitempty"`
	PlayerTags       []TagData  `json:"player_tags,omitempty"`
}

type PlayerEvent struct {
	Account             string      `json:"account"`
	Email               string      `json:"email"`
	ApiKey              string      `json:"api_key"`
	GlobalMerchantID    string      `json:"global_merchant_id"`
	GlobalPlayerID      string      `json:"global_player_id"`
	GlobalPlayerLevelID string      `json:"global_player_level_id"`
	ID                  uint64      `json:"id"`
	MerchantID          uint64      `json:"merchant_id"`
	LastActiveAt        time.Time   `json:"last_active_at"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	PlayerLevel         PlayerLevel `json:"player_level,omitempty"`
}

type PlayerLevel struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// PlayerData 從 KDS 接收的玩家數據
type PlayerData struct {
	GlobalPlayerID string `json:"global_player_id"`
	Account        string `json:"account"`
	Email          string `json:"email,omitempty"`
	Status         string `json:"status"`
}

// LevelData 玩家等級數據
type LevelData struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// TagData 玩家標籤數據
type TagData struct {
	Tag struct {
		GlobalTagID string `json:"global_tag_id"`
		Name        string `json:"name"`
	} `json:"tag"`
}

// ManagerSyncEvent 管理員同步事件數據
type ManagerSyncEvent struct {
	GlobalMerchantID string      `json:"global_merchant_id"`
	Manager          ManagerData `json:"manager"`
}

type ManagerEvent struct {
	GlobalMerchantID string    `json:"global_merchant_id"`
	GlobalManagerID  string    `json:"global_manager_id"`
	ID               uint64    `json:"id"`
	MerchantID       uint64    `json:"merchant_id"`
	Account          string    `json:"account"`
	Email            string    `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        string    `json:"deleted_at,omitempty"`
}

// ManagerData 從 KDS 接收的管理員數據
type ManagerData struct {
	ID              int    `json:"id"`
	GlobalManagerID string `json:"global_manager_id"`
	Account         string `json:"account"`
	Email           string `json:"email"`
	DeletedAt       string `json:"deleted_at,omitempty"`
}

// IdentityMerchantSyncEvent 發送到 KDS 的商戶同步事件
type IdentityMerchantSyncEvent struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	APIKey           string `json:"api_key"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	DeletedAt        string `json:"deleted_at,omitempty"`
}

// IdentityPlayerSyncEvent 發送到 KDS 的玩家同步事件
type IdentityPlayerSyncEvent struct {
	GlobalMerchantID string  `json:"global_merchant_id"`
	GlobalPlayerID   string  `json:"global_player_id"`
	ID               uint64  `json:"id"`
	MerchantID       uint64  `json:"merchant_id"`
	APIKey           string  `json:"api_key"`
	Account          string  `json:"account"`
	Email            *string `json:"email,omitempty"`
	LastActiveAt     string  `json:"last_active_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DeletedAt        string  `json:"deleted_at,omitempty"`
	// 新增：玩家標籤資料，統一在玩家同步事件中處理
	Tags        []*IdentityTagDataSyncEvent `json:"tags,omitempty"`
	PlayerLevel *PlayerLevel                `json:"player_level,omitempty"`
}

// IdentityManagerSyncEvent 發送到 KDS 的管理員同步事件
type IdentityManagerSyncEvent struct {
	GlobalMerchantID string  `json:"global_merchant_id"`
	GlobalManagerID  string  `json:"global_manager_id"`
	ID               uint64  `json:"id"`
	MerchantID       uint64  `json:"merchant_id"`
	Account          string  `json:"account"`
	Email            *string `json:"email,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DeletedAt        string  `json:"deleted_at,omitempty"`
}

// IdentityPlayerLevelSyncEvent 發送到 KDS 的玩家等級同步事件
type IdentityPlayerLevelSyncEvent struct {
	GlobalMerchantID    string    `json:"global_merchant_id"`
	GlobalPlayerLevelID string    `json:"global_player_level_id"`
	Name                string    `json:"name"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	DeletedAt           string    `json:"deleted_at,omitempty"`
}

type IdentityPlayerTagSyncEvent struct {
	GlobalMerchantID string                      `json:"global_merchant_id"`
	GlobalPlayerID   string                      `json:"global_player_id"`
	Tags             []*IdentityTagDataSyncEvent `json:"tags"`
}

type IdentityTagSyncEvent struct {
	GlobalMerchantID string                    `json:"global_merchant_id"`
	Tag              *IdentityTagDataSyncEvent `json:"tag"`
}

type IdentityTagDataSyncEvent struct {
	GlobalTagID string `json:"global_tag_id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

// AgentSyncEvent 代理同步事件數據 (根據新的KDS事件結構)
type AgentSyncEvent struct {
	GlobalAgentID    string     `json:"global_agent_id"`
	Account          string     `json:"account"`
	Ancestry         string     `json:"ancestry"`
	GlobalMerchantID string     `json:"global_merchant_id"`
	ID               uint64     `json:"id"`
	MerchantID       uint64     `json:"merchant_id"`
	CurrentSignInAt  *time.Time `json:"current_sign_in_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SSE 通知事件 (發送到 KDS 作為審計日誌)

// SSEBroadcastEvent 廣播推送事件
type SSEBroadcastEvent struct {
	EventID           string                 `json:"event_id"`
	MerchantID        uint64                 `json:"merchant_id"`
	GlobalMerchantID  string                 `json:"global_merchant_id"`
	NotificationType  string                 `json:"notification_type"`
	Title             string                 `json:"title"`
	Message           string                 `json:"message"`
	Priority          string                 `json:"priority"`
	TargetPlayerCount int                    `json:"target_player_count"`
	OnlinePlayerCount int                    `json:"online_player_count"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
}

// SSESendEvent 個別推送事件
type SSESendEvent struct {
	EventID          string                 `json:"event_id"`
	MerchantID       uint64                 `json:"merchant_id"`
	GlobalMerchantID string                 `json:"global_merchant_id"`
	PlayerID         uint64                 `json:"player_id"`
	GlobalPlayerID   string                 `json:"global_player_id"`
	NotificationType string                 `json:"notification_type"`
	Title            string                 `json:"title"`
	Message          string                 `json:"message"`
	Priority         string                 `json:"priority"`
	IsOnline         bool                   `json:"is_online"`
	TargetPodID      string                 `json:"target_pod_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// SSEPlayerConnectEvent 玩家連接事件
type SSEPlayerConnectEvent struct {
	PlayerID         uint64    `json:"player_id"`
	GlobalPlayerID   string    `json:"global_player_id"`
	MerchantID       uint64    `json:"merchant_id"`
	GlobalMerchantID string    `json:"global_merchant_id"`
	PodID            string    `json:"pod_id"`
	ClientIP         string    `json:"client_ip,omitempty"`
	UserAgent        string    `json:"user_agent,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// SSEPlayerDisconnectEvent 玩家斷線事件
type SSEPlayerDisconnectEvent struct {
	PlayerID         uint64        `json:"player_id"`
	GlobalPlayerID   string        `json:"global_player_id"`
	MerchantID       uint64        `json:"merchant_id"`
	GlobalMerchantID string        `json:"global_merchant_id"`
	PodID            string        `json:"pod_id"`
	ConnectionTime   time.Duration `json:"connection_time_seconds"`
	Timestamp        time.Time     `json:"timestamp"`
}
