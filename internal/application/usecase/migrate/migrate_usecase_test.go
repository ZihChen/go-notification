package migrate

import (
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMigrateUseCase_MapToCampaign(t *testing.T) {
	// 準備測試數據
	mockLogger := helper.NewMockLogger()
	mockLegacyDB := &gorm.DB{} // 簡化的 mock
	mockMessageRepo := mocks.NewMessageCampaignRepositoryMock(t)
	mockPlayerRepo := mocks.NewPlayerRepositoryMock(t)
	mockMerchantRepo := mocks.NewMerchantRepositoryMock(t)
	mockPlayerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)

	// 創建 use case
	uc := NewMigrateUseCase(
		mockLegacyDB,
		mockMessageRepo,
		mockPlayerRepo,
		mockMerchantRepo,
		mockPlayerMessageRepo,
		mockLogger,
	).(*migrateUseCase)

	// 測試資料
	now := time.Now()
	legacy := LegacyNotification{
		ID:        1,
		Category:  1,
		Item:      2,
		Title:     "測試標題",
		Content:   "測試內容",
		Focus:     8,
		AutoSend:  true,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "test_user",
		DeletedAt: nil,
	}

	merchant := &entity.Merchant{
		ID:   1,
		Name: "test_merchant",
	}

	// 執行測試
	result := uc.mapToCampaign(legacy, merchant)

	// 驗證結果
	assert.Equal(t, "member", result.Category)
	assert.Equal(t, "identity_verification", result.Item)
	assert.Equal(t, "success", result.TriggerType)
	assert.Equal(t, uint64(1), result.MerchantID)
	assert.Equal(t, "測試標題", result.Title)
	assert.Equal(t, "測試內容", result.Content)
	assert.Equal(t, "sent", result.Status)
	assert.Equal(t, "player", result.Target)
	assert.True(t, result.AutoSend)
	assert.Equal(t, "test_user", result.CreatedBy)
	assert.Equal(t, now, result.CreatedAt)
	// UpdatedAt 現在使用當前時間，所以檢查是否在合理的時間範圍內
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 1*time.Second)
}

func TestMigrateUseCase_MapCategory(t *testing.T) {
	uc := &migrateUseCase{}

	tests := []struct {
		input    int
		expected string
	}{
		{1, "member"},
		{2, "bonus"},
		{3, "others"},
		{999, "others"}, // 未知值應該映射為 others
	}

	for _, test := range tests {
		result := uc.mapCategory(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestMigrateUseCase_MapItem(t *testing.T) {
	uc := &migrateUseCase{}

	tests := []struct {
		input    int
		expected string
	}{
		{1, "registration"},
		{2, "identity_verification"},
		{3, "bank_card"},
		{4, "others"},
		{5, "event"},
		{6, "all"},
		{7, "mission"},
		{999, "others"}, // 未知值應該映射為 others
	}

	for _, test := range tests {
		result := uc.mapItem(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestMigrateUseCase_MapStatus(t *testing.T) {
	uc := &migrateUseCase{}

	now := time.Now()

	// 測試 nil (未刪除)
	result := uc.mapStatus(nil)
	assert.Equal(t, "sent", result)

	// 測試有刪除時間
	result = uc.mapStatus(&now)
	assert.Equal(t, "cancelled", result)
}

func TestMigrateUseCase_MapTarget(t *testing.T) {
	uc := &migrateUseCase{}

	tests := []struct {
		input    int
		expected string
	}{
		{1, "high_activity"},
		{2, "low_activity"},
		{3, "not_activity"},
		{8, "player"},
		{9, "level"},
		{10, "tag"},
		{11, "all"},
		{999, "all"}, // 未知值應該映射為 all
	}

	for _, test := range tests {
		result := uc.mapTarget(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestMigrateUseCase_GenerateGlobalID(t *testing.T) {
	uc := &migrateUseCase{}

	// 測試生成的 UUID 格式正確性
	id1 := uc.generateGlobalID()
	id2 := uc.generateGlobalID()

	// 每次生成的 UUID 應該是不同的（隨機性）
	assert.NotEqual(t, id1, id2, "每次應該產生不同的 UUID")
	assert.NotEmpty(t, id1, "UUID 不應該為空")
	assert.NotEmpty(t, id2, "UUID 不應該為空")

	// 驗證 UUID 格式 (36個字符，包含4個連字符)
	assert.Len(t, id1, 36, "UUID 長度應該是36個字符")
	assert.Len(t, id2, 36, "UUID 長度應該是36個字符")
}

func TestMigrationStats(t *testing.T) {
	stats := &inbound.MigrationStats{}

	// 測試初始狀態
	assert.Equal(t, 0, stats.ProcessedCount)
	assert.Equal(t, 0, stats.SuccessCount)
	assert.Equal(t, 0, stats.SkippedCount)
	assert.Equal(t, 0, stats.ErrorCount)

	// 測試添加成功
	stats.AddSuccess()
	assert.Equal(t, 1, stats.ProcessedCount)
	assert.Equal(t, 1, stats.SuccessCount)

	// 測試添加跳過
	stats.AddSkipped()
	assert.Equal(t, 2, stats.ProcessedCount)
	assert.Equal(t, 1, stats.SkippedCount)

	// 測試添加錯誤
	err := assert.AnError
	stats.AddError(err)
	assert.Equal(t, 3, stats.ProcessedCount)
	assert.Equal(t, 1, stats.ErrorCount)
	assert.Len(t, stats.Errors, 1)

	// 測試字串表示
	result := stats.String()
	expected := "處理: 3, 成功: 1, 跳過: 1, 錯誤: 1"
	assert.Equal(t, expected, result)
}
