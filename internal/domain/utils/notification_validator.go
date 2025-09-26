package utils

import (
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
)

// NotificationValidator 通知類型驗證器
type NotificationValidator struct{}

// NewNotificationValidator 創建通知驗證器
func NewNotificationValidator() *NotificationValidator {
	return &NotificationValidator{}
}

// ValidateNotificationType 驗證通知類型
func (v *NotificationValidator) ValidateNotificationType(value *uint8) error {
	// 檢查是否為 nil
	if value == nil {
		return errmsg.ErrNotificationTypeRequired
	}

	// 轉換為 NotificationType 並驗證
	nt, err := consts.NewNotificationType(*value)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 額外的業務邏輯驗證
	if err := v.validateBusinessRules(nt); err != nil {
		return err
	}

	return nil
}

// ValidateNotificationTypeValue 直接驗證 uint8 值
func (v *NotificationValidator) ValidateNotificationTypeValue(value uint8) error {
	return v.ValidateNotificationType(&value)
}

// validateBusinessRules 業務規則驗證
func (v *NotificationValidator) validateBusinessRules(nt consts.NotificationType) error {
	// 規則1: 不能為空（至少要有一種推送類型）
	if nt == 0 {
		return fmt.Errorf(
			"notification type cannot be empty: %w",
			errmsg.ErrInvalidNotificationType,
		)
	}

	// 規則2: 檢查不支援的組合（當前都支援，預留擴展）
	// 例如：未來可能某些組合不被支援

	return nil
}

// GetValidNotificationTypes 獲取所有有效的通知類型
func (v *NotificationValidator) GetValidNotificationTypes() []consts.NotificationType {
	return []consts.NotificationType{
		consts.NotificationTypeInAppOnly,       // 1 = 僅站內信
		consts.NotificationTypeAppPushOnly,     // 2 = 僅App推播
		consts.NotificationTypeInAppAndAppPush, // 3 = 站內信+App推播
		// 未來可以加入更多組合
	}
}

// GetNotificationTypeDescription 獲取通知類型描述
func (v *NotificationValidator) GetNotificationTypeDescription(value uint8) string {
	nt, err := consts.NewNotificationType(value)
	if err != nil {
		return fmt.Sprintf("invalid (%d)", value)
	}

	return fmt.Sprintf("%s (%d)", nt.String(), value)
}

// IsValidForCreate 驗證創建時的通知類型
func (v *NotificationValidator) IsValidForCreate(value *uint8) error {
	if err := v.ValidateNotificationType(value); err != nil {
		return fmt.Errorf("invalid notification type for create: %w", err)
	}

	// 創建時的額外驗證
	nt, _ := consts.NewNotificationType(*value)

	// 確保至少有一種推送方式
	if !nt.HasInApp() && !nt.HasAppPush() {
		return fmt.Errorf(
			"must specify at least one notification method: %w",
			errmsg.ErrInvalidNotificationType,
		)
	}

	return nil
}

// IsValidForUpdate 驗證更新時的通知類型
func (v *NotificationValidator) IsValidForUpdate(value *uint8) error {
	if value == nil {
		// 更新時 notification_types 可以為 nil (不更新)
		return nil
	}

	return v.ValidateNotificationType(value)
}

// SanitizeNotificationType 清理並修正通知類型值
func (v *NotificationValidator) SanitizeNotificationType(value uint8) (uint8, error) {
	// 移除不支援的位元
	sanitized := value & uint8(consts.NotificationTypeMax)

	// 確保至少有一個有效位元
	if sanitized == 0 {
		return 0, fmt.Errorf(
			"no valid notification bits found in value %d: %w",
			value,
			errmsg.ErrInvalidNotificationType,
		)
	}

	// 驗證清理後的值
	if err := v.ValidateNotificationTypeValue(sanitized); err != nil {
		return 0, fmt.Errorf("sanitized value is still invalid: %w", err)
	}

	return sanitized, nil
}
