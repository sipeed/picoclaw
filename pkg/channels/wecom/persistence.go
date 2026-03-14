package wecom

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ReqIDStore req_id 存储
type ReqIDStore struct {
	data      map[string]time.Time
	mu        sync.RWMutex
	filePath  string
	ttl       time.Duration
}

// NewReqIDStore 创建新的 req_id 存储
func NewReqIDStore(persistencePath string) *ReqIDStore {
	if persistencePath == "" {
		persistencePath = filepath.Join(os.TempDir(), "picoclaw", "wecom_ws")
	}

	filePath := filepath.Join(persistencePath, "req_ids.json")

	store := &ReqIDStore{
		data:     make(map[string]time.Time),
		filePath: filePath,
		ttl:      24 * time.Hour, // 默认24小时过期
	}

	// 加载历史数据
	store.Load()

	return store
}

// Add 添加 req_id
func (s *ReqIDStore) Add(reqID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[reqID] = time.Now()
}

// Exists 检查 req_id 是否存在
func (s *ReqIDStore) Exists(reqID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timestamp, exists := s.data[reqID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Since(timestamp) > s.ttl {
		return false
	}

	return true
}

// Save 保存到磁盘
func (s *ReqIDStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 清理过期数据
	s.cleanupLocked()

	// 创建目录
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 序列化数据
	data := struct {
		ReqIDs      map[string]time.Time `json:"req_ids"`
		LastCleanup time.Time            `json:"last_cleanup"`
	}{
		ReqIDs:      s.data,
		LastCleanup: time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(s.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.DebugCF("wecom_ws", "ReqID store saved", map[string]any{
		"count": len(s.data),
		"path":  s.filePath,
	})

	return nil
}

// Load 从磁盘加载
func (s *ReqIDStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		logger.DebugC("wecom_ws", "ReqID store file not found, starting fresh")
		return nil
	}

	// 读取文件
	jsonData, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 解析数据
	var data struct {
		ReqIDs      map[string]time.Time `json:"req_ids"`
		LastCleanup time.Time            `json:"last_cleanup"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	s.data = data.ReqIDs
	if s.data == nil {
		s.data = make(map[string]time.Time)
	}

	// 清理过期数据
	s.cleanupLocked()

	logger.DebugCF("wecom_ws", "ReqID store loaded", map[string]any{
		"count": len(s.data),
		"path":  s.filePath,
	})

	return nil
}

// Cleanup 清理过期数据
func (s *ReqIDStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
}

// cleanupLocked 清理过期数据（需要持有锁）
func (s *ReqIDStore) cleanupLocked() {
	now := time.Now()
	for reqID, timestamp := range s.data {
		if now.Sub(timestamp) > s.ttl {
			delete(s.data, reqID)
		}
	}
}

// StartAutoSave 启动自动保存
func (s *ReqIDStore) StartAutoSave(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.Save(); err != nil {
				logger.ErrorCF("wecom_ws", "Failed to auto-save req_id store", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}()
}

// Stop 停止并保存
func (s *ReqIDStore) Stop() error {
	return s.Save()
}
