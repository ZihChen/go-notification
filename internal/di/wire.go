//go:build wireinject
// +build wireinject

package di

import (
	"fmt"
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/consumer"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/migrate"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/scheduler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/worker"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/job"
	agentRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/agent"
	managerRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/manager"
	merchantRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/merchant"
	messageRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/message"
	playerRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/player"
	outboundService "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/service"
	agentUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/agent"
	levelUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/level"
	managerUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/manager"
	merchantUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/merchant"
	messageUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/message"
	migrateUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/migrate"
	playerUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/player"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	servicePort "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

// WorkerComponents 包含 worker 所需的所有組件
type WorkerComponents struct {
	Handler *worker.WorkerHandler
	Server  *asynq.Server
}

var baseSet = wire.NewSet(
	// 基礎設施層
	queue.NewQueueService,
	provideRedisClient,
	provideTracingService,
	provideDistributedLockManager,

	// 資料庫
	merchantRepo.NewMerchantRepository,
	playerRepo.NewPlayerRepository,
	managerRepo.NewManagerRepository,
	messageRepo.NewMessageCampaignRepository,
	messageRepo.NewCampaignTargetRepository,
	providePlayerMessageRepository,
	playerRepo.NewLevelRepository,
	playerRepo.NewTagRepository,
	playerRepo.NewPlayerTagRepository,
	merchantRepo.NewPushKeyRepository,
	// Agent repositories
	agentRepo.NewAgentRepository,
	agentRepo.NewAgentCampaignRepository,
	agentRepo.NewAgentMessageRepository,
	provideAgentRelationshipRepository,

	// 服務
	service.NewEventService,
	service.NewAgentService,
	providePushNotificationService,

	// 用例層
	merchantUseCase.NewMerchantUseCase,
	playerUseCase.NewPlayerUseCase,
	managerUseCase.NewManagerUseCase,
	messageUseCase.NewMessageUseCase,
	levelUseCase.NewLevelUseCase,
	playerUseCase.NewTagUseCase,
	agentUseCase.NewAgentUseCase,
)

// 事件生產者提供者 (保留作為別名)
func provideEventProducer(kdsService *kds.KDSService, logger infrastructure.Logger) servicePort.EventProducer {
	return service.NewEventService(kdsService, logger)
}

// 推播服務提供者
func providePushNotificationService(cfg *config.Config, logger infrastructure.Logger) servicePort.PushNotificationService {
	return outboundService.NewPushNotificationService(cfg.Push.BaseURL, logger)
}

// TracingService提供者
func provideTracingService() infrastructure.TracingService {
	return tracing.NewTracingService()
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
func InitializeWorkerServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*worker.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*WorkerComponents, error) {
	wire.Build(
		wire.Struct(new(WorkerComponents), "*"),
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
		provideWorkerServer,
	)
	return nil, nil
}

// 提供 worker 服務器
func provideWorkerServer(cfg *config.Config, logger infrastructure.Logger) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger)
}

// 提供 PlayerMessageRepository
func providePlayerMessageRepository(db *gorm.DB, redisManager *redisCache.Manager) repository.PlayerMessageRepository {
	return messageRepo.NewPlayerMessageRepository(db, redisManager)
}

// 提供 AgentRelationshipRepository
func provideAgentRelationshipRepository(db *gorm.DB, lockManager infrastructure.DistributedLockManager) repository.AgentRelationshipRepository {
	return agentRepo.NewAgentRelationshipRepository(db, lockManager)
}

// 提供 DistributedLockManager
func provideDistributedLockManager(redisManager *redisCache.Manager) infrastructure.DistributedLockManager {
	return redisCache.NewRedisDistributedLockManager(redisManager)
}

// 提供 PlayerMessageRepository (migrate 專用，不需要 redis)
func provideMigratePlayerMessageRepository(db *gorm.DB) repository.PlayerMessageRepository {
	return messageRepo.NewPlayerMessageRepository(db, nil)
}

// InitializeConsumer 初始化 Consumer 服務的 KDS 服務 (已廢棄)
func InitializeConsumer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager) (*kds.KDSService, error) {
	wire.Build(
		provideTracingService,
		queue.NewQueueService,
		kds.NewKDSService,
	)
	return nil, nil
}

// InitializeConsumerHandler 初始化 Consumer 服務的處理器
func InitializeConsumerHandler(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager) (*consumer.ConsumerHandler, error) {
	wire.Build(
		provideTracingService,
		queue.NewQueueService,
		kds.NewKDSService,
		consumer.NewConsumerHandler,
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

// LegacyDB 是舊系統資料庫連接的類型
type LegacyDB struct {
	*gorm.DB
}

// InitializeMigrateHandler 初始化 Migrate 服務的處理器
func InitializeMigrateHandler(cfg *config.Config, logger infrastructure.Logger, db *gorm.DB, legacyDSN string) (*migrate.MigrateHandler, error) {
	wire.Build(
		// 需要額外的 legacy DB 連接
		provideLegacyDB,
		// 為 migrate 提供一個不需要 redis 的版本
		provideMigratePlayerMessageRepository,
		// 基礎設施層 (使用傳入的 db 作為目標資料庫)
		merchantRepo.NewMerchantRepository,
		playerRepo.NewPlayerRepository,
		messageRepo.NewMessageCampaignRepository,
		merchantRepo.NewPushKeyRepository,
		providePushNotificationService,
		// Use case 層
		provideMigrateUseCase,
		// Handler 層
		migrate.NewMigrateHandler,
	)
	return nil, nil
}

// provideMigrateUseCase 創建 migrate use case，明確區分兩個資料庫連接
func provideMigrateUseCase(
	legacyDB *LegacyDB,
	messageRepo repository.MessageCampaignRepository,
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	playerMessageRepo repository.PlayerMessageRepository,
	pushKeyRepo repository.PushKeyRepository,
	pushService servicePort.PushNotificationService,
	logger infrastructure.Logger,
) inbound.MigrateUseCase {
	return migrateUseCase.NewMigrateUseCase(
		legacyDB.DB, // 提取底層的 *gorm.DB
		messageRepo,
		playerRepo,
		merchantRepo,
		playerMessageRepo,
		logger,
	)
}

// provideLegacyDB 提供舊系統資料庫連接（fatcat_staging）
func provideLegacyDB(legacyDSN string) (*LegacyDB, error) {
	db, err := connectToLegacyDatabase(legacyDSN)
	if err != nil {
		return nil, err
	}
	return &LegacyDB{DB: db}, nil
}

// connectToLegacyDatabase 連接到舊系統資料庫
func connectToLegacyDatabase(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("legacy database DSN is required")
	}

	// 創建 GORM 配置
	gormConfig := &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	}

	// 連接到 Legacy 資料庫
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to legacy database with DSN: %w", err)
	}

	// 設置連接池參數
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get legacy database connection pool: %w", err)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
