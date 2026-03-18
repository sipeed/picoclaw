package wecom

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
)

// GroupPolicy 群组策略
type GroupPolicy struct {
	AllowFrom   []string `json:"allow_from"`   // 允许的用户列表
	MentionOnly bool     `json:"mention_only"` // 是否只在被@时响应
	Prefixes    []string `json:"prefixes"`     // 触发前缀
}

// GroupManager 群组管理器
type GroupManager struct {
	policies map[string]*GroupPolicy
	mu       sync.RWMutex
}

// NewGroupManager 创建新的群组管理器
func NewGroupManager(policies map[string]config.GroupPolicyConfig) *GroupManager {
	gm := &GroupManager{
		policies: make(map[string]*GroupPolicy),
	}

	// 转换配置
	for groupID, policy := range policies {
		gm.policies[groupID] = &GroupPolicy{
			AllowFrom:   policy.AllowFrom,
			MentionOnly: policy.MentionOnly,
			Prefixes:    policy.Prefixes,
		}
	}

	return gm
}

// GetPolicy 获取群组策略
func (gm *GroupManager) GetPolicy(groupID string) *GroupPolicy {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.policies[groupID]
}

// IsAllowedInGroup 检查用户是否在群组白名单中
func (gm *GroupManager) IsAllowedInGroup(groupID, userID string) bool {
	policy := gm.GetPolicy(groupID)
	if policy == nil {
		// 没有特定策略，允许所有
		return true
	}

	if len(policy.AllowFrom) == 0 {
		return true
	}

	for _, allowed := range policy.AllowFrom {
		if allowed == userID {
			return true
		}
	}
	return false
}

// ShouldRespondInGroup 检查是否应该在群组中响应
func (gm *GroupManager) ShouldRespondInGroup(groupID string, isMentioned bool, content string) (bool, string) {
	policy := gm.GetPolicy(groupID)
	if policy == nil {
		// 没有特定策略，使用默认行为
		return true, content
	}

	// 检查是否被@或提及
	if isMentioned {
		return true, content
	}

	// 如果设置了 mention_only，且没有被@，则不响应
	if policy.MentionOnly {
		return false, content
	}

	// 检查前缀
	if len(policy.Prefixes) > 0 {
		for _, prefix := range policy.Prefixes {
			if len(content) >= len(prefix) && content[:len(prefix)] == prefix {
				return true, content[len(prefix):]
			}
		}
		return false, content
	}

	return true, content
}

// SetPolicy 设置群组策略
func (gm *GroupManager) SetPolicy(groupID string, policy *GroupPolicy) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.policies[groupID] = policy
}

// RemovePolicy 移除群组策略
func (gm *GroupManager) RemovePolicy(groupID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	delete(gm.policies, groupID)
}
