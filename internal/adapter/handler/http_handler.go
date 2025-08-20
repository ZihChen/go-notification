package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/usecaseport"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Merchant = entity.Merchant
type Player = entity.Player
type Manager = entity.Manager
type MessageCampaign = entity.MessageCampaign
type MessageListResponse = entity.MessageListResponse
type MessageCampaignListResponse = dto.MessageCampaignListResponse

// 錯誤響應模型
type ErrorResponse struct {
	Error string `json:"error" example:"An error occurred"`
}

// 成功響應模型
type SuccessResponse struct {
	Status string `json:"status" example:"success"`
}

// HTTPHandler HTTP接口處理器
type HTTPHandler struct {
	merchantUseCase usecaseport.MerchantUseCase
	playerUseCase   usecaseport.PlayerUseCase
	managerUseCase  usecaseport.ManagerUseCase
	messageUseCase  usecaseport.MessageUseCase
	logger          infraport.Logger
}

// NewHTTPHandler 創建HTTP處理器
func NewHTTPHandler(
	merchantUseCase usecaseport.MerchantUseCase,
	playerUseCase usecaseport.PlayerUseCase,
	managerUseCase usecaseport.ManagerUseCase,
	messageUseCase usecaseport.MessageUseCase,
	logger infraport.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		messageUseCase:  messageUseCase,
		logger:          logger,
	}
}

// RegisterRoutes 註冊路由
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
	// Swagger 文檔路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api/v1")

	// 商戶相關路由
	merchants := api.Group("/merchants")
	{
		merchants.GET("/:id", h.GetMerchantByID)
		merchants.GET("/global/:global_id", h.GetMerchantByGlobalID)
	}

	// 玩家相關路由
	players := api.Group("/players")
	{
		players.GET("/:id", h.GetPlayerByID)
		players.GET("/global/:global_id", h.GetPlayerByGlobalID)
		players.PUT("/:id/active", h.UpdatePlayerLastActive)
	}

	// 管理員相關路由
	managers := api.Group("/managers")
	{
		managers.GET("/:id", h.GetManagerByID)
		managers.GET("/global/:global_id", h.GetManagerByGlobalID)
	}

	// 管理端 - 訊息活動管理
	campaigns := api.Group("/message-campaigns")
	{
		campaigns.POST("", h.CreateMessageCampaign)
		campaigns.GET("", h.ListMessageCampaigns)
		campaigns.GET("/:id", h.GetMessageCampaign)
		campaigns.PUT("/:id", h.UpdateMessageCampaign)
		campaigns.DELETE("/:id", h.DeleteMessageCampaign)
	}

	// 玩家端 - 訊息查看
	messages := api.Group("/messages")
	{
		messages.GET("/player/:global_player_id", h.GetPlayerMessages)
		messages.PUT("/player/:global_player_id/:message_id/read", h.MarkMessageAsRead)
		messages.GET("/player/:global_player_id/sse", h.SSEHandler)
	}

	// 健康檢查
	router.GET("/health", h.HealthCheck)
}

// 以下是現有的方法（商戶、玩家、管理員）...
// HealthCheck 健康檢查
// @Summary 健康檢查
// @Description 檢查服務是否正常運行
// @Tags 系統
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /health [get]
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// GetMerchantByID 通過ID獲取商戶
// @Summary 通過ID獲取商戶
// @Description 根據商戶ID獲取商戶信息
// @Tags 商戶
// @Accept json
// @Produce json
// @Param id path uint64 true "商戶ID"
// @Success 200 {object} Merchant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/merchants/{id} [get]
func (h *HTTPHandler) GetMerchantByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid merchant ID",
		})
		return
	}

	merchant, err := h.merchantUseCase.GetMerchantByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Merchant not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get merchant by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get merchant",
		})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// GetMerchantByGlobalID 通過全局ID獲取商戶
func (h *HTTPHandler) GetMerchantByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global merchant ID is required",
		})
		return
	}

	merchant, err := h.merchantUseCase.GetMerchantByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Merchant not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get merchant by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get merchant",
		})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// GetPlayerByID 通過ID獲取玩家
// @Summary 通過ID獲取玩家
// @Description 根據玩家ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param id path uint64 true "玩家ID"
// @Success 200 {object} Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/players/{id} [get]
func (h *HTTPHandler) GetPlayerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	player, err := h.playerUseCase.GetPlayerByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get player by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get player",
		})
		return
	}

	c.JSON(http.StatusOK, player)
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
// @Summary 通過全局ID獲取玩家
// @Description 根據玩家全局ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param global_id path string true "玩家全局ID"
// @Success 200 {object} Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/players/global/{global_id} [get]

func (h *HTTPHandler) GetPlayerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global player ID is required",
		})
		return
	}

	player, err := h.playerUseCase.GetPlayerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get player by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get player",
		})
		return
	}

	c.JSON(http.StatusOK, player)
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
// @Router /api/v1/players/{id}/active [put]

func (h *HTTPHandler) UpdatePlayerLastActive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	if err := h.playerUseCase.UpdatePlayerLastActive(c.Request.Context(), id); err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to update player last active time",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update player",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
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
// @Router /api/v1/managers/{id} [get]

func (h *HTTPHandler) GetManagerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid manager ID",
		})
		return
	}

	manager, err := h.managerUseCase.GetManagerByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Manager not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get manager by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get manager",
		})
		return
	}

	c.JSON(http.StatusOK, manager)
}

// GetManagerByGlobalID 通過全局ID獲取管理員
// @Summary 通過全局ID獲取管理員
// @Description 根據管理員全局ID獲取管理員信息
// @Tags 管理員
// @Accept json
// @Produce json
// @Param global_id path string true "管理員全局ID"
// @Success 200 {object} Manager
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/managers/global/{global_id} [get]

