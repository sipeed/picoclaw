package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
	Created    int64 `json:"created"`
	Score      float64
	LastError  string
	IsReachable bool
}

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: OPENROUTER_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	fmt.Println("🔍 Fetching all models from OpenRouter...")
	models, err := fetchModels(apiKey)
	if err != nil {
		fmt.Printf("❌ Failed to fetch models: %v\n", err)
		os.Exit(1)
	}

	var freeModels []Model
	for _, m := range models {
		if m.Pricing.Prompt == "0" && m.Pricing.Completion == "0" {
			// Scoring logic (same as tool)
			score := 0.0
			score += float64(m.ContextLength) / 128000.0 * 0.4
			if m.Created > 0 {
				ageInDays := float64(time.Now().Unix()-m.Created) / 86400.0
				if ageInDays < 365 {
					score += (1.0 - ageInDays/365.0) * 0.2
				}
			}
			m.Score = score
			freeModels = append(freeModels, m)
		}
	}

	sort.Slice(freeModels, func(i, j int) bool {
		return freeModels[i].Score > freeModels[j].Score
	})

	fmt.Printf("✅ Found %d free models. Testing connectivity until we find 3 working ones...\n\n", len(freeModels))

	successCount := 0
	for i := range freeModels {
		if successCount >= 3 {
			break
		}
		m := &freeModels[i]
		fmt.Printf("[%d/%d] Testing %s... ", i+1, len(freeModels), m.ID)
		
		err := testModel(apiKey, m.ID)
		if err == nil {
			m.IsReachable = true
			successCount++
			fmt.Println("✅ OK")
		} else {
			m.LastError = err.Error()
			fmt.Printf("❌ FAIL (%v)\n", err)
		}
	}

	fmt.Println("\n--- FINAL RECOMMENDATIONS ---")
	header := fmt.Sprintf("%-50s | %-15s | %-10s", "Model ID", "Context", "Status")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))

	for i, m := range freeModels {
		if i >= 10 {
			break
		}
		status := "Unknown"
		if i < 5 {
			if m.IsReachable {
				status = "✅ OK"
			} else {
				status = "❌ FAIL"
			}
		}
		fmt.Printf("%-50s | %-15d | %-10s\n", m.ID, m.ContextLength, status)
	}

	for _, m := range freeModels {
		if m.IsReachable {
			fmt.Printf("\n🚀 SUCCESS! Use this model for testing: \n   go run cmd/picoclaw/main.go agent --model openrouter/%s\n", m.ID)
			break
		}
	}
}

func fetchModels(apiKey string) ([]Model, error) {
	req, _ := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []Model `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func testModel(apiKey, modelID string) error {
	payload := map[string]any{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 10,
	}
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	
	return nil
}
