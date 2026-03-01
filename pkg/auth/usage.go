package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicUsage struct {
	FiveHourUtilization float64
	SevenDayUtilization float64
}

func FetchAnthropicUsage(token string) (*AnthropicUsage, error) {
	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("insufficient scope: usage endpoint requires oauth scope")
	}

	if resp.StatusCode != http.StatusOK {
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
