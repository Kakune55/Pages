package site

import (
	"strings"
	"sync"
)

// Manager 站点管理器（运行时）
type Manager struct {
	sites map[string]*Site // domain -> site
	store Store
	mu    sync.RWMutex
}

// NewManager 创建站点管理器
func NewManager(store Store) *Manager {
	return &Manager{
		sites: make(map[string]*Site),
		store: store,
	}
}

// Load 从存储加载站点
func (m *Manager) Load() error {
	sites, err := m.store.Load()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sites = make(map[string]*Site)
	for _, site := range sites {
		if site.Enabled {
			m.sites[site.Domain] = site
		}
	}

	return nil
}

// Add 添加站点
func (m *Manager) Add(site *Site) error {
	// 先保存到存储
	if err := m.store.Add(site); err != nil {
		return err
	}

	// 再添加到内存
	if site.Enabled {
		m.mu.Lock()
		m.sites[site.Domain] = site
		m.mu.Unlock()
	}

	return nil
}

// Remove 移除站点
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	// 找到对应的域名
	var domain string
	for d, site := range m.sites {
		if site.ID == id {
			domain = d
			break
		}
	}
	if domain != "" {
		delete(m.sites, domain)
	}
	m.mu.Unlock()

	return m.store.Remove(id)
}

// Update 更新站点
func (m *Manager) Update(site *Site) error {
	if err := m.store.Update(site); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 移除旧的域名映射
	for domain, s := range m.sites {
		if s.ID == site.ID {
			delete(m.sites, domain)
			break
		}
	}

	// 如果启用则添加新映射
	if site.Enabled {
		m.sites[site.Domain] = site
	}

	return nil
}

// Get 根据域名获取站点
func (m *Manager) Get(domain string) *Site {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 移除端口号
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	return m.sites[domain]
}

// GetByID 根据 ID 获取站点
func (m *Manager) GetByID(id string) *Site {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, site := range m.sites {
		if site.ID == id {
			return site
		}
	}
	return nil
}

// List 列出所有启用的站点
func (m *Manager) List() []*Site {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sites := make([]*Site, 0, len(m.sites))
	for _, site := range m.sites {
		sites = append(sites, site)
	}
	return sites
}

// ListAll 列出所有站点（包括禁用的）
func (m *Manager) ListAll() ([]*Site, error) {
	return m.store.Load()
}

// Count 返回启用的站点数量
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sites)
}

// Exists 检查域名是否已存在
func (m *Manager) Exists(domain string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.sites[domain]
	return exists
}

// Reload 重新加载站点配置
func (m *Manager) Reload() error {
	return m.Load()
}
