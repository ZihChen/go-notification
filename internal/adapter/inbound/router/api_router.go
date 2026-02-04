package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
)

// APIRouter 處理API路由註冊
type APIRouter struct {
	handler           *api.HTTPHandler
	agentHandler      *api.AgentHandler
	sseHandler        *api.SSENotificationHandler
}

// NewAPIRouter 創建API路由器
func NewAPIRouter(
	handler *api.HTTPHandler,
	agentHandler *api.AgentHandler,
	sseHandler *api.SSENotificationHandler,
) *APIRouter {
	return &APIRouter{
		handler:      handler,
		agentHandler: agentHandler,
		sseHandler:   sseHandler,
	}
}

// RegisterRoutes 註冊API路由，使用獨立的中間件
func (r *APIRouter) RegisterRoutes(
	router *gin.Engine,
	authMiddleware gin.HandlerFunc,
	apiKeyMiddleware gin.HandlerFunc,
	jwtMiddleware gin.HandlerFunc,
) {
	// API 路由群組 (需要認證)
	api := router.Group("/api/v1")
	if authMiddleware != nil {
		api.Use(authMiddleware)
	}

	// 商戶相關路由
	merchants := api.Group("/merchants")
	{
		merchants.GET("/:id", r.handler.GetMerchantByID)
		merchants.GET("/global/:global_id", r.handler.GetMerchantByGlobalID)
	}

	// 玩家相關路由
	players := api.Group("/players")
	{
		players.GET("/:id", r.handler.GetPlayerByID)
		players.GET("/global/:global_id", r.handler.GetPlayerByGlobalID)
		players.PUT("/:id/active", r.handler.UpdatePlayerLastActive)
		players.GET("/levels", r.handler.GetPlayerLevels)
		players.GET("/tags", r.handler.GetPlayerTags)
	}

	// 管理員相關路由
	managers := api.Group("/managers")
	{
		managers.GET("/:id", r.handler.GetManagerByID)
		managers.GET("/global/:global_id", r.handler.GetManagerByGlobalID)
	}

	// 管理端 - 訊息活動管理
	campaigns := api.Group("/message-campaigns")
	{
		campaigns.POST("", r.handler.CreateMessageCampaign)
		campaigns.GET("", r.handler.ListMessageCampaigns)
		campaigns.GET("/:global_id", r.handler.GetMessageCampaign)
		campaigns.PUT("/:global_id", r.handler.UpdateMessageCampaign)
		campaigns.DELETE("/:global_id", r.handler.DeleteMessageCampaign)

		// 自動設定 API
		campaigns.GET("/auto-settings", r.handler.GetMerchantAutoSettings)
		campaigns.POST("/auto-settings", r.handler.CreateOrUpdateMerchantAutoSettings)
		campaigns.PUT("/auto-settings", r.handler.CreateOrUpdateMerchantAutoSettings)
	}

	// 玩家端 - 訊息查看
	messages := api.Group("/messages")
	{
		messages.GET("/player/:global_player_id", r.handler.GetPlayerMessages)
		messages.PUT("/player/:global_player_id/:message_id/read", r.handler.MarkMessageAsRead)
	}

	// 系統自動推播
	notifications := api.Group("/notifications")
	{
		notifications.POST("/auto-send", r.handler.SendAutoNotification)
	}

	// ========== 代理相關路由 ==========

	// 代理端 - 站內信查看
	agents := api.Group("/agents")
	{
		agents.GET("/:global_agent_id/messages", r.agentHandler.GetAgentMessages)
		agents.PUT("/:global_agent_id/messages/:message_id/read", r.agentHandler.MarkMessageAsRead)
	}

	// 管理端 - 代理訊息活動管理
	agentCampaigns := api.Group("/agent-campaigns")
	{
		agentCampaigns.POST("", r.agentHandler.CreateAgentCampaign)
		agentCampaigns.GET("", r.agentHandler.GetAgentCampaigns)
		agentCampaigns.DELETE(
			"",
			r.agentHandler.BatchDeleteAgentCampaigns,
		) // 批量刪除（agent_message異步刪除）
		agentCampaigns.GET("/:id", r.agentHandler.GetAgentCampaign)
		agentCampaigns.PUT("/:id", r.agentHandler.UpdateAgentCampaign)
		agentCampaigns.DELETE("/:id", r.agentHandler.DeleteAgentCampaign) // 單個刪除（使用路徑參數）
	}

	// ========== SSE 實時推播路由 ==========

	// 管理端 - SSE 推播 API (使用 API Key 認證)
	if r.sseHandler != nil && apiKeyMiddleware != nil {
		adminSSE := router.Group("/api/v1/admin/notifications")
		adminSSE.Use(apiKeyMiddleware)
		{
			adminSSE.POST("/broadcast", r.sseHandler.BroadcastNotification)
			adminSSE.POST("/send", r.sseHandler.SendNotification)
		}
	}

	// 玩家端 - SSE 串流 API (使用 JWT 認證)
	if r.sseHandler != nil && jwtMiddleware != nil {
		playerSSE := router.Group("/api/v1/notifications")
		playerSSE.Use(jwtMiddleware)
		{
			playerSSE.GET("/stream", r.sseHandler.StreamNotifications)
		}
	}
}
