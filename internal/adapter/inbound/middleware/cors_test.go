package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCorsConfig(t *testing.T) {
	config := defaultCorsConfig()

	assert.True(t, config.AllowAllOrigins)
	assert.Contains(t, config.AllowMethods, "GET")
	assert.Contains(t, config.AllowMethods, "POST")
	assert.Contains(t, config.AllowMethods, "PUT")
	assert.Contains(t, config.AllowMethods, "DELETE")
	assert.Contains(t, config.AllowHeaders, "Authorization")
	assert.Contains(t, config.AllowHeaders, "API-Key")
	assert.False(t, config.AllowCredentials)
	assert.Equal(t, 12, config.MaxAge)
}

func TestProductionCorsConfig(t *testing.T) {
	allowedOrigins := []string{"https://example.com", "https://api.example.com"}
	config := ProductionCorsConfig(allowedOrigins)

	assert.False(t, config.AllowAllOrigins)
	assert.Equal(t, allowedOrigins, config.AllowOrigins)
	assert.True(t, config.AllowCredentials)
	assert.Equal(t, 6, config.MaxAge)
}

func TestDefaultCorsMiddleware(t *testing.T) {
	// 設定 Gin 為測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試用 Gin 引擎 - 使用 nil 參數測試預設 CORS 行為
	router := gin.New()
	router.Use(CorsMiddleware(nil))

	// 添加測試端點
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// 測試 OPTIONS 請求 (preflight request)
	t.Run("OPTIONS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	})

	// 測試實際請求
	t.Run("Actual GET request with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.JSONEq(t, `{"message": "test"}`, w.Body.String())
	})
}

func TestCorsMiddlewareWithCustomConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 自定義 CORS 配置
	customConfig := CorsConfig{
		AllowAllOrigins:  false,
		AllowOrigins:     []string{"https://trusted-domain.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           1, // 1小時
	}

	router := gin.New()
	router.Use(Setup(customConfig))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// 測試受信任的來源
	t.Run("Trusted origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://trusted-domain.com")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://trusted-domain.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
}

func TestProductionCorsSecurityStrictness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Production must reject requests without explicit whitelist", func(t *testing.T) {
		// 生產環境沒有白名單時應該拒絕所有跨域請求
		cfg := &config.Config{
			App: config.AppConfig{
				Env: "production",
			},
			CORS: config.CORSConfig{
				Enabled: true,
				// 故意不設定 AllowedOrigins
			},
		}

		router := gin.New()
		router.Use(CorsMiddleware(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://any-external-site.com")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 生產環境沒有明確白名單時不應該設定 Access-Control-Allow-Origin
		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		assert.Empty(
			t,
			allowOrigin,
			"Production environment should not allow any origin without explicit whitelist",
		)
	})

	t.Run("Production should only allow explicitly whitelisted origins", func(t *testing.T) {
		cfg := &config.Config{
			App: config.AppConfig{
				Env: "production",
			},
			CORS: config.CORSConfig{
				Enabled: true,
				AllowedOrigins: []string{
					"https://secure-frontend.com",
					"https://admin.secure.com",
				},
				AllowCredentials: true,
			},
		}

		router := gin.New()
		router.Use(CorsMiddleware(cfg))
		router.GET("/api/sensitive", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "sensitive information"})
		})

		testCases := []struct {
			origin      string
			shouldAllow bool
		}{
			{"https://secure-frontend.com", true},
			{"https://admin.secure.com", true},
			{"https://malicious-site.com", false},
			{"http://localhost:3000", false}, // 即使是 localhost 在生產環境也不應該自動允許
			{"", false},
		}

		for _, tc := range testCases {
			t.Run("Origin: "+tc.origin, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/api/sensitive", nil)
				if tc.origin != "" {
					req.Header.Set("Origin", tc.origin)
				}

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if tc.shouldAllow {
					assert.Equal(t, tc.origin, allowOrigin)
					assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				} else {
					// 對於不被允許的來源，檢查是否沒有設置預期的 CORS 標頭
					if tc.origin != "" {
						assert.NotEqual(t, tc.origin, allowOrigin)
					} else {
						// 空 origin 的情況下，不應該有 CORS 標頭
						assert.Empty(t, allowOrigin)
					}
				}
			})
		}
	})
}
