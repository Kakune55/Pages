package main

import (
	"fmt"
	"log/slog"
	"os"

	"pages/internal/config"
	"pages/internal/logging"
	"pages/internal/server"
	"pages/internal/site"
)

const configPath = "config.toml"

func main() {
		// 加载配置
	cfg, created, err := config.LoadOrInit("config.toml", true)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}
	if created {
		slog.Info("已生成默认配置文件", "path", "config.toml")
	}

	// 设置日志级别
	logging.SetLevelWithStr(cfg.Server.LogLevel)

	// 初始化站点管理器
	sm, err := initSites(cfg)
	if err != nil {
		fmt.Printf("❌ 站点初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 创建并启动服务器
	srv := server.New(cfg, sm)
	if err := srv.Start(); err != nil {
		fmt.Printf("❌ 服务器启动失败: %v\n", err)
		os.Exit(1)
	}
}

// initConfig 初始化配置
func initConfig() (*config.Config, bool, error) {
	cfg, created, err := config.LoadOrInit(configPath, true)
	if err != nil {
		return nil, false, fmt.Errorf("加载配置失败: %w", err)
	}
	return cfg, created, nil
}

// initSites 初始化站点管理器
func initSites(cfg *config.Config) (*site.Manager, error) {
	// 创建存储
	store := site.NewFileStore(cfg.Server.DataDir)

	// 创建站点管理器
	sm := site.NewManager(store)

	// 加载站点
	if err := sm.Load(); err != nil {
		return nil, fmt.Errorf("加载站点失败: %w", err)
	}

	// 如果没有站点，创建默认站点
	if sm.Count() == 0 {
		fmt.Println("📝 未找到站点配置，创建默认站点...")
		if err := createDefaultSites(sm, cfg.Server.SitesDir); err != nil {
			return nil, err
		}
	}

	// 初始化站点目录
	initializer := site.NewInitializer(cfg.Server.SitesDir)
	if err := initializer.InitializeSites(sm.List()); err != nil {
		fmt.Printf("⚠️ 初始化站点目录失败: %v\n", err)
	}

	return sm, nil
}

// createDefaultSites 创建默认站点（支持多租户）
func createDefaultSites(sm *site.Manager, sitesDir string) error {
	defaultSites := []*site.Site{
		site.NewSite("default", "localhost"),
		site.NewSite("example", "example.localhost"),
	}

	for _, s := range defaultSites {
		if err := sm.Add(s); err != nil {
			return fmt.Errorf("添加默认站点失败: %w", err)
		}
	}

	fmt.Println("✅ 默认站点已创建")
	return nil
}