func (h *HTTPHandler) GetManagerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global manager ID is required",
		})
		return
	}

	manager, err := h.managerUseCase.GetManagerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Manager not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get manager by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get manager",
		})
		return
	}

	c.JSON(http.StatusOK, manager)
}

// 以下是新增的訊息相關方法

// CreateMessageCampaign 創建會員訊息活動
// @Summary 創建會員訊息活動
// @Description 創建新的會員訊息活動
// @Tags Message Campaign
// @Accept json
// @Produce json
// @Param request body dto.CreateMessageCampaignRequest true "創建請求"
// @Success 201 {object} entity.MessageCampaign
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/message-campaigns [post]
func (h *HTTPHandler) CreateMessageCampaign(c *gin.Context) {
	var req dto.CreateMessageCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	campaign := &entity.MessageCampaign{
		Category:      req.Category,
		Item:          req.Item,
		Title:         req.Title,
		Content:       req.Content,
		Target:        req.Target,
		AutoSend:      req.AutoSend,
		SendStartTime: req.SendStartTime,
		SendEndTime:   req.SendEndTime,
		CreatedBy:     req.CreatedBy,
	}

	if err := h.messageUseCase.CreateMessageCampaign(c.Request.Context(), campaign); err != nil {
		h.logger.ErrorLog("Failed to create message campaign",
			h.logger.String("title", req.Title),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create message campaign",
		})
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

// UpdateMessageCampaign 更新會員訊息活動
// @Summary 更新會員訊息活動
// @Description 更新指定ID的會員訊息活動
// @Tags Message Campaign
// @Accept json
// @Produce json
// @Param id path int true "Campaign ID"
// @Param request body dto.UpdateMessageCampaignRequest true "Update message campaign request"
// @Success 200 {object} entity.MessageCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/message-campaigns/{id} [put]
func (h *HTTPHandler) UpdateMessageCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	var req dto.UpdateMessageCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	campaign := &entity.MessageCampaign{
		ID:            id,
		Category:      req.Category,
		Item:          req.Item,
		Title:         req.Title,
		Content:       req.Content,
		Target:        req.Target,
		AutoSend:      req.AutoSend,
		SendStartTime: req.SendStartTime,
		SendEndTime:   req.SendEndTime,
		UpdatedBy:     &req.UpdatedBy,
	}

	if err := h.messageUseCase.UpdateMessageCampaign(c.Request.Context(), campaign); err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Message campaign not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to update message campaign",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update message campaign",
		})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// DeleteMessageCampaign 刪除會員訊息活動
// @Summary 刪除會員訊息活動
// @Description 刪除指定ID的會員訊息活動
// @Tags Message Campaign
// @Param id path int true "Campaign ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/message-campaigns/{id} [delete]
func (h *HTTPHandler) DeleteMessageCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	if err := h.messageUseCase.DeleteMessageCampaign(c.Request.Context(), id); err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Message campaign not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to delete message campaign",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete message campaign",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

// GetMessageCampaign 獲取會員訊息活動
// @Summary 獲取會員訊息活動
// @Description 根據ID獲取會員訊息活動詳情
// @Tags Message Campaign
// @Param id path int true "Campaign ID"
// @Success 200 {object} entity.MessageCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/message-campaigns/{id} [get]
func (h *HTTPHandler) GetMessageCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	campaign, err := h.messageUseCase.GetMessageCampaign(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Message campaign not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get message campaign",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get message campaign",
		})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// ListMessageCampaigns 列出會員訊息活動
// @Summary 列出會員訊息活動
// @Description 分頁列出會員訊息活動
// @Tags Message Campaign
// @Param page query int false "頁碼" default(1)
// @Param page_size query int false "每頁數量" default(10)
// @Success 200 {object} MessageCampaignListResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/message-campaigns [get]
func (h *HTTPHandler) ListMessageCampaigns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	campaigns, total, err := h.messageUseCase.ListMessageCampaigns(
		c.Request.Context(),
		page,
		pageSize,
	)
	if err != nil {
		h.logger.ErrorLog("Failed to list message campaigns", h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list message campaigns",
		})
		return
	}

	response := dto.MessageCampaignListResponse{
		Data:     campaigns,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}

	c.JSON(http.StatusOK, response)
}

// GetPlayerMessages 獲取玩家訊息列表
func (h *HTTPHandler) GetPlayerMessages(c *gin.Context) {
	globalPlayerID := c.Param("global_player_id")
	if globalPlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global player ID is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	response, err := h.messageUseCase.GetPlayerMessages(
		c.Request.Context(),
		globalPlayerID,
		page,
		pageSize,
	)
	if err != nil {
		h.logger.ErrorLog("Failed to get player messages",
			h.logger.String("global_player_id", globalPlayerID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get player messages",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// MarkMessageAsRead 標記訊息為已讀
func (h *HTTPHandler) MarkMessageAsRead(c *gin.Context) {
	globalPlayerID := c.Param("global_player_id")
	if globalPlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global player ID is required",
		})
		return
	}

	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid message ID",
		})
		return
	}

	if err := h.messageUseCase.MarkMessageAsRead(c.Request.Context(), globalPlayerID, messageID); err != nil {
		if err.Error() == "record not found or already read" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Message not found or already read",
			})
			return
		}

		h.logger.ErrorLog("Failed to mark message as read",
			h.logger.String("global_player_id", globalPlayerID),
			h.logger.UInt64("message_id", messageID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to mark message as read",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
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
	var lastUnreadCount = 0
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
					h.logger.Int("unread_count", lastUnreadCount))
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
