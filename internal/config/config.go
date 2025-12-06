package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Config 服务器配置（不包含站点数据）
type Config struct {
	Server ServerConfig `toml:"server"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port      string `toml:"port"`
	LogLevel  string `toml:"log_level"`
	DataDir   string `toml:"data_dir"`   // 数据目录（存放站点配置等）
	SitesDir  string `toml:"sites_dir"`  // 静态站点文件根目录
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     "1323",
			LogLevel: "info",
			DataDir:  "./data",
			SitesDir: "./data/sites",
		},
	}
}

// LoadOrInit 从 TOML 加载配置，如果文件不存在则创建默认配置
func LoadOrInit(path string, envOverride bool) (*Config, bool, error) {
	created := false

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		// 首次启动：先用 ENV 覆盖默认，再写入文件
		applyEnvOverrides(cfg)
		if err := writeToml(path, cfg); err != nil {
			slog.Warn("写入配置文件失败，将仅使用内存配置", "path", path, "error", err)
			return cfg, true, nil
		}
		created = true
	}

	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, created, err
	}
	cfg := &Config{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, created, err
	}

	// 存在则用环境变量覆盖配置（不写回文件）
	if envOverride {
		applyEnvOverrides(cfg)
	}

	return cfg, created, nil
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	return writeToml(path, c)
}

func writeToml[T any](path string, cfg T) error {
	b, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	// 确保目录存在
	if dir := dirOf(path); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(path, b, 0644)
}

func dirOf(path string) string {
	i := strings.LastIndexAny(path, "/\\")
	if i < 0 {
		return ""
	}
	return path[:i]
}

// applyEnvOverrides 读取环境变量并覆盖配置 不回写文件
func applyEnvOverrides(cfg *Config) {
	// Server
	if v := os.Getenv("PAGES_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("PAGES_LOG_LEVEL"); v != "" {
		cfg.Server.LogLevel = v
	}
	if v := os.Getenv("PAGES_DATA_DIR"); v != "" {
		cfg.Server.DataDir = v
	}
	if v := os.Getenv("PAGES_SITES_DIR"); v != "" {
		cfg.Server.SitesDir = v
	}
}

