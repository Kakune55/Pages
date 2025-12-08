package admin

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListCheckpoints 列出站点的所有检查点
func (h *Handler) ListCheckpoints(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")

	// 验证站点是否存在
	s := h.siteManager.GetByIDForUser(username, id)
	if s == nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "站点不存在",
		})
	}

	metadata, err := h.checkpointManager.ListCheckpoints(username, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取检查点列表失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"current":     metadata.Current,
			"checkpoints": metadata.Checkpoints,
			"total":       len(metadata.Checkpoints),
		},
	})
}

// GetCheckpoint 获取指定检查点信息
func (h *Handler) GetCheckpoint(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")
	checkpointID := c.Param("checkpoint_id")

	// 验证站点是否存在
	s := h.siteManager.GetByIDForUser(username, id)
	if s == nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "站点不存在",
		})
	}

	checkpoint, err := h.checkpointManager.GetCheckpoint(username, id, checkpointID)
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: fmt.Sprintf("检查点不存在: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    checkpoint,
	})
}

// DeleteCheckpoint 删除检查点
func (h *Handler) DeleteCheckpoint(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")
	checkpointID := c.Param("checkpoint_id")

	// 验证站点是否存在
	s := h.siteManager.GetByIDForUser(username, id)
	if s == nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "站点不存在",
		})
	}

	if err := h.checkpointManager.DeleteCheckpoint(username, id, checkpointID); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("删除检查点失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "检查点已删除",
		Data: map[string]any{
			"username":      username,
			"id":            id,
			"checkpoint_id": checkpointID,
		},
	})
}

// CheckoutCheckpoint 切换站点到指定检查点（仅切换，不创建新检查点）
func (h *Handler) CheckoutCheckpoint(c echo.Context) error {
	username := c.Param("username")
	id := c.Param("id")
	checkpointID := c.Param("checkpoint_id")

	// 验证站点是否存在
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

	sitesDirVal := c.Get("sitesDir")
	baseDir, ok := sitesDirVal.(string)
	if !ok || baseDir == "" {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "服务器配置缺失: sitesDir",
		})
	}
	rootDir := s.GetRootDir(baseDir)

	// 切换到指定检查点（不创建新检查点，仅切换 current 指针）
	if err := h.checkpointManager.CheckoutCheckpoint(username, id, checkpointID, rootDir); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("切换检查点失败: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "站点已切换到检查点",
		Data: map[string]any{
			"username":      username,
			"id":            id,
			"checkpoint_id": checkpointID,
		},
	})
}
