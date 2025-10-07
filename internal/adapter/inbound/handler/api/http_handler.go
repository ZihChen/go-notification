package api

import (
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
	merchantUseCase    inbound.MerchantUseCase
	playerUseCase      inbound.PlayerUseCase
	managerUseCase     inbound.ManagerUseCase
	messageUseCase     inbound.MessageUseCase
	playerLevelUseCase inbound.PlayerLevelUseCase
	playerTagUseCase   inbound.PlayerTagUseCase
	logger             infrastructure.Logger
}

// NewHTTPHandler 創建HTTP處理器
func NewHTTPHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	messageUseCase inbound.MessageUseCase,
	playerLevelUseCase inbound.PlayerLevelUseCase,
	playerTagUseCase inbound.PlayerTagUseCase,
	logger infrastructure.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		merchantUseCase:    merchantUseCase,
		playerUseCase:      playerUseCase,
		managerUseCase:     managerUseCase,
		messageUseCase:     messageUseCase,
		playerLevelUseCase: playerLevelUseCase,
		playerTagUseCase:   playerTagUseCase,
		logger:             logger,
	}
}

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
// @Param id path uint64 true "商戶ID - 系統內部唯一識別碼，必須為正整數" example(12345)
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
// @Param global_id path string true "商戶全局ID - 跨系統唯一識別符，由系統自動生成的UUID格式" example("550e8400-e29b-41d4-a716-446655440000")
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
// @Param id path uint64 true "玩家ID - 系統內部唯一識別碼，必須為正整數" example(67890)
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
// @Param global_id path string true "玩家全局ID - 跨系統唯一識別符，由系統自動生成的UUID格式" example("123e4567-e89b-12d3-a456-426614174000")
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
// @Param id path uint64 true "玩家ID - 系統內部唯一識別碼，必須為正整數" example(67890)
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
// @Param id path uint64 true "管理員ID - 系統內部唯一識別碼，必須為正整數" example(11223)
// @Success 200 {object} entity.Manager
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
// @Param global_id path string true "管理員全局ID - 跨系統唯一識別符，由系統自動生成的UUID格式" example("987fcdeb-51a2-43d1-9c4b-123456789abc")
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

// ========== 訊息活動管理 API ==========

