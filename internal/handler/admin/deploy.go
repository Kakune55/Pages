package admin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	"pages/internal/handler/deploy"
)

// DeploySite 上传压缩包并部署站点
func (h *Handler) DeploySite(c echo.Context) error {
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

	// 7. 重算存储使用量 (如果没有创建检查点,需要手动触发)
	if checkpoint == nil {
		_ = h.checkpointManager.StorageRecount(username, id, rootDir)
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

// GetSiteUsage 获取站点使用情况(磁盘空间等) - 从元数据缓存读取
func (h *Handler) GetSiteUsage(c echo.Context) error {
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

	// 从元数据中读取缓存的使用量信息
	usage, err := h.checkpointManager.GetStorageUsage(username, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: fmt.Sprintf("获取站点使用情况失败: %v", err),
		})
	}

	// 如果缓存为空(首次查询或元数据不存在),触发一次重算
	if usage.TotalSize == 0 && usage.DeployedSize == 0 {
		sitesDirVal := c.Get("sitesDir")
		baseDir, ok := sitesDirVal.(string)
		if !ok || baseDir == "" {
			return c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "服务器配置缺失: sitesDir",
			})
		}

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

		rootDir := s.GetRootDir(baseDir)

		// 触发重算
		if err := h.checkpointManager.StorageRecount(username, id, rootDir); err != nil {
			return c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: fmt.Sprintf("计算站点使用情况失败: %v", err),
			})
		}

		// 重新读取
		usage, err = h.checkpointManager.GetStorageUsage(username, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: fmt.Sprintf("获取站点使用情况失败: %v", err),
			})
		}
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    usage,
	})
}
