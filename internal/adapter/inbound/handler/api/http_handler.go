package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/response"
)

// ErrorResponse 錯誤響應模型
type ErrorResponse struct {
	Error string `json:"error" example:"An error occurred"`
}

// SuccessResponse 成功響應模型
type SuccessResponse struct {
	Status string `json:"status" example:"success"`
}

// HTTPHandler HTTP接口處理器
type HTTPHandler struct {
	merchantUseCase inbound.MerchantUseCase
	playerUseCase   inbound.PlayerUseCase
	managerUseCase  inbound.ManagerUseCase
	messageUseCase  inbound.MessageUseCase
	logger          infrastructure.Logger
}

// NewHTTPHandler 創建HTTP處理器
func NewHTTPHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	messageUseCase inbound.MessageUseCase,
	logger infrastructure.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		messageUseCase:  messageUseCase,
		logger:          logger,
	}
}

// HealthCheck 以下是現有的方法（商戶、玩家、管理員）...
// HealthCheck 健康檢查
// @Summary 健康檢查
// @Description 檢查服務是否正常運行，包含基本服務狀態
// @Tags 系統
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /health [get]
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	healthStatus := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"services":  gin.H{},
	}

	services := healthStatus["services"].(gin.H)

	// 檢查數據庫連接 - 通過嘗試獲取商戶來測試數據庫
	if _, err := h.merchantUseCase.GetMerchantByID(ctx, 1); err != nil {
		// 如果獲取商戶失敗，檢查是否是"not found"錯誤（說明數據庫連接正常）
		if err.Error() != "merchant not found" && err.Error() != "record not found" {
			services["database"] = gin.H{
				"status": "unhealthy",
				"error":  "database connection failed",
			}
			h.logger.ErrorWithContext(
				ctx,
				"Database health check failed",
				h.logger.Error("error", err),
			)

			// 返回503服務不可用
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database service unavailable",
			})
			return
		}
	}

	services["database"] = gin.H{
		"status": "healthy",
	}

	// 返回健康狀態
	c.JSON(http.StatusOK, healthStatus)
}

// GetMerchantByID 通過ID獲取商戶
// @Summary 通過ID獲取商戶
// @Description 根據商戶ID獲取商戶信息
// @Tags 商戶
// @Accept json
// @Produce json
// @Param id path uint64 true "商戶ID"
// @Success 200 {object} entity.Merchant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/merchants/{id} [get]
func (h *HTTPHandler) GetMerchantByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid merchant ID", err.Error()).Return()
	}

	merchant, err := h.merchantUseCase.GetMerchantByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "merchant not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to get merchant", err.Error()).Return()
	}
	response.OK(c).Data(merchant).Return()
}

// GetMerchantByGlobalID 通過全局ID獲取商戶
// @Summary 通過全局ID獲取商戶
// @Description 根據商戶全局ID獲取商戶信息
// @Tags 商戶
// @Accept json
// @Produce json
// @Param global_id path string true "商戶全局ID"
// @Success 200 {object} entity.Merchant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/merchants/global/{global_id} [get]
func (h *HTTPHandler) GetMerchantByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "global merchant ID is required").Return()
	}

	merchant, err := h.merchantUseCase.GetMerchantByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "merchant not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to get merchant", err.Error()).Return()
	}
	response.OK(c).Data(merchant).Return()
}

// GetPlayerByID 通過ID獲取玩家
// @Summary 通過ID獲取玩家
// @Description 根據玩家ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param id path uint64 true "玩家ID"
// @Success 200 {object} entity.Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/players/{id} [get]
func (h *HTTPHandler) GetPlayerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid player ID", err.Error()).Return()
	}

	player, err := h.playerUseCase.GetPlayerByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "player not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to get player", err.Error()).Return()
	}
	response.OK(c).Data(player).Return()
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
// @Summary 通過全局ID獲取玩家
// @Description 根據玩家全局ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param global_id path string true "玩家全局ID"
// @Success 200 {object} entity.Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/players/global/{global_id} [get]
func (h *HTTPHandler) GetPlayerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "global player ID is required").Return()
	}

	player, err := h.playerUseCase.GetPlayerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "player not found", err.Error()).Return()
		}

		response.InternalServerError(c, "failed to get player", err.Error()).Return()
	}
	response.OK(c).Data(player).Return()
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
// @Summary 更新玩家最後活躍時間
// @Description 將玩家的最後活躍時間更新為當前時間
// @Tags 玩家
// @Accept json
// @Produce json
// @Param id path uint64 true "玩家ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/players/{id}/active [put]
func (h *HTTPHandler) UpdatePlayerLastActive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid player ID", err.Error()).Return()
	}

	if err = h.playerUseCase.UpdatePlayerLastActive(c.Request.Context(), id); err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "player not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to update player last active", err.Error()).Return()
	}
	response.OK(c).Data(gin.H{
		"playerID": id,
	}).Return()
}