// CreateMessageCampaign 創建會員訊息活動
// @Summary 創建會員訊息活動
// @Description 創建新的會員訊息活動，支持站內信和App推播兩種通知方式
// @Description 參數說明：
// @Description - category: 消息類型，可選值：member(會員消息)、bonus(紅利消息)、others(其他消息)
// @Description - item: 消息項目，可選值：registration(註冊)、identity_verification(身分驗證)、bank_card(銀行卡)、others(其他)、event(活動)、all(全部)、mission(任務)
// @Description - notification_types: 推送類型位元遮罩，1=站內信，2=App推播，3=兩者皆有，範圍1-7
// @Description - target: 目標用戶，可選值：high_activity(高活躍)、low_activity(低活躍)、not_activity(無活躍)、player(指定玩家)、level(玩家等級)、tag(玩家標籤)、all(全部用戶)
// @Description - target_detail: 目標詳情，當target為player時填入玩家賬號數組，當target為level/tag時填入對應ID字符串數組
// @Description - status: 活動狀態，可選值：draft(草稿)、scheduled(已排程)
// @Tags 會員訊息活動
// @Accept json
// @Produce json
// @Param request body dto.CreateMessageCampaignRequest true "創建消息活動請求"
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
// @Description 更新指定GlobalID的會員訊息活動，支持站內信和App推播兩種通知方式
// @Description 參數說明：
// @Description - category: 消息類型，可選值：member(會員消息)、bonus(紅利消息)、others(其他消息)
// @Description - item: 消息項目，可選值：registration(註冊)、identity_verification(身分驗證)、bank_card(銀行卡)、others(其他)、event(活動)、all(全部)、mission(任務)
// @Description - notification_types: 推送類型位元遮罩，1=站內信，2=App推播，4=其他，可組合使用，範圍1-7，可選參數
// @Description - target: 目標用戶，可選值：high_activity(高活躍)、low_activity(低活躍)、not_activity(無活躍)、player(指定玩家)、level(玩家等級)、tag(玩家標籤)、all(全部用戶)
// @Description - target_detail: 目標詳情，當target為player時填入玩家賬號數組，當target為level/tag時填入對應ID字符串數組，可選參數
// @Description - status: 活動狀態，可選值：draft(草稿)、scheduled(已排程)
// @Tags 會員訊息活動
// @Accept json
// @Produce json
// @Param global_id path string true "訊息活動全局ID - 跨系統唯一識別符，用於標識特定訊息活動" example("msg-550e8400-e29b-41d4-a716-446655440000")
// @Param request body dto.UpdateMessageCampaignRequest true "更新消息活動請求"
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
// @Param global_id path string true "訊息活動全局ID - 跨系統唯一識別符，用於標識特定訊息活動" example("msg-550e8400-e29b-41d4-a716-446655440000")
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
// @Param global_id path string true "訊息活動全局ID - 跨系統唯一識別符，用於標識特定訊息活動" example("msg-550e8400-e29b-41d4-a716-446655440000")
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
// @Description 分頁列出會員訊息活動，支持多條件篩選
// @Description 查詢參數說明：
// @Description - page: 頁碼，從1開始，預設為1
// @Description - page_size: 每頁數量，範圍1-100，預設為10
// @Description - category: 消息類型篩選，可選值：member(會員消息)、bonus(紅利消息)、others(其他消息)
// @Description - item: 消息項目篩選，可選值：registration(註冊)、identity_verification(身份驗證)、bank_card(銀行卡)、others(其他)、event(活動)、all(全部)、mission(任務)
// @Description - status: 狀態篩選，可選值：draft(草稿)、scheduled(已排程)、sent(已發送)、cancelled(已取消)、failed(發送失敗)，支持多選
// @Description - show_auto_send: 是否顯示系統自動創建的訊息，預設為false
// @Description - created_by: 創建者篩選
// @Description - start_at: 創建時間起始篩選，格式：2024-01-01T00:00:00Z
// @Description - end_at: 創建時間結束篩選，格式：2024-01-01T23:59:59Z
// @Tags 會員訊息活動
// @Param page query int false "頁碼" minimum(1) default(1) example(1)
// @Param page_size query int false "每頁數量" minimum(1) maximum(100) default(10) example(10)
// @Param category query string false "消息類型" Enums(member, bonus, others) example("member")
// @Param item query string false "消息項目" Enums(registration, identity_verification, bank_card, others, event, all, mission) example("registration")
// @Param status query []string false "狀態篩選" Enums(draft, scheduled, sent, cancelled, failed) example("scheduled,sent")
// @Param show_auto_send query bool false "是否顯示系統自動創建" default(false) example(false)
// @Param created_by query string false "創建者" example("admin@example.com")
// @Param start_at query string false "創建起始時間" format(date-time) example("2024-01-01T00:00:00Z")
// @Param end_at query string false "創建結束時間" format(date-time) example("2024-01-01T23:59:59Z")
// @Param include_deleted query bool false "是否包含已刪除記錄" default(false) example(false)
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
// @Param global_player_id path string true "全域玩家ID - 跨系統玩家唯一識別符，用於標識特定玩家" example("player-123e4567-e89b-12d3-a456-426614174000")
// @Param page query int false "頁碼 - 分頁查詢的第几頁，必須大於0" minimum(1) default(1) example(1)
// @Param page_size query int false "每頁數量 - 每頁返回的記錄數，範圍1-50" minimum(1) maximum(50) default(10) example(10)
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
// @Param global_player_id path string true "全域玩家ID - 跨系統玩家唯一識別符，用於標識特定玩家" example("player-123e4567-e89b-12d3-a456-426614174000")
// @Param message_id path uint64 true "訊息ID - 系統內部訊息唯一識別碼，必須為正整數" example(98765)
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
// @Description 為特定商戶建立或更新會員訊息自動派發設定，支持批量設定多個自動發送規則
// @Description 參數說明：
// @Description - settings: 自動設定項目列表，最少1個最多10個
// @Description   - category: 消息類型，可選值：member(會員消息)、bonus(紅利消息)、others(其他消息)
// @Description   - item: 消息項目，可選值：registration(註冊)、identity_verification(身份驗證)、bank_card(銀行卡)、others(其他)、event(活動)、all(全部)、mission(任務)
// @Description   - trigger_type: 觸發類型，可選值：success(成功)、failure(失敗)
// @Description   - title: 消息標題，必填，最大255個字符
// @Description   - content: 消息內容，必填
// @Tags 會員訊息自動設定
// @Accept json
// @Produce json
// @Param merchant_id path string true "商戶ID" example("merchant-123e4567-e89b-12d3-a456-426614174000")
// @Param request body dto.MerchantAutoSettingsRequest true "商戶自動設定請求"
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

