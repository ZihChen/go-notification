package api

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/response"
)

// AgentHandler 代理HTTP接口處理器
type AgentHandler struct {
	agentUseCase inbound.AgentUseCase
	logger       infrastructure.Logger
}

// NewAgentHandler 創建代理處理器
func NewAgentHandler(
	agentUseCase inbound.AgentUseCase,
	logger infrastructure.Logger,
) *AgentHandler {
	return &AgentHandler{
		agentUseCase: agentUseCase,
		logger:       logger,
	}
}

// ========== 代理訊息活動管理 API ==========

// CreateAgentCampaign 創建代理訊息活動
// @Summary 創建代理訊息活動
// @Description 創建新的代理訊息活動，支持全代理、指定代理、代理線等目標類型
// @Description 參數說明：
// @Description - status: 訊息狀態，必填，可選值：draft(草稿)、scheduled(排程)
// @Description - target_type: 目標類型，可選值：all(全部代理)、specific(指定代理)、line(代理線)
// @Description - target_details: 目標詳情，當target_type為specific時填入代理帳號陣列，當target_type為line時填入上級代理帳號
// @Description - scheduled_at: 排程時間，根據status決定是否必填
// @Description   * 預約發送：status=draft，scheduled_at可帶可不帶
// @Description   * 立即發送：status=scheduled，scheduled_at不得帶入
// @Description   * 預約發送：status=scheduled，scheduled_at必須帶入
// @Description
// @Description 範例請求（預約發送）：
// @Description ```json
// @Description {
// @Description   "title": "Test title",
// @Description   "content": "test45678",
// @Description   "status": "scheduled",
// @Description   "target_type": "all",
// @Description   "target_details": [],
// @Description   "scheduled_at": "2025-11-11T18:00:00Z",
// @Description   "created_by": "winston888"
// @Description }
// @Description ```
// @Description
// @Description 範例請求（立即發送）：
// @Description ```json
// @Description {
// @Description   "title": "Immediate Message",
// @Description   "content": "Immediate sending",
// @Description   "status": "scheduled",
// @Description   "target_type": "all",
// @Description   "target_details": [],
// @Description   "created_by": "winston888"
// @Description }
// @Description ```
// @Tags 代理訊息活動
// @Accept json
// @Produce json
// @Param request body dto.CreateAgentCampaignRequest true "創建代理訊息活動請求"
// @Success 201 {object} entity.AgentCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns [post]
func (h *AgentHandler) CreateAgentCampaign(c *gin.Context) {
	req := &dto.CreateAgentCampaignRequest{
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, "invalid request format", err.Error()).Return()
		return
	}
	h.logger.DebugWithContext(c, "CreateAgentCampaign", h.logger.Any("request", req))

	campaign, err := h.agentUseCase.CreateAgentCampaign(c.Request.Context(), req)
	if err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to create agent campaign",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to create agent campaign", err.Error()).Return()
		return
	}

	response.CreatedSuccess(c).Data(campaign).Return()
}

