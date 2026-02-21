package miniapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// buildInitData constructs a valid initData string from params and a bot token.
func buildInitData(params map[string]string, botToken string) string {
	// Build data-check-string
	var pairs []string
	for k, v := range params {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// Compute secret key
	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	// Compute hash
	hashMac := hmac.New(sha256.New, secretKey)
	hashMac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(hashMac.Sum(nil))

	// Build query string
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("hash", hash)
	return values.Encode()
}

func TestValidateInitData(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

	t.Run("valid initData", func(t *testing.T) {
		params := map[string]string{
			"query_id": "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":     `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		if !ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned false for valid data")
		}
	})

	t.Run("tampered data", func(t *testing.T) {
		params := map[string]string{
			"query_id": "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":     `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		// Tamper with the data
		initData = strings.Replace(initData, "Vlad", "Evil", 1)
		if ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned true for tampered data")
		}
	})

	t.Run("wrong bot token", func(t *testing.T) {
		params := map[string]string{
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		if ValidateInitData(initData, "wrong-token") {
			t.Error("ValidateInitData() returned true for wrong bot token")
		}
	})

	t.Run("missing hash", func(t *testing.T) {
		if ValidateInitData("auth_date=1234567890", botToken) {
			t.Error("ValidateInitData() returned true for missing hash")
		}
	})

	t.Run("empty initData", func(t *testing.T) {
		if ValidateInitData("", botToken) {
			t.Error("ValidateInitData() returned true for empty initData")
		}
	})

	t.Run("invalid query string", func(t *testing.T) {
		if ValidateInitData("%%%invalid", botToken) {
			t.Error("ValidateInitData() returned true for invalid query string")
		}
	})
}
