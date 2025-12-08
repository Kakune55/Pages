package site

import (
	"strings"
	"sync"
	"sync/atomic"
)

// ManagerLockFree 站点管理器
// 原子化
type ManagerLockFree struct {
	sites atomic.Value // 存储 map[string]*SiteSnapshot
	store Store
	mu    sync.Mutex // 仅用于写操作
}

// SiteSnapshot 站点快照
// 用于无锁读取，避免每次请求都加锁
type SiteSnapshot struct {
	ID       string
	Username string
	Domain   string
	Index    string
	Enabled  bool
	RootDir  string
}

// NewManagerLockFree 创建无锁站点管理器
func NewManagerLockFree(store Store) *ManagerLockFree {
	m := &ManagerLockFree{
		store: store,
	}
	m.sites.Store(make(map[string]*SiteSnapshot))
	return m
}

// Load 从存储加载站点
func (m *ManagerLockFree) Load() error {
	sites, err := m.store.Load()
	if err != nil {
		return err
	}

	newSites := make(map[string]*SiteSnapshot)
	for _, site := range sites {
		if site.Enabled {
			newSites[site.Domain] = &SiteSnapshot{
				ID:       site.ID,
				Username: site.Username,
				Domain:   site.Domain,
				Index:    site.Index,
				Enabled:  site.Enabled,
				RootDir:  site.GetRelativeRootDir(),
			}
		}
	}

	m.sites.Store(newSites)
	return nil
}

// Add 添加站点
func (m *ManagerLockFree) Add(site *Site) error {
	if err := m.store.Add(site); err != nil {
		return err
	}

	if site.Enabled {
		m.mu.Lock()
		oldSites := m.sites.Load().(map[string]*SiteSnapshot)
		newSites := m.copyMap(oldSites)
		newSites[site.Domain] = &SiteSnapshot{
			ID:       site.ID,
			Username: site.Username,
			Domain:   site.Domain,
			Index:    site.Index,
			Enabled:  site.Enabled,
			RootDir:  site.GetRelativeRootDir(),
		}
		m.sites.Store(newSites)
		m.mu.Unlock()
	}

	return nil
}

// Remove 移除站点
func (m *ManagerLockFree) Remove(id string) error {
	m.mu.Lock()
	oldSites := m.sites.Load().(map[string]*SiteSnapshot)
	newSites := make(map[string]*SiteSnapshot)
	for domain, snap := range oldSites {
		if snap.ID != id {
			newSites[domain] = snap
		}
	}
	m.sites.Store(newSites)
	m.mu.Unlock()

	return m.store.Remove(id)
}

// Update 更新站点
func (m *ManagerLockFree) Update(site *Site) error {
	if err := m.store.Update(site); err != nil {
		return err
	}

	m.mu.Lock()
	oldSites := m.sites.Load().(map[string]*SiteSnapshot)
	newSites := make(map[string]*SiteSnapshot)
	
	// 复制并移除旧域名
	for domain, snap := range oldSites {
		if snap.ID != site.ID || snap.Username != site.Username {
			newSites[domain] = snap
		}
	}
	
	// 添加新映射
	if site.Enabled {
		newSites[site.Domain] = &SiteSnapshot{
			ID:       site.ID,
			Username: site.Username,
			Domain:   site.Domain,
			Index:    site.Index,
			Enabled:  site.Enabled,
			RootDir:  site.GetRelativeRootDir(),
		}
	}
	
	m.sites.Store(newSites)
	m.mu.Unlock()

	return nil
}

// Get 根据域名获取站点快照
// 这是最高频的操作，完全无锁，性能最优
func (m *ManagerLockFree) Get(domain string) *SiteSnapshot {
	// 移除端口号
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	return sites[domain]
}

// GetByID 根据 ID 获取站点快照
func (m *ManagerLockFree) GetByID(id string) *SiteSnapshot {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	for _, snap := range sites {
		if snap.ID == id {
			return snap
		}
	}
	return nil
}

// GetByIDForUser 根据租户和 ID 获取站点快照
func (m *ManagerLockFree) GetByIDForUser(username, id string) *SiteSnapshot {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	for _, snap := range sites {
		if snap.ID == id && snap.Username == username {
			return snap
		}
	}
	return nil
}

// List 列出所有启用的站点快照
func (m *ManagerLockFree) List() []*SiteSnapshot {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	result := make([]*SiteSnapshot, 0, len(sites))
	for _, snap := range sites {
		result = append(result, snap)
	}
	return result
}

// ListForUser 列出指定租户的所有启用的站点快照
func (m *ManagerLockFree) ListForUser(username string) []*SiteSnapshot {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	result := make([]*SiteSnapshot, 0)
	for _, snap := range sites {
		if snap.Username == username {
			result = append(result, snap)
		}
	}
	return result
}

// ListAll 列出所有站点（包括禁用的）
func (m *ManagerLockFree) ListAll() ([]*Site, error) {
	return m.store.Load()
}

// ListAllForUser 列出指定租户的所有站点（包括禁用的）
func (m *ManagerLockFree) ListAllForUser(username string) ([]*Site, error) {
	return m.store.LoadForUser(username)
}

// Count 返回启用的站点数量
func (m *ManagerLockFree) Count() int {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	return len(sites)
}

// CountForUser 返回指定租户启用的站点数量
func (m *ManagerLockFree) CountForUser(username string) int {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	count := 0
	for _, snap := range sites {
		if snap.Username == username {
			count++
		}
	}
	return count
}

// Exists 检查域名是否已存在
func (m *ManagerLockFree) Exists(domain string) bool {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	_, exists := sites[domain]
	return exists
}

// ExistsForUser 检查租户内的站点 ID 是否已存在
func (m *ManagerLockFree) ExistsForUser(username, id string) bool {
	sites := m.sites.Load().(map[string]*SiteSnapshot)
	for _, snap := range sites {
		if snap.Username == username && snap.ID == id {
			return true
		}
	}
	return false
}

// Reload 重新加载站点配置
func (m *ManagerLockFree) Reload() error {
	return m.Load()
}

// copyMap 辅助函数：复制 map
func (m *ManagerLockFree) copyMap(src map[string]*SiteSnapshot) map[string]*SiteSnapshot {
	dst := make(map[string]*SiteSnapshot, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
