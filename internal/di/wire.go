//go:build wireinject

package di

import (
	"fmt"
	"time"

	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/consumer"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/migrate"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/scheduler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/worker"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/job"
	failedTaskEventRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository"
	agentRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/agent"
	managerRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/manager"
	merchantRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/merchant"
	messageRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/message"
	playerRepo "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/repository/player"
	outboundService "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/service"
	agentUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/agent"
	failedTaskEventUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/failed_task_event"
	levelUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/level"
	managerUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/manager"
	merchantUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/merchant"
	messageUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/message"
	migrateUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/migrate"
	playerUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/player"
	sseNotificationUseCase "github.com/jvdiamondtech/ms-notification-cat/internal/application/usecase/sse_notification"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	servicePort "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/metrics"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// WorkerComponents 包含 worker 所需的所有組件
type WorkerComponents struct {
	Handler                *worker.WorkerHandler
	Server                 *asynq.Server
	FailedTaskEventUseCase inbound.FailedTaskEventUseCase
}

// WebComponents 包含 web 服務所需的所有組件
type WebComponents struct {
	HTTPHandler  *api.HTTPHandler
	AgentHandler *api.AgentHandler
	Metrics      *metrics.Metrics
}

// SSEComponents 包含 SSE 服務所需的所有組件
type SSEComponents struct {
	SSEHandler *api.SSENotificationHandler
	Metrics    *metrics.Metrics
}

var baseSet = wire.NewSet(
	// 基礎設施層
	queue.NewQueueService,
	provideRedisClient,
	provideTracingService,
	provideDistributedLockManager,
	provideMetricsService,

	// 資料庫
	provideMerchantRepository,
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
	// Failed Task Event repository
	failedTaskEventRepo.NewFailedTaskEventRepository,

	// 服務
	service.NewEventService,
	service.NewAgentService,
	providePushNotificationService,
	// SSE 服務
	providePodID,
	provideSSEManager,

	// 用例層
	merchantUseCase.NewMerchantUseCase,
	providePlayerUseCase,
	managerUseCase.NewManagerUseCase,
	provideMessageUseCase,
	levelUseCase.NewLevelUseCase,
	providePlayerTagUseCase,
	agentUseCase.NewAgentUseCase,
	failedTaskEventUseCase.NewFailedTaskEventUseCase,
	// SSE 用例層
	provideSSENotificationUseCase,
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

// provideMetrics 提供 Metrics 服務 (用於 WebComponents)
func provideMetrics(cfg *config.Config, log infrastructure.Logger) (*metrics.Metrics, error) {
	return metrics.NewMetrics(cfg, log)
}

// provideMetricsService 提供 MetricsService 介面 (用於 UseCase 注入)
func provideMetricsService(cfg *config.Config, log infrastructure.Logger) (infrastructure.MetricsService, error) {
	return metrics.NewMetrics(cfg, log)
}

// provideMessageUseCase 提供 MessageUseCase
func provideMessageUseCase(
	campaignRepo repository.MessageCampaignRepository,
	campaignTargetRepo repository.CampaignTargetRepository,
	merchantRepo repository.MerchantRepository,
	playerMessageRepo repository.PlayerMessageRepository,
	playerRepo repository.PlayerRepository,
	levelRepo repository.LevelRepository,
	tagRepo repository.TagRepository,
	pushApiKeyRepo repository.PushKeyRepository,
	pushService servicePort.PushNotificationService,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	metricsService infrastructure.MetricsService,
) inbound.MessageUseCase {
	return messageUseCase.NewMessageUseCase(
		campaignRepo,
		campaignTargetRepo,
		merchantRepo,
		playerMessageRepo,
		playerRepo,
		levelRepo,
		tagRepo,
		pushApiKeyRepo,
		pushService,
		logger,
		tracingService,
		metricsService,
	)
}

// ProvideMessageCampaignTriggerJob 提供訊息活動觸發器Job
func ProvideMessageCampaignTriggerJob(
	messageUseCase inbound.MessageUseCase,
	logger infrastructure.Logger,
	metricsService infrastructure.MetricsService,
) *job.MessageCampaignTriggerJob {
	return job.NewMessageCampaignTriggerJob(
		messageUseCase,
		logger,
		metricsService,
	)
}

// ProvideAgentCampaignTriggerJob 提供代理活動觸發器Job
func ProvideAgentCampaignTriggerJob(
	agentUseCase inbound.AgentUseCase,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	distributedLockMgr infrastructure.DistributedLockManager,
	metricsService infrastructure.MetricsService,
) *job.AgentCampaignTriggerJob {
	return job.NewAgentCampaignTriggerJob(
		agentUseCase,
		logger,
		tracingService,
		distributedLockMgr,
		metricsService,
	)
}

// InitializeWebServer 初始化 Web 服務的 HTTP 處理器
func InitializeWebServer(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*api.HTTPHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		api.NewHTTPHandler,
	)
	return nil, nil
}

// InitializeAgentHandler 初始化代理處理器
func InitializeAgentHandler(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*api.AgentHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		api.NewAgentHandler,
	)
	return nil, nil
}

