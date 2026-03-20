package miniapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// initDataMaxAge is the maximum age of initData before it is considered expired.
const initDataMaxAge = 24 * time.Hour

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		initData := r.URL.Query().Get("initData")
		if initData == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing initData")
			return
		}
		if !ValidateInitData(initData, h.botToken) {
			writeJSONError(w, http.StatusUnauthorized, "invalid initData")
			return
		}
		if len(h.allowList) > 0 {
			userID, _ := extractUserFromInitData(initData)
			if userID == "" || !isAllowed(userID, h.allowList) {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
		}
		next(w, r)
	}
}

// isAllowed checks whether userID matches any entry in the allow list.
// Logic mirrors BaseChannel.IsAllowed without importing channels package.
func isAllowed(userID string, allowList []string) bool {
	if len(allowList) == 0 {
		return true
	}
	for _, allowed := range allowList {
		trimmed := strings.TrimPrefix(allowed, "@")
		allowedID := trimmed
		if idx := strings.Index(trimmed, "|"); idx > 0 {
			allowedID = trimmed[:idx]
		}
		if userID == allowed || userID == trimmed || userID == allowedID {
			return true
		}
	}
	return false
}

// extractUserFromInitData parses user.id from the initData query string.
// initData contains a "user" param with JSON like {"id":123456,...}.
func extractUserFromInitData(initData string) (userID, chatID string) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return "", ""
	}
	userJSON := values.Get("user")
	if userJSON == "" {
		return "", ""
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil || user.ID == 0 {
		return "", ""
	}
	id := fmt.Sprintf("%d", user.ID)
	// For Mini App commands, chatID = userID (private chat)
	return id, id
}

// ValidateInitData verifies the Telegram WebApp initData HMAC-SHA256 signature
// and checks that auth_date is not older than initDataMaxAge.
// See https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func ValidateInitData(initData, botToken string) bool {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return false
	}

	// Check auth_date freshness
	if authDateStr := values.Get("auth_date"); authDateStr != "" {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err != nil {
			return false
		}
		if time.Since(time.Unix(authDate, 0)) > initDataMaxAge {
			return false
		}
	}

	// Build the data-check-string: sort all key=value pairs except "hash",
	// join with newlines.
	var pairs []string
	for key := range values {
		if key == "hash" {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, values.Get(key)))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC-SHA256("WebAppData", bot_token)
	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	// hash = HMAC-SHA256(secret_key, data_check_string)
	hashMac := hmac.New(sha256.New, secretKey)
	hashMac.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(hashMac.Sum(nil))

	return hmac.Equal([]byte(computedHash), []byte(receivedHash))
}
