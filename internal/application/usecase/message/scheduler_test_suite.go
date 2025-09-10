package message

import (
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/test/factories"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
)

// SchedulerTestSuite 排程測試套件
type SchedulerTestSuite struct {
	useCase           *MessageUseCase
	campaignRepo      *mocks.MessageCampaignRepositoryMock
	playerRepo        *mocks.PlayerRepositoryMock
	playerMessageRepo *mocks.PlayerMessageRepositoryMock
	factory           *factories.TestDataFactory
	logger            *helper.MockLogger
}

// setupSchedulerTestSuite 設定排程測試套件
func setupSchedulerTestSuite(t *testing.T) *SchedulerTestSuite {
	// 創建Mock實例
	campaignRepo := mocks.NewMessageCampaignRepositoryMock(t)
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	playerMessageRepo := mocks.NewPlayerMessageRepositoryMock(t)
	logger := helper.NewMockLogger()

	// 創建UseCase實例
	useCase := &MessageUseCase{
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		logger:            logger,
	}

	// 創建測試數據工廠
	factory := factories.NewTestDataFactory()

	return &SchedulerTestSuite{
		useCase:           useCase,
		campaignRepo:      campaignRepo,
		playerRepo:        playerRepo,
		playerMessageRepo: playerMessageRepo,
		factory:           factory,
		logger:            logger,
	}
}

// tearDown 清理測試套件
func (s *SchedulerTestSuite) tearDown() {
	if s.campaignRepo != nil {
		s.campaignRepo.Reset()
	}
	if s.playerRepo != nil {
		s.playerRepo.Reset()
	}
	if s.playerMessageRepo != nil {
		s.playerMessageRepo.Reset()
	}
}