package utils

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryPolicy 定义重试策略
type RetryPolicy struct {
	MaxRetries    int              // 最大重试次数
	BaseDelay     time.Duration    // 基础延迟时间
	MaxDelay      time.Duration    // 最大延迟时间
	Multiplier    float64          // 延迟倍数
	JitterFactor  float64          // 随机抖动因子 (0.0-1.0)
	RetryableFunc func(error) bool // 是否可重试的错误判断函数
}

// 默认重试策略
var DefaultRetryPolicy = &RetryPolicy{
	MaxRetries:   3,
	BaseDelay:    500 * time.Millisecond,
	MaxDelay:     8 * time.Second,
	Multiplier:   2.0,
	JitterFactor: 0.2,
	RetryableFunc: func(err error) bool {
		return IsTemporaryError(err) || IsRateLimitedError(err)
	},
}

// ShouldRetry 判断错误是否可以重试
func ShouldRetry(err error) bool {
	return DefaultRetryPolicy.RetryableFunc(err)
}

// IsTemporaryError 检查是否为临时性错误
func IsTemporaryError(err error) bool {
	var tempErr interface{ Temporary() bool }
	if errors.As(err, &tempErr) {
		return tempErr.Temporary()
	}

	if err == nil {
		return false
	}

	// 简单检查常见临时错误关键字
	errStr := err.Error()
	return ContainsAny(errStr, []string{
		"timeout", "connection refused", "connection reset", "network is unreachable",
		"i/o timeout", "eof", "broken pipe", "too many requests",
	})
}

// IsRateLimitedError 检查是否为准入控制错误
func IsRateLimitedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return ContainsAny(errStr, []string{
		"too many requests", "rate limit", "rate-limited", "429", "quota exceeded",
	})
}

// ContainsAny 检查字符串是否包含任一子串
func ContainsAny(str string, substrs []string) bool {
	for _, sub := range substrs {
		if containsIgnoreCase(str, sub) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(str, substr string) bool {
	s := str
	sub := substr

	if len(s) < len(sub) {
		return false
	}

	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			sc, sz := lower(s[i+j])
			suc, suz := lower(sub[j])

			if sz != suz || sc != suc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// toLower 将字节转换为小写（简化版，主要用于 ASCII 字符）
func lower(b byte) (byte, int) {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A'), 1
	}
	return b, 1
}

// Backoff 计算退避时间
func (rp *RetryPolicy) Backoff(attempt int) time.Duration {
	if attempt <= 0 {
		return rp.BaseDelay
	}

	// 计算指数退避延迟
	delay := float64(rp.BaseDelay) * math.Pow(rp.Multiplier, float64(attempt))

	// 应用最大延迟限制
	if delay > float64(rp.MaxDelay) {
		delay = float64(rp.MaxDelay)
	}

	// 添加随机抖动
	if rp.JitterFactor > 0 {
		jitter := (rand.Float64() * 2 * rp.JitterFactor) - rp.JitterFactor
		jitter = math.Max(jitter, -1) // ensure jitter >= -1
		delay = delay * (1 + jitter)
	}

	// 确保 delay 是非负数
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

var rng = rand.New(&lockedSource{src: rand.NewSource(time.Now().UnixNano())})

type lockedSource struct {
	src  rand.Source
	lock sync.Mutex
}

func (r *lockedSource) Int63() int64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.src.Int63()
}

func (r *lockedSource) Seed(seed int64) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.src.Seed(seed)
}

// DoWithRetry 执行带有重试逻辑的函数
func (rp *RetryPolicy) DoWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= rp.MaxRetries; attempt++ {
		lastErr = fn()

		if lastErr == nil {
			return nil
		}

		// 如果是最后一次尝试，则不再重试
		if attempt == rp.MaxRetries {
			break
		}

		// 检查错误是否可以重试
		if rp.RetryableFunc != nil && !rp.RetryableFunc(lastErr) {
			break
		}

		// 计算下一次重试的延时
		delay := rp.Backoff(attempt)

		// 如果计算出的延时为0，跳过重试等待
		if delay <= 0 {
			continue
		}

		// 使用上下文进行延时等待，以便响应取消操作
		select {
		case <-time.After(delay):
			// Continue with next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// WithCustomRetryPolicy 允许指定自定义重试策略执行函数
func WithCustomRetryPolicy(policy *RetryPolicy) *RetryPolicy {
	if policy == nil {
		return DefaultRetryPolicy
	}
	return policy
}
