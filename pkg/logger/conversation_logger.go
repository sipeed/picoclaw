package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ConversationRecord 对话记录
type ConversationRecord struct {
	Timestamp  string `json:"timestamp"`
	SessionKey string `json:"session_key"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Channel    string `json:"channel,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
}

// ConversationLogger 记录对话日志
type ConversationLogger struct {
	config   config.ConversationLogConfig
	mu       sync.Mutex
	logFiles map[string]*os.File // 按日期缓存的文件句柄
	baseDir  string
}

var (
	conversationLogger *ConversationLogger
	conversationOnce   sync.Once
)

// InitConversationLogger 初始化全局对话日志记录器
func InitConversationLogger(cfg config.ConversationLogConfig, workspace string) {
	conversationOnce.Do(func() {
		logDir := cfg.LogDir
		if logDir == "" {
			// 默认路径: workspace/logs/conversations/
			logDir = filepath.Join(workspace, "logs", "conversations")
		}

		// 展开 ~ 路径
		if strings.HasPrefix(logDir, "~") {
			home, _ := os.UserHomeDir()
			logDir = filepath.Join(home, logDir[1:])
		}

		conversationLogger = &ConversationLogger{
			config:   cfg,
			logFiles: make(map[string]*os.File),
			baseDir:  logDir,
		}

		// 确保目录存在
		os.MkdirAll(logDir, 0755)

		// 启动清理旧日志的 goroutine
		if cfg.MaxFiles > 0 {
			go conversationLogger.cleanupOldLogs()
		}
	})
}

// GetConversationLogger 获取全局对话日志记录器
func GetConversationLogger() *ConversationLogger {
	return conversationLogger
}

// IsEnabled 检查日志是否启用
func (l *ConversationLogger) IsEnabled() bool {
	return l != nil && l.config.Enabled
}

// Log 记录一条对话
func (l *ConversationLogger) Log(record *ConversationRecord) error {
	if !l.IsEnabled() {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取当天的日志文件
	dateKey := time.Now().Format("2006-01-02")
	file, err := l.getLogFile(dateKey)
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}

	// 写入 JSON 行
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	_, err = fmt.Fprintf(file, "%s\n", data)
	if err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// getLogFile 获取或创建指定日期的日志文件
func (l *ConversationLogger) getLogFile(dateKey string) (*os.File, error) {
	if file, ok := l.logFiles[dateKey]; ok {
		return file, nil
	}

	// 创建新文件
	filename := fmt.Sprintf("%s.jsonl", dateKey)
	path := filepath.Join(l.baseDir, filename)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	l.logFiles[dateKey] = file
	return file, nil
}

// cleanupOldLogs 清理旧日志文件
func (l *ConversationLogger) cleanupOldLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		l.doCleanup()
	}
}

// doCleanup 执行清理
func (l *ConversationLogger) doCleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 读取目录中的所有日志文件
	entries, err := os.ReadDir(l.baseDir)
	if err != nil {
		return
	}

	// 收集所有日志文件
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			files = append(files, entry.Name())
		}
	}

	// 如果文件数量超过限制，删除最旧的
	if len(files) > l.config.MaxFiles {
		// 按文件名排序（文件名包含日期）
		// 删除最旧的文件
		for i := 0; i < len(files)-l.config.MaxFiles; i++ {
			path := filepath.Join(l.baseDir, files[i])
			os.Remove(path)

			// 关闭可能打开的文件句柄
			dateKey := strings.TrimSuffix(files[i], ".jsonl")
			if file, ok := l.logFiles[dateKey]; ok {
				file.Close()
				delete(l.logFiles, dateKey)
			}
		}
	}
}

// Close 关闭所有打开的文件句柄
func (l *ConversationLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var lastErr error
	for dateKey, file := range l.logFiles {
		if err := file.Close(); err != nil {
			lastErr = err
		}
		delete(l.logFiles, dateKey)
	}

	return lastErr
}