// UpdateAgentCampaign 更新代理訊息活動
// @Summary 更新代理訊息活動
// @Description 更新指定ID的代理訊息活動
// @Description 參數說明：
// @Description - status: 訊息狀態，可選，可選值：draft(草稿)、scheduled(排程)
// @Description - target_type: 目標類型，可選值：all(全部代理)、specific(指定代理)、line(代理線)
// @Description - target_details: 目標詳情，當target_type為specific時填入代理帳號陣列，當target_type為line時填入上級代理帳號
// @Description - scheduled_at: 排程時間，根據status決定是否必填
// @Description   * 預約發送：status=draft，scheduled_at可帶可不帶
// @Description   * 立即發送：status=scheduled，scheduled_at不得帶入
// @Description   * 預約發送：status=scheduled，scheduled_at必須帶入
// @Description
// @Description 範例請求（預約發送）：
// @Description ```json
// @Description {
// @Description   "title": "Test update",
// @Description   "content": "hello mey friend",
// @Description   "status": "scheduled",
// @Description   "target_type": "specific",
// @Description   "target_details": ["fatcat-john01", "fatcat-yoyo"],
// @Description   "scheduled_at": "2025-11-11T18:00:00Z",
// @Description   "updated_by": "winston888"
// @Description }
// @Description ```
// @Description
// @Description 範例請求（立即發送）：
// @Description ```json
// @Description {
// @Description   "title": "Immediate Update",
// @Description   "content": "Immediate update message",
// @Description   "status": "scheduled",
// @Description   "target_type": "all",
// @Description   "target_details": [],
// @Description   "updated_by": "winston888"
// @Description }
// @Description ```
// @Tags 代理訊息活動
// @Accept json
// @Produce json
// @Param id path uint64 true "代理訊息活動ID - 系統內部唯一識別碼，必須為正整數" example(12345)
// @Param request body dto.UpdateAgentCampaignRequest true "更新代理訊息活動請求"
// @Success 200 {object} entity.AgentCampaign
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns/{id} [put]
func (h *AgentHandler) UpdateAgentCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid campaign ID", err.Error()).Return()
		return
	}

	req := &dto.UpdateAgentCampaignRequest{
		ID:               id,
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request format", err.Error()).Return()
		return
	}
	h.logger.DebugWithContext(c, "UpdateAgentCampaign", h.logger.Any("request", req))

	campaign, err := h.agentUseCase.UpdateAgentCampaign(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "agent campaign not found", err.Error()).Return()
			return
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to update agent campaign",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to update agent campaign", err.Error()).Return()
		return
	}

	response.OK(c).Data(campaign).Return()
}

// GetAgentCampaign 獲取代理訊息活動
// @Summary 獲取代理訊息活動
// @Description 根據ID獲取代理訊息活動詳情
// @Tags 代理訊息活動
// @Param id path uint64 true "代理訊息活動ID - 系統內部唯一識別碼，必須為正整數" example(12345)
// @Success 200 {object} dto.AgentCampaignResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns/{id} [get]
func (h *AgentHandler) GetAgentCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid campaign ID", err.Error()).Return()
		return
	}

	campaign, err := h.agentUseCase.GetAgentCampaign(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "agent campaign not found", err.Error()).Return()
			return
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to get agent campaign",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to get agent campaign", err.Error()).Return()
		return
	}

	response.OK(c).Data(campaign).Return()
}

// GetAgentCampaigns 列出代理訊息活動
// @Summary 列出代理訊息活動
// @Description 分頁列出代理訊息活動，支持多條件篩選
// @Description 查詢參數說明：
// @Description - page: 頁碼，從1開始，預設為1
// @Description - page_size: 每頁數量，範圍1-100，預設為10
// @Description - status: 狀態篩選（多選），可選值：draft(草稿)、scheduled(已排程)、sent(已發送)、cancelled(已取消)
// @Description - created_by: 創建者篩選
// @Description - created_start_at: 創建時間區間開始（ISO 8601格式）
// @Description - created_end_at: 創建時間區間結束（ISO 8601格式）
// @Description - scheduled_start_at: 發送時間區間開始（ISO 8601格式）
// @Description - scheduled_end_at: 發送時間區間結束（ISO 8601格式）
// @Tags 代理訊息活動
// @Param page query int false "頁碼" minimum(1) default(1) example(1)
// @Param page_size query int false "每頁數量" minimum(1) maximum(100) default(10) example(10)
// @Param status query string false "狀態篩選（逗號分隔）" example("draft,scheduled,sent")
// @Param created_by query string false "創建者" example("admin@example.com")
// @Param created_start_at query string false "創建時間區間開始" format(date-time) example("2024-01-01T00:00:00Z")
// @Param created_end_at query string false "創建時間區間結束" format(date-time) example("2024-12-31T23:59:59Z")
// @Param scheduled_start_at query string false "發送時間區間開始" format(date-time) example("2024-01-01T00:00:00Z")
// @Param scheduled_end_at query string false "發送時間區間結束" format(date-time) example("2024-12-31T23:59:59Z")
// @Success 200 {object} dto.AgentCampaignListResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns [get]
func (h *AgentHandler) GetAgentCampaigns(c *gin.Context) {
	query := &dto.AgentCampaignsQuery{
		GlobalMerchantID: c.GetString("global_merchant_id"),
	}
	if err := c.ShouldBindQuery(query); err != nil {
		response.BadRequest(c, "invalid query parameters", err.Error()).Return()
	}

	// 設定預設值
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	campaignsResponse, err := h.agentUseCase.GetAgentCampaigns(
		c.Request.Context(),
		query,
	)
	if err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to get agent campaigns",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to get agent campaigns", err.Error()).Return()
	}

	response.OK(c).
		Data(campaignsResponse.Campaigns).
		Pagination(campaignsResponse.Page, campaignsResponse.PageSize, campaignsResponse.Total).
		Return()
}

