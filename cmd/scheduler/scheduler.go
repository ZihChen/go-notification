package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler/scheduler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

// Command 創建並返回scheduler子命令
func Command() *cobra.Command {
	schedulerCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Start the scheduler service",
		Long:  `Start the scheduler service to handle scheduled tasks and cron jobs`,
		Run:   runScheduler,
	}

	return schedulerCmd
}

func init() {
	cmd.AddCommand(Command())
}

const (
	gracefulShutdownTime = 30 * time.Second
)

type services struct {
	tracer           *tracing.Tracer
	db               *mysql.Database
	redisManager     *redis.Manager
	schedulerHandler *scheduler.Handler
}

// runScheduler 啟動Scheduler服務
func runScheduler(cobraCmd *cobra.Command, args []string) {
	// 獲取配置和日誌
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	// 輸出配置資訊用於除錯追蹤
	cfg.PrintConfig()

	// 主程序的Context
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化所有服務
	svc, err := initializeServices(rootCtx, cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize scheduler services",
			logger.Error("err", err),
		)
	}
	defer svc.cleanup(rootCtx, logger)

	// 初始化 Cron 調度器，使用秒級精度
	cronManager := cron.New(cron.WithSeconds())

	// 註冊所有排程任務
	svc.schedulerHandler.RegisterJobs(cronManager)
	logger.InfoWithContext(rootCtx, "Scheduler jobs registered successfully")

	// 啟動 Cron 調度器
	cronManager.Start()
	logger.InfoWithContext(rootCtx, "Scheduler service started successfully")

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(rootCtx, "Shutting down scheduler...")
	rootCancel()

	// 停止 Cron 調度器
	cronCtx := cronManager.Stop()

	// 等待優雅關閉或超時
	select {
	case <-cronCtx.Done():
		logger.InfoWithContext(
			rootCtx,
			"All scheduled jobs stopped gracefully",
		)
	case <-time.After(gracefulShutdownTime):
		logger.WarnWithContext(
			rootCtx,
			"Force shutdown - some jobs may still be running",
		)
	}

	logger.InfoWithContext(rootCtx, "Scheduler service exited")
}

func initializeServices(
	ctx context.Context,
	cfg *config.Config,
	logger infrastructure.Logger,
) (*services, error) {
	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized scheduler tracer!")

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized database connection!")

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	if err = redisManager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized Redis connection!")

	// 使用Wire初始化Scheduler組件
	schedulerHandler, err := di.InitializeSchedulerComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scheduler components: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized scheduler components!")

	return &services{
		tracer:           tracer,
		db:               db,
		redisManager:     redisManager,
		schedulerHandler: schedulerHandler,
	}, nil
}

func (s *services) cleanup(ctx context.Context, logger infrastructure.Logger) {
	// 關閉追蹤器
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorLog("Failed to shutdown tracer", logger.Error("err", err))
	}

	// 關閉資料庫連線
	if err := s.db.Close(); err != nil {
		logger.ErrorLog("Failed to close database connection", logger.Error("err", err))
	} else {
		logger.InfoWithContext(ctx, "Database connection closed successfully")
	}

	// 關閉Redis連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorLog("Failed to close Redis connection", logger.Error("err", err))
	} else {
		logger.InfoWithContext(ctx, "Redis connection closed successfully")
	}
}
