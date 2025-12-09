package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"pages/internal/analytics"
	"pages/internal/config"
	"pages/internal/logging"
	"pages/internal/server"
	"pages/internal/site"
)

const configPath = "config.toml"

func main() {
	fmt.Println(" ██████╗  █████╗  ██████╗ ███████╗███████╗")
	fmt.Println(" ██╔══██╗██╔══██╗██╔════╝ ██╔════╝██╔════╝")
	fmt.Println(" ██████╔╝███████║██║  ███╗█████╗  ███████╗")
	fmt.Println(" ██╔═══╝ ██╔══██║██║   ██║██╔══╝  ╚════██║")
	fmt.Println(" ██║     ██║  ██║╚██████╔╝███████╗███████║")
	fmt.Println(" ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝")
	// 加载配置
	cfg, created, err := config.LoadOrInit(configPath, true)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}
	if created {
		slog.Info("已生成默认配置文件", "path", configPath)
	}

	// 设置日志级别
	logging.SetLevelWithStr(cfg.Server.LogLevel)

	// 初始化站点管理器
	sm, err := initSites(cfg)
	if err != nil {
		fmt.Printf("站点初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化统计管理器
	analyticsDir := filepath.Join(cfg.Server.DataDir, "analytics")
	am := analytics.NewManager(analyticsDir)

	// 创建并启动服务器
	srv := server.New(cfg, sm, am)
	
	// 在 goroutine 中启动服务器
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务器异常退出", "error", err)
			os.Exit(1)
		}
	}()
	
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅停止服务器
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("服务器关闭失败", "error", err)
		os.Exit(1)
	}

	// 停止统计管理器
	slog.Info("正在保存统计数据...")
	am.StopAll()
	
	slog.Info("服务器已安全退出")
	slog.Info("Bye!")
}


// initSites 初始化站点管理器
func initSites(cfg *config.Config) (*site.ManagerLockFree, error) {
	// 创建存储
	store := site.NewFileStore(cfg.Server.DataDir)

	// 创建站点管理器
	sm := site.NewManagerLockFree(store)

	// 加载站点
	if err := sm.Load(); err != nil {
		return nil, fmt.Errorf("加载站点失败: %w", err)
	}

	// 如果没有站点，创建默认站点
	if sm.Count() == 0 {
		fmt.Println("未找到站点配置，创建默认站点...")
			if err := createDefaultSites(sm); err != nil {
				return nil, err
			}
	}

	// 初始化站点目录
	initializer := site.NewInitializer(cfg.Server.SitesDir)
	snapshots := sm.List()
	// 将 SiteSnapshot 转换为 Site 对象
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
	if err := initializer.InitializeSites(sites); err != nil {
		fmt.Printf("初始化站点目录失败: %v\n", err)
	}

	return sm, nil
}

// createDefaultSites 创建默认站点（支持多租户）
func createDefaultSites(sm *site.ManagerLockFree) error {
	defaultSites := []*site.Site{
		site.NewSite("default", "localhost"),
		site.NewSite("example", "example.localhost"),
	}

	for _, s := range defaultSites {
		if err := sm.Add(s); err != nil {
			return fmt.Errorf("添加默认站点失败: %w", err)
		}
	}

	fmt.Println("默认站点已创建")
	return nil
}
