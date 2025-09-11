package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

// MessageCampaignRepositoryMock 統一的 MessageCampaign Repository Mock
type MessageCampaignRepositoryMock struct {
	*BaseMock
}

// NewMessageCampaignRepositoryMock 創建新的 MessageCampaign Repository Mock
func NewMessageCampaignRepositoryMock(t *testing.T) *MessageCampaignRepositoryMock {
	return &MessageCampaignRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MessageCampaignRepositoryMock) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindScheduledCampaigns(
	ctx context.Context,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) UpdateSentCount(
	ctx context.Context,
	id uint64,
	count int64,
) error {
	args := m.Called(ctx, id, count)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) UpdateStatus(
	ctx context.Context,
	id uint64,
	status uint8,
) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) Create(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) Update(
	ctx context.Context,
	campaign *entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) UpsertAutoSettings(
	ctx context.Context,
	campaigns []*entity.MessageCampaign,
) error {
	args := m.Called(ctx, campaigns)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) DeleteAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) error {
	args := m.Called(ctx, merchantID)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) UpdateFields(
	ctx context.Context,
	id uint64,
	updates map[string]interface{},
	includeDeleted bool,
) error {
	args := m.Called(ctx, id, updates, includeDeleted)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MessageCampaignRepositoryMock) FindAllWithOptions(
	ctx context.Context,
	query *dto.MessageCampaignsQuery,
) ([]*entity.MessageCampaign, int, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Int(1), args.Error(2)
}

