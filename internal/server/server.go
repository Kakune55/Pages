package server

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"pages/internal/config"
	"pages/internal/handler"
	"pages/internal/middleware"
	"pages/internal/site"
)

// Server 应用服务器
type Server struct {
	echo        *echo.Echo
	config      *config.Config
	siteManager *site.ManagerLockFree
	initializer *site.Initializer
}

// New 创建新的服务器实例
func New(cfg *config.Config, sm *site.ManagerLockFree) *Server {
	e := echo.New()
	e.HideBanner = true

	s := &Server{
		echo:        e,
		config:      cfg,
		siteManager: sm,
		initializer: site.NewInitializer(cfg.Server.SitesDir),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware 设置中间件
func (s *Server) setupMiddleware() {
	// 日志中间件
	s.echo.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // 将错误转发给全局错误处理程序，以便其决定适当的响应状态码
		LogValuesFunc: func(c echo.Context, v echomw.RequestLoggerValues) error {
			if v.Error == nil {
				slog.LogAttrs(context.Background(), slog.LevelInfo, "REQ",
					slog.Int("status", v.Status),
					slog.String("uri", v.URI),
				)
			} else {
				slog.LogAttrs(context.Background(), slog.LevelError, "REQ_ERR",
					slog.Int("status", v.Status),
					slog.String("uri", v.URI),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))

	// 恢复中间件
	s.echo.Use(echomw.Recover())

	// CORS 中间件
	s.echo.Use(echomw.CORS())

	// 设置站点目录到context
	s.echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("sitesDir", s.config.Server.SitesDir)
			return next(c)
		}
	})
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 管理 API（在静态文件中间件之前注册，优先级更高）
	adminGroup := s.echo.Group("/_api")
	adminGroup.Use(echomw.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		adminUser := s.config.Server.AdminUser
		adminPass := s.config.Server.AdminPass
		return username == adminUser && password == adminPass, nil
	}))
	
	// 检查点存储在站点目录的父级 checkpoints 目录
	checkpointsDir := s.config.Server.SitesDir + "-checkpoints"
	adminHandler := handler.NewAdminHandler(s.siteManager, s.initializer, checkpointsDir)
	adminHandler.RegisterRoutes(adminGroup)

	// 静态文件服务（作为最后的中间件，处理所有其他请求）
	s.echo.Use(middleware.StaticFileServer(s.siteManager))
}

// Start 启动服务器
func (s *Server) Start() error {
	s.printStartupInfo()
	return s.echo.Start(":" + s.config.Server.Port)
}

// Shutdown 优雅停止服务器
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("正在优雅关闭服务器...")
	
	// 关闭 HTTP 服务器
	if err := s.echo.Shutdown(ctx); err != nil {
		slog.Error("HTTP服务器关闭失败", "error", err)
		return err
	}
	
	slog.Info("服务器已关闭")
	return nil
}

// printStartupInfo 打印启动信息
func (s *Server) printStartupInfo() {
	slog.Info("服务器启动",
		slog.String("port", s.config.Server.Port),
		slog.String("sites_dir", s.config.Server.SitesDir),
	)
	slog.Info("已加载Pages站点数量",
		slog.Int("count", len(s.siteManager.List())),
	)
	slog.Info("管理API路径",
		slog.String("url", "/_api"),
	)
}

// Echo 返回 Echo 实例（用于扩展路由等）
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// SiteManager 返回站点管理器
func (s *Server) SiteManager() *site.ManagerLockFree {
	return s.siteManager
}

// Config 返回配置
func (s *Server) Config() *config.Config {
	return s.config
}
