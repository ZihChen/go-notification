package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

// Command 創建並返回worker
func Command() *cobra.Command {
	workerCmd := &cobra.Command{
		Use:   "worker",
		Short: "Start the worker service",
		Long:  `Start the worker service to process tasks from the queue`,
		Run:   runWorker,
	}

	return workerCmd
}

func init() {
	cmd.AddCommand(Command())
}

const (
	gracefulShutdownTime = 10 * time.Second
)

type services struct {
	tracer        *tracing.Tracer
	db            *mysql.Database
	redisManager  *redis.Manager
	workerHandler *handler.WorkerHandler
	workerServer  *asynq.Server
}

// runWorker 啟動Worker
func runWorker(cobraCmd *cobra.Command, args []string) {
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
			"Failed to initialize worker services",
			logger.Error("err", err),
		)
	}
	defer svc.cleanup(rootCtx, logger)

	// 創建追蹤 span
	ctx, rootSpan := tracing.StartSpan(rootCtx, "WorkerService")
	defer tracing.SpanEnd(rootSpan)

	// 創建並註冊任務處理器
	mux := asynq.NewServeMux()
	svc.workerHandler.RegisterHandlers(mux)
	logger.InfoWithContext(ctx, "Task handlers registered successfully")

	// 記錄Worker啟動
	logger.InfoWithContext(ctx, "Starting worker service",
		logger.String("redis", cfg.Redis.Domain),
		logger.Int("redis_port", cfg.Redis.Port))
	tracing.TraceEvent(rootSpan, "Starting worker service")

	// 啟動Worker服務器
	go func() {
		if serverErr := svc.workerServer.Start(mux); serverErr != nil &&
			!errors.Is(serverErr, asynq.ErrServerClosed) {
			logger.FatalWithContext(
				ctx,
				"Failed to start worker server",
				logger.Error("err", serverErr),
			)
			tracing.RecordSpanError(rootSpan, serverErr)
		}
	}()

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(ctx, "Shutting down worker...")
	rootCancel()

	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down worker service")

	// 優雅關閉Worker
	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTime)
	defer cancel()

	done := make(chan struct{})
	go func() {
		svc.workerServer.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		logger.InfoWithContext(ctx, "Worker service shutdown completed gracefully")
	case <-shutdownCtx.Done():
		logger.WarnWithContext(ctx, "Worker service shutdown timeout - forcing exit")
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Worker service exited gracefully")
	logger.InfoWithContext(ctx, "Worker service exited")
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
	logger.InfoWithContext(ctx, "Successfully initialized worker tracer!")

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

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker components: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized worker components!")

	return &services{
		tracer:        tracer,
		db:            db,
		redisManager:  redisManager,
		workerHandler: components.Handler,
		workerServer:  components.Server,
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