func (m *MessageCampaignRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindActiveByFocus(
	ctx context.Context,
	focus uint8,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx, focus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindAutoSettingsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindAutoSettingByCategoryItemTrigger(
	ctx context.Context,
	merchantID uint64,
	category uint8,
	item uint8,
	triggerType string,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, merchantID, category, item, triggerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

// 統一的設定方法
func (m *MessageCampaignRepositoryMock) SetupSuccess() {
	// 可以根據需要設定常見的成功場景
}

func (m *MessageCampaignRepositoryMock) SetupError() {
	// 可以根據需要設定常見的錯誤場景
}

func (m *MessageCampaignRepositoryMock) SetupEmpty() {
	// 可以根據需要設定空結果場景
}

func (m *MessageCampaignRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerRepositoryMock 統一的 Player Repository Mock
type PlayerRepositoryMock struct {
	*BaseMock
}

func NewPlayerRepositoryMock(t *testing.T) *PlayerRepositoryMock {
	return &PlayerRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerRepositoryMock) FindByTargetType(
	ctx context.Context,
	target uint8,
	offset, limit int,
) ([]*entity.Player, error) {
	args := m.Called(ctx, target, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) CountByTargetType(ctx context.Context, target uint8) (int64, error) {
	args := m.Called(ctx, target)
	return args.Get(0).(int64), args.Error(1)
}

func (m *PlayerRepositoryMock) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Create(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Update(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Upsert(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) SetupSuccess() {}
func (m *PlayerRepositoryMock) SetupError()   {}
func (m *PlayerRepositoryMock) SetupEmpty()   {}
func (m *PlayerRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerMessageRepositoryMock 統一的 PlayerMessage Repository Mock
type PlayerMessageRepositoryMock struct {
	*BaseMock
}

func NewPlayerMessageRepositoryMock(t *testing.T) *PlayerMessageRepositoryMock {
	return &PlayerMessageRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerMessageRepositoryMock) CheckMessageExistsBatch(
	ctx context.Context,
	playerIDs []uint64,
	campaignID uint64,
) (map[uint64]bool, error) {
	args := m.Called(ctx, playerIDs, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uint64]bool), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) CreateBatchOptimized(
	ctx context.Context,
	messages []*entity.PlayerMessage,
	batchSize int,
) error {
	args := m.Called(ctx, messages, batchSize)
	return args.Error(0)
}

func (m *PlayerMessageRepositoryMock) CheckMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
) (bool, error) {
	args := m.Called(ctx, globalPlayerID, campaignID)
	return args.Bool(0), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) Create(
	ctx context.Context,
	message *entity.PlayerMessage,
) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *PlayerMessageRepositoryMock) CreateBatch(
	ctx context.Context,
	messages []*entity.PlayerMessage,
) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *PlayerMessageRepositoryMock) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.PlayerMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PlayerMessage), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) FindByPlayerID(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) ([]*entity.PlayerMessage, int, error) {
	args := m.Called(ctx, globalPlayerID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.PlayerMessage), args.Int(1), args.Error(2)
}

func (m *PlayerMessageRepositoryMock) GetPlayerMessageStats(
	ctx context.Context,
	globalPlayerID string,
) (*dto.PlayerMessageStats, error) {
	args := m.Called(ctx, globalPlayerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PlayerMessageStats), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) MarkAsRead(
	ctx context.Context,
	globalPlayerID string,
	messageID uint64,
) error {
	args := m.Called(ctx, globalPlayerID, messageID)
	return args.Error(0)
}

func (m *PlayerMessageRepositoryMock) SetupSuccess() {}
func (m *PlayerMessageRepositoryMock) SetupError()   {}
func (m *PlayerMessageRepositoryMock) SetupEmpty()   {}
func (m *PlayerMessageRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// MerchantRepositoryMock 統一的 Merchant Repository Mock
type MerchantRepositoryMock struct {
	*BaseMock
}

func NewMerchantRepositoryMock(t *testing.T) *MerchantRepositoryMock {
	return &MerchantRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MerchantRepositoryMock) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantRepositoryMock) FirstOrCreate(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Update(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Upsert(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) SetupSuccess() {}
func (m *MerchantRepositoryMock) SetupError()   {}
func (m *MerchantRepositoryMock) SetupEmpty()   {}
func (m *MerchantRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// LevelRepositoryMock 統一的 Level Repository Mock
type LevelRepositoryMock struct {
	*BaseMock
}

func NewLevelRepositoryMock(t *testing.T) *LevelRepositoryMock {
	return &LevelRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *LevelRepositoryMock) Upsert(ctx context.Context, level *entity.Level) error {
	args := m.Called(ctx, level)
	return args.Error(0)
}

func (m *LevelRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Level), args.Error(1)
}

func (m *LevelRepositoryMock) SetupSuccess() {}
func (m *LevelRepositoryMock) SetupError()   {}
func (m *LevelRepositoryMock) SetupEmpty()   {}
func (m *LevelRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// TagRepositoryMock 統一的 Tag Repository Mock
type TagRepositoryMock struct {
	*BaseMock
}

func NewTagRepositoryMock(t *testing.T) *TagRepositoryMock {
	return &TagRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *TagRepositoryMock) Upsert(ctx context.Context, tag *entity.Tag) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}

func (m *TagRepositoryMock) BatchUpsert(ctx context.Context, tags []*entity.Tag) error {
	args := m.Called(ctx, tags)
	return args.Error(0)
}

func (m *TagRepositoryMock) FindByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Tag, error) {
	args := m.Called(ctx, globalIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Tag), args.Error(1)
}

func (m *TagRepositoryMock) SetupSuccess() {}
func (m *TagRepositoryMock) SetupError()   {}
func (m *TagRepositoryMock) SetupEmpty()   {}
func (m *TagRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// PlayerTagRepositoryMock 統一的 PlayerTag Repository Mock
type PlayerTagRepositoryMock struct {
	*BaseMock
}

func NewPlayerTagRepositoryMock(t *testing.T) *PlayerTagRepositoryMock {
	return &PlayerTagRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerTagRepositoryMock) BatchUpdate(
	ctx context.Context,
	playerID uint64,
	tagIDs []uint64,
) error {
	args := m.Called(ctx, playerID, tagIDs)
	return args.Error(0)
}

func (m *PlayerTagRepositoryMock) SetupSuccess() {}
func (m *PlayerTagRepositoryMock) SetupError()   {}
func (m *PlayerTagRepositoryMock) SetupEmpty()   {}
func (m *PlayerTagRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// ManagerRepositoryMock 統一的 Manager Repository Mock
type ManagerRepositoryMock struct {
	*BaseMock
}

func NewManagerRepositoryMock(t *testing.T) *ManagerRepositoryMock {
	return &ManagerRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *ManagerRepositoryMock) FindByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerRepositoryMock) FirstOrCreate(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Create(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Update(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Upsert(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) SetupSuccess() {}
func (m *ManagerRepositoryMock) SetupError()   {}
func (m *ManagerRepositoryMock) SetupEmpty()   {}
func (m *ManagerRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}
