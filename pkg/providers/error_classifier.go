package providers

import (
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"syscall"
)

// Common patterns in Go HTTP error messages
var httpStatusPatterns = []*regexp.Regexp{
	regexp.MustCompile(`status[:\s]+(\d{3})`),
	regexp.MustCompile(`http[/\s]+\d*\.?\d*\s+(\d{3})`),
	regexp.MustCompile(`\b([3-5]\d{2})\b`),
}

// errorPattern defines a single pattern (string or regex) for error classification.
type errorPattern struct {
	substring string
	regex     *regexp.Regexp
}

func substr(s string) errorPattern { return errorPattern{substring: s} }
func rxp(r string) errorPattern    { return errorPattern{regex: regexp.MustCompile("(?i)" + r)} }

// Error patterns organized by FailoverReason, matching OpenClaw production (~40 patterns).
var (
	rateLimitPatterns = []errorPattern{
		rxp(`rate[_ ]limit`),
		substr("too many requests"),
		substr("429"),
		substr("exceeded your current quota"),
		rxp(`exceeded.*quota`),
		rxp(`resource has been exhausted`),
		rxp(`resource.*exhausted`),
		substr("resource_exhausted"),
		substr("quota exceeded"),
		substr("usage limit"),
	}

	overloadedPatterns = []errorPattern{
		rxp(`overloaded_error`),
		rxp(`"type"\s*:\s*"overloaded_error"`),
		substr("overloaded"),
	}

	timeoutPatterns = []errorPattern{
		substr("timeout"),
		substr("timed out"),
		substr("deadline exceeded"),
		substr("context deadline exceeded"),
	}

	networkPatterns = []errorPattern{
		substr("connection reset"),
		substr("reset by peer"),
		substr("connection refused"),
		substr("connection aborted"),
		substr("broken pipe"),
		substr("use of closed network connection"),
		substr("network is unreachable"),
		substr("host is unreachable"),
		substr("no such host"),
		substr("temporary failure in name resolution"),
		substr("server misbehaving"),
		substr("read tcp"),
		substr("write tcp"),
		substr("dial tcp"),
		substr("tls:"),
		substr("x509:"),
		substr("certificate"),
		substr("handshake"),
		substr("unexpected eof"),
		substr("read: eof"),
		substr("write: eof"),
	}

	billingPatterns = []errorPattern{
		rxp(`\b402\b`),
		substr("payment required"),
		substr("insufficient credits"),
		substr("credit balance"),
		substr("plans & billing"),
		substr("insufficient balance"),
	}

	authPatterns = []errorPattern{
		rxp(`invalid[_ ]?api[_ ]?key`),
		substr("incorrect api key"),
		substr("invalid token"),
		substr("authentication"),
		substr("re-authenticate"),
		substr("oauth token refresh failed"),
		substr("unauthorized"),
		substr("forbidden"),
		substr("access denied"),
		substr("expired"),
		substr("token has expired"),
		rxp(`\b401\b`),
		rxp(`\b403\b`),
		substr("no credentials found"),
		substr("no api key found"),
	}

	formatPatterns = []errorPattern{
		substr("string should match pattern"),
		substr("tool_use.id"),
		substr("tool_use_id"),
		substr("messages.1.content.1.tool_use.id"),
		substr("invalid request format"),
	}
	// modelNotFoundPatterns detects errors indicating the model is unavailable at
	// the provider, which should trigger fallback to the next candidate.
	modelNotFoundPatterns = []errorPattern{
		substr("no endpoints found"),
		substr("model not found"),
		substr("does not exist"),
		rxp(`model .* not available`),
		substr("model_not_found"),
		rxp(`\b404\b`),
	}

	contextOverflowPatterns = []errorPattern{
		rxp(`context[_ ]?length[_ ]?exceeded`),
		rxp(`context[_ ]?window[_ ]?exceeded`),
		substr("maximum context length"),
		substr("token limit"),
		substr("too many tokens"),
		substr("prompt is too long"),
		substr("request too large"),
	}

	imageDimensionPatterns = []errorPattern{
		rxp(`image dimensions exceed max`),
	}

	imageSizePatterns = []errorPattern{
		rxp(`image exceeds.*mb`),
	}

	// contextWindowPatterns detects context window / token limit exhaustion errors.
	contextWindowPatterns = []errorPattern{
		rxp(`context.*(window|length).*exceed`),
		rxp(`max.*token.*(limit|exceed)`),
		rxp(`total tokens.*exceed`),
		substr("context_length_exceeded"),
		substr("maximum context length"),
		rxp(`invalidparameter.*token`),
		rxp(`tokens?.*exceed`),
		rxp(`exceeds? the (model|max).*token`),
	}

	// Transient HTTP status codes that map to timeout (server-side failures).
	transientStatusCodes = map[int]bool{
		500: true, 502: true, 503: true,
		521: true, 522: true, 523: true, 524: true,
		529: true,
	}
)

