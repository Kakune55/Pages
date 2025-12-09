package admin

import (
	"github.com/labstack/echo/v4"
)

// RegisterRoutes 注册管理路由
func (h *Handler) RegisterRoutes(g *echo.Group) {
	// 用户资源
	userGroup := g.Group("/users/:username")
	
	// 用户站点管理
	userGroup.GET("/sites", h.ListUserSites)
	userGroup.POST("/sites", h.CreateUserSite)
	userGroup.GET("/sites/:id", h.GetSite)
	userGroup.PUT("/sites/:id", h.UpdateSite)
	userGroup.DELETE("/sites/:id", h.DeleteSite)

	// 站点部署管理
	userGroup.POST("/sites/:id/deploy", h.DeploySite)
	userGroup.GET("/sites/:id/usage", h.GetSiteUsage)
	
	// 用户用量
	userGroup.GET("/usage", h.GetUserUsage)

	// 检查点管理
	userGroup.GET("/sites/:id/checkpoints", h.ListCheckpoints)
	userGroup.GET("/sites/:id/checkpoints/:checkpoint_id", h.GetCheckpoint)
	userGroup.DELETE("/sites/:id/checkpoints/:checkpoint_id", h.DeleteCheckpoint)
	userGroup.POST("/sites/:id/checkpoints/:checkpoint_id/checkout", h.CheckoutCheckpoint)

	// 系统管理
	systemGroup := g.Group("/system")
	systemGroup.POST("/reload", h.Reload)
	systemGroup.GET("/health", h.Health)
}
