package deploy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NormalizeDirectory 检测并整理目录结构
// 如果目录中只有一个顶层文件夹且没有顶层文件，将该文件夹的内容提升到上层
// 返回整理后的目录路径（可能是原目录或新创建的临时目录）
func NormalizeDirectory(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}

	// 过滤隐藏文件和系统文件
	var visibleEntries []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		// 跳过 .DS_Store, __MACOSX, .git 等
		if strings.HasPrefix(name, ".") || name == "__MACOSX" {
			continue
		}
		visibleEntries = append(visibleEntries, entry)
	}

	// 如果没有可见内容，返回原目录
	if len(visibleEntries) == 0 {
		return extractDir, nil
	}

	// 如果只有一个条目且是文件夹，需要展平
	if len(visibleEntries) == 1 && visibleEntries[0].IsDir() {
		nestedDir := filepath.Join(extractDir, visibleEntries[0].Name())
		
		// 创建临时目录用于存放展平后的内容
		normalizedDir, err := os.MkdirTemp("", "deploy-normalized-*")
		if err != nil {
			return "", fmt.Errorf("创建临时目录失败: %w", err)
		}

		// 移动嵌套目录的内容到新的临时目录
		if err := moveDirectoryContents(nestedDir, normalizedDir); err != nil {
			os.RemoveAll(normalizedDir)
			return "", fmt.Errorf("展平目录失败: %w", err)
		}

		return normalizedDir, nil
	}

	// 多个顶层条目或包含顶层文件，无需展平
	return extractDir, nil
}

// moveDirectoryContents 将 src 目录中的所有内容移动到 dst 目录
func moveDirectoryContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// 尝试重命名（同一文件系统内是原子操作）
		if err := os.Rename(srcPath, dstPath); err != nil {
			// 重命名失败（可能跨文件系统），使用复制+删除
			if entry.IsDir() {
				if err := copyDirectory(srcPath, dstPath); err != nil {
					return fmt.Errorf("复制目录 %s 失败: %w", entry.Name(), err)
				}
			} else {
				if err := copyFile(srcPath, dstPath); err != nil {
					return fmt.Errorf("复制文件 %s 失败: %w", entry.Name(), err)
				}
			}
			// 复制成功后删除源文件
			if err := os.RemoveAll(srcPath); err != nil {
				return fmt.Errorf("删除源文件 %s 失败: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// AtomicReplaceDirectory 原子性地替换目标目录
// 支持 Windows 兼容性和重试机制
func AtomicReplaceDirectory(oldDir, newDir string) error {
	// 确保目标目录的父目录存在
	parentDir := filepath.Dir(oldDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("创建父目录失败: %w", err)
	}

	// 生成备份目录名
	backupDir := oldDir + ".backup." + fmt.Sprintf("%d", time.Now().Unix())

	// 1. 如果旧目录存在，先重命名为备份
	if _, err := os.Stat(oldDir); err == nil {
		if err := renameWithRetry(oldDir, backupDir, 5); err != nil {
			return fmt.Errorf("备份旧目录失败: %w", err)
		}
		defer func() {
			// 清理备份目录
			os.RemoveAll(backupDir)
		}()
	}

	// 2. 将新目录重命名为目标目录
	if err := renameWithRetry(newDir, oldDir, 5); err != nil {
		// 失败时尝试恢复备份
		if _, statErr := os.Stat(backupDir); statErr == nil {
			renameWithRetry(backupDir, oldDir, 3)
		}
		return fmt.Errorf("替换目录失败: %w", err)
	}

	return nil
}

// renameWithRetry 带重试的重命名操作（Windows 兼容）
func renameWithRetry(src, dst string, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := os.Rename(src, dst)
		if err == nil {
			return nil
		}
		lastErr = err

		// Windows 下可能因为文件被占用而失败，等待后重试
		if i < maxRetries-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("重命名失败（已重试 %d 次）: %w", maxRetries, lastErr)
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Close()
}

// copyDirectory 递归复制目录
func copyDirectory(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
