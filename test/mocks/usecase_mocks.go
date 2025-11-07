package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/stretchr/testify/mock"
)

// MessageUseCaseMock 統一的 MessageUseCase Mock
type MessageUseCaseMock struct {
	*BaseMock
}

var _ inbound.MessageUseCase = (*MessageUseCaseMock)(nil)

// NewMessageUseCaseMock 創建新的 MessageUseCase Mock
func NewMessageUseCaseMock(t *testing.T) *MessageUseCaseMock {
	return &MessageUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MessageUseCaseMock) CreateMessageCampaign(
	ctx context.Context,
	campaign *dto.CreateMessageCampaignRequest,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *MessageUseCaseMock) UpdateMessageCampaign(
	ctx context.Context,
	campaign *dto.UpdateMessageCampaignRequest,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *MessageUseCaseMock) DeleteMessageCampaign(ctx context.Context, globalID string) error {
	args := m.Called(ctx, globalID)
	return args.Error(0)
}

func (m *MessageUseCaseMock) GetMessageCampaign(
	ctx context.Context,
	globalID string,
) (*dto.MessageCampaignResponse, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MessageCampaignResponse), args.Error(1)
}

func (m *MessageUseCaseMock) ListMessageCampaigns(
	ctx context.Context,
	req *dto.ListMessageCampaignsRequest,
) (*dto.MessageCampaignListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MessageCampaignListResponse), args.Error(1)
}

func (m *MessageUseCaseMock) GetPlayerMessages(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) (*dto.MessageListResponse, error) {
	args := m.Called(ctx, globalPlayerID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MessageListResponse), args.Error(1)
}

func (m *MessageUseCaseMock) MarkMessageAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	args := m.Called(ctx, globalPlayerID, messageID)
	return args.Error(0)
}

func (m *MessageUseCaseMock) ProcessScheduledCampaigns(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MessageUseCaseMock) SendCampaignToPlayers(ctx context.Context, campaignID uint64) error {
	args := m.Called(ctx, campaignID)
	return args.Error(0)
}

func (m *MessageUseCaseMock) SendCampaignToPlayersAsync(
	ctx context.Context,
	campaignID uint64,
) error {
	args := m.Called(ctx, campaignID)
	return args.Error(0)
}

func (m *MessageUseCaseMock) ProcessPlayer(ctx context.Context, globalPlayerID string) error {
	args := m.Called(ctx, globalPlayerID)
	return args.Error(0)
}

func (m *MessageUseCaseMock) GetMerchantAutoSettings(
	ctx context.Context,
	globalMerchantID string,
) (*dto.MerchantAutoSettingsResponse, error) {
	args := m.Called(ctx, globalMerchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MerchantAutoSettingsResponse), args.Error(1)
}

func (m *MessageUseCaseMock) CreateOrUpdateMerchantAutoSettings(
	ctx context.Context,
	req *dto.MerchantAutoSettingsRequest,
) (*dto.AutoSettingsOperationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AutoSettingsOperationResponse), args.Error(1)
}

func (m *MessageUseCaseMock) SendAutoNotification(
	ctx context.Context,
	req *dto.SendAutoNotificationRequest,
) (*dto.SendAutoNotificationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.SendAutoNotificationResponse), args.Error(1)
}

func (m *MessageUseCaseMock) SetupSuccess() {
	// Setup common success scenarios for MessageUseCase
	m.On("CreateMessageCampaign", mock.Anything, mock.Anything).Return(nil)
	m.On("UpdateMessageCampaign", mock.Anything, mock.Anything).Return(nil)
	m.On("DeleteMessageCampaign", mock.Anything, mock.Anything).Return(nil)
	m.On("ProcessScheduledCampaigns", mock.Anything).Return(nil)
	m.On("SendCampaignToPlayers", mock.Anything, mock.Anything).Return(nil)
	m.On("SendCampaignToPlayersAsync", mock.Anything, mock.Anything).Return(nil)
	m.On("ProcessPlayer", mock.Anything, mock.Anything).Return(nil)
	m.On("MarkMessageAsRead", mock.Anything, mock.Anything, mock.Anything).Return(nil)
}

func (m *MessageUseCaseMock) SetupError() {
	// Setup common error scenarios
	m.On("CreateMessageCampaign", mock.Anything, mock.Anything).Return(mock.AnythingOfType("error"))
	m.On("UpdateMessageCampaign", mock.Anything, mock.Anything).Return(mock.AnythingOfType("error"))
	m.On("DeleteMessageCampaign", mock.Anything, mock.Anything).Return(mock.AnythingOfType("error"))
	m.On("ProcessScheduledCampaigns", mock.Anything).Return(mock.AnythingOfType("error"))
}

