// Package cache provides caching functionality for LLM providers
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// CacheableProvider wraps an existing provider with caching functionality
type CacheableProvider struct {
	provider providers.LLMProvider
	cache    *CacheProvider
}

// NewCachedProvider creates a new provider with caching capabilities
func NewCachedProvider(provider providers.LLMProvider, cacheProvider *CacheProvider) *CacheableProvider {
	return &CacheableProvider{
		provider: provider,
		cache:    cacheProvider,
	}
}

// Chat method with caching logic
func (cp *CacheableProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	// Generate cache key based on request parameters
	key := cp.cache.GenerateKey(sliceAny(messages), model, sliceAny(tools), options)

	// Try to get cached response
	if cachedResponse, err := cp.cache.Get(ctx, key); err == nil {
		// We have a cached response, convert it to providers.LLMResponse
		fmt.Println("[CACHE HIT] Returning cached response for key:", key)
		return convertToProvidersResponse(cachedResponse), nil
	} else {
		fmt.Println("[CACHE MISS] Request not cached, calling provider directly")
		// No cache hit, call the underlying provider
		response, err := cp.provider.Chat(ctx, messages, tools, model, options)
		if err != nil {
			return nil, err
		}

		// Store response in cache
		cacheResponse := convertFromProvidersResponse(response)
		if cacheErr := cp.cache.Set(ctx, key, cacheResponse, nil); cacheErr != nil {
			// Log error but don't fail the operation
			fmt.Printf("Cache set error for key %s: %v\n", key, cacheErr)
		} else {
			fmt.Println("[CACHE STORE] Response cached for key:", key)
		}

		return response, nil
	}
}

// GetDefaultModel delegation
func (cp *CacheableProvider) GetDefaultModel() string {
	return cp.provider.GetDefaultModel()
}

// Helper to convert []providers.Message to []any for compatibility
func sliceAny[T any](slice []T) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

// Convert our internal LLMResponse to providers.LLMResponse
func convertToProvidersResponse(internal *LLMResponse) *providers.LLMResponse {
	usageInfo := (*providers.UsageInfo)(nil)
	if internal.Usage != nil {
		usageInfo = &providers.UsageInfo{
			PromptTokens:     internal.Usage.PromptTokens,
			CompletionTokens: internal.Usage.CompletionTokens,
			TotalTokens:      internal.Usage.TotalTokens,
		}
	}

	toolCalls := make([]providers.ToolCall, len(internal.ToolCalls))
	for i, tc := range internal.ToolCalls {
		args := make(map[string]interface{})
		for k, v := range tc.Arguments {
			args[k] = v
		}

		extraContent := (*providers.ExtraContent)(nil)
		if internal.ToolCalls[i].ExtraContent != nil && internal.ToolCalls[i].ExtraContent.Google != nil {
			extraContent = &providers.ExtraContent{
				Google: &providers.GoogleExtra{
					ThoughtSignature: internal.ToolCalls[i].ExtraContent.Google.ThoughtSignature,
				},
			}
		}

		toolCalls[i] = providers.ToolCall{
			ID:               tc.ID,
			Name:             tc.Name,
			Arguments:        args,
			ThoughtSignature: tc.ThoughtSignature,
			ExtraContent:     extraContent,
		}
	}

	reasoningDetails := make([]providers.ReasoningDetail, len(internal.ReasoningDetails))
	for i, rd := range internal.ReasoningDetails {
		reasoningDetails[i] = providers.ReasoningDetail{
			Text: rd.Text,
		}
	}

	extraContent := (*providers.ExtraContent)(nil)
	if internal.ExtraContent != nil && internal.ExtraContent.Google != nil {
		extraContent = &providers.ExtraContent{
			Google: &providers.GoogleExtra{
				ThoughtSignature: internal.ExtraContent.Google.ThoughtSignature,
			},
		}
	}

	return &providers.LLMResponse{
		Content:          internal.Content,
		ReasoningContent: internal.ReasoningContent,
		Reasoning:        internal.Reasoning,
		ReasoningDetails: reasoningDetails,
		ToolCalls:        toolCalls,
		FinishReason:     internal.FinishReason,
		Usage:            usageInfo,
		Intent:           internal.Intent,
		ExtraContent:     extraContent,
	}
}

// Convert providers.LLMResponse to our internal LLMResponse
func convertFromProvidersResponse(prov *providers.LLMResponse) *LLMResponse {
	usageInfo := (*UsageInfo)(nil)
	if prov.Usage != nil {
		usageInfo = &UsageInfo{
			PromptTokens:     prov.Usage.PromptTokens,
			CompletionTokens: prov.Usage.CompletionTokens,
			TotalTokens:      prov.Usage.TotalTokens,
		}
	}

	toolCalls := make([]ToolCall, len(prov.ToolCalls))
	for i, tc := range prov.ToolCalls {
		args := make(map[string]interface{})
		for k, v := range tc.Arguments {
			args[k] = v
		}

		extraContent := (*ExtraContent)(nil)
		if prov.ToolCalls[i].ExtraContent != nil && prov.ToolCalls[i].ExtraContent.Google != nil {
			extraContent = &ExtraContent{
				Google: &GoogleExtra{
					ThoughtSignature: prov.ToolCalls[i].ExtraContent.Google.ThoughtSignature,
				},
			}
		}

		toolCalls[i] = ToolCall{
			ID:               tc.ID,
			Name:             tc.Name,
			Arguments:        args,
			ThoughtSignature: tc.ThoughtSignature,
			ExtraContent:     extraContent,
		}
	}

	reasoningDetails := make([]ReasoningDetail, len(prov.ReasoningDetails))
	for i, rd := range prov.ReasoningDetails {
		reasoningDetails[i] = ReasoningDetail{
			Text: rd.Text,
		}
	}

	extraContent := (*ExtraContent)(nil)
	if prov.ExtraContent != nil && prov.ExtraContent.Google != nil {
		extraContent = &ExtraContent{
			Google: &GoogleExtra{
				ThoughtSignature: prov.ExtraContent.Google.ThoughtSignature,
			},
		}
	}

	return &LLMResponse{
		Content:          prov.Content,
		ReasoningContent: prov.ReasoningContent,
		Reasoning:        prov.Reasoning,
		ReasoningDetails: reasoningDetails,
		ToolCalls:        toolCalls,
		FinishReason:     prov.FinishReason,
		Usage:            usageInfo,
		Intent:           prov.Intent,
		ExtraContent:     extraContent,
	}
}
