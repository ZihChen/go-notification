package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
)

// SSENotificationHandler SSE 通知 HTTP 處理器
type SSENotificationHandler struct {
	useCase inbound.SSENotificationUseCase
	logger  infrastructure.Logger
}

// NewSSENotificationHandler 創建 SSE 通知處理器實例
func NewSSENotificationHandler(
	useCase inbound.SSENotificationUseCase,
	logger infrastructure.Logger,
) *SSENotificationHandler {
	return &SSENotificationHandler{
		useCase: useCase,
		logger:  logger,
	}
}

// BroadcastNotification 廣播推送通知
// @Summary 廣播推送通知給所有線上玩家
// @Description 透過 Redis Pub/Sub 廣播到所有 SSE Service Pods
// @Tags SSE Notifications (Admin)
// @Accept json
// @Produce json
// @Param request body dto.BroadcastRequest true "廣播請求"
// @Success 200 {object} dto.BroadcastResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/notifications/broadcast [post]
// @Security ApiKeyAuth
func (h *SSENotificationHandler) BroadcastNotification(c *gin.Context) {
	var req dto.BroadcastRequest

	// 綁定並驗證請求
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 調用 UseCase
	response, err := h.useCase.BroadcastNotification(c.Request.Context(), &req)
	if err != nil {
		h.logger.ErrorWithContext(c.Request.Context(), "Failed to broadcast notification",
			h.logger.Error("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to broadcast notification",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendNotification 個別推送通知
// @Summary 個別推送通知給特定玩家清單
// @Description 透過玩家路由表查詢目標 Pod，並通過 Pub/Sub 轉發
// @Tags SSE Notifications (Admin)
// @Accept json
// @Produce json
// @Param request body dto.SendNotificationRequest true "個別推送請求"
// @Success 200 {object} dto.SendNotificationResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/notifications/send [post]
// @Security ApiKeyAuth
func (h *SSENotificationHandler) SendNotification(c *gin.Context) {
	var req dto.SendNotificationRequest

	// 綁定並驗證請求
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 調用 UseCase
	response, err := h.useCase.SendNotification(c.Request.Context(), &req)
	if err != nil {
		h.logger.ErrorWithContext(c.Request.Context(), "Failed to send notification",
			h.logger.Error("error", err),
			h.logger.Int("target_count", len(req.PlayerIDs)))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send notification",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// StreamNotifications SSE 長連接串流
// @Summary 建立 SSE 長連接接收實時通知
// @Description 玩家連線時註冊、推送離線訊息、保持連接
// @Tags SSE Notifications (Player)
// @Produce text/event-stream
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/notifications/stream [get]
// @Security BearerAuth
func (h *SSENotificationHandler) StreamNotifications(c *gin.Context) {
	// 從 context 中獲取 Player ID（由 JWT middleware 注入）
	playerIDValue, exists := c.Get("player_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Player ID not found in context",
		})
		return
	}

	playerID, ok := playerIDValue.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid player ID format",
		})
		return
	}

	// 創建 SSE Writer
	writer, err := NewGinSSEWriter(c)
	if err != nil {
		h.logger.ErrorWithContext(c.Request.Context(), "Failed to create SSE writer",
			h.logger.Error("error", err),
			h.logger.String("player_id", playerID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create SSE connection",
			"details": err.Error(),
		})
		return
	}

	// 調用 UseCase 處理 SSE 串流
	// 這個方法會阻塞直到連接關閉
	if err := h.useCase.StreamNotifications(c.Request.Context(), playerID, writer); err != nil {
		h.logger.ErrorWithContext(c.Request.Context(), "SSE stream error",
			h.logger.Error("error", err),
			h.logger.String("player_id", playerID))
		// SSE 連接錯誤通常意味著客戶端已斷線，不需要返回 JSON 錯誤
	}
}
