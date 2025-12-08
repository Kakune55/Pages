package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"pages/internal/site"
)

// Reload 热重载站点配置
func (h *Handler) Reload(c echo.Context) error {
	if err := h.siteManager.Reload(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("重载失败: %v", err),
		})
	}

	sites := h.siteManager.List()

	// 重新初始化站点目录
	if h.initializer != nil {
		snapshots := h.siteManager.List()
		// 将 SiteSnapshot 转换为 Site 对象以便初始化
		sites := make([]*site.Site, len(snapshots))
		for i, snap := range snapshots {
			sites[i] = &site.Site{
				ID:       snap.ID,
				Username: snap.Username,
				Domain:   snap.Domain,
				Index:    snap.Index,
				Enabled:  snap.Enabled,
			}
		}
		_ = h.initializer.InitializeSites(sites)
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: fmt.Sprintf("重载成功，当前 %d 个站点已生效", len(sites)),
		Data: map[string]any{
			"sites_count": len(sites),
			"reloaded_at": time.Now(),
		},
	})
}

// Health 健康检查
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"status":      "healthy",
			"sites_count": h.siteManager.Count(),
			"timestamp":   time.Now(),
		},
	})
}
