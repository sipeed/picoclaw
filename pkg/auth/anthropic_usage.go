package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	anthropicBetaHeader = "oauth-2025-04-20"
	anthropicAPIVersion = "2023-06-01"
)

// anthropicUsageURL is the endpoint for fetching OAuth usage stats.
// It is a var (not const) to allow overriding in tests.
var anthropicUsageURL = "https://api.anthropic.com/api/oauth/usage"

func setAnthropicUsageURL(url string) { anthropicUsageURL = url }

type AnthropicUsage struct {
	FiveHourUtilization float64
	SevenDayUtilization float64
}

func FetchAnthropicUsage(token string) (*AnthropicUsage, error) {
	req, err := http.NewRequest("GET", anthropicUsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Anthropic-Version", anthropicAPIVersion)
	req.Header.Set("Anthropic-Beta", anthropicBetaHeader)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading usage response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("insufficient scope: usage endpoint requires oauth scope")
		}
		return nil, fmt.Errorf("usage request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		FiveHour struct {
			Utilization float64 `json:"utilization"`
		} `json:"five_hour"`
		SevenDay struct {
			Utilization float64 `json:"utilization"`
		} `json:"seven_day"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing usage response: %w", err)
	}

	return &AnthropicUsage{
		FiveHourUtilization: result.FiveHour.Utilization,
		SevenDayUtilization: result.SevenDay.Utilization,
	}, nil
}
