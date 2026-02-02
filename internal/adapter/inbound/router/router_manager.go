package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/middleware"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/metrics"
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
	metrics       *metrics.Metrics
}

// NewRouterManager 創建路由管理器
func NewRouterManager(handler *api.HTTPHandler, agentHandler *api.AgentHandler, m *metrics.Metrics) *Manager {
	return &Manager{
		apiRouter:     NewAPIRouter(handler, agentHandler),
		swaggerRouter: NewSwaggerRouter(),
		healthRouter:  NewHealthRouter(handler),
		pprofRouter:   NewPprofRouter(),
		metrics:       m,
	}
}

// SetupRoutersWithMiddleware 配置所有中間件並註冊路由
func (rm *Manager) SetupRoutersWithMiddleware(router *gin.Engine, cfg *config.Config) {
	// 加入全局Middleware
	router.Use(
		middleware.CorsMiddleware(cfg),        // 使用環境感知的 CORS 配置
		middleware.MetricsMiddleware(rm.metrics), // Metrics 收集中間件
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
	rm.registerRoutes(router, cfg, authMiddleware)
}

// registerRoutes 註冊所有路由，每個路由器使用各自的中間件
func (rm *Manager) registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	middleware gin.HandlerFunc,
) {
	// 註冊 API 路由 (使用認證中間件)
	rm.apiRouter.RegisterRoutes(router, middleware)

	// 註冊 Swagger 路由 (不使用認證中間件)
	rm.swaggerRouter.RegisterRoutes(router, cfg)

	// 註冊健康檢查路由 (不使用認證中間件)
	rm.healthRouter.RegisterRoutes(router)
}
