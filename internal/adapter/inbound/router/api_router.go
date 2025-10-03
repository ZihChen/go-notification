package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/adapter/inbound/handler/api"
)

// APIRouter 處理API路由註冊
type APIRouter struct {
	handler *api.HTTPHandler
}

// NewAPIRouter 創建API路由器
func NewAPIRouter(handler *api.HTTPHandler) *APIRouter {
	return &APIRouter{
		handler: handler,
	}
}

// RegisterRoutes 註冊API路由，使用獨立的中間件
func (r *APIRouter) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
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
}
