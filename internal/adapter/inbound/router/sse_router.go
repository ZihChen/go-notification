package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/middleware"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/metrics"
)

// SSERouterManager SSE 專用路由管理器
// 只包含 SSE 相關的路由和中間件
type SSERouterManager struct {
	sseHandler *api.SSENotificationHandler
	metrics    *metrics.Metrics
}

// NewSSERouterManager 創建 SSE 路由管理器
func NewSSERouterManager(
	sseHandler *api.SSENotificationHandler,
	m *metrics.Metrics,
) *SSERouterManager {
	return &SSERouterManager{
		sseHandler: sseHandler,
		metrics:    m,
	}
}

// SetupRoutersWithMiddleware 配置所有中間件並註冊 SSE 路由
func (rm *SSERouterManager) SetupRoutersWithMiddleware(router *gin.Engine, cfg *config.Config) {
	// 加入全局 Middleware
	router.Use(
		middleware.CorsMiddleware(cfg),           // CORS 配置
		middleware.MetricsMiddleware(rm.metrics), // Metrics 收集
		middleware.ErrorHandler(),                // 錯誤處理
	)

	// 創建 API Key 認證中間件（用於 Admin API）
	var apiKeyMiddleware gin.HandlerFunc
	if cfg.Auth.Enabled {
		apiKeyMiddleware = middleware.AuthMiddleware(middleware.AuthConfig{
			APIKeys:        cfg.Auth.APIKeys,
			HeaderKey:      "API-Key",
			EncryptionType: cfg.Auth.EncryptionType,
		})
	}

	// 創建 JWT 認證中間件（用於 Player API）
	var jwtMiddleware gin.HandlerFunc
	if cfg.SSE.JWTSecretKey != "" {
		jwtMiddleware = middleware.JWTAuthMiddleware(middleware.JWTConfig{
			SecretKey: []byte(cfg.SSE.JWTSecretKey),
			HeaderKey: "Authorization",
		})
	}

	// 註冊 SSE 路由
	rm.registerSSERoutes(router, apiKeyMiddleware, jwtMiddleware)

	// 註冊健康檢查路由（SSE Service 專用）
	rm.registerHealthRoutes(router)

	// 註冊 Swagger 路由（可選，用於 API 文檔）
	swaggerRouter := NewSwaggerRouter()
	swaggerRouter.RegisterRoutes(router, cfg)
}

// registerSSERoutes 註冊 SSE 相關路由
func (rm *SSERouterManager) registerSSERoutes(
	router *gin.Engine,
	apiKeyMiddleware gin.HandlerFunc,
	jwtMiddleware gin.HandlerFunc,
) {
	// ========== Admin SSE 推播 API (API Key 認證) ==========
	if rm.sseHandler != nil && apiKeyMiddleware != nil {
		adminSSE := router.Group("/api/v1/admin/notifications")
		adminSSE.Use(apiKeyMiddleware)
		{
			adminSSE.POST("/broadcast", rm.sseHandler.BroadcastNotification)
			adminSSE.POST("/send", rm.sseHandler.SendNotification)
		}
	}

	// ========== Player SSE 串流 API (JWT 認證) ==========
	if rm.sseHandler != nil && jwtMiddleware != nil {
		playerSSE := router.Group("/api/v1/notifications")
		playerSSE.Use(jwtMiddleware)
		{
			playerSSE.GET("/stream", rm.sseHandler.StreamNotifications)
		}
	}
}

// registerHealthRoutes 註冊健康檢查路由（SSE Service 專用）
func (rm *SSERouterManager) registerHealthRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		// 簡單的健康檢查
		health.GET("/live", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "sse-notification",
			})
		})

		// 就緒檢查（可以檢查 Redis 連接等）
		health.GET("/ready", func(c *gin.Context) {
			// TODO: 可以在這裡檢查 Redis Pub/Sub 是否正常
			c.JSON(200, gin.H{
				"status":  "ready",
				"service": "sse-notification",
			})
		})
	}
}
