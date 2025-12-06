package site

import (
	"fmt"
	"os"
	"path/filepath"
)

// Initializer 站点初始化器
type Initializer struct {
	sitesDir string // 站点文件根目录
}

// NewInitializer 创建站点初始化器
func NewInitializer(sitesDir string) *Initializer {
	return &Initializer{
		sitesDir: sitesDir,
	}
}

// InitializeSites 初始化站点目录和示例文件
func (i *Initializer) InitializeSites(sites []*Site) error {
	for _, site := range sites {
		if err := i.initializeSite(site); err != nil {
			return err
		}
	}
	return nil
}

// initializeSite 初始化单个站点
func (i *Initializer) initializeSite(site *Site) error {
	// 根据站点信息自动生成目录路径
	siteDir := site.GetRootDir(i.sitesDir)

	// 创建目录
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", siteDir, err)
	}

	// 创建示例 index.html
	indexPath := filepath.Join(siteDir, site.Index)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		html := i.generateDefaultHTML(site)
		if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
			return fmt.Errorf("创建 %s 失败: %w", indexPath, err)
		}
	}

	return nil
}

// generateDefaultHTML 生成默认的 HTML 内容
func (i *Initializer) generateDefaultHTML(site *Site) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - 静态站点</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 100px auto;
            padding: 20px;
            text-align: center;
        }
        h1 { color: #333; }
        p { color: #666; }
        .domain { color: #007bff; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🎉 欢迎访问</h1>
    <p>这是绑定到 <span class="domain">%s</span> 的静态站点</p>
	<p>Powered by Pages 静态站点服务器</p>
</body>
</html>`, site.Domain, site.Domain)
}
