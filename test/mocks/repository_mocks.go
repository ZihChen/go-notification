package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/aggregate"
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
	status string,
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

func (m *MessageCampaignRepositoryMock) GetByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) GetByLegacyID(
	ctx context.Context,
	legacyID uint,
) (*entity.MessageCampaign, error) {
	args := m.Called(ctx, legacyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindWithLegacyIDPaginated(
	ctx context.Context,
	limit, offset int,
) ([]*entity.MessageCampaign, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MessageCampaign), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) CountWithLegacyID(
	ctx context.Context,
) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MessageCampaignRepositoryMock) FindActiveByFocus(
	ctx context.Context,
	focus string,
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
	category string,
	item string,
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
	campaignID uint64,
	targetType string,
	offset, limit int,
) ([]*entity.Player, error) {
	args := m.Called(ctx, campaignID, targetType, offset, limit)
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

func (m *PlayerRepositoryMock) FindByAccount(
	ctx context.Context,
	account string,
) (*entity.Player, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) FindByAccounts(
	ctx context.Context,
	accounts []string,
) ([]*entity.Player, error) {
	args := m.Called(ctx, accounts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) GetByGlobalPlayerID(
	ctx context.Context,
	globalPlayerID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalPlayerID)
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

func (m *PlayerRepositoryMock) GetPlayerTagIDs(
	ctx context.Context,
	playerID uint64,
) ([]uint64, error) {
	args := m.Called(ctx, playerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *PlayerRepositoryMock) SetupSuccess() {}
func (m *PlayerRepositoryMock) SetupError()   {}
func (m *PlayerRepositoryMock) SetupEmpty()   {}
func (m *PlayerRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// CampaignTargetRepositoryMock 統一的 CampaignTarget Repository Mock
type CampaignTargetRepositoryMock struct {
	*BaseMock
}

func NewCampaignTargetRepositoryMock(t *testing.T) *CampaignTargetRepositoryMock {
	return &CampaignTargetRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *CampaignTargetRepositoryMock) Create(
	ctx context.Context,
	target *entity.CampaignTarget,
) error {
	args := m.Called(ctx, target)
	return args.Error(0)
}

func (m *CampaignTargetRepositoryMock) CreateBatch(
	ctx context.Context,
	targets []*entity.CampaignTarget,
) error {
	args := m.Called(ctx, targets)
	return args.Error(0)
}

func (m *CampaignTargetRepositoryMock) DeleteByCampaignID(
	ctx context.Context,
	campaignID uint64,
) error {
	args := m.Called(ctx, campaignID)
	return args.Error(0)
}

func (m *CampaignTargetRepositoryMock) FindCampaignIDsByPlayerCriteria(
	ctx context.Context,
	merchantID, playerID, levelID uint64,
	tagIDs []uint64,
) ([]uint64, error) {
	args := m.Called(ctx, merchantID, playerID, levelID, tagIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *CampaignTargetRepositoryMock) FindByCampaignID(
	ctx context.Context,
	campaignID uint64,
) ([]*entity.CampaignTarget, error) {
	args := m.Called(ctx, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.CampaignTarget), args.Error(1)
}

func (m *CampaignTargetRepositoryMock) FindByTargetType(
	ctx context.Context,
	targetType string,
	merchantID uint64,
) ([]*entity.CampaignTarget, error) {
	args := m.Called(ctx, targetType, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.CampaignTarget), args.Error(1)
}

func (m *CampaignTargetRepositoryMock) SetupSuccess() {}
func (m *CampaignTargetRepositoryMock) SetupError()   {}
func (m *CampaignTargetRepositoryMock) SetupEmpty()   {}
func (m *CampaignTargetRepositoryMock) Reset() {
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

func (m *PlayerMessageRepositoryMock) CheckAutoMessageExists(
	ctx context.Context,
	globalPlayerID string,
	campaignID uint64,
	withinMinutes int,
) (bool, error) {
	args := m.Called(ctx, globalPlayerID, campaignID, withinMinutes)
	return args.Bool(0), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) CreateAutoNotification(
	ctx context.Context,
	message *entity.PlayerMessage,
	timeWindowMinutes int,
) error {
	args := m.Called(ctx, message, timeWindowMinutes)
	return args.Error(0)
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

func (m *PlayerMessageRepositoryMock) FindByPlayerIDWithCampaign(
	ctx context.Context,
	globalPlayerID string,
	page, pageSize int,
) ([]*aggregate.PlayerMessageAggregate, int, error) {
	args := m.Called(ctx, globalPlayerID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*aggregate.PlayerMessageAggregate), args.Int(1), args.Error(2)
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

func (m *PlayerMessageRepositoryMock) GetByPlayerAndCampaign(
	ctx context.Context,
	playerID uint64,
	campaignID uint64,
) (*entity.PlayerMessage, error) {
	args := m.Called(ctx, playerID, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PlayerMessage), args.Error(1)
}

func (m *PlayerMessageRepositoryMock) Update(
	ctx context.Context,
	message *entity.PlayerMessage,
) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *PlayerMessageRepositoryMock) FindExistingCampaignIDs(
	ctx context.Context,
	globalPlayerID string,
	campaignIDs []uint64,
) ([]uint64, error) {
	args := m.Called(ctx, globalPlayerID, campaignIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
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

func (m *MerchantRepositoryMock) GetByName(
	ctx context.Context,
	name string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
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

func (m *LevelRepositoryMock) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.Level, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Level), args.Error(1)
}

func (m *LevelRepositoryMock) FindByIDs(
	ctx context.Context,
	ids []uint64,
) ([]*entity.Level, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Level), args.Error(1)
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

func (m *TagRepositoryMock) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.Tag, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Tag), args.Error(1)
}

func (m *TagRepositoryMock) FindByIDs(
	ctx context.Context,
	ids []uint64,
) ([]*entity.Tag, error) {
	args := m.Called(ctx, ids)
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

// PushKeyRepositoryMock 統一的 PushKey Repository Mock
type PushKeyRepositoryMock struct {
	*BaseMock
}

// NewPushKeyRepositoryMock 創建新的 PushKey Repository Mock
func NewPushKeyRepositoryMock(t *testing.T) *PushKeyRepositoryMock {
	return &PushKeyRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PushKeyRepositoryMock) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*entity.PushKey, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PushKey), args.Error(1)
}

func (m *PushKeyRepositoryMock) FindByGlobalMerchantID(
	ctx context.Context,
	globalMerchantID string,
) (*entity.PushKey, error) {
	args := m.Called(ctx, globalMerchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PushKey), args.Error(1)
}

func (m *PushKeyRepositoryMock) SetupSuccess() {}
func (m *PushKeyRepositoryMock) SetupError()   {}
func (m *PushKeyRepositoryMock) SetupEmpty()   {}
func (m *PushKeyRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentRepositoryMock 統一的 Agent Repository Mock
type AgentRepositoryMock struct {
	*BaseMock
}

func NewAgentRepositoryMock(t *testing.T) *AgentRepositoryMock {
	return &AgentRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentRepositoryMock) GetByGlobalID(
	ctx context.Context,
	globalAgentID string,
) (*entity.Agent, error) {
	args := m.Called(ctx, globalAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) GetByAccount(
	ctx context.Context,
	account string,
) (*entity.Agent, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) Upsert(ctx context.Context, agent *entity.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *AgentRepositoryMock) FindAgents(
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

func (m *AgentRepositoryMock) QueryAgentsByRelationship(
	ctx context.Context,
	parentAgentID string,
) ([]uint64, error) {
	args := m.Called(ctx, parentAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *AgentRepositoryMock) ProcessAgentsByRelationshipInBatches(
	ctx context.Context,
	parentAgentID string,
	batchSize int,
	processor func(ctx context.Context, agentIDs []uint64) (processedCount int, err error),
) (totalProcessed int, err error) {
	args := m.Called(ctx, parentAgentID, batchSize, processor)
	return args.Int(0), args.Error(1)
}

func (m *AgentRepositoryMock) BatchGetAgentsByIDs(
	ctx context.Context,
	agentIDs []uint64,
) ([]*entity.Agent, error) {
	args := m.Called(ctx, agentIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) BatchGetAgentsByAccounts(
	ctx context.Context,
	accounts []string,
) ([]*entity.Agent, error) {
	args := m.Called(ctx, accounts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) BatchGetOrCreateAgentsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
	merchantID uint64,
) (map[string]uint64, error) {
	args := m.Called(ctx, globalIDs, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]uint64), args.Error(1)
}

func (m *AgentRepositoryMock) FindActiveAgents(
	ctx context.Context,
	merchantID uint64,
	activeThreshold time.Time,
	limit, offset int,
) ([]*entity.Agent, error) {
	args := m.Called(ctx, merchantID, activeThreshold, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) BatchGetActiveAgentsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
	activeThreshold time.Time,
) ([]*entity.Agent, error) {
	args := m.Called(ctx, globalIDs, activeThreshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Agent), args.Error(1)
}

func (m *AgentRepositoryMock) SetupSuccess() {}
func (m *AgentRepositoryMock) SetupError()   {}
func (m *AgentRepositoryMock) SetupEmpty()   {}
func (m *AgentRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentCampaignRepositoryMock 統一的 AgentCampaign Repository Mock
type AgentCampaignRepositoryMock struct {
	*BaseMock
}

func NewAgentCampaignRepositoryMock(t *testing.T) *AgentCampaignRepositoryMock {
	return &AgentCampaignRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentCampaignRepositoryMock) Create(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) (*entity.AgentCampaign, error) {
	args := m.Called(ctx, campaign)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentCampaign), args.Error(1)
}

func (m *AgentCampaignRepositoryMock) GetByID(
	ctx context.Context,
	id uint64,
) (*entity.AgentCampaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentCampaign), args.Error(1)
}

func (m *AgentCampaignRepositoryMock) Update(
	ctx context.Context,
	campaign *entity.AgentCampaign,
) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *AgentCampaignRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *AgentCampaignRepositoryMock) UpdateFields(
	ctx context.Context,
	id uint64,
	columns map[string]interface{},
) error {
	args := m.Called(ctx, id, columns)
	return args.Error(0)
}

func (m *AgentCampaignRepositoryMock) List(
	ctx context.Context,
	query *dto.AgentCampaignsQuery,
) ([]*entity.AgentCampaign, int, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.AgentCampaign), args.Int(1), args.Error(2)
}

func (m *AgentCampaignRepositoryMock) GetScheduledCampaigns(
	ctx context.Context,
	currentTime time.Time,
) ([]*entity.AgentCampaign, error) {
	args := m.Called(ctx, currentTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentCampaign), args.Error(1)
}

func (m *AgentCampaignRepositoryMock) FindSentCampaignsForBackfill(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.AgentCampaign, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentCampaign), args.Error(1)
}

func (m *AgentCampaignRepositoryMock) FindSentCampaignsForBackfillPaginated(
	ctx context.Context,
	merchantID uint64,
	targetTypes []string,
	limit, offset int,
) ([]*entity.AgentCampaign, error) {
	args := m.Called(ctx, merchantID, targetTypes, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentCampaign), args.Error(1)
}

func (m *AgentCampaignRepositoryMock) SetupSuccess() {}
func (m *AgentCampaignRepositoryMock) SetupError()   {}
func (m *AgentCampaignRepositoryMock) SetupEmpty()   {}
func (m *AgentCampaignRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentMessageRepositoryMock 統一的 AgentMessage Repository Mock
type AgentMessageRepositoryMock struct {
	*BaseMock
}

func NewAgentMessageRepositoryMock(t *testing.T) *AgentMessageRepositoryMock {
	return &AgentMessageRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentMessageRepositoryMock) Create(
	ctx context.Context,
	message *entity.AgentMessage,
) (*entity.AgentMessage, error) {
	args := m.Called(ctx, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentMessage), args.Error(1)
}

func (m *AgentMessageRepositoryMock) GetByID(
	ctx context.Context,
	id uint64,
) (*entity.AgentMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AgentMessage), args.Error(1)
}

func (m *AgentMessageRepositoryMock) Update(
	ctx context.Context,
	message *entity.AgentMessage,
) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *AgentMessageRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *AgentMessageRepositoryMock) CreateBatch(
	ctx context.Context,
	messages []*entity.AgentMessage,
) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *AgentMessageRepositoryMock) CheckMessageExistsBatch(
	ctx context.Context,
	agentIDs []uint64,
	campaignID uint64,
) (map[uint64]bool, error) {
	args := m.Called(ctx, agentIDs, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uint64]bool), args.Error(1)
}

func (m *AgentMessageRepositoryMock) ListByAgent(
	ctx context.Context,
	query *dto.AgentMessagesQuery,
) ([]*entity.AgentMessage, int, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entity.AgentMessage), args.Int(1), args.Error(2)
}

func (m *AgentMessageRepositoryMock) MarkAsRead(
	ctx context.Context,
	messageID uint64,
	agentID uint64,
) error {
	args := m.Called(ctx, messageID, agentID)
	return args.Error(0)
}

func (m *AgentMessageRepositoryMock) ExistsMessage(
	ctx context.Context,
	campaignID uint64,
	agentID uint64,
) (bool, error) {
	args := m.Called(ctx, campaignID, agentID)
	return args.Bool(0), args.Error(1)
}

func (m *AgentMessageRepositoryMock) CheckCampaignMessageExistsBatch(
	ctx context.Context,
	agentID uint64,
	campaignIDs []uint64,
) (map[uint64]bool, error) {
	args := m.Called(ctx, agentID, campaignIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uint64]bool), args.Error(1)
}