// ========== 玩家等級管理 API ==========

// GetPlayerLevels 獲取玩家等級列表
// @Summary 獲取玩家等級列表
// @Description 獲取當前商戶的所有玩家等級
// @Tags 玩家等級
// @Accept json
// @Produce json
// @Success 200 {object} dto.LevelListResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/players/levels [get]
func (h *HTTPHandler) GetPlayerLevels(c *gin.Context) {
	merchantGlobalID := c.GetString("global_merchant_id")
	merchant, err := h.merchantUseCase.GetMerchantByGlobalID(c.Request.Context(), merchantGlobalID)
	if err != nil {
		response.InternalServerError(c, "failed to get merchant", err.Error()).Return()
		return
	}

	levels, err := h.playerLevelUseCase.GetLevelsByMerchantID(c.Request.Context(), merchant.ID)
	if err != nil {
		response.InternalServerError(c, "failed to get player levels", err.Error()).Return()
		return
	}

	response.OK(c).Data(levels).Return()
}

// ========== 玩家標籤管理 API ==========

// GetPlayerTags 獲取玩家標籤列表
// @Summary 獲取玩家標籤列表
// @Description 獲取當前商戶的所有玩家標籤
// @Tags 玩家標籤
// @Accept json
// @Produce json
// @Success 200 {object} dto.TagListResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/players/tags [get]
func (h *HTTPHandler) GetPlayerTags(c *gin.Context) {
	merchantGlobalID := c.GetString("global_merchant_id")
	merchant, err := h.merchantUseCase.GetMerchantByGlobalID(c.Request.Context(), merchantGlobalID)
	if err != nil {
		response.InternalServerError(c, "failed to get merchant", err.Error()).Return()
		return
	}

	tags, err := h.playerTagUseCase.GetTagsByMerchantID(c.Request.Context(), merchant.ID)
	if err != nil {
		response.InternalServerError(c, "failed to get player tags", err.Error()).Return()
		return
	}

	response.OK(c).Data(tags).Return()
}

// SendAutoNotification 發送系統自動推播訊息
// @Summary 發送系統自動推播訊息
// @Description 根據指定的category、item、trigger_type向特定玩家發送系統自動推播訊息
// @Description 參數說明：
// @Description - global_player_id: 玩家全局ID，必填，格式為UUID
// @Description - category: 訊息類別，可選值：member(會員訊息)、bonus(紅利訊息)
// @Description - item: 訊息項目，可選值：registration(註冊)、identity_verification(身份驗證)、bank_card(銀行卡)、mission(任務)
// @Description - trigger_type: 觸發類型，可選值：success(成功)、failure(失敗)
// @Tags 系統自動推播
// @Accept json
// @Produce json
// @Param request body dto.SendAutoNotificationRequest true "自動推播訊息請求"
// @Success 200 {object} dto.SendAutoNotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/notifications/auto-send [post]
func (h *HTTPHandler) SendAutoNotification(c *gin.Context) {
	var req dto.SendAutoNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request format", err.Error()).Return()
		return
	}

	result, err := h.messageUseCase.SendAutoNotification(c.Request.Context(), &req)
	if err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to send auto notification",
			h.logger.Error("err", err),
			h.logger.String("global_player_id", req.GlobalPlayerID),
			h.logger.String("category", req.Category),
			h.logger.String("item", req.Item),
			h.logger.String("trigger_type", req.TriggerType),
		)
		response.InternalServerError(c, "failed to send auto notification", err.Error()).Return()
		return
	}

	response.OK(c).Data(result).Return()
}
