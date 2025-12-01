package entity

import (
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageCampaign_ValidateNotificationTypes(t *testing.T) {
	tests := []struct {
		name         string
		campaign     *MessageCampaign
		isCreate     bool
		expectError  bool
		errorMessage string
	}{
		{
			name: "Valid notification types for create",
			campaign: &MessageCampaign{
				NotificationTypes: uint8(consts.NotificationTypeInAppOnly),
			},
			isCreate:    true,
			expectError: false,
		},
		{
			name: "Invalid notification types for create - zero",
			campaign: &MessageCampaign{
				NotificationTypes: 0,
			},
			isCreate:    true,
			expectError: true,
		},
		{
			name: "Valid notification types for update",
			campaign: &MessageCampaign{
				NotificationTypes: uint8(consts.NotificationTypeAppPushOnly),
			},
			isCreate:    false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.campaign.ValidateNotificationTypes(tt.isCreate)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessageCampaign_HasNotifications(t *testing.T) {
	tests := []struct {
		name              string
		notificationTypes uint8
		expectInApp       bool
		expectAppPush     bool
	}{
		{
			name:              "In-app only",
			notificationTypes: uint8(consts.NotificationTypeInAppOnly),
			expectInApp:       true,
			expectAppPush:     false,
		},
		{
			name:              "App push only",
			notificationTypes: uint8(consts.NotificationTypeAppPushOnly),
			expectInApp:       false,
			expectAppPush:     true,
		},
		{
			name:              "Both types",
			notificationTypes: uint8(consts.NotificationTypeInAppAndAppPush),
			expectInApp:       true,
			expectAppPush:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign := &MessageCampaign{
				NotificationTypes: tt.notificationTypes,
			}

			assert.Equal(t, tt.expectInApp, campaign.HasInAppNotification())
			assert.Equal(t, tt.expectAppPush, campaign.HasAppPushNotification())
		})
	}
}

func TestMessageCampaign_SetPlayerTargetDetail(t *testing.T) {
	campaign := &MessageCampaign{}
	accounts := []string{"player1", "player2", "player3"}

	err := campaign.SetPlayerTargetDetail(accounts)
	require.NoError(t, err)

	assert.Equal(t, consts.TargetPlayer, campaign.Target)
	assert.NotNil(t, campaign.TargetDetail)
}

func TestMessageCampaign_SetLevelTargetDetail(t *testing.T) {
	campaign := &MessageCampaign{}
	levelIDs := []string{"1", "2", "3"}

	err := campaign.SetLevelTargetDetail(levelIDs)
	require.NoError(t, err)

	assert.Equal(t, consts.TargetLevel, campaign.Target)
	assert.NotNil(t, campaign.TargetDetail)
}

func TestMessageCampaign_SetLevelTargetDetail_InvalidID(t *testing.T) {
	campaign := &MessageCampaign{}
	levelIDs := []string{"1", "invalid", "3"}

	err := campaign.SetLevelTargetDetail(levelIDs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid level IDs")
}

func TestMessageCampaign_StatusMethods(t *testing.T) {
	campaign := &MessageCampaign{}

	// 測試草稿狀態
	campaign.Status = consts.MessageCampaignStatusDraft
	assert.True(t, campaign.IsDraft())
	assert.False(t, campaign.IsScheduled())
	assert.False(t, campaign.IsSent())

	// 測試已排程狀態
	campaign.Status = consts.MessageCampaignStatusScheduled
	assert.False(t, campaign.IsDraft())
	assert.True(t, campaign.IsScheduled())
	assert.False(t, campaign.IsSent())

	// 測試已發送狀態
	campaign.Status = consts.MessageCampaignStatusSent
	assert.False(t, campaign.IsDraft())
	assert.False(t, campaign.IsScheduled())
	assert.True(t, campaign.IsSent())
}

func TestMessageCampaign_MarkMethods(t *testing.T) {
	campaign := &MessageCampaign{
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldTime := campaign.UpdatedAt

	// 測試標記為已排程
	campaign.MarkAsScheduled()
	assert.Equal(t, consts.MessageCampaignStatusScheduled, campaign.Status)
	assert.True(t, campaign.UpdatedAt.After(oldTime))

	// 測試標記為已發送
	campaign.MarkAsSent(100)
	assert.Equal(t, consts.MessageCampaignStatusSent, campaign.Status)
	assert.Equal(t, int64(100), campaign.RealSentCount)
	assert.NotNil(t, campaign.SendEndTime)

}