// ClassifyError classifies an error into a FailoverError with reason.
// Returns nil if the error is not classifiable (unknown errors should not trigger fallback).
func ClassifyError(err error, provider, model string) *FailoverError {
	if err == nil {
		return nil
	}

	// Context cancellation: user abort, never fallback.
	if err == context.Canceled {
		return nil
	}

	// Context deadline exceeded: treat as timeout, always fallback.
	if err == context.DeadlineExceeded {
		return &FailoverError{
			Reason:   FailoverTimeout,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	msg := strings.ToLower(err.Error())

	// Concrete transport errors should continue the fallback chain even when
	// providers do not expose a structured HTTP status.
	if reason := classifyByErrorType(err); reason != "" {
		return &FailoverError{
			Reason:   reason,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	// Image dimension/size errors: non-retriable, non-fallback.
	if IsImageDimensionError(msg) || IsImageSizeError(msg) {
		return &FailoverError{
			Reason:   FailoverFormat,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	// Try HTTP status code extraction first.
	if status := extractHTTPStatus(msg); status > 0 {
		if reason := classifyByStatus(status); reason != "" {
			return &FailoverError{
				Reason:   reason,
				Provider: provider,
				Model:    model,
				Status:   status,
				Wrapped:  err,
			}
		}
	}

	// Message pattern matching (priority order from OpenClaw).
	if reason := classifyByMessage(msg); reason != "" {
		return &FailoverError{
			Reason:   reason,
			Provider: provider,
			Model:    model,
			Wrapped:  err,
		}
	}

	return nil
}

// classifyByErrorType maps concrete transport-layer error types to a retryable
// fallback reason before message heuristics are applied.
func classifyByErrorType(err error) FailoverReason {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return FailoverNetwork
	}

	for _, transportErr := range []error{
		syscall.ECONNRESET,
		syscall.ECONNABORTED,
		syscall.ECONNREFUSED,
		syscall.ETIMEDOUT,
		syscall.EHOSTUNREACH,
		syscall.ENETUNREACH,
		syscall.EPIPE,
	} {
		if errors.Is(err, transportErr) {
			if transportErr == syscall.ETIMEDOUT {
				return FailoverTimeout
			}
			return FailoverNetwork
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return FailoverTimeout
		}
		return FailoverNetwork
	}

	return ""
}

// classifyByStatus maps HTTP status codes to FailoverReason.
func classifyByStatus(status int) FailoverReason {
	switch {
	case status == 401 || status == 403:
		return FailoverAuth
	case status == 402:
		return FailoverBilling
	case status == 408:
		return FailoverTimeout
	case status == 429:
		return FailoverRateLimit
	case status == 400:
		return FailoverFormat
	case status == 404:
		return FailoverModelNotFound
	case transientStatusCodes[status]:
		return FailoverTimeout
	}
	return ""
}

// classifyByMessage matches error messages against patterns.
// Priority order matters (from OpenClaw classifyFailoverReason).
func classifyByMessage(msg string) FailoverReason {
	if matchesAny(msg, rateLimitPatterns) {
		return FailoverRateLimit
	}
	if matchesAny(msg, overloadedPatterns) {
		return FailoverRateLimit // Overloaded treated as rate_limit
	}
	if matchesAny(msg, billingPatterns) {
		return FailoverBilling
	}
	if matchesAny(msg, timeoutPatterns) {
		return FailoverTimeout
	}
	if matchesAny(msg, networkPatterns) {
		return FailoverNetwork
	}
	if matchesAny(msg, authPatterns) {
		return FailoverAuth
	}
	if matchesAny(msg, formatPatterns) {
		return FailoverFormat
	}
	if matchesAny(msg, contextOverflowPatterns) {
		return FailoverContextOverflow
	}
	if matchesAny(msg, modelNotFoundPatterns) {
		return FailoverModelNotFound
	}
	return ""
}

// extractHTTPStatus extracts an HTTP status code from an error message.
// Looks for patterns like "status: 429", "status 429", "http/1.1 429", "http 429", or standalone "429".
func extractHTTPStatus(msg string) int {
	for _, p := range httpStatusPatterns {
		if m := p.FindStringSubmatch(msg); len(m) > 1 {
			return parseDigits(m[1])
		}
	}
	return 0
}

// IsImageDimensionError returns true if the message indicates an image dimension error.
func IsImageDimensionError(msg string) bool {
	return matchesAny(msg, imageDimensionPatterns)
}

// IsImageSizeError returns true if the message indicates an image file size error.
func IsImageSizeError(msg string) bool {
	return matchesAny(msg, imageSizePatterns)
}

// IsContextWindowError returns true if the error indicates a context window / token limit
// exhaustion. Uses targeted regex/substring patterns to reduce false positives
// (e.g. "invalidparameter" is scoped to token-related errors only).
func IsContextWindowError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return matchesAny(msg, contextWindowPatterns)
}

// matchesAny checks if msg matches any of the patterns.
func matchesAny(msg string, patterns []errorPattern) bool {
	for _, p := range patterns {
		if p.regex != nil {
			if p.regex.MatchString(msg) {
				return true
			}
		} else if p.substring != "" {
			if strings.Contains(msg, p.substring) {
				return true
			}
		}
	}
	return false
}

// parseDigits converts a string of digits to an int.
func parseDigits(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
