package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/response"
)

// JWTConfig JWT 認證配置
type JWTConfig struct {
	SecretKey []byte // JWT 簽名密鑰
	HeaderKey string // Header 名稱 (預設: "Authorization")
}

// JWTClaims JWT Payload 結構
type JWTClaims struct {
	PlayerID   string `json:"player_id"`   // 玩家 ID
	MerchantID string `json:"merchant_id"` // 商戶 ID (可選)
	jwt.RegisteredClaims
}

// JWTAuthMiddleware JWT 認證中介軟體（用於 SSE 串流端點）
// 驗證 Bearer Token，提取 Player ID 並注入到 Gin Context
func JWTAuthMiddleware(config JWTConfig) gin.HandlerFunc {
	// 設定預設 header key
	headerKey := config.HeaderKey
	if headerKey == "" {
		headerKey = "Authorization"
	}

	return func(c *gin.Context) {
		// 從 Header 取得 Authorization Token
		authHeader := c.GetHeader(headerKey)
		if authHeader == "" {
			response.Unauthorized(c, "Missing authorization token", "The request is missing the authorization header").
				Abort()
			return
		}

		// 檢查 Bearer 前綴
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			response.Unauthorized(c, "Invalid authorization format", "Authorization header must use Bearer scheme").
				Abort()
			return
		}

		// 提取 Token
		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
		if tokenString == "" {
			response.Unauthorized(c, "Empty token", "Token string is empty").
				Abort()
			return
		}

		// 解析並驗證 JWT Token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// 驗證簽名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.SecretKey, nil
		})

		if err != nil {
			response.Unauthorized(c, "Invalid token", err.Error()).
				Abort()
			return
		}

		// 驗證 Token 有效性
		if !token.Valid {
			response.Unauthorized(c, "Token is invalid", "The provided token is not valid").
				Abort()
			return
		}

		// 提取 Claims
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			response.Unauthorized(c, "Invalid token claims", "Unable to extract claims from token").
				Abort()
			return
		}

		// 驗證 Player ID 是否存在
		if claims.PlayerID == "" {
			response.Unauthorized(c, "Missing player ID in token", "Token does not contain player_id claim").
				Abort()
			return
		}

		// 將 Player ID 和 Merchant ID 注入到 Context（供後續 Handler 使用）
		c.Set("player_id", claims.PlayerID)
		if claims.MerchantID != "" {
			c.Set("merchant_id", claims.MerchantID)
		}

		c.Next()
	}
}
