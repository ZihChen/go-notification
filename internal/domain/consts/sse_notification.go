package consts

import "time"

// 通知類型常數
const (
	NotificationTypeActivity     = "activity"     // 活動通知
	NotificationTypePromotion    = "promotion"    // 促銷通知
	NotificationTypeSystem       = "system"       // 系統通知
	NotificationTypeAnnouncement = "announcement" // 公告通知
)

// 優先級常數
const (
	NotificationPriorityLow      = "low"      // 低優先級
	NotificationPriorityMedium   = "medium"   // 中優先級
	NotificationPriorityHigh     = "high"     // 高優先級
	NotificationPriorityCritical = "critical" // 緊急優先級
)

// 推送狀態常數
const (
	NotificationStatusQueued    = "queued"    // 已排隊
	NotificationStatusDelivered = "delivered" // 已遞送
	NotificationStatusFailed    = "failed"    // 遞送失敗
)

// TTL 和限制常數
const (
	NotificationDefaultTTL        = 24 * time.Hour     // 預設通知有效期 24 小時
	NotificationOfflineMessageTTL = 7 * 24 * time.Hour // 離線訊息保留 7 天
	NotificationMaxTitleLength    = 100                // 標題最大長度
	NotificationMaxMessageLength  = 500                // 訊息最大長度
	NotificationMaxBatchSize      = 1000               // 單次推送最多玩家數量
	NotificationMaxOfflineQueue   = 100                // 單個玩家最多保留離線訊息數量
)

// Redis Keys 常數
const (
	RedisKeySSEPlayerRoutes   = "sse:player_routes" // Hash: 玩家路由表 (playerID → podID)
	RedisKeySSEOnlineCount    = "sse:online_count"  // String: 線上玩家計數器
	RedisKeySSEPodStatsPrefix = "sse:pod_stats:"    // Hash: Pod 統計資訊前綴
	RedisKeySSEOfflinePrefix  = "sse:offline:"      // Stream: 離線訊息前綴 (sse:offline:{playerID})

	// 簡化別名（向後兼容）
	SSEPlayerRoutesKey    = RedisKeySSEPlayerRoutes // 玩家路由表 Key
	SSEOnlineCountKey     = RedisKeySSEOnlineCount  // 線上計數 Key
	SSEOfflineMessagesKey = "sse:offline:%s"        // 離線訊息 Key 格式化字串
)

// Redis Pub/Sub Channel 常數
const (
	RedisChannelSSEBroadcast = "sse:broadcast" // Pub/Sub: 廣播頻道 (所有 Pod 訂閱)
	RedisChannelSSEPodPrefix = "sse:pod:"      // Pub/Sub: Pod 專屬頻道前綴 (sse:pod:{podID})

	// 簡化別名（向後兼容）
	SSEBroadcastChannel = RedisChannelSSEBroadcast // 廣播頻道
	SSEPodChannel       = "sse:pod:%s"             // Pod 頻道格式化字串
)

// SSE 事件類型常數
const (
	SSEEventTypeConnected    = "connected"    // 連接成功事件
	SSEEventTypeNotification = "notification" // 通知事件
	SSEEventTypePing         = "ping"         // 心跳檢測事件
	SSEEventTypeError        = "error"        // 錯誤事件
)

// SSE 心跳間隔
const (
	SSEHeartbeatInterval = 30 * time.Second // 心跳間隔 30 秒
)
