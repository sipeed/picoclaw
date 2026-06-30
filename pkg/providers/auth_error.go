package providers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

type AuthErrorKind string

const (
	AuthErrorInvalidAPIKey AuthErrorKind = "invalid_api_key"
	AuthErrorMissingAPIKey AuthErrorKind = "missing_api_key"
	AuthErrorExpiredToken  AuthErrorKind = "expired_token"
	AuthErrorGeneric       AuthErrorKind = "auth"
)

var (
	invalidAPIKeyPattern = regexp.MustCompile(
		`(?i)\b(?:invalid|incorrect|malformed|wrong)[-_\s]+(?:api[-_\s]*)?key\b|\b(?:api[-_\s]*)?key[-_\s]+(?:is[-_\s]+)?(?:invalid|incorrect|malformed|wrong)\b|\binvalid[-_]?api[-_]?key\b`,
	)
	missingAPIKeyPattern = regexp.MustCompile(
		`(?i)\b(?:missing|no|not[-_\s]+configured)[-_\s]+(?:api[-_\s]*)?key\b|\bno[-_\s]+credentials[-_\s]+found\b|\bapi[-_\s]+key[-_\s]+not[-_\s]+configured\b`,
	)
	expiredTokenPattern = regexp.MustCompile(
		`(?i)\b(?:oauth[-_\s]+token[-_\s]+refresh[-_\s]+failed|re-authenticate|token[-_\s]+(?:has[-_\s]+)?expired|expired[-_\s]+token)\b`,
	)
	genericAuthPattern = regexp.MustCompile(
		`(?i)\b(?:unauthorized|forbidden|authentication[-_\s]+(?:failed|required)|access[-_\s]+denied)\b`,
	)
)

func ClassifyAuthError(err error) (AuthErrorKind, bool) {
	if err == nil {
		return "", false
	}

	var exhausted *FallbackExhaustedError
	if errors.As(err, &exhausted) && exhausted != nil {
		return classifyFallbackExhaustedAuthError(exhausted)
	}

	msg := authErrorText(err)
	if missingAPIKeyPattern.MatchString(msg) {
		return AuthErrorMissingAPIKey, true
	}
	if expiredTokenPattern.MatchString(msg) {
		return AuthErrorExpiredToken, true
	}
	if invalidAPIKeyPattern.MatchString(msg) {
		return AuthErrorInvalidAPIKey, true
	}

	if hasStructuredAuthError(err) || genericAuthPattern.MatchString(msg) {
		return AuthErrorGeneric, true
	}
	return "", false
}

func authErrorText(err error) string {
	var parts []string
	if err != nil {
		parts = append(parts, err.Error())
	}

	var httpErr *common.HTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		parts = append(parts, httpErr.BodyPreview)
	}

	return strings.Join(parts, "\n")
}

func hasStructuredAuthError(err error) bool {
	var failErr *FailoverError
	if errors.As(err, &failErr) && failErr != nil && failErr.Reason == FailoverAuth {
		return true
	}

	var httpErr *common.HTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		return httpErr.StatusCode == 401 || httpErr.StatusCode == 403
	}

	return false
}

func classifyFallbackExhaustedAuthError(err *FallbackExhaustedError) (AuthErrorKind, bool) {
	if err == nil {
		return "", false
	}

	var authMessages []string
	nonSkipped := 0
	for _, attempt := range err.Attempts {
		if attempt.Skipped {
			continue
		}
		nonSkipped++
		if !attemptIsAuthFailure(attempt) {
			return "", false
		}
		if attempt.Error != nil {
			authMessages = append(authMessages, authErrorText(attempt.Error))
		}
	}
	if nonSkipped == 0 {
		return "", false
	}

	msg := strings.Join(authMessages, "\n")
	if missingAPIKeyPattern.MatchString(msg) {
		return AuthErrorMissingAPIKey, true
	}
	if expiredTokenPattern.MatchString(msg) {
		return AuthErrorExpiredToken, true
	}
	if invalidAPIKeyPattern.MatchString(msg) {
		return AuthErrorInvalidAPIKey, true
	}
	return AuthErrorGeneric, true
}

func attemptIsAuthFailure(attempt FallbackAttempt) bool {
	if attempt.Reason == FailoverAuth {
		return true
	}
	if attempt.Error == nil {
		return false
	}
	var failErr *FailoverError
	if errors.As(attempt.Error, &failErr) && failErr != nil && failErr.Reason == FailoverAuth {
		return true
	}
	var httpErr *common.HTTPError
	if errors.As(attempt.Error, &httpErr) && httpErr != nil {
		return httpErr.StatusCode == 401 || httpErr.StatusCode == 403
	}
	return false
}
