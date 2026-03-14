// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk access token management

package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	dingtalkAPIBase      = "https://api.dingtalk.com"
	tokenRefreshInterval = 5 * time.Minute
	tokenExpireBuffer    = 5 * time.Minute // Refresh token 5 minutes before expiry
)

// accessTokenResponse 钉钉 access_token 响应
type accessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireIn    int64  `json:"expireIn"` // seconds
}

// refreshAccessToken refreshes the access token from DingTalk API
func (c *DingTalkChannel) refreshAccessToken() error {
	url := fmt.Sprintf("%s/v1.0/oauth2/accessToken", dingtalkAPIBase)

	body := map[string]string{
		"appKey":    c.clientID,
		"appSecret": c.clientSecret,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp accessTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("empty access token in response")
	}

	c.tokenMu.Lock()
	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpireIn) * time.Second)
	c.tokenMu.Unlock()

	logger.DebugCF("dingtalk", "Access token refreshed successfully", map[string]any{
		"expires_in": tokenResp.ExpireIn,
	})
	return nil
}

// getAccessToken returns the current valid access token
// Returns empty string if token is expired or about to expire
func (c *DingTalkChannel) getAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()

	if c.accessToken == "" {
		return ""
	}

	// Check if token is about to expire (within buffer time)
	if time.Now().After(c.tokenExpiry.Add(-tokenExpireBuffer)) {
		return "" // Token expired or about to expire
	}

	return c.accessToken
}

// ensureValidToken ensures we have a valid access token, refreshing if necessary
func (c *DingTalkChannel) ensureValidToken(ctx context.Context) (string, error) {
	token := c.getAccessToken()
	if token != "" {
		return token, nil
	}

	// Need to refresh
	if err := c.refreshAccessToken(); err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}

	return c.getAccessToken(), nil
}

// tokenRefreshLoop runs periodically to refresh the access token
func (c *DingTalkChannel) tokenRefreshLoop() {
	ticker := time.NewTicker(tokenRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.config.ProactiveSend {
				// Check if token needs refresh
				if c.getAccessToken() == "" {
					if err := c.refreshAccessToken(); err != nil {
						logger.ErrorCF("dingtalk", "Failed to refresh access token", map[string]any{
							"error": err.Error(),
						})
					}
				}
			}
		}
	}
}