// InitializeWebComponents 初始化 Web 服務的所有組件
func InitializeWebComponents(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*WebComponents, error) {
	wire.Build(
		wire.Struct(new(WebComponents), "*"),
		baseSet,
		kds.NewKDSService,
		api.NewHTTPHandler,
		api.NewAgentHandler,
		provideMetrics,
	)
	return nil, nil
}

// InitializeSSEComponents 初始化 SSE 服務的所有組件
func InitializeSSEComponents(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*SSEComponents, error) {
	wire.Build(
		wire.Struct(new(SSEComponents), "*"),
		// SSE 基礎依賴
		provideTracingService,
		provideDistributedLockManager,
		provideMetricsService,
		kds.NewKDSService,
		queue.NewQueueService,
		service.NewEventService,
		// SSE 核心組件
		providePodID,
		provideSSEManager,
		provideSSENotificationUseCase,
		provideSSENotificationHandler,
		provideMetrics,
	)
	return nil, nil
}

// InitializeWorkerServer 初始化 Worker 服務的處理器
func InitializeWorkerServer(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*worker.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*WorkerComponents, error) {
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
func provideWorkerServer(cfg *config.Config, logger infrastructure.Logger, failedTaskUseCase inbound.FailedTaskEventUseCase) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger, failedTaskUseCase)
}

// 提供 PlayerMessageRepository
func providePlayerMessageRepository(db *gorm.DB, lockManager infrastructure.DistributedLockManager) repository.PlayerMessageRepository {
	return messageRepo.NewPlayerMessageRepository(db, lockManager)
}

// 提供 AgentRelationshipRepository
func provideAgentRelationshipRepository(db *gorm.DB, lockManager infrastructure.DistributedLockManager) repository.AgentRelationshipRepository {
	return agentRepo.NewAgentRelationshipRepository(db, lockManager)
}

// 提供 DistributedLockManager
func provideDistributedLockManager(cacheManager infrastructure.CacheManager) infrastructure.DistributedLockManager {
	return redisCache.NewRedisDistributedLockManager(cacheManager)
}

// 提供帶快取的 MerchantRepository
func provideMerchantRepository(db *gorm.DB, cacheManager infrastructure.CacheManager) repository.MerchantRepository {
	return merchantRepo.NewMerchantRepository(db, cacheManager)
}

// 提供 PlayerMessageRepository (migrate 專用，不需要 redis)
func provideMigratePlayerMessageRepository(db *gorm.DB) repository.PlayerMessageRepository {
	return messageRepo.NewPlayerMessageRepository(db, nil)
}

// 提供 MerchantRepository (migrate 專用，不需要快取)
func provideMigrateMerchantRepository(db *gorm.DB) repository.MerchantRepository {
	return merchantRepo.NewMerchantRepository(db, nil)
}

