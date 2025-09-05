package dto

import (
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// CreateMessageCampaignRequest 创建消息活动请求
type CreateMessageCampaignRequest struct {
	GlobalMerchantID string     `json:"global_merchant_id" swaggerignore:"true"`                                                                                                                                                                           // 商户全局ID，由中间件自动设置
	Category         uint8      `json:"category"                                binding:"required,min=1,max=3"            example:"1"                    swaggertype:"integer" enums:"1,2,3"           validate:"required,min=1,max=3"`                    // 消息类型：1=会员消息，2=红利消息，3=其他消息
	Item             uint8      `json:"item"                                    binding:"required,min=1,max=6"            example:"1"                    swaggertype:"integer" enums:"1,2,3,4,5,6"     validate:"required,min=1,max=6"`                    // 消息项目：1=注册，2=实名认证，3=存款，4=提款，5=投注，6=全部
	TriggerType      string     `json:"trigger_type"                            binding:"omitempty,oneof=success failure" example:"success"                                    enums:"success,failure"`                                                    // 触发类型：success=成功，failure=失败（可选）
	Title            string     `json:"title"                                   binding:"required,max=255"                example:"欢迎新用户"                                                              validate:"required,max=255"`                        // 消息标题，最大255个字符
	Content          string     `json:"content"                                 binding:"required"                        example:"恭喜您成功注册成为我们的会员！"                                                    validate:"required"`                                // 消息内容，必填
	Target           uint8      `json:"target"                                  binding:"required"                        example:"1"                    swaggertype:"integer"                         validate:"required"`                                // 目标用户：1=所有用户，2=特定用户组
	SendStartTime    *time.Time `json:"send_start_time"                                                                   example:"2024-01-01T10:00:00Z"                                                                               format:"date-time"` // 发送开始时间（可选）
	SendEndTime      *time.Time `json:"send_end_time"                                                                     example:"2024-01-01T18:00:00Z"                                                                               format:"date-time"` // 发送结束时间（可选）
	CreatedBy        string     `json:"created_by"                              binding:"required,max=100"                example:"admin@example.com"                                                  validate:"required,max=100"`                        // 创建者，最大100个字符
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
	GlobalID         string     `json:"global_id"          swaggerignore:"true"`                                                                                                                                                                           // 活动全局ID，由路径参数获取
	GlobalMerchantID string     `json:"global_merchant_id" swaggerignore:"true"`                                                                                                                                                                           // 商户全局ID，由中间件自动设置
	Category         uint8      `json:"category"                                binding:"required,min=1,max=3"            example:"1"                    swaggertype:"integer" enums:"1,2,3"           validate:"required,min=1,max=3"`                    // 消息类型：1=会员消息，2=红利消息，3=其他消息
	Item             uint8      `json:"item"                                    binding:"required,min=1,max=6"            example:"1"                    swaggertype:"integer" enums:"1,2,3,4,5,6"     validate:"required,min=1,max=6"`                    // 消息项目：1=注册，2=实名认证，3=存款，4=提款，5=投注，6=全部
	TriggerType      string     `json:"trigger_type"                            binding:"omitempty,oneof=success failure" example:"success"                                    enums:"success,failure"`                                                    // 触发类型：success=成功，failure=失败（可选）
	Title            string     `json:"title"                                   binding:"required,max=255"                example:"欢迎新用户"                                                              validate:"required,max=255"`                        // 消息标题，最大255个字符
	Content          string     `json:"content"                                 binding:"required"                        example:"恭喜您成功注册成为我们的会员！"                                                    validate:"required"`                                // 消息内容，必填
	Target           uint8      `json:"target"                                  binding:"required"                        example:"1"                    swaggertype:"integer"                         validate:"required"`                                // 目标用户：1=所有用户，2=特定用户组
	SendStartTime    *time.Time `json:"send_start_time"                                                                   example:"2024-01-01T10:00:00Z"                                                                               format:"date-time"` // 发送开始时间（可选）
	SendEndTime      *time.Time `json:"send_end_time"                                                                     example:"2024-01-01T18:00:00Z"                                                                               format:"date-time"` // 发送结束时间（可选）
	UpdatedBy        string     `json:"updated_by"                              binding:"required,max=100"                example:"admin@example.com"                                                  validate:"required,max=100"`                        // 更新者，最大100个字符
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
	Category    uint8  `json:"category"     binding:"required,min=1,max=3"           example:"1"               swaggertype:"integer" enums:"1,2,3"           validate:"required,min=1,max=3"` // 消息类型：1=会员消息，2=红利消息，3=其他消息
	Item        uint8  `json:"item"         binding:"required,min=1,max=6"           example:"1"               swaggertype:"integer" enums:"1,2,3,4,5,6"     validate:"required,min=1,max=6"` // 消息项目：1=注册，2=实名认证，3=存款，4=提款，5=投注，6=全部
	TriggerType string `json:"trigger_type" binding:"required,oneof=success failure" example:"success"                               enums:"success,failure"`                                 // 触发类型：success=成功，failure=失败
	Title       string `json:"title"        binding:"required,max=255"               example:"欢迎新用户"                                                         validate:"required,max=255"`     // 消息标题，最大255个字符
	Content     string `json:"content"      binding:"required"                       example:"恭喜您成功注册成为我们的会员！"                                               validate:"required"`             // 消息内容，必填
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
