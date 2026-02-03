package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
)

// GinSSEWriter Gin 框架的 SSE Writer 實作
type GinSSEWriter struct {
	ctx     *gin.Context
	flusher http.Flusher
	closed  bool
}

// NewGinSSEWriter 創建 Gin SSE Writer 實例
func NewGinSSEWriter(c *gin.Context) (inbound.SSEWriter, error) {
	// 檢查是否支援 Flusher（SSE 必需）
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	// 設置 SSE Headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // 禁用 Nginx 緩衝

	return &GinSSEWriter{
		ctx:     c,
		flusher: flusher,
		closed:  false,
	}, nil
}

// Write 寫入 SSE 事件
// event: 事件類型 (connected, notification, ping, error)
// data: JSON 格式的事件數據
func (w *GinSSEWriter) Write(event string, data string) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	// SSE 格式: event: {event}\ndata: {data}\n\n
	_, err := fmt.Fprintf(w.ctx.Writer, "event: %s\ndata: %s\n\n", event, data)
	return err
}

// Flush 立即推送緩衝區的數據到客戶端
func (w *GinSSEWriter) Flush() error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	w.flusher.Flush()
	return nil
}

// Close 關閉 SSE 連接
func (w *GinSSEWriter) Close() error {
	if w.closed {
		return nil
	}

	w.closed = true
	// Gin 會自動處理連接關閉，這裡只標記狀態
	return nil
}
