package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
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

// runWorker 啟動Worker
func runWorker(cobraCmd *cobra.Command, args []string) {
	// 獲取配置和日誌
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 主程序的Context
	rootCtx := context.Background()
	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		logger.FatalLog("Failed to initialize tracer", logger.Error("err", err))
	}
	defer func() {
		err = tracer.Shutdown(context.Background())
		if err != nil {
			logger.ErrorWithContext(
				rootCtx,
				"Failed to shutdown tracer",
				logger.Error("err", err),
			)
		}
	}()
	logger.InfoWithContext(rootCtx, "Successfully initialized tracer!")

	ctx, rootSpan := tracing.StartSpan(context.Background(), "WorkerService")
	defer rootSpan.End()

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		logger.FatalLog("Failed to initialize database", logger.Error("err", err))
	}
	defer func() {
		err = db.Close() // 主程序結束後關閉DB連線
		if err != nil {
			logger.ErrorLog("Failed to close database connection", logger.Error("err", err))
		}
		logger.InfoLog("Database connection closed successfull")
	}()

	if err = db.Ping(); err != nil {
		logger.FatalLog("Failed to ping database", logger.Error("err", err))
	}

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	defer func() {
		err = redisManager.Close() // 主程序結束後關閉Redis連線
		if err != nil {
			logger.ErrorLog("Failed to close Redis connection", logger.Error("err", err))
		}
	}()
	if err = redisManager.Connect(rootCtx); err != nil {
		logger.FatalLog("Failed to connect to Redis", logger.Error("err", err))
	}

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		logger.FatalLog("Failed to initialize worker components", logger.Error("err", err))
		rootSpan.RecordError(err)
		return
	}

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	logger.InfoLog("Starting worker service",
		logger.String("redis", cfg.Redis.Domain),
		logger.Int("redis_port", cfg.Redis.Port))
	tracing.TraceEvent(rootSpan, "Starting worker service")

	components.Handler.RegisterHandlers(mux)
	logger.InfoLog("Task handlers registered")

	go func() {
		if err := components.Server.Start(mux); err != nil {
			if err != asynq.ErrServerClosed {
				logger.FatalLog("Failed to start worker server", logger.Error("err", err))
				rootSpan.RecordError(err)
			}
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoLog("Shutting down worker...")
	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down worker service")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 關閉Worker
	done := make(chan struct{})
	go func() {
		components.Server.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		logger.InfoLog("Worker service shutdown completed gracefully")
	case <-shutdownCtx.Done():
		logger.WarnLog("Worker service shutdown timeout - forcing exit")
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Worker service exited gracefully")

	logger.InfoLog("Worker exited")
}
