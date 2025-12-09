package analytics

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	checkpointFileName = "stats_checkpoint.json"
	flushInterval      = 1 * time.Minute
	channelBufferSize  = 1000
)

// SiteWorker 单个站点的统计工作者
type SiteWorker struct {
	SiteID    string
	BaseDir   string // 站点日志存储目录
	Stats     *SiteStats
	LogChan   chan AccessLog
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewSiteWorker 创建新的工作者
func NewSiteWorker(siteID, baseDir string) *SiteWorker {
	return &SiteWorker{
		SiteID:   siteID,
		BaseDir:  baseDir,
		Stats:    NewSiteStats(),
		LogChan:  make(chan AccessLog, channelBufferSize),
		stopChan: make(chan struct{}),
	}
}

// Start 启动工作者
func (w *SiteWorker) Start() {
	// 确保目录存在
	if err := os.MkdirAll(w.BaseDir, 0755); err != nil {
		slog.Error("无法创建统计目录", "site", w.SiteID, "error", err)
		return
	}

	// 加载快照
	w.loadCheckpoint()

	w.wg.Add(1)
	go w.run()
}

// Stop 停止工作者
func (w *SiteWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func (w *SiteWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case log := <-w.LogChan:
			w.processLog(log)
		case <-ticker.C:
			w.saveCheckpoint()
		case <-w.stopChan:
			// 处理剩余的日志
			for len(w.LogChan) > 0 {
				log := <-w.LogChan
				w.processLog(log)
			}
			w.saveCheckpoint()
			return
		}
	}
}

func (w *SiteWorker) processLog(log AccessLog) {
	// 1. 更新内存统计
	w.updateStats(log)
}

func (w *SiteWorker) updateStats(log AccessLog) {
	w.Stats.mu.Lock()
	defer w.Stats.mu.Unlock()

	date := log.Time.Format("2006-01-02")

	// 初始化 Today
	if w.Stats.Today == nil || w.Stats.Today.Date != date {
		// 如果是新的一天，把旧的 Today 归档到 History
		if w.Stats.Today != nil {
			w.Stats.History[w.Stats.Today.Date] = w.Stats.Today
		}
		
		// 检查 History 中是否已有今天的数据 (可能是重启后加载的)
		if stats, ok := w.Stats.History[date]; ok {
			w.Stats.Today = stats
		} else {
			w.Stats.Today = NewDailyStats(date)
		}
	}

	// 累加数据
	s := w.Stats.Today
	s.PV++
	s.Bytes += log.BytesSent
	s.TotalDuration += log.Duration
	if log.StatusCode >= 400 {
		s.ErrorCount++
	}

	// UV 统计
	if s.uvMap == nil {
		s.uvMap = make(map[string]struct{})
	}
	if _, ok := s.uvMap[log.IP]; !ok {
		s.uvMap[log.IP] = struct{}{}
		s.UV++
	}
}

func (w *SiteWorker) saveCheckpoint() {
	w.Stats.mu.RLock()
	defer w.Stats.mu.RUnlock()

	filePath := filepath.Join(w.BaseDir, checkpointFileName)
	f, err := os.Create(filePath)
	if err != nil {
		slog.Error("无法创建快照文件", "site", w.SiteID, "error", err)
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(w.Stats); err != nil {
		slog.Error("写入快照失败", "site", w.SiteID, "error", err)
	}
}

func (w *SiteWorker) loadCheckpoint() {
	filePath := filepath.Join(w.BaseDir, checkpointFileName)
	f, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		slog.Error("无法打开快照文件", "site", w.SiteID, "error", err)
		return
	}
	defer f.Close()

	w.Stats.mu.Lock()
	defer w.Stats.mu.Unlock()

	if err := json.NewDecoder(f).Decode(w.Stats); err != nil {
		slog.Error("解析快照失败", "site", w.SiteID, "error", err)
		// 如果解析失败，可能文件损坏，保持空状态
		return
	}
	
	// 恢复 uvMap
	if w.Stats.Today != nil {
		w.Stats.Today.uvMap = make(map[string]struct{})
	}
	for _, s := range w.Stats.History {
		s.uvMap = make(map[string]struct{})
	}
}

// GetStats 获取统计数据的副本
func (w *SiteWorker) GetStats() DailyStats {
	w.Stats.mu.RLock()
	defer w.Stats.mu.RUnlock()
	
	if w.Stats.Today == nil {
		return *NewDailyStats(time.Now().Format("2006-01-02"))
	}
	return *w.Stats.Today
}

// GetFullStats 获取完整统计数据（包含历史）
func (w *SiteWorker) GetFullStats() *SiteStats {
	w.Stats.mu.RLock()
	defer w.Stats.mu.RUnlock()

	// 创建副本以避免并发读写 map 导致 panic
	copy := &SiteStats{
		History: make(map[string]*DailyStats, len(w.Stats.History)),
	}
	
	if w.Stats.Today != nil {
		today := *w.Stats.Today
		copy.Today = &today
	} else {
		copy.Today = NewDailyStats(time.Now().Format("2006-01-02"))
	}

	for k, v := range w.Stats.History {
		val := *v
		copy.History[k] = &val
	}

	return copy
}
