package dto

import (
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// CreateMessageCampaignRequest 创建消息活动请求
type CreateMessageCampaignRequest struct {
	GlobalMerchantID string     `json:"global_merchant_id"      swaggerignore:"true"`                                                                                                                                                                                                                                                           // 商户全局ID，由中间件自动设置
	Category         string     `json:"category"                                     binding:"required,oneof=member bonus others"                                                   example:"member"               enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"` // 消息类型：member=会员消息，bonus=红利消息，others=其他消息
	Item             string     `json:"item"                                         binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration"         enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                           // 消息项目
	TriggerType      string     `json:"trigger_type"                                 binding:"omitempty,oneof=success failure"                                                      example:"success"              enums:"success,failure"`                                                                                                     // 触发类型：success=成功，failure=失败（可选）
	Title            string     `json:"title"                                        binding:"required,max=255"                                                                     example:"欢迎新用户"                                                                                              validate:"required,max=255"`                   // 消息标题，最大255个字符
	Content          string     `json:"content"                                      binding:"required"                                                                             example:"恭喜您成功注册成为我们的会员！"                                                                                    validate:"required"`                           // 消息内容，必填
	Target           string     `json:"target"                                       binding:"required,oneof=high_activity low_activity not_activity player level tag all"          example:"all"                  enums:"high_activity,low_activity,not_activity,player,level,tag,all"          validate:"required"`                           // 目标用户
	TargetDetail     []string   `json:"target_detail,omitempty"                                                                                                                     example:"['123', '456']"                                                                                     validate:"omitempty"`                          // 目标详情：player为账号数组，level/tag为ID字符串数组
	Status           string     `json:"status"                                       binding:"required,oneof=draft scheduled"                                                       example:"scheduled"`
	SendStartTime    *time.Time `json:"send_start_time"                                                                                                                             example:"2024-01-01T10:00:00Z"                                                                                                                             format:"date-time"` // 发送开始时间（可选）
	SendEndTime      *time.Time `json:"send_end_time"                                                                                                                               example:"2024-01-01T18:00:00Z"                                                                                                                             format:"date-time"` // 发送结束时间（可选）
	CreatedBy        string     `json:"created_by"                                   binding:"required,max=100"                                                                     example:"admin@example.com"                                                                                  validate:"required,max=100"`                                      // 创建者，最大100个字符
}

type ListMessageCampaignsRequest struct {
	Page         int      `form:"page"           json:"page"`
	PageSize     int      `form:"page_size"      json:"page_size"`
	Category     string   `form:"category"       json:"category"`       // 類型：member, bonus, others
	Item         string   `form:"item"           json:"item"`           // 項目：registration, identity_verification, bank_card, others, event, all, mission
	Status       []string `form:"status"         json:"status"`         // 狀態：draft, scheduled, sent, cancelled, failed
	ShowAutoSend bool     `form:"show_auto_send" json:"show_auto_send"` // 是否顯示站內系統建立
	CreatedBy    string   `form:"created_by"     json:"created_by"`     // 建立者
	StartAt      string   `form:"start_at"       json:"start_at"`       // 建立起始時間
	EndAt        string   `form:"end_at"         json:"end_at"`         // 建立結束時間
}

// UpdateMessageCampaignRequest 更新訊息活動請求
type UpdateMessageCampaignRequest struct {
	GlobalID         string     `json:"global_id"               swaggerignore:"true"`                                                                                                                                                                                                                                                           // 活动全局ID，由路径参数获取
	GlobalMerchantID string     `json:"global_merchant_id"      swaggerignore:"true"`                                                                                                                                                                                                                                                           // 商户全局ID，由中间件自动设置
	Category         string     `json:"category"                                     binding:"required,oneof=member bonus others"                                                   example:"member"               enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"` // 消息类型：member=会员消息，bonus=红利消息，others=其他消息
	Item             string     `json:"item"                                         binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration"         enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                           // 消息项目
	TriggerType      string     `json:"trigger_type"                                 binding:"omitempty,oneof=success failure"                                                      example:"success"              enums:"success,failure"`                                                                                                     // 触发类型：success=成功，failure=失败（可选）
	Title            string     `json:"title"                                        binding:"required,max=255"                                                                     example:"欢迎新用户"                                                                                              validate:"required,max=255"`                   // 消息标题，最大255个字符
	Content          string     `json:"content"                                      binding:"required"                                                                             example:"恭喜您成功注册成为我们的会员！"                                                                                    validate:"required"`                           // 消息内容，必填
	Target           string     `json:"target"                                       binding:"required,oneof=high_activity low_activity not_activity player level tag all"          example:"all"                  enums:"high_activity,low_activity,not_activity,player,level,tag,all"          validate:"required"`                           // 目标用户
	TargetDetail     []string   `json:"target_detail,omitempty"                                                                                                                     example:"['123', '456']"                                                                                     validate:"omitempty"`                          // 目标详情：player为账号数组，level/tag为ID字符串数组
	Status           string     `json:"status"                                       binding:"required,oneof=draft scheduled"                                                       example:"scheduled"`
	SendStartTime    *time.Time `json:"send_start_time"                                                                                                                             example:"2024-01-01T10:00:00Z"                                                                                                                             format:"date-time"` // 发送开始时间（可选）
	SendEndTime      *time.Time `json:"send_end_time"                                                                                                                               example:"2024-01-01T18:00:00Z"                                                                                                                             format:"date-time"` // 发送结束时间（可选）
	UpdatedBy        string     `json:"updated_by"                                   binding:"required,max=100"                                                                     example:"admin@example.com"                                                                                  validate:"required,max=100"`                                      // 更新者，最大100个字符
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
	Category    string `json:"category"     binding:"required,oneof=member bonus others"                                                   example:"member"          enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"` // 消息类型：member=会员消息，bonus=红利消息，others=其他消息
	Item        string `json:"item"         binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration"    enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                           // 消息项目
	TriggerType string `json:"trigger_type" binding:"required,oneof=success failure"                                                       example:"success"         enums:"success,failure"`                                                                                                     // 触发类型：success=成功，failure=失败
	Title       string `json:"title"        binding:"required,max=255"                                                                     example:"欢迎新用户"                                                                                         validate:"required,max=255"`                   // 消息标题，最大255个字符
	Content     string `json:"content"      binding:"required"                                                                             example:"恭喜您成功注册成为我们的会员！"                                                                               validate:"required"`                           // 消息内容，必填
}

// MerchantAutoSettingsRequest 商戶自動設定請求
type MerchantAutoSettingsRequest struct {
	GlobalMerchantID string            `json:"global_merchant_id" swaggerignore:"true"`                                                                  // 商户全局ID，由中间件自动设置
	Settings         []AutoSettingItem `json:"settings"                                binding:"required,min=1,max=10" validate:"required,min=1,max=10"` // 自动设置项目列表，最少1个，最多10个
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
