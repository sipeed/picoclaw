package config

import "time"

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries   int           `json:"max_retries" yaml:"max_retries"`     // 最大重试次数
	BaseDelay    time.Duration `json:"base_delay" yaml:"base_delay"`       // 基础延迟时间
	MaxDelay     time.Duration `json:"max_delay" yaml:"max_delay"`         // 最大延迟时间
	Multiplier   float64       `json:"multiplier" yaml:"multiplier"`       // 延迟倍数
	JitterFactor float64       `json:"jitter_factor" yaml:"jitter_factor"` // 随机抖动因子 0.0-1.0
}

// GetDefaultRetryConfig 获取默认重试配置
func GetDefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		BaseDelay:    500 * time.Millisecond,
		MaxDelay:     8 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}
