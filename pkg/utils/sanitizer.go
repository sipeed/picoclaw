// PicoClaw - Ultra-lightweight personal AI assistant
// License: MIT

package utils

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// SanitizerConfig 脱敏配置
type SanitizerConfig struct {
	Enabled        bool                `json:"enabled"`
	Keywords       []KeywordRule       `json:"keywords"`
	CustomPatterns []CustomPatternRule `json:"custom_patterns"`
}

// KeywordRule 关键词规则
type KeywordRule struct {
	Word string `json:"word"`
	Tag  string `json:"tag"`
}

// CustomPatternRule 自定义正则规则
type CustomPatternRule struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Tag     string `json:"tag"`
}

// compiledPattern 编译后的正则规则
type compiledPattern struct {
	regex *regexp.Regexp
	tag   string
	name  string
}

// Sanitizer 脱敏器
type Sanitizer struct {
	config   SanitizerConfig
	patterns []compiledPattern
	mu       sync.RWMutex
}

// 全局脱敏器实例
var globalSanitizer *Sanitizer
var globalSanitizerMu sync.RWMutex

// 内置正则规则（按优先级排序）
var builtinPatterns = []compiledPattern{
	// 邮箱
	{regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), "[EMAIL]", "email"},
	// 中国手机号
	{regexp.MustCompile(`\b1[3-9]\d{9}\b`), "[PHONE]", "phone"},
	// 中国身份证号（18位）
	{regexp.MustCompile(`\b\d{17}[\dXx]\b`), "[ID_CARD]", "id_card"},
	// API Key（OpenAI 格式）
	{regexp.MustCompile(`(?i)\bsk-[a-zA-Z0-9]{20,}\b`), "[API_KEY]", "api_key"},
	// 信用卡号
	{regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), "[CREDIT_CARD]", "credit_card"},
	// IP 地址
	{regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), "[IP]", "ip"},
}

// NewSanitizer 创建脱敏器
func NewSanitizer(config SanitizerConfig) *Sanitizer {
	s := &Sanitizer{
		config:   config,
		patterns: make([]compiledPattern, 0),
	}

	// 添加内置规则
	s.patterns = append(s.patterns, builtinPatterns...)

	// 添加自定义正则规则
	for _, cp := range config.CustomPatterns {
		re, err := regexp.Compile(cp.Pattern)
		if err != nil {
			continue // 忽略无效正则
		}
		s.patterns = append(s.patterns, compiledPattern{
			regex: re,
			tag:   cp.Tag,
			name:  cp.Name,
		})
	}

	return s
}

// InitGlobalSanitizer 初始化全局脱敏器
func InitGlobalSanitizer(config SanitizerConfig) {
	globalSanitizerMu.Lock()
	defer globalSanitizerMu.Unlock()
	globalSanitizer = NewSanitizer(config)
}

// GetGlobalSanitizer 获取全局脱敏器
func GetGlobalSanitizer() *Sanitizer {
	globalSanitizerMu.RLock()
	defer globalSanitizerMu.RUnlock()
	return globalSanitizer
}

// SanitizeResult 脱敏结果
type SanitizeResult struct {
	Sanitized string            // 脱敏后的文本
	Mappings  map[string]string // 占位符 -> 原始值 的映射
}