func (m *MessageUseCaseMock) SetupEmpty() {
	// Setup empty/not found scenarios
	m.On("GetMessageCampaign", mock.Anything, mock.Anything).
		Return(nil, mock.AnythingOfType("error"))
	m.On("ListMessageCampaigns", mock.Anything, mock.Anything).
		Return(&dto.MessageCampaignListResponse{
			Campaigns: []dto.MessageCampaignResponse{},
			Total:     0,
		}, nil)
	m.On("GetPlayerMessages", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&dto.MessageListResponse{
			Messages: []dto.MessageSummary{},
			Total:    0,
		}, nil)
}

func (m *MessageUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// MerchantUseCaseMock 統一的 MerchantUseCase Mock
type MerchantUseCaseMock struct {
	*BaseMock
}

var _ inbound.MerchantUseCase = (*MerchantUseCaseMock)(nil)

// NewMerchantUseCaseMock 創建新的 MerchantUseCase Mock
func NewMerchantUseCaseMock(t *testing.T) *MerchantUseCaseMock {
	return &MerchantUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MerchantUseCaseMock) SyncMerchant(ctx context.Context, data *event.MerchantEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MerchantUseCaseMock) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*dto.MerchantResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MerchantResponse), args.Error(1)
}

func (m *MerchantUseCaseMock) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.MerchantResponse, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MerchantResponse), args.Error(1)
}

func (m *MerchantUseCaseMock) SetupSuccess() {}
func (m *MerchantUseCaseMock) SetupError()   {}
func (m *MerchantUseCaseMock) SetupEmpty()   {}
func (m *MerchantUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerUseCaseMock 統一的 PlayerUseCase Mock
type PlayerUseCaseMock struct {
	*BaseMock
}

var _ inbound.PlayerUseCase = (*PlayerUseCaseMock)(nil)

// NewPlayerUseCaseMock 創建新的 PlayerUseCase Mock
func NewPlayerUseCaseMock(t *testing.T) *PlayerUseCaseMock {
	return &PlayerUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerUseCaseMock) SyncPlayer(ctx context.Context, data *event.PlayerEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *PlayerUseCaseMock) GetPlayerByID(
	ctx context.Context,
	id uint64,
) (*dto.PlayerResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PlayerResponse), args.Error(1)
}

func (m *PlayerUseCaseMock) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.PlayerResponse, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PlayerResponse), args.Error(1)
}

func (m *PlayerUseCaseMock) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *PlayerUseCaseMock) SetupSuccess() {}
func (m *PlayerUseCaseMock) SetupError()   {}
func (m *PlayerUseCaseMock) SetupEmpty()   {}
func (m *PlayerUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// ManagerUseCaseMock 統一的 ManagerUseCase Mock
type ManagerUseCaseMock struct {
	*BaseMock
}

var _ inbound.ManagerUseCase = (*ManagerUseCaseMock)(nil)

// NewManagerUseCaseMock 創建新的 ManagerUseCase Mock
func NewManagerUseCaseMock(t *testing.T) *ManagerUseCaseMock {
	return &ManagerUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *ManagerUseCaseMock) SyncManager(ctx context.Context, data *event.ManagerEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *ManagerUseCaseMock) GetManagerByID(
	ctx context.Context,
	id uint64,
) (*dto.ManagerResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ManagerResponse), args.Error(1)
}

func (m *ManagerUseCaseMock) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*dto.ManagerResponse, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ManagerResponse), args.Error(1)
}

func (m *ManagerUseCaseMock) SetupSuccess() {}
func (m *ManagerUseCaseMock) SetupError()   {}
func (m *ManagerUseCaseMock) SetupEmpty()   {}
func (m *ManagerUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerLevelUseCaseMock 統一的 PlayerLevelUseCase Mock
type PlayerLevelUseCaseMock struct {
	*BaseMock
}

var _ inbound.PlayerLevelUseCase = (*PlayerLevelUseCaseMock)(nil)

// NewPlayerLevelUseCaseMock 創建新的 PlayerLevelUseCase Mock
func NewPlayerLevelUseCaseMock(t *testing.T) *PlayerLevelUseCaseMock {
	return &PlayerLevelUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerLevelUseCaseMock) SyncPlayerLevel(
	ctx context.Context,
	data *event.IdentityPlayerLevelSyncEvent,
) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *PlayerLevelUseCaseMock) GetLevelsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*dto.LevelListResponse, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LevelListResponse), args.Error(1)
}

