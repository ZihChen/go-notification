package dto

import (
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

type CreateMessageCampaignRequest struct {
	Category         uint8      `json:"category"           binding:"required,min=1,max=3"`
	Item             uint8      `json:"item"               binding:"required,min=1,max=6"`
	Title            string     `json:"title"              binding:"required,max=255"`
	Content          string     `json:"content"            binding:"required"`
	Target           uint8      `json:"target"             binding:"required"`
	AutoSend         bool       `json:"auto_send"`
	TotalTargetCount int        `json:"total_target_count"`
	SendStartTime    *time.Time `json:"send_start_time"`
	SendEndTime      *time.Time `json:"send_end_time"`
	CreatedBy        string     `json:"created_by"         binding:"required,max=100"`
}

// UpdateMessageCampaignRequest 更新訊息活動請求
type UpdateMessageCampaignRequest struct {
	Category         uint8      `json:"category"           binding:"required,min=1,max=3"`
	Item             uint8      `json:"item"               binding:"required,min=1,max=6"`
	Title            string     `json:"title"              binding:"required,max=255"`
	Content          string     `json:"content"            binding:"required"`
	Target           uint8      `json:"target"             binding:"required"`
	AutoSend         bool       `json:"auto_send"`
	TotalTargetCount int        `json:"total_target_count"`
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

// MessageCampaignListResponse 活動列表響應
type MessageCampaignListResponse struct {
	Data     []*entity.MessageCampaign `json:"data"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
	Total    int                       `json:"total"`
}
