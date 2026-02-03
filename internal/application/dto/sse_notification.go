package dto

// ==================== 後端推播 API DTO ====================

// BroadcastRequest 廣播推送請求
type BroadcastRequest struct {
	Title     string  `json:"title" binding:"required,max=100"`
	Message   string  `json:"message" binding:"required,max=500"`
	Type      string  `json:"type" binding:"required,oneof=activity promotion system announcement"`
	Priority  string  `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	IconURL   *string `json:"icon_url" binding:"omitempty,url"`
	ActionURL *string `json:"action_url" binding:"omitempty,url"`
}

// BroadcastResponse 廣播推送回應
type BroadcastResponse struct {
	NotificationID  string `json:"notification_id"`   // 通知 ID
	TotalOnline     int    `json:"total_online"`      // 線上玩家總數
	SuccessfulSends int    `json:"successful_sends"`  // 成功推送數量
	FailedSends     int    `json:"failed_sends"`      // 失敗推送數量
}

// SendNotificationRequest 個別推送請求
type SendNotificationRequest struct {
	PlayerIDs []string `json:"player_ids" binding:"required,min=1,max=1000,dive,required"`
	Title     string   `json:"title" binding:"required,max=100"`
	Message   string   `json:"message" binding:"required,max=500"`
	Type      string   `json:"type" binding:"required,oneof=activity promotion system announcement"`
	Priority  string   `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	IconURL   *string  `json:"icon_url" binding:"omitempty,url"`
	ActionURL *string  `json:"action_url" binding:"omitempty,url"`
}

// SendNotificationResponse 個別推送回應
type SendNotificationResponse struct {
	NotificationID  string `json:"notification_id"`   // 通知 ID
	TotalTargets    int    `json:"total_targets"`     // 目標玩家總數
	SuccessfulSends int    `json:"successful_sends"`  // 成功推送數量
	FailedSends     int    `json:"failed_sends"`      // 失敗推送數量
}
