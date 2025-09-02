package dto

import (
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

type CreateMessageCampaignRequest struct {
	GlobalMerchantID string     `json:"global_merchant_id"`
	Category         uint8      `json:"category"           binding:"required,min=1,max=3"`
	Item             uint8      `json:"item"               binding:"required,min=1,max=6"`
	TriggerType      string     `json:"trigger_type"       binding:"omitempty,oneof=success failure"`
	Title            string     `json:"title"              binding:"required,max=255"`
	Content          string     `json:"content"            binding:"required"`
	Target           uint8      `json:"target"             binding:"required"`
	SendStartTime    *time.Time `json:"send_start_time"`
	SendEndTime      *time.Time `json:"send_end_time"`
	CreatedBy        string     `json:"created_by"         binding:"required,max=100"`
}

type ListMessageCampaignsRequest struct {
	Page         int     `form:"page"           json:"page"`
	PageSize     int     `form:"page_size"      json:"page_size"`
	Category     uint8   `form:"category"       json:"category"`       // 類型：1=member, 2=bonus, 3=others
	Item         uint8   `form:"item"           json:"item"`           // 項目：1=registration, 2=identity_verification, ..., 6=all
	Status       []uint8 `form:"status"         json:"status"`         // 狀態：1=草稿 2=已排程 3=已發送 4=已取消 5=已歸檔
	ShowAutoSend bool    `form:"show_auto_send" json:"show_auto_send"` // 是否顯示站內系統建立
	CreatedBy    string  `form:"created_by"     json:"created_by"`     // 建立者
	StartAt      string  `form:"start_at"       json:"start_at"`       // 建立起始時間
	EndAt        string  `form:"end_at"         json:"end_at"`         // 建立結束時間
}

// UpdateMessageCampaignRequest 更新訊息活動請求
type UpdateMessageCampaignRequest struct {
	GlobalID         string     `json:"global_id"`
	GlobalMerchantID string     `json:"global_merchant_id"`
	Category         uint8      `json:"category"           binding:"required,min=1,max=3"`
	Item             uint8      `json:"item"               binding:"required,min=1,max=6"`
	TriggerType      string     `json:"trigger_type"       binding:"omitempty,oneof=success failure"`
	Title            string     `json:"title"              binding:"required,max=255"`
	Content          string     `json:"content"            binding:"required"`
	Target           uint8      `json:"target"             binding:"required"`
	SendStartTime    *time.Time `json:"send_start_time"`
	SendEndTime      *time.Time `json:"send_end_time"`
	UpdatedBy        string     `json:"updated_by"         binding:"required,max=100"`
}

// ErrorResponse 錯誤響應
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse 成功響應
type SuccessResponse struct {
	Status string `json:"status"`
}

// AutoSettingItem 自動設定項目
type AutoSettingItem struct {
	Category    uint8  `json:"category"     binding:"required,min=1,max=3"`
	Item        uint8  `json:"item"         binding:"required,min=1,max=6"`
	TriggerType string `json:"trigger_type" binding:"required,oneof=success failure"`
	Title       string `json:"title"        binding:"required,max=255"`
	Content     string `json:"content"      binding:"required"`
}

// MerchantAutoSettingsRequest 商戶自動設定請求
type MerchantAutoSettingsRequest struct {
	GlobalMerchantID string            `json:"global_merchant_id"`
	Settings         []AutoSettingItem `json:"settings"           binding:"required,min=1,max=10"`
}

// MerchantAutoSettingsResponse 商戶自動設定回應
type MerchantAutoSettingsResponse struct {
	GlobalMerchantID string                    `json:"global_merchant_id"`
	Settings         []*entity.MessageCampaign `json:"settings"`
}

// AutoSettingsOperationResponse 自動設定操作回應
type AutoSettingsOperationResponse struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	CreatedCount     int    `json:"created_count,omitempty"`
	UpdatedCount     int    `json:"updated_count,omitempty"`
	Operation        string `json:"operation"` // create, update
}
