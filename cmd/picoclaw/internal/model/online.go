package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type modelEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type modelsAPIResponse struct {
	Data []modelEntry `json:"data"`
}

// fetchOpenAIModels GETs <baseURL>/models with Bearer auth and accepts both the
// {data:[…]} envelope and a bare array shape used by various OpenAI-compatible servers.
func fetchOpenAIModels(baseURL, apiKey string) ([]modelEntry, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api base is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}

	url := strings.TrimRight(baseURL, "/") + "/models"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// {"data": [...]} envelope. Distinguish "envelope shape with empty list"
	// from "object without a data key" via Data being non-nil after unmarshal:
	// json.Unmarshal sets Data to []modelEntry{} for `{"data":[]}` but leaves
	// it as nil when "data" is absent or null.
	var envelope modelsAPIResponse
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Data != nil {
		return envelope.Data, nil
	}

	// Bare-array shape, including `[]`.
	var arr []modelEntry
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}

	preview := body
	if len(preview) > 256 {
		preview = preview[:256]
	}
	return nil, fmt.Errorf("decode response: unrecognized shape: %s", strings.TrimSpace(string(preview)))
}
