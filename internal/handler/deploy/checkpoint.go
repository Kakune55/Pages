package deploy

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Checkpoint 表示一个站点部署检查点
type Checkpoint struct {
	ID          string    `json:"id"`           // 检查点唯一标识（时间戳+哈希）
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	FileSize    int64     `json:"file_size"`    // 备份文件大小
	FileName    string    `json:"file_name"`    // 原始上传文件名
	Note        string    `json:"note"`         // 备注信息
	Source      string    `json:"source"`       // 来源："deploy" 或 "manual"
	Description string    `json:"description"`  // 描述信息
}

// SiteCheckpointMetadata 站点检查点元数据
type SiteCheckpointMetadata struct {
	SiteID       string       `json:"site_id"`        // 站点 ID
	Username     string       `json:"username"`       // 租户用户名
	Current      string       `json:"current"`        // 当前激活的检查点 ID
	Checkpoints  []Checkpoint `json:"checkpoints"`    // 所有检查点列表
	StorageUsage *DiskUsage   `json:"storage_usage"`  // 存储使用量缓存
	UpdatedAt    time.Time    `json:"updated_at"`     // 最后更新时间
}

// CheckpointManager 管理检查点
type CheckpointManager struct {
	baseDir string // 检查点存储根目录
}

// NewCheckpointManager 创建检查点管理器
func NewCheckpointManager(baseDir string) *CheckpointManager {
	return &CheckpointManager{
		baseDir: baseDir,
	}
}

// getCheckpointDir 获取指定站点的检查点目录
func (m *CheckpointManager) getCheckpointDir(username, siteID string) string {
	return filepath.Join(m.baseDir, username, siteID)
}

// getCheckpointsSubDir 获取检查点文件存储子目录
func (m *CheckpointManager) getCheckpointsSubDir(username, siteID string) string {
	return filepath.Join(m.getCheckpointDir(username, siteID), "checkpoints")
}

// getCheckpointPath 获取检查点文件路径
func (m *CheckpointManager) getCheckpointPath(username, siteID, checkpointID string) string {
	return filepath.Join(m.getCheckpointsSubDir(username, siteID), checkpointID+".tar.gz")
}

// getSiteMetadataPath 获取站点检查点元数据文件路径 (metadata.json)
func (m *CheckpointManager) getSiteMetadataPath(username, siteID string) string {
	return filepath.Join(m.getCheckpointDir(username, siteID), "metadata.json")
}

// CreateCheckpoint 创建检查点（从目录打包为 tar.gz）- 仅在部署时调用
func (m *CheckpointManager) CreateCheckpoint(username, siteID, sourceDir, originalFileName string) (*Checkpoint, error) {
	checkpointsDir := m.getCheckpointsSubDir(username, siteID)
	if err := os.MkdirAll(checkpointsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建检查点目录失败: %w", err)
	}

	// 生成检查点 ID（时间戳 + 内容哈希前8位）
	timestamp := time.Now().Format("20060102-150405")
	hash := m.calculateDirHash(sourceDir)
	checkpointID := fmt.Sprintf("%s-%s", timestamp, hash[:8])

	checkpoint := &Checkpoint{
		ID:          checkpointID,
		CreatedAt:   time.Now(),
		FileName:    originalFileName,
		Source:      "deploy",
		Description: fmt.Sprintf("部署: %s", originalFileName),
	}

	// 打包目录为 tar.gz
	archivePath := m.getCheckpointPath(username, siteID, checkpointID)
	if err := m.packDirectory(sourceDir, archivePath); err != nil {
		return nil, fmt.Errorf("打包检查点失败: %w", err)
	}

	// 获取文件大小
	stat, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("获取检查点文件信息失败: %w", err)
	}
	checkpoint.FileSize = stat.Size()

	// 加载元数据
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		os.Remove(archivePath) // 清理已创建的备份文件
		return nil, fmt.Errorf("加载站点元数据失败: %w", err)
	}

	// 添加新检查点到元数据
	metadata.Checkpoints = append(metadata.Checkpoints, *checkpoint)
	metadata.Current = checkpointID
	metadata.UpdatedAt = time.Now()

	// 保存站点元数据
	if err := m.saveSiteMetadata(metadata); err != nil {
		os.Remove(archivePath) // 清理已创建的备份文件
		return nil, fmt.Errorf("保存站点元数据失败: %w", err)
	}

	// 重算存储使用量 (忽略错误,不影响检查点创建)
	_ = m.StorageRecount(username, siteID, sourceDir)

	return checkpoint, nil
}

