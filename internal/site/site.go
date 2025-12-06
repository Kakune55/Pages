package site

import "time"

// Site 站点数据结构
type Site struct {
	ID        string    `json:"id" toml:"id"`                 // 站点唯一标识（可用于目录名）
	Domain    string    `json:"domain" toml:"domain"`         // 绑定的域名
	RootDir   string    `json:"root_dir" toml:"root_dir"`     // 静态文件根目录
	Index     string    `json:"index" toml:"index"`           // 默认首页文件
	Enabled   bool      `json:"enabled" toml:"enabled"`       // 是否启用
	CreatedAt time.Time `json:"created_at" toml:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at" toml:"updated_at"` // 更新时间
}

// NewSite 创建新站点
func NewSite(id, domain, rootDir string) *Site {
	now := time.Now()
	return &Site{
		ID:        id,
		Domain:    domain,
		RootDir:   rootDir,
		Index:     "index.html",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Update 更新站点信息
func (s *Site) Update(domain, rootDir, index string) {
	if domain != "" {
		s.Domain = domain
	}
	if rootDir != "" {
		s.RootDir = rootDir
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
