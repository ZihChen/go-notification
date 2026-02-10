package sse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	routermgr "github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/router"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

const (
	defaultPort     = 8081 // SSE Service 預設使用不同端口
	shutdownTimeout = 5 * time.Second
)

var (
	port int
)

// services 包含所有需要清理的服務
type services struct {
	tracer        *tracing.Tracer
	db            *mysql.Database
	redisManager  *redis.Manager
	sseComponents *di.SSEComponents
}

// Command 創建並返回 sse 子命令
func Command() *cobra.Command {
	sseCmd := &cobra.Command{
		Use:   "sse",
		Short: "Start the SSE notification server",
		Long:  `Start the SSE (Server-Sent Events) notification server for real-time push notifications`,
		Run:   runSSEServer,
	}

	sseCmd.Flags().IntVarP(&port, "port", "p", 0, "server port (default is from config or 8081)")

	return sseCmd
}

func init() {
	cmd.AddCommand(Command())
}

// runSSEServer 啟動 SSE 服務
func runSSEServer(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化所有服務
	svc, err := initializeServices(rootCtx, cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize SSE services",
			logger.Error("err", err),
		)
	}
	defer svc.cleanup(rootCtx, logger)

	// 決定服務端口
	serverPort := determinePort(port, cfg)

	// 設置 Gin 模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 創建 Gin 路由
	router := gin.Default()

	// 使用 SSE Router 管理器配置路由
	sseRouterManager := routermgr.NewSSERouterManager(
		svc.sseComponents.SSEHandler,
		svc.sseComponents.Metrics,
	)
	sseRouterManager.SetupRoutersWithMiddleware(router, cfg)

	// 創建 HTTP 服務器（針對長連接優化）
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", serverPort),
		Handler:      router,
		ReadTimeout:  5 * time.Minute,  // SSE 長連接需要較長的讀取超時
		WriteTimeout: 5 * time.Minute,  // SSE 長連接需要較長的寫入超時
		IdleTimeout:  10 * time.Minute, // 空閒連接超時
	}
	server.SetKeepAlivesEnabled(true)

	// 在後台運行服務器
	go func() {
		logger.InfoWithContext(
			rootCtx,
			"Starting SSE notification server",
			logger.Int("port", serverPort),
			logger.String("pod_id", svc.sseComponents.PodID),
		)
		if serverErr := server.ListenAndServe(); serverErr != nil &&
			!errors.Is(serverErr, http.ErrServerClosed) {
			logger.FatalWithContext(
				rootCtx,
				"Failed to start SSE server",
				logger.Error("err", serverErr),
			)
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(rootCtx, "Shutting down SSE server...")

	// 創建帶超時的上下文用於優雅關閉
	shutdownCtx, cancel := context.WithTimeout(rootCtx, shutdownTimeout)
	defer cancel()

	// 關閉 HTTP 服務器
	if err = server.Shutdown(shutdownCtx); err != nil {
		logger.ErrorWithContext(
			rootCtx,
			"Failed to gracefully shutdown SSE server",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(rootCtx, "SSE server shutdown gracefully")
	}

	// 額外檢查是否還有活動連接
	select {
	case <-shutdownCtx.Done():
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			logger.WarnWithContext(rootCtx, "SSE server shutdown timeout exceeded, forcing exit")
		}
	default:
		logger.InfoWithContext(rootCtx, "All SSE connections closed gracefully")
	}
	logger.InfoWithContext(rootCtx, "SSE server exited")
}

// initializeServices 初始化所有必要的服務
func initializeServices(
	ctx context.Context,
	cfg *config.Config,
	logger infrastructure.Logger,
) (*services, error) {
	// 初始化追蹤器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized SSE tracer!")

	// 初始化 DB 連線（SSE 可能需要查詢玩家資料）
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized database connection!")

	// 初始化 Redis 連線（SSE 核心依賴）
	redisManager := redis.NewRedisManager(cfg)
	if err = redisManager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized Redis connection!")

	// 使用 Wire 初始化 SSE 組件
	sseComponents, err := di.InitializeSSEComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SSE components: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized SSE components!")

	return &services{
		tracer:        tracer,
		db:            db,
		redisManager:  redisManager,
		sseComponents: sseComponents,
	}, nil
}

// cleanup 清理所有服務資源
func (s *services) cleanup(ctx context.Context, logger infrastructure.Logger) {
	// 關閉 Metrics 服務
	if s.sseComponents.Metrics != nil {
		if err := s.sseComponents.Metrics.Shutdown(ctx); err != nil {
			logger.ErrorWithContext(
				ctx,
				"Failed to shutdown metrics",
				logger.Error("err", err),
			)
		} else {
			logger.InfoWithContext(ctx, "Metrics service shutdown successfully")
		}
	}

	// 關閉追蹤器
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to shutdown tracer",
			logger.Error("err", err),
		)
	}

	// 關閉資料庫連線
	if err := s.db.Close(); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to close database connection",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(ctx, "Database connection closed successfully")
	}

	// 關閉 Redis 連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to close Redis connection",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(ctx, "Redis connection closed successfully")
	}
}

// determinePort 決定使用的端口
func determinePort(cmdPort int, cfg *config.Config) int {
	// 優先使用命令行參數
	if cmdPort != 0 {
		return cmdPort
	}
	// 其次使用配置文件中的 SSE 端口（如果有的話）
	// 注意：這裡可以在 Config 中新增 SSE.Port 配置
	// 最後使用默認值
	return defaultPort
}