// CheckoutCheckpoint 切换到指定检查点（仅更新 current 指针，不创建新检查点）
func (m *CheckpointManager) CheckoutCheckpoint(username, siteID, checkpointID, targetDir string) error {
	// 检查检查点是否存在
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return fmt.Errorf("加载站点元数据失败: %w", err)
	}

	// 验证检查点存在
	found := false
	for _, cp := range metadata.Checkpoints {
		if cp.ID == checkpointID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("检查点不存在: %s", checkpointID)
	}

	archivePath := m.getCheckpointPath(username, siteID, checkpointID)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return fmt.Errorf("检查点文件不存在: %s", checkpointID)
	}

	// 清空目标目录
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("清空目标目录失败: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 解压检查点到目标目录
	if err := ExtractTarGzSimple(archivePath, targetDir); err != nil {
		return fmt.Errorf("解压检查点失败: %w", err)
	}

	// 更新 current 指针
	metadata.Current = checkpointID
	metadata.UpdatedAt = time.Now()

	// 保存更新后的元数据
	if err := m.saveSiteMetadata(metadata); err != nil {
		return err
	}

	// 重算存储使用量 (忽略错误,不影响切换操作)
	_ = m.StorageRecount(username, siteID, targetDir)

	return nil
}

// ListCheckpoints 列出指定站点的所有检查点
func (m *CheckpointManager) ListCheckpoints(username, siteID string) (*SiteCheckpointMetadata, error) {
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return nil, err
	}

	// 按创建时间倒序排序
	sort.Slice(metadata.Checkpoints, func(i, j int) bool {
		return metadata.Checkpoints[i].CreatedAt.After(metadata.Checkpoints[j].CreatedAt)
	})

	return metadata, nil
}

// DeleteCheckpoint 删除指定检查点
func (m *CheckpointManager) DeleteCheckpoint(username, siteID, checkpointID string) error {
	// 加载元数据
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return fmt.Errorf("加载站点元数据失败: %w", err)
	}

	// 检查是否是当前激活的检查点
	if metadata.Current == checkpointID {
		return fmt.Errorf("无法删除当前激活的检查点")
	}

	// 从元数据中移除检查点
	newCheckpoints := make([]Checkpoint, 0, len(metadata.Checkpoints))
	found := false
	for _, cp := range metadata.Checkpoints {
		if cp.ID == checkpointID {
			found = true
			continue
		}
		newCheckpoints = append(newCheckpoints, cp)
	}

	if !found {
		return fmt.Errorf("检查点不存在: %s", checkpointID)
	}

	metadata.Checkpoints = newCheckpoints
	metadata.UpdatedAt = time.Now()

	// 删除备份文件
	archivePath := m.getCheckpointPath(username, siteID, checkpointID)
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除检查点文件失败: %w", err)
	}

	metadata.StorageUsage = nil

	// 保存更新后的元数据
	return m.saveSiteMetadata(metadata)
}

// GetCheckpoint 获取指定检查点信息
func (m *CheckpointManager) GetCheckpoint(username, siteID, checkpointID string) (*Checkpoint, error) {
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return nil, err
	}

	for _, cp := range metadata.Checkpoints {
		if cp.ID == checkpointID {
			return &cp, nil
		}
	}

	return nil, fmt.Errorf("检查点不存在: %s", checkpointID)
}

