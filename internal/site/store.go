package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store 站点存储接口
type Store interface {
	Load() ([]*Site, error)
	LoadForUser(username string) ([]*Site, error)
	Save(sites []*Site) error
	Add(site *Site) error
	Remove(id string) error
	RemoveForUser(username, id string) error
	Update(site *Site) error
}

// FileStore 基于文件的站点存储
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileStore 创建文件存储
func NewFileStore(dataDir string) *FileStore {
	return &FileStore{
		path: filepath.Join(dataDir, "sites.json"),
	}
}

// Load 从文件加载站点列表
func (s *FileStore) Load() ([]*Site, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 文件不存在时返回空列表
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return []*Site{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("读取站点文件失败: %w", err)
	}

	var sites []*Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return nil, fmt.Errorf("解析站点文件失败: %w", err)
	}

	return sites, nil
}

// LoadForUser 加载指定租户的站点
func (s *FileStore) LoadForUser(username string) ([]*Site, error) {
	sites, err := s.Load()
	if err != nil {
		return nil, err
	}

	userSites := make([]*Site, 0)
	for _, site := range sites {
		if site.Username == username {
			userSites = append(userSites, site)
		}
	}
	return userSites, nil
}

// Save 保存站点列表到文件
func (s *FileStore) Save(sites []*Site) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveInternal(sites)
}

// saveInternal 内部保存方法（不加锁）
func (s *FileStore) saveInternal(sites []*Site) error {
	// 确保目录存在
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := json.MarshalIndent(sites, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化站点失败: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("写入站点文件失败: %w", err)
	}

	return nil
}

// Add 添加站点
func (s *FileStore) Add(site *Site) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sites, err := s.loadInternal()
	if err != nil {
		return err
	}

	// 检查同租户的 ID 是否已存在
	for _, existing := range sites {
		if existing.Username == site.Username && existing.ID == site.ID {
			return fmt.Errorf("站点 ID %s 在租户 %s 中已存在", site.ID, site.Username)
		}
		if existing.Domain == site.Domain {
			return fmt.Errorf("域名 %s 已被绑定", site.Domain)
		}
	}

	sites = append(sites, site)
	return s.saveInternal(sites)
}

// Remove 移除站点
func (s *FileStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sites, err := s.loadInternal()
	if err != nil {
		return err
	}

	found := false
	newSites := make([]*Site, 0, len(sites))
	for _, site := range sites {
		if site.ID != id {
			newSites = append(newSites, site)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("站点 %s 不存在", id)
	}

	return s.saveInternal(newSites)
}

// RemoveForUser 移除指定租户的站点
func (s *FileStore) RemoveForUser(username, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sites, err := s.loadInternal()
	if err != nil {
		return err
	}

	found := false
	newSites := make([]*Site, 0, len(sites))
	for _, site := range sites {
		if site.Username == username && site.ID == id {
			found = true
		} else {
			newSites = append(newSites, site)
		}
	}

	if !found {
		return fmt.Errorf("站点 %s 不在租户 %s 中", id, username)
	}

	return s.saveInternal(newSites)
}

// Update 更新站点
func (s *FileStore) Update(site *Site) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sites, err := s.loadInternal()
	if err != nil {
		return err
	}

	found := false
	for i, existing := range sites {
		if existing.ID == site.ID && existing.Username == site.Username {
			sites[i] = site
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("站点 %s 不在租户 %s 中", site.ID, site.Username)
	}

	return s.saveInternal(sites)
}

// loadInternal 内部加载方法（不加锁）
func (s *FileStore) loadInternal() ([]*Site, error) {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return []*Site{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("读取站点文件失败: %w", err)
	}

	var sites []*Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return nil, fmt.Errorf("解析站点文件失败: %w", err)
	}

	return sites, nil
}

// GetPath 返回存储文件路径
func (s *FileStore) GetPath() string {
	return s.path
}
