package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/http/middleware"
)

// Manager 路由管理器，協調所有路由器
// 提供多種配置方式：
// 1. registerRoutes - 僅註冊路由，需手動配置中間件
// 2. SetupRoutersWithMiddleware - 自動配置所有中間件並註冊基本路由
type Manager struct {
	apiRouter     *APIRouter
	swaggerRouter *SwaggerRouter
	healthRouter  *HealthRouter
	pprofRouter   *PprofRouter
}

// NewRouterManager 創建路由管理器
func NewRouterManager(handler *api.HTTPHandler) *Manager {
	return &Manager{
		apiRouter:     NewAPIRouter(handler),
		swaggerRouter: NewSwaggerRouter(),
		healthRouter:  NewHealthRouter(handler),
		pprofRouter:   NewPprofRouter(),
	}
}

// SetupRoutersWithMiddleware 配置所有中間件並註冊路由
func (rm *Manager) SetupRoutersWithMiddleware(router *gin.Engine, cfg *config.Config) {
	// 配置CORS中間件 - 為開發環境和Swagger提供寬鬆設定
	corsConfig := cors.DefaultConfig()

	// 開發環境使用寬鬆的CORS設定
	// 在生產環境中應該限制為具體的域名
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"}
	corsConfig.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"API-Key",
		"X-Requested-With",
		"Accept",
		"Cache-Control",
		"X-File-Name",
	}
	corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	// 對於開發環境，暫時禁用credentials以避免CORS衝突
	corsConfig.AllowCredentials = false
	corsConfig.MaxAge = 12 * 60 * 60 // 12小時

	// 加入全局Middleware
	router.Use(
		cors.New(corsConfig),
		middleware.ErrorHandler(),
	)

	// 創建認證中間件配置（如果啟用）
	var authMiddleware gin.HandlerFunc
	if cfg.Auth.Enabled {
		authMiddleware = middleware.AuthMiddleware(middleware.AuthConfig{
			APIKeys:        cfg.Auth.APIKeys,
			HeaderKey:      cfg.Auth.HeaderKey,
			EncryptionType: cfg.Auth.EncryptionType,
		})
	}

	// 註冊所有路由
	rm.registerRoutes(router, authMiddleware)
}

// registerRoutes 註冊所有路由，每個路由器使用各自的中間件
func (rm *Manager) registerRoutes(router *gin.Engine, middleware gin.HandlerFunc) {
	// 註冊 API 路由 (使用認證中間件)
	rm.apiRouter.RegisterRoutes(router, middleware)

	// 註冊 Swagger 路由 (不使用認證中間件)
	rm.swaggerRouter.RegisterRoutes(router)

	// 註冊健康檢查路由 (不使用認證中間件)
	rm.healthRouter.RegisterRoutes(router)
}
