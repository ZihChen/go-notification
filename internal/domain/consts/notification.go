package consts

import "github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"

// NotificationType 推送類型位元遮罩常數
type NotificationType uint8

const (
	// 基礎推送類型位元定義
	NotificationTypeInApp     NotificationType = 1 << 0 // 1 = 站內信
	NotificationTypeAppPush   NotificationType = 1 << 1 // 2 = App推播
	NotificationTypeReserved1 NotificationType = 1 << 2 // 4 = 預留擴展
	NotificationTypeReserved2 NotificationType = 1 << 3 // 8 = 預留擴展
	NotificationTypeReserved3 NotificationType = 1 << 4 // 16 = 預留擴展
)

// 常用組合推送類型
const (
	NotificationTypeInAppOnly       NotificationType = NotificationTypeInApp                             // 1 = 僅站內信
	NotificationTypeAppPushOnly     NotificationType = NotificationTypeAppPush                           // 2 = 僅App推播
	NotificationTypeInAppAndAppPush NotificationType = NotificationTypeInApp | NotificationTypeAppPush   // 3 = 站內信+App推播
	NotificationTypeAll             NotificationType = NotificationTypeInApp | NotificationTypeAppPush | // 7 = 所有類型（當前支援）
		NotificationTypeReserved1
)

// 邊界值定義
const (
	NotificationTypeMin    NotificationType = 1  // 最小有效值
	NotificationTypeMax    NotificationType = 7  // 最大有效值（當前支援範圍）
	NotificationTypeFuture NotificationType = 31 // 未來擴展的最大值（5位元）
)

// 驗證函數
func (nt NotificationType) IsValid() bool {
	// 檢查是否在有效範圍內
	if nt < NotificationTypeMin || nt > NotificationTypeMax {
		return false
	}

	// 檢查是否有未定義的位元被設置
	validBits := NotificationTypeInApp | NotificationTypeAppPush | NotificationTypeReserved1
	return (nt & ^validBits) == 0
}

// 檢查是否包含站內信
func (nt NotificationType) HasInApp() bool {
	return (nt & NotificationTypeInApp) != 0
}

// 檢查是否包含App推播
func (nt NotificationType) HasAppPush() bool {
	return (nt & NotificationTypeAppPush) != 0
}

// 檢查是否包含特定類型
func (nt NotificationType) Has(checkType NotificationType) bool {
	return (nt & checkType) != 0
}

// 轉換為字符串描述
func (nt NotificationType) String() string {
	if !nt.IsValid() {
		return "invalid"
	}

	var parts []string
	if nt.HasInApp() {
		parts = append(parts, "in_app")
	}
	if nt.HasAppPush() {
		parts = append(parts, "app_push")
	}
	if nt.Has(NotificationTypeReserved1) {
		parts = append(parts, "reserved_1")
	}

	if len(parts) == 0 {
		return "none"
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "|"
		}
		result += part
	}
	return result
}

// 從 uint8 安全轉換
func NewNotificationType(value uint8) (NotificationType, error) {
	nt := NotificationType(value)
	if !nt.IsValid() {
		return 0, errmsg.ErrInvalidNotificationType
	}
	return nt, nil
}

// 驗證並轉換為 uint8
func (nt NotificationType) ToUint8() (uint8, error) {
	if !nt.IsValid() {
		return 0, errmsg.ErrInvalidNotificationType
	}
	return uint8(nt), nil
}