// DeleteAgentCampaign 刪除代理訊息活動
// @Summary 刪除代理訊息活動
// @Description 刪除指定ID的代理訊息活動
// @Tags 代理訊息活動
// @Param id path uint64 true "代理訊息活動ID - 系統內部唯一識別碼，必須為正整數" example(12345)
// @Param updated_by query string true "執行刪除操作的用戶標識" example("admin001")
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns/{id} [delete]
func (h *AgentHandler) DeleteAgentCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid campaign ID", err.Error()).Return()
	}

	// 獲取執行刪除操作的用戶標識
	updatedBy := c.Query("updated_by")
	if updatedBy == "" {
		response.BadRequest(c, "updated_by parameter is required", "updated_by query parameter must be provided").
			Return()
		return
	}

	if err = h.agentUseCase.DeleteAgentCampaign(c.Request.Context(), id, updatedBy); err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "agent campaign not found", err.Error()).Return()
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to delete agent campaign",
			h.logger.Error("err", err),
		)
		response.InternalServerError(c, "failed to delete agent campaign", err.Error()).Return()
	}

	response.DeletedSuccess(c).Return()
}

// BatchDeleteAgentCampaigns 批量刪除代理訊息活動
// @Summary 批量刪除代理訊息活動
// @Description 根據ID列表批量刪除代理訊息活動，支援高效能批量操作
// @Description
// @Description **請求說明:**
// @Description - ids: 要删除的活動ID數組，必須提供至少一個有效ID
// @Description - updated_by: 執行刪除操作的用戶標識，必須提供
// @Description - 無效ID（不存在或不能删除）會被自動跳過
// @Description - 只有狀態允許删除的活動才會被處理
// @Description
// @Description **響應說明:**
// @Description - 200: 批量删除成功（部分成功也返回200）
// @Description - 400: 請求參數錯誤
// @Description - 500: 服務器內部錯誤
// @Tags 代理訊息活動
// @Accept json
// @Produce json
// @Param request body dto.BatchDeleteAgentCampaignsRequest true "批量刪除請求" example({"ids": [1, 2, 3, 4], "updated_by": "admin001"})
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agent-campaigns [delete]
func (h *AgentHandler) BatchDeleteAgentCampaigns(c *gin.Context) {
	var req dto.BatchDeleteAgentCampaignsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload", err.Error()).Return()
		return
	}

	// 驗證ID列表是否為空
	if len(req.IDs) == 0 {
		response.BadRequest(c, "ids array cannot be empty", "at least one campaign ID must be provided").
			Return()
		return
	}

	// 驗證ID是否有效
	for i, id := range req.IDs {
		if id == 0 {
			response.BadRequest(c, "invalid campaign ID", fmt.Sprintf("ID at index %d is zero", i)).
				Return()
			return
		}
	}

	if err := h.agentUseCase.BatchDeleteAgentCampaigns(c.Request.Context(), &req); err != nil {
		if err.Error() == "no valid campaigns found for deletion" {
			response.BadRequest(c, "no valid campaigns found for deletion", err.Error()).Return()
			return
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to batch delete agent campaigns",
			h.logger.Error("err", err),
			h.logger.Any("campaign_ids", req.IDs),
		)
		response.InternalServerError(c, "failed to batch delete agent campaigns", err.Error()).
			Return()
		return
	}

	response.DeletedSuccess(c).Return()
}

