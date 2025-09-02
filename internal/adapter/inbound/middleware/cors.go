package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CorsConfig CORS 中間件配置
type CorsConfig struct {
	AllowAllOrigins  bool     `json:"allow_all_origins"`
	AllowOrigins     []string `json:"allow_origins"`
	AllowMethods     []string `json:"allow_methods"`
	AllowHeaders     []string `json:"allow_headers"`
	ExposeHeaders    []string `json:"expose_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"` // in hours
}

// defaultCorsConfig 返回適合開發環境的預設 CORS 配置
func defaultCorsConfig() CorsConfig {
	return CorsConfig{
		AllowAllOrigins: true,
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"API-Key",
			"X-Requested-With",
			"Accept",
			"Cache-Control",
			"X-File-Name",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: false, // 開發環境禁用以避免 CORS 衝突
		MaxAge:           12,    // 12小時
	}
}

// ProductionCorsConfig 返回適合生產環境的嚴格 CORS 配置
func ProductionCorsConfig(allowedOrigins []string) CorsConfig {
	return CorsConfig{
		AllowAllOrigins: false,
		AllowOrigins:    allowedOrigins,
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"API-Key",
		},
		ExposeHeaders: []string{
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           6, // 6小時
	}
}

// Setup 創建 CORS 中間件
func Setup(config CorsConfig) gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()

	// 配置 origins
	if config.AllowAllOrigins {
		corsConfig.AllowAllOrigins = true
	} else if len(config.AllowOrigins) > 0 {
		corsConfig.AllowOrigins = config.AllowOrigins
	}

	// 配置允許的方法
	if len(config.AllowMethods) > 0 {
		corsConfig.AllowMethods = config.AllowMethods
	}

	// 配置允許的 headers
	if len(config.AllowHeaders) > 0 {
		corsConfig.AllowHeaders = config.AllowHeaders
	}

	// 配置暴露的 headers
	if len(config.ExposeHeaders) > 0 {
		corsConfig.ExposeHeaders = config.ExposeHeaders
	}

	// 配置 credentials
	corsConfig.AllowCredentials = config.AllowCredentials

	// 配置 MaxAge (轉換小時為秒)
	if config.MaxAge > 0 {
		corsConfig.MaxAge = time.Duration(config.MaxAge) * time.Hour
	}

	return cors.New(corsConfig)
}

// CorsMiddleware 返回使用預設配置的 CORS 中間件
func CorsMiddleware() gin.HandlerFunc {
	return Setup(defaultCorsConfig())
}
