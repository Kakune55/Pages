package admin

import (
	"pages/internal/handler/deploy"
	"pages/internal/site"
)

// Handler 管理接口处理器
type Handler struct {
	siteManager       *site.ManagerLockFree
	initializer       *site.Initializer
	checkpointManager *deploy.CheckpointManager
}

// NewHandler 创建管理接口处理器
func NewHandler(sm *site.ManagerLockFree, init *site.Initializer, checkpointsDir string) *Handler {
	return &Handler{
		siteManager:       sm,
		initializer:       init,
		checkpointManager: deploy.NewCheckpointManager(checkpointsDir),
	}
}

// Response 通用响应结构
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