// GetManagerByID 通過ID獲取管理員
// @Summary 通過ID獲取管理員
// @Description 根據管理員ID獲取管理員信息
// @Tags 管理員
// @Accept json
// @Produce json
// @Param id path uint64 true "管理員ID"
// @Success 200 {object} Manager
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/managers/{id} [get]

func (h *HTTPHandler) GetManagerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid Manager ID", err.Error()).Return()
	}

	manager, err := h.managerUseCase.GetManagerByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			response.BadRequest(c, "manager not found", err.Error()).Return()
		}
		response.BadRequest(c, "failed to get manager", err.Error()).Return()
	}
	response.OK(c).Data(manager).Return()
}

// GetManagerByGlobalID 通過全局ID獲取管理員
// @Summary 通過全局ID獲取管理員
// @Description 根據管理員全局ID獲取管理員信息
// @Tags 管理員
// @Accept json
// @Produce json
// @Param global_id path string true "管理員全局ID"
// @Success 200 {object} entity.Manager
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/managers/global/{global_id} [get]
func (h *HTTPHandler) GetManagerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "global manager ID is required").Return()
	}

	manager, err := h.managerUseCase.GetManagerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "manager not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to get manager", err.Error()).Return()
	}
	response.OK(c).Data(manager).Return()
}

// 以下是新增的訊息相關方法

// CreateMessageCampaign 創建會員訊息活動
// @Summary 創建會員訊息活動
// @Description 創建新的會員訊息活動
// @Tags 會員訊息活動
// @Accept json
// @Produce json
// @Param request body dto.CreateMessageCampaignRequest true "創建請求"
// @Success 201 {object} entity.MessageCampaign
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns [post]
func (h *HTTPHandler) CreateMessageCampaign(c *gin.Context) {
	req := &dto.CreateMessageCampaignRequest{
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, "invalid request format", err.Error()).Return()
	}

	if err := h.messageUseCase.CreateMessageCampaign(c.Request.Context(), req); err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to create message campaign",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to create message campaign", err.Error()).Return()
	}
	response.CreatedSuccess(c).Return()
}

// UpdateMessageCampaign 更新會員訊息活動
// @Summary 更新會員訊息活動
// @Description 更新指定GlobalID的會員訊息活動
// @Tags 會員訊息活動
// @Accept json
// @Produce json
// @Param id path string true "Campaign Global ID"
// @Param request body dto.UpdateMessageCampaignRequest true "Update message campaign request"
// @Success 200 {object} entity.MessageCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns/{global_id} [put]
func (h *HTTPHandler) UpdateMessageCampaign(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "invalid campaign global_id", "global_id parameter is required").
			Return()
	}

	req := &dto.UpdateMessageCampaignRequest{
		GlobalID:         globalID,
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request format", err.Error()).Return()
	}

	if err = h.messageUseCase.UpdateMessageCampaign(c.Request.Context(), req); err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "message campaign not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to update message campaign", err.Error()).Return()
	}
	response.OK(c).Data(req).Return()
}

// DeleteMessageCampaign 刪除會員訊息活動
// @Summary 刪除會員訊息活動
// @Description 刪除指定GlobalID的會員訊息活動
// @Tags 會員訊息活動
// @Param global_id path string true "Campaign Global ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns/{global_id} [delete]
func (h *HTTPHandler) DeleteMessageCampaign(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "invalid campaign global_id", "global_id parameter is required").
			Return()
	}

	if err := h.messageUseCase.DeleteMessageCampaign(c.Request.Context(), globalID); err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "message campaign not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to delete message campaign", err.Error()).Return()
	}
	response.DeletedSuccess(c).Return()
}

// GetMessageCampaign 獲取會員訊息活動
// @Summary 獲取會員訊息活動
// @Description 根據GlobalID獲取會員訊息活動詳情
// @Tags 會員訊息活動
// @Param global_id path string true "Campaign Global ID"
// @Success 200 {object} entity.MessageCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns/{global_id} [get]
func (h *HTTPHandler) GetMessageCampaign(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		response.BadRequest(c, "invalid campaign global_id", "global_id parameter is required").
			Return()
	}

	campaign, err := h.messageUseCase.GetMessageCampaign(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "message campaign not found", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to get message campaign", err.Error()).Return()
		return
	}
	response.OK(c).Data(campaign).Return()
}

