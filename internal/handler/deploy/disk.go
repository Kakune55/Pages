package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiskUsage 磁盘使用情况统计
type DiskUsage struct {
	DeployedSize    int64  `json:"deployed_size"`     // 当前部署的站点大小(字节)
	CheckpointsSize int64  `json:"checkpoints_size"`  // 所有检查点归档大小(字节)
	TotalSize       int64  `json:"total_size"`        // 总使用大小(字节)
	DeployedSizeHR  string `json:"deployed_size_h"`  // 人类可读格式
	CheckpointsSizeHR string `json:"checkpoints_size_h"`
	TotalSizeHR     string `json:"total_size_h"`
	FileCount       int64  `json:"file_count"`        // 文件总数
	CheckpointCount int    `json:"checkpoint_count"`  // 检查点数量
}

// GetDirectoryUsage 获取指定站点的完整磁盘使用情况
func GetDirectoryUsage(rootDir string) (*DiskUsage, error) {
	usage := &DiskUsage{}

	// 1. 计算部署目录大小
	if _, err := os.Stat(rootDir); err == nil {
		deployedSize, fileCount, err := calculateDirSize(rootDir)
		if err != nil {
			return nil, fmt.Errorf("计算部署目录大小失败: %w", err)
		}
		usage.DeployedSize = deployedSize
		usage.FileCount = fileCount
		usage.DeployedSizeHR = formatBytes(deployedSize)
	}

	// 计算总大小
	usage.TotalSize = usage.DeployedSize + usage.CheckpointsSize
	usage.TotalSizeHR = formatBytes(usage.TotalSize)

	return usage, nil
}

// GetCheckpointsUsage 获取检查点目录的使用情况
func GetCheckpointsUsage(checkpointsDir string) (int64, int, error) {
	if _, err := os.Stat(checkpointsDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var totalSize int64
	var count int

	checkpointsSubDir := filepath.Join(checkpointsDir, "checkpoints")
	if _, err := os.Stat(checkpointsSubDir); err == nil {
		err := filepath.Walk(checkpointsSubDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".gz" {
				totalSize += info.Size()
				count++
			}
			return nil
		})
		if err != nil {
			return 0, 0, fmt.Errorf("遍历检查点目录失败: %w", err)
		}
	}

	return totalSize, count, nil
}

// calculateDirSize 递归计算目录大小和文件数量
func calculateDirSize(path string) (int64, int64, error) {
	var totalSize int64
	var fileCount int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// 忽略无法访问的文件,继续统计其他文件
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	return totalSize, fileCount, nil
}

// formatBytes 将字节数格式化为人类可读的格式
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	// KB, MB, GB, TB, PB
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}