package analytics

import (
	"sync"
	"time"
)

// AccessLog 访问日志
type AccessLog struct {
	Time       time.Time `json:"time"`
	IP         string    `json:"ip"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	Duration   int64     `json:"duration_ms"` // ms
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	BytesSent  int64     `json:"bytes_sent"`
}

// DailyStats 每日统计聚合
type DailyStats struct {
	Date          string `json:"date"`           // 日期 "2006-01-02"
	PV            int64  `json:"pv"`             // 页面浏览量
	UV            int64  `json:"uv"`             // 独立访客 (简单计数)
	Bytes         int64  `json:"bytes"`          // 流量 (字节)
	TotalDuration int64  `json:"total_duration"` // 总响应时间 (用于计算平均值)
	ErrorCount    int64  `json:"error_count"`    // 错误数 (状态码 >= 400)
	
	// 简单的 UV 统计辅助 (不序列化)
	// 在实际生产中，应该使用 HyperLogLog 或 BloomFilter，这里为了轻量使用 map
	// 注意：这会消耗内存，如果 UV 很大，需要优化
	uvMap map[string]struct{} `json:"-"`
}

// SiteStats 单个站点的所有统计数据
type SiteStats struct {
	mu sync.RWMutex // 读写锁，保护并发访问

	// 实时统计 (当天)
	Today *DailyStats `json:"today"`

	// 历史统计 (最近 N 天)
	History map[string]*DailyStats `json:"history"`
}

func NewSiteStats() *SiteStats {
	return &SiteStats{
		History: make(map[string]*DailyStats),
	}
}

func NewDailyStats(date string) *DailyStats {
	return &DailyStats{
		Date:  date,
		uvMap: make(map[string]struct{}),
	}
}