// ListMessageCampaigns 列出會員訊息活動
// @Summary 列出會員訊息活動
// @Description 分頁列出會員訊息活動
// @Tags 會員訊息活動
// @Param page query int false "頁碼" default(1)
// @Param page_size query int false "每頁數量" default(10)
// @Param include_deleted query bool false "是否包含已刪除的活動" default(false)
// @Param status query []int false "狀態篩選 (1=草稿, 2=已排程, 3=已發送, 4=已取消)"
// @Success 200 {object} dto.MessageCampaignListResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns [get]
func (h *HTTPHandler) ListMessageCampaigns(c *gin.Context) {
	var req dto.ListMessageCampaignsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid query parameters", err.Error()).Return()
		return
	}

	// 設定預設值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	campaignsResponse, err := h.messageUseCase.ListMessageCampaigns(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		response.InternalServerError(c, "failed to list message campaigns", err.Error()).Return()
	}

	response.OK(c).
		Data(campaignsResponse.Campaigns).
		Pagination(campaignsResponse.Page, campaignsResponse.PageSize, campaignsResponse.Total).
		Return()
}

// GetPlayerMessages 獲取玩家訊息列表
// @Summary 獲取玩家訊息列表
// @Description 分頁獲取指定玩家的訊息列表
// @Tags 玩家訊息
// @Accept json
// @Produce json
// @Param global_player_id path string true "全域玩家ID"
// @Param page query int false "頁碼" default(1)
// @Param page_size query int false "每頁數量" default(10)
// @Success 200 {object} dto.MessageListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/messages/player/{global_player_id} [get]
func (h *HTTPHandler) GetPlayerMessages(c *gin.Context) {
	globalPlayerID := c.Param("global_player_id")
	if globalPlayerID == "" {
		response.BadRequest(c, "global player ID is required").Return()
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	res, err := h.messageUseCase.GetPlayerMessages(
		c.Request.Context(),
		globalPlayerID,
		page,
		pageSize,
	)
	if err != nil {
		response.InternalServerError(c, "failed to get player messages", err.Error()).Return()
	}
	response.OK(c).Data(res).Return()
}

// MarkMessageAsRead 標記訊息為已讀
// @Summary 標記訊息為已讀
// @Description 將指定玩家的指定訊息標記為已讀
// @Tags 玩家訊息
// @Accept json
// @Produce json
// @Param global_player_id path string true "全域玩家ID"
// @Param message_id path uint64 true "訊息ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/messages/player/{global_player_id}/{message_id}/read [put]
func (h *HTTPHandler) MarkMessageAsRead(c *gin.Context) {
	globalPlayerID := c.Param("global_player_id")
	if globalPlayerID == "" {
		response.BadRequest(c, "global player ID is required").Return()
	}

	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid message ID", err.Error()).Return()
	}

	if err = h.messageUseCase.MarkMessageAsRead(c.Request.Context(), globalPlayerID, messageID); err != nil {
		if err.Error() == "record not found or already read" {
			response.NotFound(c, "message not found or already read", err.Error()).Return()
		}
		response.InternalServerError(c, "failed to mark message as read", err.Error()).Return()
	}

	response.OK(c).Data(gin.H{
		"messageID": messageID,
	}).Return()
}

// GetMerchantAutoSettings 獲取商戶自動設定
// @Summary 獲取商戶自動設定
// @Description 根據商戶ID獲取會員訊息自動派發設定
// @Tags 會員訊息自動設定
// @Accept json
// @Produce json
// @Param merchant_id path string true "商戶ID"
// @Success 200 {object} dto.MerchantAutoSettingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns/auto-settings [get]
func (h *HTTPHandler) GetMerchantAutoSettings(c *gin.Context) {
	GlobalMerchantID := c.GetString("global_merchant_id")
	result, err := h.messageUseCase.GetMerchantAutoSettings(c.Request.Context(), GlobalMerchantID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "merchant not found or has no auto settings").Return()
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to get merchant auto settings",
			h.logger.Error("err", err),
			h.logger.String("global_merchant_id", GlobalMerchantID),
		)
		response.InternalServerError(c, "failed to get auto settings", err.Error()).Return()
	}
	response.OK(c).Data(result).Return()
}

