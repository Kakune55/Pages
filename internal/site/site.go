package site

import (
	"fmt"
	"path/filepath"
	"time"
)

// Site 站点数据结构（支持多租户）
type Site struct {
	ID        string    `json:"id" toml:"id"`                   // 站点唯一标识（可用于目录名）
	Username  string    `json:"username" toml:"username"`       // 租户用户名（默认为"default"）
	Domain    string    `json:"domain" toml:"domain"`           // 绑定的域名
	Index     string    `json:"index" toml:"index"`             // 默认首页文件
	Enabled   bool      `json:"enabled" toml:"enabled"`         // 是否启用
	CreatedAt time.Time `json:"created_at" toml:"created_at"`   // 创建时间
	UpdatedAt time.Time `json:"updated_at" toml:"updated_at"`   // 更新时间
}

// NewSite 创建新站点（默认租户为"default"）
func NewSite(id, domain string) *Site {
	return NewSiteForUser(id, domain, "default")
}

// NewSiteForUser 为指定租户创建新站点
func NewSiteForUser(id, domain, username string) *Site {
	now := time.Now()
	if username == "" {
		username = "default"
	}
	return &Site{
		ID:        id,
		Username:  username,
		Domain:    domain,
		Index:     "index.html",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetRootDir 获取站点根目录（自动生成）
func (s *Site) GetRootDir(basePath string) string {
	return filepath.Join(basePath, s.Username, s.ID)
}

// GetRelativeRootDir 获取相对路径的根目录
func (s *Site) GetRelativeRootDir() string {
	return fmt.Sprintf("data/sites/%s/%s", s.Username, s.ID)
}

// Update 更新站点信息
func (s *Site) Update(domain, index string) {
	if domain != "" {
		s.Domain = domain
	}
	if index != "" {
		s.Index = index
	}
	s.UpdatedAt = time.Now()
}

// Enable 启用站点
func (s *Site) Enable() {
	s.Enabled = true
	s.UpdatedAt = time.Now()
}

// Disable 禁用站点
func (s *Site) Disable() {
	s.Enabled = false
	s.UpdatedAt = time.Now()
}