func (m *AgentMessageRepositoryMock) GetMessageStats(
	ctx context.Context,
	agentID uint64,
) (*dto.AgentMessageStats, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AgentMessageStats), args.Error(1)
}

func (m *AgentMessageRepositoryMock) SetupSuccess() {}
func (m *AgentMessageRepositoryMock) SetupError()   {}
func (m *AgentMessageRepositoryMock) SetupEmpty()   {}
func (m *AgentMessageRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}

// AgentRelationshipRepositoryMock 統一的 AgentRelationship Repository Mock
type AgentRelationshipRepositoryMock struct {
	*BaseMock
}

func NewAgentRelationshipRepositoryMock(t *testing.T) *AgentRelationshipRepositoryMock {
	return &AgentRelationshipRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *AgentRelationshipRepositoryMock) BatchUpdate(
	ctx context.Context,
	parentID uint64,
	relationships []*entity.AgentRelationship,
) error {
	args := m.Called(ctx, parentID, relationships)
	return args.Error(0)
}

func (m *AgentRelationshipRepositoryMock) GetByParentID(
	ctx context.Context,
	parentID uint64,
) ([]*entity.AgentRelationship, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentRelationship), args.Error(1)
}

func (m *AgentRelationshipRepositoryMock) GetByChildID(
	ctx context.Context,
	childID uint64,
) ([]*entity.AgentRelationship, error) {
	args := m.Called(ctx, childID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentRelationship), args.Error(1)
}

func (m *AgentRelationshipRepositoryMock) FindDescendants(
	ctx context.Context,
	parentID uint64,
	maxDepth int,
) ([]*entity.AgentRelationship, error) {
	args := m.Called(ctx, parentID, maxDepth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentRelationship), args.Error(1)
}

func (m *AgentRelationshipRepositoryMock) FindAncestors(
	ctx context.Context,
	childID uint64,
	maxDepth int,
) ([]*entity.AgentRelationship, error) {
	args := m.Called(ctx, childID, maxDepth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentRelationship), args.Error(1)
}

func (m *AgentRelationshipRepositoryMock) FindByPathHash(
	ctx context.Context,
	pathHash string,
) ([]*entity.AgentRelationship, error) {
	args := m.Called(ctx, pathHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.AgentRelationship), args.Error(1)
}

func (m *AgentRelationshipRepositoryMock) SetupSuccess() {}
func (m *AgentRelationshipRepositoryMock) SetupError()   {}
func (m *AgentRelationshipRepositoryMock) SetupEmpty()   {}
func (m *AgentRelationshipRepositoryMock) Reset() {
	m.Mock = mock.Mock{}
}
