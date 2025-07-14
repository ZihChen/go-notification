package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

var (
	port int
)

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

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		logger.FatalLog("Failed to initialize tracer", logger.Error("err", err))
	}
	defer func() {
		err = tracer.Shutdown(context.Background())
		if err != nil {
			logger.ErrorLog("Failed to shutdown web tracer", logger.Error("err", err))
		}
	}()
	logger.InfoLog("Successfully initialized web tracer!")

	ctx, rootSpan := tracing.StartSpan(context.Background(), "WebService")
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

	// 使用Wire初始化HTTP處理器
	httpHandler, err := di.InitializeWebServer(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	) // 使用di包中的函數
	if err != nil {
		logger.FatalLog("Failed to initialize web server", logger.Error("err", err))
	}
	// 使用命令行指定的端口或配置中的端口
	if port == 0 {
		port = cfg.App.Port
	} else {
		port = 8080 // 默認端口
	}

	// 設置 Gin 模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 創建 Gin 路由
	router := gin.Default()

	// 註冊路由
	httpHandler.RegisterRoutes(router)

	// 創建HTTP服務器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	// 在後台運行服務器
	go func() {
		logger.InfoLog("Starting web server", logger.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.FatalLog("Failed to start server", logger.Error("err", err))
		}
	}()
	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoLog("Shutting down server...")

	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down web server")

	// 創建上下文用於通知服務器關閉
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.FatalLog("Server forced to shutdown", logger.Error("err", err))
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Web server exited gracefully")

	logger.InfoLog("Server exited")

}
