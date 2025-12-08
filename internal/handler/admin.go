package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"pages/internal/handler/deploy"
	"pages/internal/site"
)

// AdminHandler 管理接口处理器
type AdminHandler struct {
	siteManager        *site.ManagerLockFree
	initializer        *site.Initializer
	checkpointManager  *deploy.CheckpointManager
}

// NewAdminHandler 创建管理接口处理器
func NewAdminHandler(sm *site.ManagerLockFree, init *site.Initializer, checkpointsDir string) *AdminHandler {
	return &AdminHandler{
		siteManager:       sm,
		initializer:       init,
		checkpointManager: deploy.NewCheckpointManager(checkpointsDir),
	}
}

// RegisterRoutes 注册管理路由
func (h *AdminHandler) RegisterRoutes(g *echo.Group) {
	// 站点管理
	g.GET("/sites", h.ListSites)
	g.POST("/sites", h.CreateSite)
	g.GET("/sites/:username/:id", h.GetSite)
	g.PUT("/sites/:username/:id", h.UpdateSite)
	g.DELETE("/sites/:username/:id", h.DeleteSite)
	g.POST("/sites/:username/:id/deploy", h.DeploySite)

	// 检查点管理
	g.GET("/sites/:username/:id/checkpoints", h.ListCheckpoints)
	g.GET("/sites/:username/:id/checkpoints/:checkpoint_id", h.GetCheckpoint)
	g.DELETE("/sites/:username/:id/checkpoints/:checkpoint_id", h.DeleteCheckpoint)
	g.POST("/sites/:username/:id/checkpoints/:checkpoint_id/checkout", h.CheckoutCheckpoint)

	// 热重载
	g.POST("/reload", h.Reload)

	// 健康检查
	g.GET("/health", h.Health)
}

// Response 通用响应结构
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
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
	ID       string `json:"id" validate:"required"`
	Username string `json:"username"` // 租户用户名（可选，默认为"default"）
	Domain   string `json:"domain" validate:"required"`
	Index    string `json:"index"`
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

	if req.ID == "" || req.Domain == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "id 和 domain 为必填字段",
		})
	}

	// 默认租户为"default"
	username := req.Username
	if username == "" {
		username = "default"
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
func (h *AdminHandler) GetSite(c echo.Context) error {
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
func (h *AdminHandler) UpdateSite(c echo.Context) error {
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
func (h *AdminHandler) DeleteSite(c echo.Context) error {
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

// DeploySite 上传压缩包并部署站点
func (h *AdminHandler) DeploySite(c echo.Context) error {
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

	sitesDirVal := c.Get("sitesDir")
	baseDir, ok := sitesDirVal.(string)
	if !ok || baseDir == "" {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "服务器配置缺失: sitesDir",
		})
	}
	rootDir := s.GetRootDir(baseDir)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "缺少上传文件字段 file",
		})
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "无法读取上传文件",
		})
	}
	defer src.Close()

	// 1. 保存上传文件到临时文件
	tmpFile, err := os.CreateTemp("", "deploy-archive-*")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "创建临时文件失败",
		})
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "保存上传文件失败",
		})
	}
	if err := tmpFile.Close(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "关闭临时文件失败",
		})
	}

	// 2. 创建临时解压目录
	tmpExtractDir, err := os.MkdirTemp("", "deploy-extract-*")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "创建临时解压目录失败",
		})
	}
	defer os.RemoveAll(tmpExtractDir)

	// 3. 根据文件类型解压到临时目录
	filename := strings.ToLower(fileHeader.Filename)
	switch {
	case strings.HasSuffix(filename, ".zip"):
		if err := deploy.ExtractZip(tmpPath, tmpExtractDir); err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: fmt.Sprintf("解压 zip 失败: %v", err),
			})
		}
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
		if err := deploy.ExtractTarGz(tmpPath, tmpExtractDir); err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: fmt.Sprintf("解压 tar.gz 失败: %v", err),
			})
		}
	case strings.HasSuffix(filename, ".tar"):
		if err := deploy.ExtractTar(tmpPath, tmpExtractDir); err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: fmt.Sprintf("解压 tar 失败: %v", err),
			})
		}
	default:
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "仅支持 zip、tar 或 tar.gz 压缩包",
		})
	}

	// 4. 检测并整理目录结构（展平单层嵌套）
	normalizedDir, err := deploy.NormalizeDirectory(tmpExtractDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("整理目录结构失败: %v", err),
		})
	}
	// 如果创建了新的临时目录，确保清理
	if normalizedDir != tmpExtractDir {
		defer os.RemoveAll(normalizedDir)
	}

	// 5. 创建检查点（如果站点目录已存在）
	var checkpoint *deploy.Checkpoint
	if _, err := os.Stat(rootDir); err == nil {
		// 站点目录存在，创建检查点
		checkpoint, err = h.checkpointManager.CreateCheckpoint(username, id, rootDir, fileHeader.Filename)
		if err != nil {
			// 检查点创建失败不中断部署，只记录错误
			c.Logger().Warnf("创建检查点失败 (继续部署): %v", err)
		}
	}

	// 6. 原子性替换站点目录
	if err := deploy.AtomicReplaceDirectory(rootDir, normalizedDir); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("部署失败: %v", err),
		})
	}

	result := map[string]any{
		"username": username,
		"id":       id,
	}
	if checkpoint != nil {
		result["checkpoint"] = checkpoint
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "站点已部署",
		Data:    result,
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
func (h *AdminHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"status":      "healthy",
			"sites_count": h.siteManager.Count(),
			"timestamp":   time.Now(),
		},
	})
}

// ListCheckpoints 列出站点的所有检查点
func (h *AdminHandler) ListCheckpoints(c echo.Context) error {
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
func (h *AdminHandler) GetCheckpoint(c echo.Context) error {
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
func (h *AdminHandler) DeleteCheckpoint(c echo.Context) error {
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
	})
}

// CheckoutCheckpoint 切换站点到指定检查点（仅切换，不创建新检查点）
func (h *AdminHandler) CheckoutCheckpoint(c echo.Context) error {
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
