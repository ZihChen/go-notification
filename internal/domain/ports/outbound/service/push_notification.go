package service

import (
	"context"
)

// PushNotificationRequest App推播請求
type PushNotificationRequest struct {
	Accounts []string `json:"accounts"`  // 玩家帳號列表
	MsgTitle string   `json:"msg_title"` // 推播標題
	MsgBody  string   `json:"msg_body"`  // 推播內容
}

// PushNotificationResponse App推播回應
type PushNotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// PushNotificationService App推播服務介面
type PushNotificationService interface {
	// SendPushNotification 發送App推播通知
	SendPushNotification(
		ctx context.Context,
		apiKey string,
		req *PushNotificationRequest,
	) (*PushNotificationResponse, error)
}
