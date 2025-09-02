//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler/scheduler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/job"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/repository"
	usecase2 "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/usecase"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/usecase/message_campaign"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/queue"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// WorkerComponents 包含 worker 所需的所有組件
type WorkerComponents struct {
	Handler *handler.WorkerHandler
	Server  *asynq.Server
}

var baseSet = wire.NewSet(
	// 基礎設施層
	queue.NewQueueService,
	provideRedisClient,

	// 資料庫
	repository.NewMerchantRepository,
	repository.NewPlayerRepository,
	repository.NewManagerRepository,
	repository.NewMessageCampaignRepository,
	repository.NewPlayerMessageRepository,
	repository.NewLevelRepository,
	repository.NewTagRepository,
	repository.NewPlayerTagRepository,

	// 服務
	provideEventProducer,

	// 用例層
	usecase2.NewMerchantUseCase,
	usecase2.NewPlayerUseCase,
	usecase2.NewManagerUseCase,
	message_campaign.NewMessageUseCase,
	usecase2.NewLevelUseCase,
	usecase2.NewTagUseCase,
)

// 事件生產者提供者
func provideEventProducer(kdsService *kds.KDSService, logger infrastructure.Logger) service.EventProducer {
	return kdsService
}

// InitializeWebServer 初始化 Web 服務的 HTTP 處理器
func InitializeWebServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*api.HTTPHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		api.NewHTTPHandler,
	)
	return nil, nil
}

// InitializeWorkerServer 初始化 Worker 服務的處理器
func InitializeWorkerServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*handler.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		handler.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*WorkerComponents, error) {
	wire.Build(
		wire.Struct(new(WorkerComponents), "*"),
		baseSet,
		kds.NewKDSService,
		handler.NewWorkerHandler,
		provideWorkerServer,
	)
	return nil, nil
}

// 提供 worker 服務器
func provideWorkerServer(cfg *config.Config, logger infrastructure.Logger) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger)
}

// InitializeConsumer 初始化 Consumer 服務的 KDS 服務
func InitializeConsumer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager) (*kds.KDSService, error) {
	wire.Build(
		queue.NewQueueService,
		kds.NewKDSService,
	)
	return nil, nil
}

func provideRedisClient(manager *redisCache.Manager) (*redis.Client, error) {
	redisInstance, err := manager.GetClient()
	if err != nil {
		return nil, err
	}
	return redisInstance, nil
}

// InitializeSchedulerComponents 初始化 Scheduler 服務的處理器
func InitializeSchedulerComponents(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*scheduler.Handler, error) {
	wire.Build(
		baseSet,
		job.NewMessageCampaignTriggerJob,
		job.NewRegistry,
		scheduler.NewSchedulerHandler,
	)
	return nil, nil
}