func (m *PlayerLevelUseCaseMock) SetupSuccess() {}
func (m *PlayerLevelUseCaseMock) SetupError()   {}
func (m *PlayerLevelUseCaseMock) SetupEmpty()   {}
func (m *PlayerLevelUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerTagUseCaseMock 統一的 PlayerTagUseCase Mock
type PlayerTagUseCaseMock struct {
	*BaseMock
}

var _ inbound.PlayerTagUseCase = (*PlayerTagUseCaseMock)(nil)

// NewPlayerTagUseCaseMock 創建新的 PlayerTagUseCase Mock
func NewPlayerTagUseCaseMock(t *testing.T) *PlayerTagUseCaseMock {
	return &PlayerTagUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerTagUseCaseMock) SyncPlayerTags(
	ctx context.Context,
	data *event.IdentityPlayerTagSyncEvent,
) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *PlayerTagUseCaseMock) SyncTag(
	ctx context.Context,
	data *event.IdentityTagSyncEvent,
) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *PlayerTagUseCaseMock) GetTagsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*dto.TagListResponse, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TagListResponse), args.Error(1)
}

func (m *PlayerTagUseCaseMock) SetupSuccess() {}
func (m *PlayerTagUseCaseMock) SetupError()   {}
func (m *PlayerTagUseCaseMock) SetupEmpty()   {}
func (m *PlayerTagUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentUseCaseMock 統一的 AgentUseCase Mock
type AgentUseCaseMock struct {
	*BaseMock
}

var _ inbound.AgentUseCase = (*AgentUseCaseMock)(nil)

// NewAgentUseCaseMock 創建新的 AgentUseCase Mock
func NewAgentUseCaseMock(t *testing.T) *AgentUseCaseMock {
	return &AgentUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentUseCaseMock) SyncAgentDataWithRelationships(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	args := m.Called(ctx, agentEvent)
	return args.Error(0)
}

func (m *AgentUseCaseMock) SyncCurrentAgentUpsert(
	ctx context.Context,
	agentEvent *event.AgentSyncEvent,
) error {
	args := m.Called(ctx, agentEvent)
	return args.Error(0)
}

func (m *AgentUseCaseMock) GetAgentByGlobalID(
	ctx context.Context,
	globalAgentID string,
) (*entity.Agent, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Agent), args.Error(1)
}

func (m *AgentUseCaseMock) GetActiveAgents(
	ctx context.Context,
	merchantID uint64,
	limit, offset int,
) ([]*entity.Agent, error) {
	args := m.Called(ctx, merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Agent), args.Error(1)
}

func (m *AgentUseCaseMock) CreateAgentCampaign(
	ctx context.Context,
	req *dto.CreateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentCampaign), args.Error(1)
}

func (m *AgentUseCaseMock) UpdateAgentCampaign(
	ctx context.Context,
	req *dto.UpdateAgentCampaignRequest,
) (*entity.AgentCampaign, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentCampaign), args.Error(1)
}

func (m *AgentUseCaseMock) GetAgentCampaign(
	ctx context.Context,
	id uint64,
) (*entity.AgentCampaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentCampaign), args.Error(1)
}

func (m *AgentUseCaseMock) GetAgentCampaigns(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) (*dto.AgentCampaignListResponse, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AgentCampaignListResponse), args.Error(1)
}

func (m *AgentUseCaseMock) DeleteAgentCampaign(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *AgentUseCaseMock) GetAgentMessages(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) (*dto.AgentMessageListResponse, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AgentMessageListResponse), args.Error(1)
}

func (m *AgentUseCaseMock) MarkMessageAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	args := m.Called(ctx, messageID, agentID)
	return args.Error(0)
}

func (m *AgentUseCaseMock) SendMessageToCampaignTargets(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *AgentUseCaseMock) GetScheduledCampaigns(
	ctx context.Context,
) ([]*entity.AgentCampaign, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentCampaign), args.Error(1)
}

func (m *AgentUseCaseMock) FilterActiveAgents(agentIDs []string) []string {
	args := m.Called(agentIDs)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *AgentUseCaseMock) SetupSuccess() {}
func (m *AgentUseCaseMock) SetupError()   {}
func (m *AgentUseCaseMock) SetupEmpty()   {}
func (m *AgentUseCaseMock) Reset() {
	m.Mock = mock.Mock{}
}
