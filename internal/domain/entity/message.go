package entity

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/utils"
)

// MessageCampaign 會員訊息活動模型
type MessageCampaign struct {
	ID                uint64     `json:"id"`
	Category          string     `json:"category"`     // member, bonus, others
	Item              string     `json:"item"`         // registration, identity_verification, bank_card, others, event, all, mission
	TriggerType       string     `json:"trigger_type"` // success, failure (for auto-send settings)
	Title             string     `json:"title"`
	MerchantID        uint64     `json:"merchant_id"`
	GlobalID          string     `json:"global_id"`
	LegacyID          *uint      `json:"legacy_id,omitempty"`     // 舊系統的 notification ID，用於資料遷移
	Content           string     `json:"content"`                 // 可包含 HTML Tag
	AppContent        *string    `json:"app_content,omitempty"`   // App推播內容
	NotificationTypes uint8      `json:"notification_types"`      // 推送類型位元遮罩: 1=站內信, 2=App推播, 4=其他
	Target            string     `json:"target"`                  // high_activity, low_activity, not_activity, player, level, tag, all
	TargetDetail      *string    `json:"target_detail,omitempty"` // JSON string containing target-specific details (player accounts, level names, tag names)
	Status            string     `json:"status"`                  // draft, scheduled, sent, cancelled, failed
	AutoSend          bool       `json:"auto_send"`               // 是否為系統自動訊息
	Active            bool       `json:"active"`                  // 是否啟用自動發送（僅適用於AutoSend=true的訊息）
	RealSentCount     int64      `json:"real_sent_count"`         // 實際成功發送人數
	SendStartTime     *time.Time `json:"send_start_time,omitempty"`
	SendEndTime       *time.Time `json:"send_end_time,omitempty"`
	CreatedBy         string     `json:"created_by"`           // 建立者帳號或名稱
	UpdatedBy         *string    `json:"updated_by,omitempty"` // 最後更新者帳號或名稱
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

// ==== 通知類型管理業務方法 ====

// ValidateNotificationTypes 驗證通知類型
func (m *MessageCampaign) ValidateNotificationTypes(isCreate bool) error {
	validator := utils.NewNotificationValidator()
	if isCreate {
		return validator.IsValidForCreate(&m.NotificationTypes)
	}
	return validator.IsValidForUpdate(&m.NotificationTypes)
}

// HasInAppNotification 檢查是否包含站內信通知
func (m *MessageCampaign) HasInAppNotification() bool {
	notificationType := consts.NotificationType(m.NotificationTypes)
	return notificationType.HasInApp()
}

// HasAppPushNotification 檢查是否包含App推播通知
func (m *MessageCampaign) HasAppPushNotification() bool {
	notificationType := consts.NotificationType(m.NotificationTypes)
	return notificationType.HasAppPush()
}

// ==== 目標類型處理業務方法 ====

// SetPlayerTargetDetail 設置玩家目標詳情
func (m *MessageCampaign) SetPlayerTargetDetail(playerAccounts []string) error {
	m.Target = consts.TargetPlayer
	targetDetailBytes, err := json.Marshal(playerAccounts)
	if err != nil {
		return fmt.Errorf("marshal player target detail: %w", err)
	}
	targetDetailStr := string(targetDetailBytes)
	m.TargetDetail = &targetDetailStr
	return nil
}

// SetLevelTargetDetail 設置等級目標詳情
func (m *MessageCampaign) SetLevelTargetDetail(levelIDs []string) error {
	m.Target = consts.TargetLevel
	// 驗證ID格式
	if err := m.validateStringIDs(levelIDs); err != nil {
		return fmt.Errorf("invalid level IDs: %w", err)
	}
	targetDetailBytes, err := json.Marshal(levelIDs)
	if err != nil {
		return fmt.Errorf("marshal level target detail: %w", err)
	}
	targetDetailStr := string(targetDetailBytes)
	m.TargetDetail = &targetDetailStr
	return nil
}

// SetTagTargetDetail 設置標籤目標詳情
func (m *MessageCampaign) SetTagTargetDetail(tagIDs []string) error {
	m.Target = consts.TargetTag
	// 驗證ID格式
	if err := m.validateStringIDs(tagIDs); err != nil {
		return fmt.Errorf("invalid tag IDs: %w", err)
	}
	targetDetailBytes, err := json.Marshal(tagIDs)
	if err != nil {
		return fmt.Errorf("marshal tag target detail: %w", err)
	}
	targetDetailStr := string(targetDetailBytes)
	m.TargetDetail = &targetDetailStr
	return nil
}

// ==== 活動狀態管理業務方法 ====

// IsScheduled 檢查是否為已排程狀態
func (m *MessageCampaign) IsScheduled() bool {
	return m.Status == consts.MessageCampaignStatusScheduled
}

// IsSent 檢查是否為已發送狀態
func (m *MessageCampaign) IsSent() bool {
	return m.Status == consts.MessageCampaignStatusSent
}

// IsDraft 檢查是否為草稿狀態
func (m *MessageCampaign) IsDraft() bool {
	return m.Status == consts.MessageCampaignStatusDraft
}

// MarkAsScheduled 標記為已排程
func (m *MessageCampaign) MarkAsScheduled() {
	m.Status = consts.MessageCampaignStatusScheduled
	m.UpdatedAt = time.Now()
}

// MarkAsSent 標記為已發送
func (m *MessageCampaign) MarkAsSent(sentCount int64) {
	m.Status = consts.MessageCampaignStatusSent
	m.RealSentCount = sentCount
	now := time.Now()
	m.SendEndTime = &now
	m.UpdatedAt = now
}

// MarkAsCancelled 標記為已取消
func (m *MessageCampaign) MarkAsCancelled() {
	m.Status = consts.MessageCampaignStatusCancelled
	m.UpdatedAt = time.Now()
}

// ==== 內容處理業務方法 ====

// ==== 輔助方法 ====

// validateStringIDs 驗證字符串ID格式
func (m *MessageCampaign) validateStringIDs(stringIDs []string) error {
	if len(stringIDs) == 0 {
		return nil
	}
	for _, strID := range stringIDs {
		if _, err := strconv.ParseUint(strID, 10, 64); err != nil {
			return fmt.Errorf("invalid ID format '%s': %w", strID, err)
		}
	}
	return nil
}

// convertStringIDsToUint64 將字符串ID數組轉換為uint64數組
func (m *MessageCampaign) convertStringIDsToUint64(stringIDs []string) ([]uint64, error) {
	if len(stringIDs) == 0 {
		return nil, nil
	}
	ids := make([]uint64, len(stringIDs))
	for i, strID := range stringIDs {
		id, err := strconv.ParseUint(strID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format '%s': %w", strID, err)
		}
		ids[i] = id
	}
	return ids, nil
}

// PushKey 商戶推播API金鑰模型
type PushKey struct {
	ID               uint64    `json:"id"`
	GlobalMerchantID string    `json:"global_merchant_id"`
	MerchantID       uint64    `json:"merchant_id"`
	Key              string    `json:"key"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PlayerMessage 會員訊息模型
type PlayerMessage struct {
	ID             uint64    `json:"id"`
	GlobalPlayerID string    `json:"global_player_id"`
	PlayerID       uint64    `json:"player_id"`
	CampaignID     uint64    `json:"campaign_id"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
