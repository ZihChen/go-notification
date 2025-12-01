package factories

import (
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// TestDataFactory 統一的測試資料工廠
type TestDataFactory struct {
	sequence uint64
}

// NewTestDataFactory 創建新的測試資料工廠
func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{sequence: 1}
}

// nextID 獲取下一個唯一ID
func (f *TestDataFactory) nextID() uint64 {
	id := f.sequence
	f.sequence++
	return id
}

// MessageCampaignBuilder 消息活動建構器
type MessageCampaignBuilder struct {
	campaign *entity.MessageCampaign
}

// CreateMessageCampaign 創建消息活動建構器
func (f *TestDataFactory) CreateMessageCampaign() *MessageCampaignBuilder {
	id := f.nextID()
	now := time.Now()

	campaign := &entity.MessageCampaign{
		ID:                id,
		GlobalID:          fmt.Sprintf("campaign_%d", id),
		MerchantID:        1,
		Title:             fmt.Sprintf("Test Campaign %d", id),
		Content:           fmt.Sprintf("Test content for campaign %d", id),
		Target:            consts.TargetAll,
		Category:          consts.CategoryMember,
		Item:              consts.ItemOthers,
		Status:            consts.MessageCampaignStatusScheduled,
		NotificationTypes: 1, // 預設只有站內信
		SendStartTime:     &now,
		RealSentCount:     0,
		AutoSend:          false,
		Active:            true, // 預設為啟用狀態
		TriggerType:       "manual",
		CreatedBy:         "test",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	return &MessageCampaignBuilder{campaign: campaign}
}

func (b *MessageCampaignBuilder) WithID(id uint64) *MessageCampaignBuilder {
	b.campaign.ID = id
	return b
}

func (b *MessageCampaignBuilder) WithGlobalID(globalID string) *MessageCampaignBuilder {
	b.campaign.GlobalID = globalID
	return b
}

func (b *MessageCampaignBuilder) WithMerchantID(merchantID uint64) *MessageCampaignBuilder {
	b.campaign.MerchantID = merchantID
	return b
}

func (b *MessageCampaignBuilder) WithTitle(title string) *MessageCampaignBuilder {
	b.campaign.Title = title
	return b
}

func (b *MessageCampaignBuilder) WithContent(content string) *MessageCampaignBuilder {
	b.campaign.Content = content
	return b
}

func (b *MessageCampaignBuilder) WithTarget(target string) *MessageCampaignBuilder {
	b.campaign.Target = target
	return b
}

func (b *MessageCampaignBuilder) WithStatus(status string) *MessageCampaignBuilder {
	b.campaign.Status = status
	return b
}

func (b *MessageCampaignBuilder) WithSentCount(count int64) *MessageCampaignBuilder {
	b.campaign.RealSentCount = count
	return b
}

func (b *MessageCampaignBuilder) WithActive(active bool) *MessageCampaignBuilder {
	b.campaign.Active = active
	return b
}

func (b *MessageCampaignBuilder) WithSendStartTime(
	sendStartTime *time.Time,
) *MessageCampaignBuilder {
	b.campaign.SendStartTime = sendStartTime
	return b
}

func (b *MessageCampaignBuilder) Build() *entity.MessageCampaign {
	// 創建副本避免修改原始數據
	campaign := *b.campaign
	return &campaign
}

// PlayerBuilder 玩家建構器
type PlayerBuilder struct {
	player *entity.Player
}

// CreatePlayer 創建玩家建構器
func (f *TestDataFactory) CreatePlayer() *PlayerBuilder {
	id := f.nextID()
	now := time.Now()

	player := &entity.Player{
		ID:             id,
		GlobalPlayerID: fmt.Sprintf("player_%d", id),
		MerchantID:     1,
		LevelID:        1,
		Account:        fmt.Sprintf("test_account_%d", id),
		APIKey:         fmt.Sprintf("api_key_%d", id),
		LastActiveAt:   &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return &PlayerBuilder{player: player}
}

func (b *PlayerBuilder) WithID(id uint64) *PlayerBuilder {
	b.player.ID = id
	return b
}

func (b *PlayerBuilder) WithGlobalID(globalID string) *PlayerBuilder {
	b.player.GlobalPlayerID = globalID
	return b
}

func (b *PlayerBuilder) WithMerchantID(merchantID uint64) *PlayerBuilder {
	b.player.MerchantID = merchantID
	return b
}

func (b *PlayerBuilder) WithLevelID(levelID uint64) *PlayerBuilder {
	b.player.LevelID = levelID
	return b
}

func (b *PlayerBuilder) WithLastActivity(lastActivity *time.Time) *PlayerBuilder {
	b.player.LastActiveAt = lastActivity
	return b
}

func (b *PlayerBuilder) Build() *entity.Player {
	player := *b.player
	return &player
}

// PlayerMessageBuilder 玩家消息建構器
type PlayerMessageBuilder struct {
	message *entity.PlayerMessage
}

// CreatePlayerMessage 創建玩家消息建構器
func (f *TestDataFactory) CreatePlayerMessage() *PlayerMessageBuilder {
	id := f.nextID()
	now := time.Now()

	message := &entity.PlayerMessage{
		ID:             id,
		GlobalPlayerID: fmt.Sprintf("player_%d", id),
		PlayerID:       id,
		CampaignID:     1,
		IsRead:         false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return &PlayerMessageBuilder{message: message}
}

func (b *PlayerMessageBuilder) WithID(id uint64) *PlayerMessageBuilder {
	b.message.ID = id
	return b
}

func (b *PlayerMessageBuilder) WithGlobalPlayerID(globalPlayerID string) *PlayerMessageBuilder {
	b.message.GlobalPlayerID = globalPlayerID
	return b
}

func (b *PlayerMessageBuilder) WithPlayerID(playerID uint64) *PlayerMessageBuilder {
	b.message.PlayerID = playerID
	return b
}

func (b *PlayerMessageBuilder) WithCampaignID(campaignID uint64) *PlayerMessageBuilder {
	b.message.CampaignID = campaignID
	return b
}

func (b *PlayerMessageBuilder) WithIsRead(isRead bool) *PlayerMessageBuilder {
	b.message.IsRead = isRead
	return b
}

func (b *PlayerMessageBuilder) Build() *entity.PlayerMessage {
	message := *b.message
	return &message
}

// 批次創建方法
// CreateMultiplePlayers 批次創建玩家
func (f *TestDataFactory) CreateMultiplePlayers(count int) []*entity.Player {
	players := make([]*entity.Player, count)
	for i := 0; i < count; i++ {
		players[i] = f.CreatePlayer().Build()
	}
	return players
}

// CreateMultiplePlayerMessages 批次創建玩家消息
func (f *TestDataFactory) CreateMultiplePlayerMessages(
	count int,
	campaignID uint64,
) []*entity.PlayerMessage {
	messages := make([]*entity.PlayerMessage, count)
	for i := 0; i < count; i++ {
		messages[i] = f.CreatePlayerMessage().
			WithCampaignID(campaignID).
			Build()
	}
	return messages
}

// 預設場景工廠方法
// CreateScheduledCampaign 創建已排程的活動
func (f *TestDataFactory) CreateScheduledCampaign() *entity.MessageCampaign {
	now := time.Now()
	return f.CreateMessageCampaign().
		WithStatus(consts.MessageCampaignStatusScheduled).
		WithSendStartTime(&now).
		Build()
}

// CreateFailedCampaign 創建失敗的活動
func (f *TestDataFactory) CreateFailedCampaign() *entity.MessageCampaign {
	return f.CreateMessageCampaign().
		WithStatus(consts.MessageCampaignStatusFailed).
		WithSentCount(1500). // 部分發送
		Build()
}

// CreateActivePlayer 創建活躍玩家
func (f *TestDataFactory) CreateActivePlayer() *entity.Player {
	recentTime := time.Now().Add(-10 * 24 * time.Hour) // 10天前活躍
	return f.CreatePlayer().
		WithLastActivity(&recentTime).
		Build()
}

// CreateInactivePlayer 創建不活躍玩家
func (f *TestDataFactory) CreateInactivePlayer() *entity.Player {
	oldTime := time.Now().Add(-120 * 24 * time.Hour) // 120天前活躍
	return f.CreatePlayer().
		WithLastActivity(&oldTime).
		Build()
}

// 錯誤場景數據
// CreateEmptyPlayerBatch 創建空玩家批次
func (f *TestDataFactory) CreateEmptyPlayerBatch() []*entity.Player {
	return []*entity.Player{}
}

// CreateLargePlayerBatch 創建大批次玩家（測試性能）
func (f *TestDataFactory) CreateLargePlayerBatch(size int) []*entity.Player {
	return f.CreateMultiplePlayers(size)
}

// CreateMerchant 創建商戶
func (f *TestDataFactory) CreateMerchant() *entity.Merchant {
	id := f.nextID()
	return &entity.Merchant{
		ID:               id,
		GlobalMerchantID: fmt.Sprintf("merchant_%d", id),
		Name:             fmt.Sprintf("Test Merchant %d", id),
		DisplayName:      fmt.Sprintf("Test Display Name %d", id),
		APIKey:           fmt.Sprintf("test-api-key-%d", id),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// CreatePushKey 創建推播金鑰
func (f *TestDataFactory) CreatePushKey() *entity.PushKey {
	id := f.nextID()
	return &entity.PushKey{
		ID:               id,
		GlobalMerchantID: fmt.Sprintf("merchant_%d", id),
		MerchantID:       id,
		Key:              fmt.Sprintf("test-push-api-key-%d", id),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}
