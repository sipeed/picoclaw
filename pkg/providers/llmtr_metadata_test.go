package providers

import "testing"

func TestLLMTRProviderMetadata(t *testing.T) {
	t.Parallel()

	if !IsSupportedModelProvider("llmtr") {
		t.Fatal("llmtr should be a supported model provider")
	}
	if !IsCreatableModelProvider("llmtr") {
		t.Fatal("llmtr should be creatable from the Web UI")
	}
	if !IsDefaultModelProvider("llmtr") {
		t.Fatal("llmtr should be allowed as the default chat model")
	}
	if !IsModelProviderFetchable("llmtr") {
		t.Fatal("llmtr should allow /models fetch")
	}

	var option ModelProviderOption
	for _, candidate := range ModelProviderOptions() {
		if candidate.ID == "llmtr" {
			option = candidate
			break
		}
	}
	if option.ID == "" {
		t.Fatal("llmtr missing from ModelProviderOptions")
	}
	if option.DisplayName != "LLMTR" {
		t.Fatalf("DisplayName = %q, want LLMTR", option.DisplayName)
	}
	if option.Domain != "llmtr.com" {
		t.Fatalf("Domain = %q, want llmtr.com", option.Domain)
	}
	if option.DefaultAPIBase != "https://llmtr.com/v1" {
		t.Fatalf("DefaultAPIBase = %q, want https://llmtr.com/v1", option.DefaultAPIBase)
	}
	if option.IconSlug != "" {
		t.Fatalf("IconSlug = %q, want empty (no built-in logo)", option.IconSlug)
	}
	if len(option.CommonModels) != 0 {
		t.Fatalf("CommonModels = %v, want none", option.CommonModels)
	}
	if !option.httpAPI {
		t.Fatal("llmtr should use the shared OpenAI-compatible HTTP client")
	}

	provider, modelID := SplitModelProviderAndID("llmtr/catalog-model", "")
	if provider != "llmtr" || modelID != "catalog-model" {
		t.Fatalf("SplitModelProviderAndID = %q, %q, want llmtr, catalog-model", provider, modelID)
	}
}
