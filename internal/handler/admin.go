package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"pages/internal/site"
)

// AdminHandler 管理接口处理器
type AdminHandler struct {
	siteManager *site.Manager
	initializer *site.Initializer
}

// NewAdminHandler 创建管理接口处理器
func NewAdminHandler(sm *site.Manager, init *site.Initializer) *AdminHandler {
	return &AdminHandler{
		siteManager: sm,
		initializer: init,
	}
}

// RegisterRoutes 注册管理路由
func (h *AdminHandler) RegisterRoutes(g *echo.Group) {
	// 站点管理
	g.GET("/sites", h.ListSites)
	g.POST("/sites", h.CreateSite)
	g.GET("/sites/:id", h.GetSite)
	g.PUT("/sites/:id", h.UpdateSite)
	g.DELETE("/sites/:id", h.DeleteSite)

	// 热重载
	g.POST("/reload", h.Reload)

	// 健康检查
	g.GET("/health", h.Health)
}

// Response 通用响应结构
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ListSites 列出所有站点
func (h *AdminHandler) ListSites(c echo.Context) error {
	sites, err := h.siteManager.ListAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取站点列表失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"sites": sites,
			"total": len(sites),
		},
	})
}

// CreateSiteRequest 创建站点请求
type CreateSiteRequest struct {
	ID      string `json:"id" validate:"required"`
	Domain  string `json:"domain" validate:"required"`
	RootDir string `json:"root_dir" validate:"required"`
	Index   string `json:"index"`
}

// CreateSite 创建站点
func (h *AdminHandler) CreateSite(c echo.Context) error {
	var req CreateSiteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "请求参数错误",
		})
	}

	if req.ID == "" || req.Domain == "" || req.RootDir == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "id, domain, root_dir 为必填字段",
		})
	}

	// 创建站点
	s := site.NewSite(req.ID, req.Domain, req.RootDir)
	if req.Index != "" {
		s.Index = req.Index
	}

	if err := h.siteManager.Add(s); err != nil {
		return c.JSON(http.StatusConflict, Response{
			Success: false,
			Message: fmt.Sprintf("创建站点失败: %v", err),
		})
	}

	// 初始化站点目录
	if h.initializer != nil {
		_ = h.initializer.InitializeSites([]*site.Site{s})
	}

	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: "站点创建成功",
		Data:    s,
	})
}

// GetSite 获取单个站点
func (h *AdminHandler) GetSite(c echo.Context) error {
	id := c.Param("id")

	s := h.siteManager.GetByID(id)
	if s == nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "站点不存在",
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    s,
	})
}

// UpdateSiteRequest 更新站点请求
type UpdateSiteRequest struct {
	Domain  string `json:"domain"`
	RootDir string `json:"root_dir"`
	Index   string `json:"index"`
	Enabled *bool  `json:"enabled"`
}

// UpdateSite 更新站点
func (h *AdminHandler) UpdateSite(c echo.Context) error {
	id := c.Param("id")

	s := h.siteManager.GetByID(id)
	if s == nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "站点不存在",
		})
	}

	var req UpdateSiteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "请求参数错误",
		})
	}

	// 更新字段
	if req.Domain != "" {
		s.Domain = req.Domain
	}
	if req.RootDir != "" {
		s.RootDir = req.RootDir
	}
	if req.Index != "" {
		s.Index = req.Index
	}
	if req.Enabled != nil {
		s.Enabled = *req.Enabled
	}
	s.UpdatedAt = time.Now()

	if err := h.siteManager.Update(s); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("更新站点失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "站点更新成功",
		Data:    s,
	})
}

// DeleteSite 删除站点
func (h *AdminHandler) DeleteSite(c echo.Context) error {
	id := c.Param("id")

	if err := h.siteManager.Remove(id); err != nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: fmt.Sprintf("删除站点失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "站点删除成功",
	})
}

// Reload 热重载站点配置
func (h *AdminHandler) Reload(c echo.Context) error {
	if err := h.siteManager.Reload(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("重载失败: %v", err),
		})
	}

	sites := h.siteManager.List()

	// 重新初始化站点目录
	if h.initializer != nil {
		_ = h.initializer.InitializeSites(sites)
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: fmt.Sprintf("重载成功，当前 %d 个站点已生效", len(sites)),
		Data: map[string]interface{}{
			"sites_count": len(sites),
			"reloaded_at": time.Now(),
		},
	})
}

// Health 健康检查
func (h *AdminHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"status":      "healthy",
			"sites_count": h.siteManager.Count(),
			"timestamp":   time.Now(),
		},
	})
}
