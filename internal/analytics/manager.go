package analytics

import (
	"path/filepath"
	"sync"
)

// Manager 统计管理器
type Manager struct {
	baseDir string // 统计数据根目录
	workers map[string]*SiteWorker
	mu      sync.RWMutex
}

// NewManager 创建管理器
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
		workers: make(map[string]*SiteWorker),
	}
}

// GetWorker 获取或创建站点的 Worker
func (m *Manager) GetWorker(username, siteID string) *SiteWorker {
	key := username + ":" + siteID
	m.mu.RLock()
	w, ok := m.workers[key]
	m.mu.RUnlock()
	
	if ok {
		return w
	}

	// 双重检查锁定
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if w, ok = m.workers[key]; ok {
		return w
	}

	// 创建新 Worker
	// 目录结构: <baseDir>/<username>/<site_id>/
	siteDir := filepath.Join(m.baseDir, username, siteID)
	w = NewSiteWorker(siteID, siteDir)
	w.Start()
	m.workers[key] = w
	
	return w
}

// GetStats 获取站点统计数据
func (m *Manager) GetStats(username, siteID string, full bool) (any, error) {
	// 尝试获取正在运行的 Worker
	key := username + ":" + siteID
	m.mu.RLock()
	w, ok := m.workers[key]
	m.mu.RUnlock()

	if ok {
		if full {
			return w.GetFullStats(), nil
		}
		return w.GetStats(), nil
	}

	// 如果 Worker 不在运行，临时启动一个
	w = m.GetWorker(username, siteID)
	if full {
		return w.GetFullStats(), nil
	}
	return w.GetStats(), nil
}

// StopAll 停止所有 Worker
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	for _, w := range m.workers {
		wg.Add(1)
		go func(worker *SiteWorker) {
			defer wg.Done()
			worker.Stop()
		}(w)
	}
	wg.Wait()
}
