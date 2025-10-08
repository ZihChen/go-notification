package dto

import (
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// CreateMessageCampaignRequest 创建消息活动请求
type CreateMessageCampaignRequest struct {
	GlobalMerchantID  string     `json:"global_merchant_id"      swaggerignore:"true"`                                                                                                                                                                                                                                                                                      // 商户全局ID，由中间件自动设置
	Category          string     `json:"category"                                     binding:"required,oneof=member bonus others"                                                   example:"member"                       enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"`                    // 消息类型：member=会员消息，bonus=红利消息，others=其他消息。可選值：member, bonus, others
	Item              string     `json:"item"                                         binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration"                 enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                                              // 消息项目。可選值：registration(註冊), identity_verification(身份驗證), bank_card(銀行卡), others(其他), event(活動), all(全部), mission(任務)
	Title             string     `json:"title"                                        binding:"required,max=255"                                                                     example:"欢迎新用户"                                                                                                      validate:"required,max=255"`                                      // 消息标题，最大255个字符
	Content           string     `json:"content"                                      binding:"required"                                                                             example:"恭喜您成功注册成为我们的会员！"                                                                                            validate:"required"`                                              // 消息内容，必填
	AppContent        *string    `json:"app_content,omitempty"                                                                                                                       example:"恭喜您成功注册！"                                                                                                   validate:"omitempty,max=255"`                                     // App推播內容，可選，最大255個字符
	NotificationTypes *uint8     `json:"notification_types"                           binding:"required,min=1,max=7"                                                                 example:"3"                                                                                                          validate:"required,min=1,max=7"`                                  // 推送類型位元遮罩：1=站內信，2=App推播，4=其他，可組合使用，範圍1-7，必填。示例：1(僅站內信), 2(僅App推播), 3(站內信+App推播)
	Target            string     `json:"target"                                       binding:"required,oneof=high_activity low_activity not_activity player level tag all"          example:"player"                       enums:"high_activity,low_activity,not_activity,player,level,tag,all"          validate:"required"`                                              // 目标用户。可選值：high_activity(高活躍), low_activity(低活躍), not_activity(無活躍), player(指定玩家), level(玩家等級), tag(玩家標籤), all(全部用戶)
	TargetDetail      []string   `json:"target_detail,omitempty"                                                                                                                     example:"['winston123', 'winston888']"                                                                               validate:"omitempty"`                                             // 目标详情：player为账号数组，level/tag为ID字符串数组
	Status            string     `json:"status"                                       binding:"required,oneof=draft scheduled"                                                       example:"scheduled"                    enums:"draft,scheduled"                                                       validate:"required"`                                              // 活动状态。可選值：draft(草稿), scheduled(已排程)
	SendStartTime     *time.Time `json:"send_start_time"                                                                                                                             example:"2024-01-01T10:00:00Z"                                                                                                                                     format:"date-time"` // 发送开始时间（可选）
	SendEndTime       *time.Time `json:"send_end_time"                                                                                                                               example:"2024-01-01T18:00:00Z"                                                                                                                                     format:"date-time"` // 发送结束时间（可选）
	CreatedBy         string     `json:"created_by"                                   binding:"required,max=100"                                                                     example:"admin@example.com"                                                                                          validate:"required,max=100"`                                      // 创建者，最大100个字符
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
	GlobalID          string     `json:"global_id"                    swaggerignore:"true"`                                                                                                                                                                                                                                                                              // 活动全局ID，由路径参数获取
	GlobalMerchantID  string     `json:"global_merchant_id"           swaggerignore:"true"`                                                                                                                                                                                                                                                                              // 商户全局ID，由中间件自动设置
	Category          string     `json:"category"                                          binding:"required,oneof=member bonus others"                                                   example:"member"               enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"`                    // 消息类型：member=会员消息，bonus=红利消息，others=其他消息。可選值：member, bonus, others
	Item              string     `json:"item"                                              binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration"         enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                                              // 消息项目。可選值：registration(註冊), identity_verification(身份驗證), bank_card(銀行卡), others(其他), event(活動), all(全部), mission(任務)
	Title             string     `json:"title"                                             binding:"required,max=255"                                                                     example:"欢迎新用户"                                                                                              validate:"required,max=255"`                                      // 消息标题，最大255个字符
	Content           string     `json:"content"                                           binding:"required"                                                                             example:"恭喜您成功注册成为我们的会员！"                                                                                    validate:"required"`                                              // 消息内容，必填
	AppContent        *string    `json:"app_content,omitempty"                                                                                                                            example:"恭喜您成功注册！"                                                                                           validate:"omitempty,max=255"`                                     // App推播內容，可選，最大255個字符
	NotificationTypes *uint8     `json:"notification_types,omitempty"                      binding:"omitempty,min=1,max=7"                                                                example:"3"                                                                                                  validate:"omitempty,min=1,max=7"`                                 // 推送類型位元遮罩：1=站內信，2=App推播，4=其他，可組合使用，範圍1-7，可選。示例：1(僅站內信), 2(僅App推播), 3(站內信+App推播)
	Target            string     `json:"target"                                            binding:"required,oneof=high_activity low_activity not_activity player level tag all"          example:"all"                  enums:"high_activity,low_activity,not_activity,player,level,tag,all"          validate:"required"`                                              // 目标用户。可選值：high_activity(高活躍), low_activity(低活躍), not_activity(無活躍), player(指定玩家), level(玩家等級), tag(玩家標籤), all(全部用戶)
	TargetDetail      []string   `json:"target_detail,omitempty"                                                                                                                          example:"['123', '456']"                                                                                     validate:"omitempty"`                                             // 目标详情：player为账号数组，level/tag为ID字符串数组
	Status            string     `json:"status"                                            binding:"required,oneof=draft scheduled"                                                       example:"scheduled"            enums:"draft,scheduled"                                                       validate:"required"`                                              // 活动状态。可選值：draft(草稿), scheduled(已排程)
	SendStartTime     *time.Time `json:"send_start_time"                                                                                                                                  example:"2024-01-01T10:00:00Z"                                                                                                                             format:"date-time"` // 发送开始时间（可选）
	SendEndTime       *time.Time `json:"send_end_time"                                                                                                                                    example:"2024-01-01T18:00:00Z"                                                                                                                             format:"date-time"` // 发送结束时间（可选）
	UpdatedBy         string     `json:"updated_by"                                        binding:"required,max=100"                                                                     example:"admin@example.com"                                                                                  validate:"required,max=100"`                                      // 更新者，最大100个字符
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
	Category    string `json:"category"     binding:"required,oneof=member bonus others"                                                   example:"member"       enums:"member,bonus,others"                                                   validate:"required,oneof=member bonus others"` // 消息类型：member=会员消息，bonus=红利消息，others=其他消息。可選值：member, bonus, others
	Item        string `json:"item"         binding:"required,oneof=registration identity_verification bank_card others event all mission" example:"registration" enums:"registration,identity_verification,bank_card,others,event,all,mission" validate:"required"`                           // 消息项目。可選值：registration(註冊), identity_verification(身份驗證), bank_card(銀行卡), others(其他), event(活動), all(全部), mission(任務)
	TriggerType string `json:"trigger_type" binding:"required,oneof=success failure"                                                       example:"success"      enums:"success,failure"`                                                                                                     // 触发类型：success=成功，failure=失败。可選值：success, failure
	Title       string `json:"title"        binding:"required,max=255"                                                                     example:"註冊成功"                                                                                       validate:"required,max=255"`                   // 消息标题，最大255个字符
	Content     string `json:"content"      binding:"required"                                                                             example:"恭喜，註冊成功"                                                                                    validate:"required"`                           // 消息内容，必填
	Active      bool   `json:"active"       binding:"required"                                                                             example:"true"`                                                                                                                                     // 是否啟用自動發送，必填
}

// MerchantAutoSettingsRequest 商戶自動設定請求
// @Description 需要提供全部6種組合：member+registration+success, member+identity_verification+success, member+identity_verification+failure, member+bank_card+failure, bonus+mission+success, bonus+mission+failure
type MerchantAutoSettingsRequest struct {
	GlobalMerchantID string            `json:"global_merchant_id" swaggerignore:"true"`                                                                  // 商户全局ID，由中间件自动设置
	Settings         []AutoSettingItem `json:"settings"                                binding:"required,min=1,max=10" validate:"required,min=1,max=10"` // 自动设置项目列表，必須包含全部6種組合
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

// SendAutoNotificationRequest 發送系統自動推播訊息請求
type SendAutoNotificationRequest struct {
	GlobalPlayerID string `json:"global_player_id" binding:"required"                                                            example:"player-123e4567-e89b-12d3-a456-426614174000" validate:"required"`                                                                                    // 玩家全局ID，必填，格式為UUID
	Category       string `json:"category"         binding:"required,oneof=member bonus"                                         example:"member"                                      validate:"required,oneof=member bonus"    enums:"member,bonus"`                                         // 訊息類別：member=會員訊息，bonus=紅利訊息。可選值：member, bonus
	Item           string `json:"item"             binding:"required,oneof=registration identity_verification bank_card mission" example:"registration"                                validate:"required"                       enums:"registration,identity_verification,bank_card,mission"` // 訊息項目：registration=註冊，identity_verification=身分驗證，bank_card=銀行卡，mission=任務。可選值：registration, identity_verification, bank_card, mission
	TriggerType    string `json:"trigger_type"     binding:"required,oneof=success failure"                                      example:"success"                                     validate:"required,oneof=success failure" enums:"success,failure"`                                      // 觸發類型：success=成功，failure=失敗。可選值：success, failure
}

// SendAutoNotificationResponse 發送系統自動推播訊息回應
type SendAutoNotificationResponse struct {
	PlayerID     string   `json:"player_id"`               // 玩家ID
	Category     string   `json:"category"`                // 訊息類別
	Item         string   `json:"item"`                    // 訊息項目
	TriggerType  string   `json:"trigger_type"`            // 觸發類型
	Status       string   `json:"status"`                  // 發送狀態：sent=已發送，not_found=找不到對應訊息設定，failed=發送失敗
	SentChannels []string `json:"sent_channels,omitempty"` // 已發送的渠道：["in_app", "push"]
	Message      string   `json:"message,omitempty"`       // 狀態訊息
}
