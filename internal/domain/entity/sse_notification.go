package entity

import "time"

// SSENotification SSE 實時推播通知實體
// 注意：此實體僅用於記憶體中的訊息傳遞，不進行 MySQL 持久化
type SSENotification struct {
	ID        string    `json:"id"`                   // UUID v4
	Title     string    `json:"title"`                // 標題 (最多 100 字元)
	Message   string    `json:"message"`              // 訊息內容 (最多 500 字元)
	Type      string    `json:"type"`                 // 通知類型: activity|promotion|system|announcement
	Priority  string    `json:"priority"`             // 優先級: low|medium|high|critical
	IconURL   *string   `json:"icon_url,omitempty"`   // 圖示 URL (選填)
	ActionURL *string   `json:"action_url,omitempty"` // 點擊後跳轉的 URL (選填)
	CreatedAt time.Time `json:"created_at"`           // 建立時間
	ExpiresAt time.Time `json:"expires_at"`           // 過期時間
}

// IsExpired 檢查通知是否已過期
func (n *SSENotification) IsExpired() bool {
	return time.Now().After(n.ExpiresAt)
}