// Sanitize 对文本进行脱敏
func (s *Sanitizer) Sanitize(text string) SanitizeResult {
	if s == nil || !s.config.Enabled {
		return SanitizeResult{
			Sanitized: text,
			Mappings:  make(map[string]string),
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	mappings := make(map[string]string)
	result := text

	// 先处理关键词（精确匹配）
	for _, kw := range s.config.Keywords {
		if kw.Word == "" {
			continue
		}
		placeholder := s.findUniquePlaceholder(kw.Tag, mappings)
		if strings.Contains(result, kw.Word) {
			mappings[placeholder] = kw.Word
			result = strings.ReplaceAll(result, kw.Word, placeholder)
		}
	}

	// 再处理正则规则
	for _, p := range s.patterns {
		matches := p.regex.FindAllString(result, -1)
		for i, match := range matches {
			// 检查是否已经被脱敏（避免重复脱敏）
			if s.isAlreadySanitized(match) {
				continue
			}
			placeholder := s.findUniquePlaceholderIndexed(p.tag, i, mappings)
			mappings[placeholder] = match
			result = strings.Replace(result, match, placeholder, 1)
		}
	}

	return SanitizeResult{
		Sanitized: result,
		Mappings:  mappings,
	}
}

// Restore 还原脱敏内容
func (s *Sanitizer) Restore(text string, mappings map[string]string) string {
	if s == nil || len(mappings) == 0 {
		return text
	}

	result := text

	// 按占位符长度降序排序，避免短占位符先被替换导致长占位符无法匹配
	type kv struct {
		key   string
		value string
	}
	var sorted []kv
	for k, v := range mappings {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].key) > len(sorted[j].key)
	})

	for _, kv := range sorted {
		result = strings.ReplaceAll(result, kv.key, kv.value)
	}

	return result
}

// findUniquePlaceholder 找到唯一的占位符
func (s *Sanitizer) findUniquePlaceholder(tag string, existing map[string]string) string {
	base := tag
	if base == "" {
		base = "[REDACTED]"
	}

	// 如果 tag 已经是完整格式（如 [EMAIL]），直接使用
	if strings.HasPrefix(base, "[") && strings.HasSuffix(base, "]") {
		placeholder := base
		// 检查是否已存在
		for i := 0; ; i++ {
			if i == 0 {
				if _, exists := existing[placeholder]; !exists {
					return placeholder
				}
			} else {
				numbered := placeholder[:len(placeholder)-1] + "_" + intToString(i) + "]"
				if _, exists := existing[numbered]; !exists {
					return numbered
				}
			}
		}
	}

	// 否则添加方括号
	placeholder := "[" + base + "]"
	for i := 0; ; i++ {
		if i == 0 {
			if _, exists := existing[placeholder]; !exists {
				return placeholder
			}
		} else {
			numbered := "[" + base + "_" + intToString(i) + "]"
			if _, exists := existing[numbered]; !exists {
				return numbered
			}
		}
	}
}

// findUniquePlaceholderIndexed 找到带索引的唯一占位符
func (s *Sanitizer) findUniquePlaceholderIndexed(tag string, idx int, existing map[string]string) string {
	base := tag
	if base == "" {
		base = "[REDACTED]"
	}

	// 确保 tag 有方括号
	if !strings.HasPrefix(base, "[") {
		base = "[" + base + "]"
	}

	// 如果是第一个匹配，使用基础格式
	if idx == 0 {
		if _, exists := existing[base]; !exists {
			return base
		}
	}

	// 否则使用带索引的格式
	for i := idx; ; i++ {
		numbered := base[:len(base)-1] + "_" + intToString(i) + "]"
		if _, exists := existing[numbered]; !exists {
			return numbered
		}
	}
}

// isAlreadySanitized 检查文本是否已被脱敏
func (s *Sanitizer) isAlreadySanitized(text string) bool {
	// 检查是否是占位符格式
	return strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]")
}

// intToString 简单的整数转字符串
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	var negative bool
	if n < 0 {
		negative = true
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// SanitizeText 全局脱敏函数
func SanitizeText(text string) SanitizeResult {
	s := GetGlobalSanitizer()
	if s == nil {
		return SanitizeResult{
			Sanitized: text,
			Mappings:  make(map[string]string),
		}
	}
	return s.Sanitize(text)
}

// RestoreText 全局还原函数
func RestoreText(text string, mappings map[string]string) string {
	s := GetGlobalSanitizer()
	if s == nil {
		return text
	}
	return s.Restore(text, mappings)
}