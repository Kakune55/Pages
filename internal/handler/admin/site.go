package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"pages/internal/site"
)

// ListUserSites 列出指定用户的所有站点
func (h *Handler) ListUserSites(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "用户名不能为空",
		})
	}

	sites, err := h.siteManager.ListAllForUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取站点列表失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"sites": sites,
			"total": len(sites),
		},
	})
}

// CreateSiteRequest 创建站点请求
type CreateSiteRequest struct {
	ID     string `json:"id" validate:"required"`
	Domain string `json:"domain" validate:"required"`
	Index  string `json:"index"`
}

// CreateUserSite 为指定用户创建站点
func (h *Handler) CreateUserSite(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "用户名不能为空",
		})
	}

	var req CreateSiteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "请求参数错误",
		})
	}

	if req.ID == "" || req.Domain == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "id 和 domain 为必填字段",
		})
	}

	// 创建站点（路径自动生成）
	s := site.NewSiteForUser(req.ID, req.Domain, username)
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
func (h *Handler) GetSite(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")

	s, err := h.siteManager.GetFullSiteByIDForUser(username, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取站点失败: %v", err),
		})
	}
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
	Index   string `json:"index"`
	Enabled *bool  `json:"enabled"`
}

// UpdateSite 更新站点
func (h *Handler) UpdateSite(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")

	s, err := h.siteManager.GetFullSiteByIDForUser(username, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取站点失败: %v", err),
		})
	}
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

	// 更新字段（不允许修改ID、Username、RootDir）
	if req.Domain != "" {
		s.Domain = req.Domain
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
func (h *Handler) DeleteSite(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")

	if err := h.siteManager.RemoveForUser(username, id); err != nil {
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