// ========== 代理站內信管理 API ==========

// GetAgentMessages 獲取代理站內信列表
// @Summary 獲取代理站內信列表
// @Description 分頁獲取指定代理的站內信列表，按創建時間由近到遠排序
// @Tags 代理站內信
// @Accept json
// @Produce json
// @Param global_agent_id path string true "全域代理ID - 跨系統代理唯一識別符，用於標識特定代理" example("agent-123e4567-e89b-12d3-a456-426614174000")
// @Param page query int false "頁碼 - 分頁查詢的第几頁，必須大於0" minimum(1) default(1) example(1)
// @Param page_size query int false "每頁數量 - 每頁返回的記錄數，範圍1-50" minimum(1) maximum(50) default(10) example(10)
// @Success 200 {object} dto.AgentMessageListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agents/{global_agent_id}/messages [get]
func (h *AgentHandler) GetAgentMessages(c *gin.Context) {
	globalAgentID := c.Param("global_agent_id")
	if globalAgentID == "" {
		response.BadRequest(c, "global agent ID is required").Return()
	}

	var query dto.AgentMessagesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query parameters", err.Error()).Return()
	}

	// 設定全域代理ID
	query.GlobalAgentID = globalAgentID

	// 設定預設值
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 50 {
		query.PageSize = 10
	}

	messagesResponse, err := h.agentUseCase.GetAgentMessages(
		c.Request.Context(),
		&query,
	)
	if err != nil {
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to get agent messages",
			h.logger.Error("err", err),
			h.logger.String("global_agent_id", globalAgentID),
		)
		response.InternalServerError(c, "failed to get agent messages", err.Error()).Return()
	}

	response.OK(c).Data(messagesResponse).Return()
}

// MarkMessageAsRead 標記訊息為已讀
// @Summary 標記訊息為已讀
// @Description 將指定代理的指定訊息標記為已讀
// @Tags 代理站內信
// @Accept json
// @Produce json
// @Param global_agent_id path string true "全域代理ID - 跨系統代理唯一識別符，用於標識特定代理" example("agent-123e4567-e89b-12d3-a456-426614174000")
// @Param message_id path uint64 true "訊息ID - 系統內部訊息唯一識別碼，必須為正整數" example(98765)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/agents/{global_agent_id}/messages/{message_id}/read [put]
func (h *AgentHandler) MarkMessageAsRead(c *gin.Context) {
	globalAgentID := c.Param("global_agent_id")
	if globalAgentID == "" {
		response.BadRequest(c, "global agent ID is required").Return()
	}

	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid message ID", err.Error()).Return()
	}

	// 首先根據globalAgentID獲取agentID
	agent, err := h.agentUseCase.GetAgentByGlobalID(c.Request.Context(), globalAgentID)
	if err != nil {
		if err.Error() == "record not found" {
			response.NotFound(c, "agent not found", err.Error()).Return()
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to get agent by global ID",
			h.logger.Error("err", err),
			h.logger.String("global_agent_id", globalAgentID),
		)
		response.InternalServerError(c, "failed to get agent", err.Error()).Return()
	}

	if err = h.agentUseCase.MarkMessageAsRead(c.Request.Context(), messageID, agent.ID); err != nil {
		if err.Error() == "record not found or already read" {
			response.NotFound(c, "message not found or already read", err.Error()).Return()
		}
		h.logger.ErrorWithContext(
			c.Request.Context(),
			"failed to mark message as read",
			h.logger.Error("err", err),
			h.logger.UInt64("message_id", messageID),
			h.logger.UInt64("agent_id", agent.ID),
		)
		response.InternalServerError(c, "failed to mark message as read", err.Error()).Return()
	}

	response.OK(c).Data(gin.H{
		"messageID":     messageID,
		"globalAgentID": globalAgentID,
	}).Return()
}