// 提供 PlayerUseCase
func providePlayerUseCase(
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	levelRepo repository.LevelRepository,
	tagRepo repository.TagRepository,
	playerTagRepo repository.PlayerTagRepository,
	eventProducer servicePort.EventProducer,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
	cacheManager infrastructure.CacheManager,
	lockManager infrastructure.DistributedLockManager,
) inbound.PlayerUseCase {
	return playerUseCase.NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		tagRepo,
		playerTagRepo,
		eventProducer,
		logger,
		tracingService,
		cacheManager,
		lockManager,
	)
}

// 提供 PlayerTagUseCase
func providePlayerTagUseCase(
	tagRepo repository.TagRepository,
	merchantRepo repository.MerchantRepository,
	playerRepo repository.PlayerRepository,
	playerTagRepo repository.PlayerTagRepository,
	logger infrastructure.Logger,
	lockManager infrastructure.DistributedLockManager,
	tracingService infrastructure.TracingService,
	cacheManager infrastructure.CacheManager,
) inbound.PlayerTagUseCase {
	return playerUseCase.NewTagUseCase(
		tagRepo,
		merchantRepo,
		playerRepo,
		playerTagRepo,
		logger,
		lockManager,
		tracingService,
		cacheManager,
	)
}

// InitializeConsumer 初始化 Consumer 服務的 KDS 服務 (已廢棄)
func InitializeConsumer(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager) (*kds.KDSService, error) {
	wire.Build(
		provideTracingService,
		provideDistributedLockManager,
		provideMetricsService,
		queue.NewQueueService,
		kds.NewKDSService,
	)
	return nil, nil
}

// InitializeConsumerHandler 初始化 Consumer 服務的處理器
func InitializeConsumerHandler(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager) (*consumer.ConsumerHandler, error) {
	wire.Build(
		provideTracingService,
		provideDistributedLockManager,
		provideMetricsService,
		queue.NewQueueService,
		kds.NewKDSService,
		consumer.NewConsumerHandler,
	)
	return nil, nil
}

func provideRedisClient(cacheManager infrastructure.CacheManager) (*redis.Client, error) {
	redisInstance, err := cacheManager.GetClient()
	if err != nil {
		return nil, err
	}
	return redisInstance, nil
}

// InitializeSchedulerComponents 初始化 Scheduler 服務的處理器
func InitializeSchedulerComponents(cfg *config.Config, logger infrastructure.Logger, cacheManager infrastructure.CacheManager, db *gorm.DB) (*scheduler.Handler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		ProvideMessageCampaignTriggerJob,
		ProvideAgentCampaignTriggerJob,
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
		provideMigrateMerchantRepository,
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

// providePodID 提供 Pod ID (從環境變數或配置讀取)
func providePodID(cfg *config.Config) string {
	podID := cfg.SSE.PodID
	if podID == "" {
		// 如果未設置，生成一個唯一的 Pod ID
		podID = fmt.Sprintf("pod-%d", time.Now().UnixNano())
	}
	return podID
}

// provideSSEManager 提供 SSE Manager (注入 Pod ID)
func provideSSEManager(
	podID string,
	cacheManager infrastructure.CacheManager,
	logger infrastructure.Logger,
) servicePort.SSEManager {
	return outboundService.NewSSEManager(podID, cacheManager, logger)
}

// provideSSENotificationUseCase 提供 SSE Notification UseCase
func provideSSENotificationUseCase(
	sseManager servicePort.SSEManager,
	eventService servicePort.EventProducer,
	logger infrastructure.Logger,
) inbound.SSENotificationUseCase {
	return sseNotificationUseCase.NewSSENotificationUseCase(sseManager, eventService, logger)
}

// provideSSENotificationHandler 提供 SSE Notification Handler
func provideSSENotificationHandler(
	useCase inbound.SSENotificationUseCase,
	logger infrastructure.Logger,
) *api.SSENotificationHandler {
	return api.NewSSENotificationHandler(useCase, logger)
}
