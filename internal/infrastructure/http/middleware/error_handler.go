package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/http/response"
)

// ErrorHandler 全局錯誤處理中間件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 處理請求過程中的錯誤
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// 根據錯誤類型返回適當的響應
			switch err.Type {
			case gin.ErrorTypePublic:
				response.NewResponse(c).
					Status(http.StatusBadRequest).
					Success(false).
					Error(response.ErrCodeBadRequest, err.Error()).
					Return()
			case gin.ErrorTypeBind:
				response.NewResponse(c).
					Status(http.StatusBadRequest).
					Success(false).
					Error(response.ErrCodeValidationFailed, "Request validation failed", err.Meta).
					Return()
			default:
				response.InternalServerError(c, "An unexpected error occurred").Return()
			}

			c.Abort()
		}
	}
}

// RateLimitExceeded 速率限制響應
func RateLimitExceeded(c *gin.Context) {
	response.NewResponse(c).
		Status(http.StatusTooManyRequests).
		Success(false).
		Error(response.ErrCodeTooManyRequests, "Rate limit exceeded").
		Meta(map[string]interface{}{
			"retry_after": 60, // seconds
		}).
		Abort()
}

// NotFoundHandler 404 處理
func NotFoundHandler(c *gin.Context) {
	response.NotFound(c, "The requested resource was not found").Return()
}
