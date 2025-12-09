package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"pages/internal/site"
)

// StaticFileServer 静态文件服务中间件
func StaticFileServer(sm *site.ManagerLockFree) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			host := c.Request().Host
			snap := sm.Get(host)

			if snap == nil {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error":   "站点未找到",
					"message": fmt.Sprintf("域名 %s 未绑定任何站点", host),
				})
			}

			// 检查站点是否已启用
			if !snap.Enabled {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"error":   "站点已禁用",
					"message": fmt.Sprintf("域名 %s 对应的站点已被管理员禁用", host),
				})
			}

			// 获取请求路径
			reqPath := c.Request().URL.Path
			if reqPath == "/" {
				reqPath = "/" + snap.Index
			}
			if strings.HasPrefix(reqPath, "/_api") || strings.HasPrefix(reqPath, "/_admin") {
				return next(c)
			}

			// 构建文件路径
			sitesDirVal := c.Get("sitesDir")
			baseDir, ok := sitesDirVal.(string)
			if !ok || baseDir == "" {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "服务器配置缺失: sitesDir",
				})
			}
			rootDir := filepath.Join(baseDir, snap.Username, snap.ID)
			filePath := filepath.Join(rootDir, reqPath)

			// 安全检查：防止路径遍历攻击
			if !isPathSafe(rootDir, filePath) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "禁止访问",
				})
			}

			// 检查文件是否存在
			info, err := os.Stat(filePath)
			if os.IsNotExist(err) {
				return handleNotFound(c, rootDir, reqPath)
			}

			// 如果是目录，尝试返回 index.html
			if info.IsDir() {
				return handleDirectory(c, filePath, snap.Index)
			}

			return c.File(filePath)
		}
	}
}

// isPathSafe 检查路径是否安全（防止路径遍历攻击）
func isPathSafe(rootDir, filePath string) bool {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return false
	}

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absFile, absRoot)
}

// handleNotFound 处理文件未找到的情况
func handleNotFound(c echo.Context, rootDir, reqPath string) error {
	// 尝试返回 404.html
	notFoundPath := filepath.Join(rootDir, "404.html")
	if _, err := os.Stat(notFoundPath); err == nil {
		return c.File(notFoundPath)
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "文件未找到",
		"path":  reqPath,
	})
}

// handleDirectory 处理目录请求
func handleDirectory(c echo.Context, dirPath, indexFile string) error {
	indexPath := filepath.Join(dirPath, indexFile)
	if _, err := os.Stat(indexPath); err == nil {
		return c.File(indexPath)
	}

	return c.JSON(http.StatusForbidden, map[string]string{
		"error": "目录访问被禁止",
	})
}
