package infrastructure

import "context"

// MetricsService 定義 Metrics 服務的抽象介面
//
//go:generate mockery --name=MetricsService --output=../../../../../../test/mocks --outpkg=mocks
type MetricsService interface {
	// IsEnabled 檢查 Metrics 是否啟用
	IsEnabled() bool

	// Shutdown 關閉 Metrics 服務
	Shutdown(ctx context.Context) error

	// === API Metrics ===

	// RecordAPIRequest 記錄 API 請求
	RecordAPIRequest(method, path, status string)

	// RecordAPIDuration 記錄 API 延遲
	RecordAPIDuration(method, path string, duration float64)

	// RecordAPIError 記錄 API 錯誤
	RecordAPIError(method, path, errorType string)

	// === Campaign Metrics ===

	// RecordCampaignProcessed 記錄 Campaign 處理
	RecordCampaignProcessed(merchantID string, campaignType string)

	// RecordCampaignDuration 記錄 Campaign 處理延遲
	RecordCampaignDuration(merchantID string, duration float64)

	// RecordCampaignError 記錄 Campaign 處理錯誤
	RecordCampaignError(merchantID string, errorType string)

	// === Message Metrics ===

	// RecordMessageSent 記錄訊息發送
	RecordMessageSent(merchantID string, notificationType string)

	// RecordMessageDelivered 記錄訊息送達
	RecordMessageDelivered(merchantID string)

	// RecordMessageFailed 記錄訊息失敗
	RecordMessageFailed(merchantID string, reason string)

	// === Event Metrics ===

	// RecordEventReceived 記錄事件接收
	RecordEventReceived(eventType string)

	// RecordEventProcessed 記錄事件處理完成
	RecordEventProcessed(eventType string, success bool)

	// RecordEventDuration 記錄事件處理延遲
	RecordEventDuration(eventType string, duration float64)

	// === Worker Metrics ===

	// RecordTaskProcessed 記錄任務處理
	RecordTaskProcessed(taskType string, success bool)

	// RecordTaskDuration 記錄任務處理延遲
	RecordTaskDuration(taskType string, duration float64)
}
