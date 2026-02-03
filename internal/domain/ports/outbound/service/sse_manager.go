package service

import (
	"context"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
)

// SSEManager SSE 連接管理器服務介面
// 負責管理 SSE 長連接、訊息推送、Redis Pub/Sub 路由、離線訊息處理
type SSEManager interface {
	// ==================== 連接管理 ====================

	// RegisterConnection 註冊玩家 SSE 連接
	// 1. 加入本地連接池 (本 Pod 記憶體)
	// 2. 更新全局玩家路由表 (Redis Hash: sse:player_routes)
	// 3. 增加線上玩家計數 (Redis String: sse:online_count)
	// 4. 訂閱 Pod 專屬 Pub/Sub 頻道 (sse:pod:{podID})
	RegisterConnection(ctx context.Context, playerID string, writer inbound.SSEWriter) error

	// UnregisterConnection 取消註冊玩家 SSE 連接
	// 1. 從本地連接池移除
	// 2. 從全局路由表移除 (Redis Hash)
	// 3. 減少線上玩家計數 (Redis String)
	UnregisterConnection(ctx context.Context, playerID string) error

	// IsPlayerOnline 檢查玩家是否在線上
	// 查詢全局路由表 (Redis Hash: sse:player_routes)
	IsPlayerOnline(ctx context.Context, playerID string) (bool, error)

	// GetOnlinePlayerCount 獲取線上玩家總數
	// 從 Redis 計數器讀取 (Redis String: sse:online_count)
	GetOnlinePlayerCount(ctx context.Context) (int, error)

	// GetOnlinePlayerIDs 獲取所有線上玩家 ID 清單
	// 從全局路由表讀取所有 keys (Redis Hash: sse:player_routes)
	GetOnlinePlayerIDs(ctx context.Context) ([]string, error)

	// ==================== 訊息推送 ====================

	// BroadcastToAll 廣播訊息給所有線上玩家
	// 1. 發布訊息到廣播頻道 (Redis Pub/Sub: sse:broadcast)
	// 2. 所有 Pod 收到訊息後，推送給各自本地連接池的玩家
	// 返回成功推送的玩家數量
	BroadcastToAll(ctx context.Context, notification *entity.SSENotification) (int, error)

	// SendToPlayer 推送訊息給單一玩家
	// 1. 查詢玩家所在 Pod (Redis Hash: sse:player_routes)
	// 2. 如果在本地，直接推送；如果在其他 Pod，透過 Pub/Sub 轉發
	// 3. 如果玩家離線，加入離線佇列 (Redis Streams)
	SendToPlayer(ctx context.Context, playerID string, notification *entity.SSENotification) error

	// SendToPlayers 批次推送訊息給多位玩家
	// 1. 批次查詢玩家路由表，按 Pod 分組
	// 2. 本地玩家直接推送，遠端玩家透過 Pub/Sub 轉發
	// 3. 離線玩家加入離線佇列
	// 返回成功推送的玩家數量
	SendToPlayers(ctx context.Context, playerIDs []string, notification *entity.SSENotification) (int, error)

	// ==================== 離線訊息處理 (Redis Streams) ====================

	// EnqueueOfflineMessage 將訊息加入玩家的離線佇列
	// 使用 Redis Streams 儲存，TTL 為 7 天
	// 單個玩家最多保留 100 條離線訊息 (MAXLEN)
	// Stream Key: sse:offline:{playerID}
	EnqueueOfflineMessage(ctx context.Context, playerID string, notification *entity.SSENotification) error

	// GetOfflineMessages 獲取玩家的離線訊息清單
	// 從 Redis Streams 讀取 (sse:offline:{playerID})
	// 按時間順序返回，最多 100 條
	GetOfflineMessages(ctx context.Context, playerID string) ([]*entity.SSENotification, error)

	// ClearOfflineMessages 清除玩家的離線訊息
	// 玩家上線後，推送完離線訊息即清空 Stream
	ClearOfflineMessages(ctx context.Context, playerID string) error
}
