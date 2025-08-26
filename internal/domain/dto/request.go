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
