package admin

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"pages/internal/analytics"
	"pages/internal/site"
)

type AnalyticsHandler struct {
	am *analytics.Manager
	sm *site.ManagerLockFree
}

func NewAnalyticsHandler(am *analytics.Manager, sm *site.ManagerLockFree) *AnalyticsHandler {
	return &AnalyticsHandler{
		am: am,
		sm: sm,
	}
}

func (h *AnalyticsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/users/:username/analytics", h.GetUserStats)
	g.GET("/users/:username/sites/:id/analytics", h.GetStats)
}

// GetUserStats 获取指定用户所有站点的今日统计
func (h *AnalyticsHandler) GetUserStats(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "参数错误",
		})
	}

	// 获取该用户所有站点（包括禁用的）
	sites, err := h.sm.ListAllForUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	result := make(map[string]analytics.DailyStats)
	for _, s := range sites {
		// 只获取今日统计
		stats, err := h.am.GetStats(username, s.ID, false)
		if err == nil {
			if ds, ok := stats.(analytics.DailyStats); ok {
				result[s.ID] = ds
			}
		}
	}

	return c.JSON(http.StatusOK, result)
}

// GetStats 获取站点统计数据
func (h *AnalyticsHandler) GetStats(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")
	full := c.QueryParam("scope") == "full"

	if username == "" || id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "参数错误",
		})
	}

	stats, err := h.am.GetStats(username, id, full)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, stats)
}
