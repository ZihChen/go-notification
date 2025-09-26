package entity

import (
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/stretchr/testify/assert"
)

func TestPlayer_DetermineFocusType(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		lastActiveAt  *time.Time
		expectedFocus string
	}{
		{
			name:          "No activity record",
			lastActiveAt:  nil,
			expectedFocus: consts.TargetNotActivity,
		},
		{
			name:          "High activity - 15 days ago",
			lastActiveAt:  &[]time.Time{now.AddDate(0, 0, -15)}[0],
			expectedFocus: consts.TargetHighActivity,
		},
		{
			name:          "High activity - 30 days ago (boundary)",
			lastActiveAt:  &[]time.Time{now.AddDate(0, 0, -30)}[0],
			expectedFocus: consts.TargetHighActivity,
		},
		{
			name:          "Low activity - 50 days ago",
			lastActiveAt:  &[]time.Time{now.AddDate(0, 0, -50)}[0],
			expectedFocus: consts.TargetLowActivity,
		},
		{
			name:          "Low activity - 100 days ago (boundary)",
			lastActiveAt:  &[]time.Time{now.AddDate(0, 0, -100)}[0],
			expectedFocus: consts.TargetLowActivity,
		},
		{
			name:          "Not activity - 150 days ago",
			lastActiveAt:  &[]time.Time{now.AddDate(0, 0, -150)}[0],
			expectedFocus: consts.TargetNotActivity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{
				LastActiveAt: tt.lastActiveAt,
			}

			focus := player.DetermineFocusType()
			assert.Equal(t, tt.expectedFocus, focus)
		})
	}
}

func TestPlayer_UpdateLastActive(t *testing.T) {
	player := &Player{
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldUpdateTime := player.UpdatedAt

	player.UpdateLastActive()

	assert.NotNil(t, player.LastActiveAt)
	assert.True(t, player.LastActiveAt.After(oldUpdateTime))
	assert.True(t, player.UpdatedAt.After(oldUpdateTime))
}

func TestPlayer_RegenerateAPIKey(t *testing.T) {
	player := &Player{
		APIKey:    "old-key",
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldAPIKey := player.APIKey
	oldUpdateTime := player.UpdatedAt

	player.RegenerateAPIKey()

	assert.NotEqual(t, oldAPIKey, player.APIKey)
	assert.NotEmpty(t, player.APIKey)
	assert.True(t, player.UpdatedAt.After(oldUpdateTime))
}

func TestPlayer_ValidateAPIKey(t *testing.T) {
	player := &Player{
		APIKey: "test-api-key",
	}

	assert.True(t, player.ValidateAPIKey("test-api-key"))
	assert.False(t, player.ValidateAPIKey("wrong-key"))
}
