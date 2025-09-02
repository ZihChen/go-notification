package router

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

// PprofRouter 處理pprof性能分析路由註冊
type PprofRouter struct{}

// NewPprofRouter 創建pprof路由器
func NewPprofRouter() *PprofRouter {
	return &PprofRouter{}
}

// RegisterRoutes 註冊pprof路由，不需要認證中間件，通常用於開發環境
func (r *PprofRouter) RegisterRoutes(router *gin.Engine) {
	// pprof 路由群組 (不需要認證，但通常僅在開發環境啟用)
	pprofGroup := router.Group("/debug/pprof")
	{
		pprofGroup.GET("/", gin.WrapF(pprof.Index))
		pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
		pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
		pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
		pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}
