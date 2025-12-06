package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

func init() {
	// 默认使用 info 级别，可以在其他地方通过 SetLevel 来调整
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		AddSource:  true,
		Level:      slog.LevelInfo,
		TimeFormat: "2006-01-02 15:04:05",
	})))
}

// SetLevel 设置日志级别
func SetLevel(level slog.Level) {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		AddSource:  true,
		Level:      level,
		TimeFormat: "2006-01-02 15:04:05",
	})))
}

// SetLevelWithStr 通过字符串设置日志级别
func SetLevelWithStr(levelStr string) {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	SetLevel(level)
}
