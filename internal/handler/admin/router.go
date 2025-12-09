package admin

import (
	"github.com/labstack/echo/v4"
)

// RegisterRoutes 注册管理路由
func (h *Handler) RegisterRoutes(g *echo.Group) {
	siteGroup := g.Group("/sites")
	
	// 站点管理
	siteGroup.GET("", h.ListSites)
	siteGroup.POST("", h.CreateSite)
	siteGroup.GET("/:username/:id", h.GetSite)
	siteGroup.PUT("/:username/:id", h.UpdateSite)
	siteGroup.DELETE("/:username/:id", h.DeleteSite)

	// 部署管理
	siteGroup.POST("/:username/:id/deploy", h.DeploySite)
	siteGroup.GET("/:username/:id/usage", h.GetSiteUsage)
	
	// 用户管理
	g.GET("/users/:username/usage", h.GetUserUsage)

	// 检查点管理
	siteGroup.GET("/:username/:id/checkpoints", h.ListCheckpoints)
	siteGroup.GET("/:username/:id/checkpoints/:checkpoint_id", h.GetCheckpoint)
	siteGroup.DELETE("/:username/:id/checkpoints/:checkpoint_id", h.DeleteCheckpoint)
	siteGroup.POST("/:username/:id/checkpoints/:checkpoint_id/checkout", h.CheckoutCheckpoint)

	// 系统管理
	siteGroup.POST("/reload", h.Reload)
	g.GET("/health", h.Health)
}
