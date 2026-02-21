package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type RetryReason string

const (
	RetryReasonTimeout     RetryReason = "timeout"
	RetryReasonServerError RetryReason = "server_error"
)

type RetryDecision struct {
	Retryable bool
	Status    int
	Reason    RetryReason
}

func IsRetryableError(err error) RetryDecision {
	if err == nil {
		return RetryDecision{}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return RetryDecision{Retryable: true, Reason: RetryReasonTimeout}
	}

	msg := err.Error()
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "Client.Timeout") {
		return RetryDecision{Retryable: true, Reason: RetryReasonTimeout}
	}

	if s, ok := ParseHTTPStatusFromError(msg); ok {
		if s >= 500 && s <= 599 {
			return RetryDecision{Retryable: true, Status: s, Reason: RetryReasonServerError}
		}
		return RetryDecision{Retryable: false, Status: s}
	}

	return RetryDecision{}
}

func ParseHTTPStatusFromError(msg string) (int, bool) {
	idx := strings.Index(msg, "Status:")
	if idx < 0 {
		return 0, false
	}

	s := strings.TrimSpace(msg[idx+len("Status:"):])
	end := 0
	for end < len(s) {
		c := s[end]
		if c < '0' || c > '9' {
			break
		}
		end++
	}
	if end == 0 {
		return 0, false
	}

	code, err := strconv.Atoi(s[:end])
	if err != nil {
		return 0, false
	}
	return code, true
}

type RetryNotifyFunc func(attempt, total int, decision RetryDecision)

type RetryConfig struct {
	Timeouts []time.Duration
	Backoffs []time.Duration
	Notify   RetryNotifyFunc
}

func DoWithRetry[T any](
	ctx context.Context,
	retry RetryConfig,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T
	if len(retry.Timeouts) == 0 {
		return fn(ctx)
	}

	var lastErr error
	for attempt := 1; attempt <= len(retry.Timeouts); attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, retry.Timeouts[attempt-1])
		val, err := fn(attemptCtx)
		cancel()

		if err == nil {
			return val, nil
		}

		lastErr = err
		if attempt == len(retry.Timeouts) {
			break
		}

		decision := IsRetryableError(err)
		if !decision.Retryable {
			break
		}

		if retry.Notify != nil {
			retry.Notify(attempt, len(retry.Timeouts), decision)
		}

		if attempt-1 < len(retry.Backoffs) {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(retry.Backoffs[attempt-1]):
			}
		}
	}

	return zero, lastErr
}

func FormatLLMRetryNotice(attempt, total int, decision RetryDecision) string {
	switch decision.Reason {
	case RetryReasonTimeout:
		return fmt.Sprintf("LLM timed out, retrying (attempt %d/%d)", attempt+1, total)
	case RetryReasonServerError:
		if decision.Status > 0 {
			return fmt.Sprintf("LLM server error (%d), retrying (attempt %d/%d)", decision.Status, attempt+1, total)
		}
		return fmt.Sprintf("LLM server error, retrying (attempt %d/%d)", attempt+1, total)
	default:
		return fmt.Sprintf("LLM call failed, retrying (attempt %d/%d)", attempt+1, total)
	}
}
