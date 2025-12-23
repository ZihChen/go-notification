package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/docs"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerRouter 處理Swagger文檔路由註冊
type SwaggerRouter struct{}

// NewSwaggerRouter 創建Swagger路由器
func NewSwaggerRouter() *SwaggerRouter {
	return &SwaggerRouter{}
}

// RegisterRoutes 註冊Swagger文檔路由，不需要認證中間件
func (r *SwaggerRouter) RegisterRoutes(router *gin.Engine, cfg *config.Config) {
	// 動態設定 Swagger Host
	// 支援多種環境變數設定方式
	var swaggerHost string
	defaultHost := cfg.Server.HttpDomain + ":8081"
	switch cfg.App.Env {
	case "local":
		swaggerHost = defaultHost
	case "development":
		swaggerHost = cfg.Server.HttpDomain
	default:
		swaggerHost = defaultHost
	}
	docs.SwaggerInfo.Host = swaggerHost
	// Swagger 文檔路由 (不需要認證)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
