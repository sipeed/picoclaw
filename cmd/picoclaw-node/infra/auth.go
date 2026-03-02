package infra

import (
	"fmt"
	"strings"
)

// BuildDeviceAuthPayload builds the string that gets signed for connect device auth
// Format: v2|deviceId|clientId|clientMode|role|scopes|signedAtMs|token|nonce
func BuildDeviceAuthPayload(params struct {
	DeviceID   string
	ClientID   string
	ClientMode string
	Role       string
	Scopes     []string
	SignedAtMs int64
	Token      string
	Nonce      string
}) string {
	scopes := strings.Join(params.Scopes, ",")
	if params.Token == "" && params.Nonce == "" {
		return ""
	}
	return strings.Join([]string{
		"v2",
		params.DeviceID,
		params.ClientID,
		params.ClientMode,
		params.Role,
		scopes,
		fmt.Sprintf("%d", params.SignedAtMs),
		params.Token,
		params.Nonce,
	}, "|")
}