// loadSiteMetadata 加载站点检查点元数据
func (m *CheckpointManager) loadSiteMetadata(username, siteID string) (*SiteCheckpointMetadata, error) {
	metadataPath := m.getSiteMetadataPath(username, siteID)

	// 如果文件不存在，返回空元数据
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return &SiteCheckpointMetadata{
			SiteID:      siteID,
			Username:    username,
			Checkpoints: []Checkpoint{},
			UpdatedAt:   time.Now(),
		}, nil
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("读取站点元数据失败: %w", err)
	}

	var metadata SiteCheckpointMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("解析站点元数据失败: %w", err)
	}

	return &metadata, nil
}

// saveSiteMetadata 保存站点检查点元数据
func (m *CheckpointManager) saveSiteMetadata(metadata *SiteCheckpointMetadata) error {
	metadataPath := m.getSiteMetadataPath(metadata.Username, metadata.SiteID)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// packDirectory 将目录打包为 tar.gz
func (m *CheckpointManager) packDirectory(sourceDir, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// 跳过根目录本身
		if relPath == "." {
			return nil
		}

		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		return nil
	})
}

// calculateDirHash 计算目录内容的哈希值
func (m *CheckpointManager) calculateDirHash(dir string) string {
	hash := sha256.New()

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)
		hash.Write([]byte(relPath))

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		io.Copy(hash, f)
		return nil
	})

	return hex.EncodeToString(hash.Sum(nil))
}

// ExtractTarGzSimple 简化版的 tar.gz 解压（不做展平处理）
func ExtractTarGzSimple(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dest, hdr.Name)
		if !isWithinRoot(dest, targetPath) {
			return fmt.Errorf("非法路径: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCheckpointsUsage 获取指定站点的检查点使用情况
func (m *CheckpointManager) GetCheckpointsUsage(username, siteID string) (totalSize int64, count int, err error) {
	checkpointsDir := m.getCheckpointsSubDir(username, siteID)
	
	if _, err := os.Stat(checkpointsDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	err = filepath.Walk(checkpointsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误,继续统计
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

	return totalSize, count, nil
}

// StorageRecount 重新计算并更新站点的存储使用量到元数据
// rootDir: 站点部署根目录
func (m *CheckpointManager) StorageRecount(username, siteID, rootDir string) error {
	// 加载现有元数据
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return fmt.Errorf("加载元数据失败: %w", err)
	}

	// 计算部署目录使用量
	var deployedSize, fileCount int64
	if _, err := os.Stat(rootDir); err == nil {
		deployedSize, fileCount, err = calculateDirSize(rootDir)
		if err != nil {
			return fmt.Errorf("计算部署目录大小失败: %w", err)
		}
	}

	// 计算检查点使用量
	checkpointsSize, checkpointCount, err := m.GetCheckpointsUsage(username, siteID)
	if err != nil {
		return fmt.Errorf("计算检查点大小失败: %w", err)
	}

	// 构建存储使用量对象
	totalSize := deployedSize + checkpointsSize
	metadata.StorageUsage = &DiskUsage{
		DeployedSize:      deployedSize,
		CheckpointsSize:   checkpointsSize,
		TotalSize:         totalSize,
		DeployedSizeHR:    formatBytes(deployedSize),
		CheckpointsSizeHR: formatBytes(checkpointsSize),
		TotalSizeHR:       formatBytes(totalSize),
		FileCount:         fileCount,
		CheckpointCount:   checkpointCount,
	}
	metadata.UpdatedAt = time.Now()

	// 保存元数据
	if err := m.saveSiteMetadata(metadata); err != nil {
		return fmt.Errorf("保存元数据失败: %w", err)
	}

	return nil
}

// GetStorageUsage 从元数据中获取缓存的存储使用量
func (m *CheckpointManager) GetStorageUsage(username, siteID string) (*DiskUsage, error) {
	metadata, err := m.loadSiteMetadata(username, siteID)
	if err != nil {
		return nil, fmt.Errorf("加载元数据失败: %w", err)
	}

	if metadata.StorageUsage == nil {
		// 如果缓存不存在,返回空的使用量信息
		return &DiskUsage{}, nil
	}

	return metadata.StorageUsage, nil
}


