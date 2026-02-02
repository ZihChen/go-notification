package web

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
	defaultPort     = 8080
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
	webComponents *di.WebComponents
}

// Command 創建並返回web子命令
func Command() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web server",
		Long:  `Start the web server to handle HTTP API requests`,
		Run:   runWebServer,
	}

	webCmd.Flags().IntVarP(&port, "port", "p", 0, "server port (default is from config)")

	return webCmd
}

func init() {
	cmd.AddCommand(Command())
}

// runWebServer 啟動Web服務
func runWebServer(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化所有服務
	svc, err := initializeServices(rootCtx, cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize services",
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

	// 使用路由管理器配置所有中間件並註冊路由
	routerManager := routermgr.NewRouterManager(
		svc.webComponents.HTTPHandler,
		svc.webComponents.AgentHandler,
		svc.webComponents.Metrics,
	)
	routerManager.SetupRoutersWithMiddleware(router, cfg)

	// 創建HTTP服務器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", serverPort),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,  // 讀取請求的超時時間
		WriteTimeout: cfg.Server.WriteTimeout, // 寫入響應的超時時間
		IdleTimeout:  cfg.Server.IdleTimeout,  // 空閒連接的超時時間
	}
	server.SetKeepAlivesEnabled(true)

	// 在後台運行服務器
	go func() {
		logger.InfoWithContext(
			rootCtx,
			"Starting web server",
			logger.Int("port", serverPort),
		)
		if serverErr := server.ListenAndServe(); serverErr != nil &&
			!errors.Is(serverErr, http.ErrServerClosed) {
			logger.FatalWithContext(
				rootCtx,
				"Failed to start server",
				logger.Error("err", serverErr),
			)
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(rootCtx, "Shutting down server...")

	// 創建帶超時的上下文用於優雅關閉
	shutdownCtx, cancel := context.WithTimeout(rootCtx, shutdownTimeout)
	defer cancel()

	// 關閉HTTP服務器，停止接受新請求，並等待現有請求完成
	if err = server.Shutdown(shutdownCtx); err != nil {
		logger.ErrorWithContext(
			rootCtx,
			"Failed to gracefully shutdown server",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(rootCtx, "HTTP server shutdown gracefully")
	}

	// 額外檢查是否還有活動連接
	select {
	case <-shutdownCtx.Done():
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			logger.WarnWithContext(rootCtx, "Server shutdown timeout exceeded, forcing exit")
		}
	default:
		logger.InfoWithContext(rootCtx, "All connections closed gracefully")
	}
	logger.InfoWithContext(rootCtx, "Server exited")
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
	logger.InfoWithContext(ctx, "Successfully initialized web tracer!")

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

	// 使用Wire初始化Web組件
	webComponents, err := di.InitializeWebComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web components: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized web components!")

	return &services{
		tracer:        tracer,
		db:            db,
		redisManager:  redisManager,
		webComponents: webComponents,
	}, nil
}

// cleanup 清理所有服務資源
func (s *services) cleanup(ctx context.Context, logger infrastructure.Logger) {
	// 關閉 Metrics 服務
	if s.webComponents.Metrics != nil {
		if err := s.webComponents.Metrics.Shutdown(ctx); err != nil {
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

	// 關閉Redis連線
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
	// 其次使用配置文件
	if cfg.App.Port != 0 {
		return cfg.App.Port
	}
	// 最後使用默認值
	return defaultPort
}
