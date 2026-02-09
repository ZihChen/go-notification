package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
)

// SSENotificationUseCase SSE 通知使用案例介面 (簡化版)
type SSENotificationUseCase interface {
	// BroadcastNotification 廣播推送通知給所有線上玩家
	// 透過 Redis Pub/Sub 廣播到所有 SSE Service Pods
	// 返回成功推送的玩家數量
	BroadcastNotification(
		ctx context.Context,
		req *dto.BroadcastRequest,
	) (*dto.BroadcastResponse, error)

	// SendNotification 個別推送通知給特定玩家清單
	// 透過玩家路由表查詢目標 Pod，並通過 Pub/Sub 轉發
	// 支援批次推送，最多 1000 位玩家
	SendNotification(
		ctx context.Context,
		req *dto.SendNotificationRequest,
	) (*dto.SendNotificationResponse, error)

	// StreamNotifications 建立 SSE 長連接，持續推送實時通知
	// 當玩家連線時：
	// 1. 註冊玩家連接到本地連接池
	// 2. 更新玩家路由表 (Redis Hash)
	// 3. 推送累積的離線訊息 (Redis Streams)
	// 4. 保持連接直到 context 取消
	StreamNotifications(ctx context.Context, playerID string, writer SSEWriter) error
}

// SSEWriter SSE 寫入器介面 (抽象化 HTTP ResponseWriter)
// 讓 Domain Layer 不依賴於具體的 HTTP 框架
type SSEWriter interface {
	// Write 寫入 SSE 事件
	// event: 事件類型 (connected, notification, ping, error)
	// data: JSON 格式的事件數據
	Write(event string, data string) error

	// Flush 立即推送緩衝區的數據到客戶端
	Flush() error

	// Close 關閉 SSE 連接
	Close() error
}