// CreateOrUpdateMerchantAutoSettings 新增或更新商戶自動設定
// @Summary 新增或更新商戶自動設定
// @Description 為特定商戶建立或更新會員訊息自動派發設定
// @Tags 會員訊息自動設定
// @Accept json
// @Produce json
// @Param merchant_id path string true "商戶ID"
// @Param request body dto.MerchantAutoSettingsRequest true "自動設定請求"
// @Success 200 {object} dto.AutoSettingsOperationResponse "更新成功"
// @Success 201 {object} dto.AutoSettingsOperationResponse "建立成功"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/message-campaigns/auto-settings [post]
// @Router /api/v1/message-campaigns/auto-settings [put]
func (h *HTTPHandler) CreateOrUpdateMerchantAutoSettings(c *gin.Context) {
	req := &dto.MerchantAutoSettingsRequest{
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	if err := c.ShouldBindJSON(req); err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"invalid request format",
			h.logger.Error("err", err),
		)
		response.BadRequest(c, "invalid request format", err.Error()).Return()
	}

	if len(req.Settings) == 0 {
		response.BadRequest(c, "settings cannot be empty").Return()
	}

	if len(req.Settings) > 10 {
		response.BadRequest(c, "settings count cannot exceed 10").Return()
	}

	result, err := h.messageUseCase.CreateOrUpdateMerchantAutoSettings(
		c.Request.Context(),
		req,
	)
	if err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to create/update merchant auto settings",
			h.logger.Error("err", err),
			h.logger.String("global_merchant_id", req.GlobalMerchantID),
			h.logger.Int("settings_count", len(req.Settings)),
		)
		response.InternalServerError(c, "failed to create/update auto settings", err.Error()).
			Return()
		return
	}

	// 根據 HTTP 方法提供不同的回應狀態碼
	if c.Request.Method == http.MethodPost {
		response.CreatedSuccess(c).Data(result).Return()
	} else {
		response.OK(c).Data(result).Return()
	}
}

// SSEHandler Server-Sent Events處理器
// @Summary Server-Sent Events訊息推送
// @Description 為指定玩家建立SSE連接，即時推送訊息更新
// @Tags 玩家訊息
// @Accept json
// @Produce text/event-stream
// @Param global_player_id path string true "全域玩家ID"
// @Success 200 {string} string "SSE stream"
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/messages/player/{global_player_id}/sse [get]
func (h *HTTPHandler) SSEHandler(c *gin.Context) {
	globalPlayerID := c.Param("global_player_id")
	if globalPlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global player ID is required",
		})
		return
	}

	// 設置SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 發送初始連接成功訊息
	_, _ = fmt.Fprintf(
		c.Writer,
		"data: {\"type\": \"connected\", \"message\": \"SSE connection established\"}\n\n",
	)
	c.Writer.Flush()

	// 獲取初始訊息統計
	response, err := h.messageUseCase.GetPlayerMessages(c.Request.Context(), globalPlayerID, 1, 10)
	if err != nil {
		h.logger.ErrorLog("Failed to get initial player messages for SSE",
			h.logger.String("global_player_id", globalPlayerID),
			h.logger.Error("err", err))
	} else {
		// 發送初始統計資訊
		statsData, _ := json.Marshal(map[string]interface{}{
			"type":  "stats",
			"stats": response.Stats,
		})
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", statsData)
		c.Writer.Flush()
	}

	// 建立ticker用於定期檢查新訊息
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 用於檢測客戶端斷線
	clientGone := c.Writer.CloseNotify()
	var lastUnreadCount int64 = 0
	if response != nil {
		lastUnreadCount = response.Stats.UnreadCount
	}

	h.logger.InfoLog("SSE connection established",
		h.logger.String("global_player_id", globalPlayerID))

	for {
		select {
		case <-clientGone:
			h.logger.InfoLog("SSE client disconnected",
				h.logger.String("global_player_id", globalPlayerID))
			return
		case <-ticker.C:
			// 定期檢查新訊息
			currentResponse, err := h.messageUseCase.GetPlayerMessages(
				c.Request.Context(),
				globalPlayerID,
				1,
				1,
			)
			if err != nil {
				h.logger.WarnLog("Failed to check messages in SSE",
					h.logger.String("global_player_id", globalPlayerID),
					h.logger.Error("err", err))
				continue
			}

			// 如果未讀數量有變化，發送更新
			if currentResponse.Stats.UnreadCount != lastUnreadCount {
				updateData, _ := json.Marshal(map[string]interface{}{
					"type":  "stats_update",
					"stats": currentResponse.Stats,
				})
				_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", updateData)
				c.Writer.Flush()
				lastUnreadCount = currentResponse.Stats.UnreadCount

				h.logger.DebugLog("SSE stats update sent",
					h.logger.String("global_player_id", globalPlayerID),
					h.logger.Int64("unread_count", lastUnreadCount))
			}

			// 發送心跳
			_, _ = fmt.Fprintf(
				c.Writer,
				"data: {\"type\": \"heartbeat\", \"timestamp\": \"%s\"}\n\n",
				time.Now().Format(time.RFC3339),
			)
			c.Writer.Flush()
		}
	}
}
