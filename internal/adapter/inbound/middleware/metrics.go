package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/metrics"
)

// MetricsMiddleware 創建 metrics middleware 來自動記錄 API 請求
func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果 metrics 未啟用，直接跳過
		if m == nil || !m.IsEnabled() {
			c.Next()
			return
		}

		start := time.Now()

		// 處理請求
		c.Next()

		// 計算請求處理時間
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()

		// 如果沒有匹配的路由，使用原始路徑
		if path == "" {
			path = c.Request.URL.Path
		}

		// 記錄 API 請求
		m.RecordAPIRequest(c.Request.Method, path, status)

		// 記錄 API 延遲
		m.RecordAPIDuration(c.Request.Method, path, duration)

		// 如果狀態碼 >= 400，記錄錯誤
		if c.Writer.Status() >= 400 {
			m.RecordAPIError(c.Request.Method, path, status)
		}
	}
}
