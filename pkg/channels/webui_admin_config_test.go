package channels

import (
	"encoding/json"
	"testing"
)

func TestMaskConfigJSON_MasksSensitiveValuesPrefix8(t *testing.T) {
	in := []byte(`{
		"gateway": {
			"token": "12345678ABCDEFGH",
			"admin_token": "ABCDEFGH12345678"
		},
		"providers": {
			"openai": {"api_key": "sk-12345678ZZZZ", "api_base": ""}
		},
		"channels": {
			"telegram": {"token": "99999999TTTT"}
		}
	}`)

	out, err := maskConfigJSON(in)
	if err != nil {
		t.Fatalf("maskConfigJSON error: %v", err)
	}

	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("unmarshal masked output: %v", err)
	}

	gateway := v["gateway"].(map[string]any)
	if got := gateway["token"].(string); got != "12345678********" {
		t.Fatalf("gateway.token = %q, want %q", got, "12345678********")
	}
	if got := gateway["admin_token"].(string); got != "ABCDEFGH********" {
		t.Fatalf("gateway.admin_token = %q, want %q", got, "ABCDEFGH********")
	}

	providers := v["providers"].(map[string]any)
	openai := providers["openai"].(map[string]any)
	if got := openai["api_key"].(string); got != "sk-12345********" {
		t.Fatalf("providers.openai.api_key = %q, want %q", got, "sk-12345********")
	}

	channels := v["channels"].(map[string]any)
	tg := channels["telegram"].(map[string]any)
	if got := tg["token"].(string); got != "99999999********" {
		t.Fatalf("channels.telegram.token = %q, want %q", got, "99999999********")
	}
}

func TestMergePreserveMaskedSecrets_PreservesOnlyMaskedNotEmpty(t *testing.T) {
	oldRaw := []byte(`{
		"gateway": {"token": "TOK_OLD_1234567890", "admin_token": "ADM_OLD_1234567890"},
		"providers": {"openai": {"api_key": "sk-OLD-1234567890", "api_base": ""}},
		"channels": {"telegram": {"token": "TG_OLD_1234567890"}}
	}`)

	newRaw := []byte(`{
		"gateway": {"token": "TOK_OLD_********", "admin_token": ""},
		"providers": {"openai": {"api_key": "sk-OLD-1********", "api_base": ""}},
		"channels": {"telegram": {"token": "TG_NEW_123"}}
	}`)

	merged, err := mergePreserveMaskedSecrets(oldRaw, newRaw)
	if err != nil {
		t.Fatalf("mergePreserveMaskedSecrets error: %v", err)
	}

	var v map[string]any
	if err := json.Unmarshal(merged, &v); err != nil {
		t.Fatalf("unmarshal merged: %v", err)
	}

	gateway := v["gateway"].(map[string]any)
	if got := gateway["token"].(string); got != "TOK_OLD_1234567890" {
		t.Fatalf("gateway.token = %q, want old preserved", got)
	}
	// empty string should not be treated as "keep"; it should remain empty (delete)
	if got := gateway["admin_token"].(string); got != "" {
		t.Fatalf("gateway.admin_token = %q, want empty string", got)
	}

	providers := v["providers"].(map[string]any)
	openai := providers["openai"].(map[string]any)
	if got := openai["api_key"].(string); got != "sk-OLD-1234567890" {
		t.Fatalf("providers.openai.api_key = %q, want old preserved", got)
	}

	channels := v["channels"].(map[string]any)
	tg := channels["telegram"].(map[string]any)
	if got := tg["token"].(string); got != "TG_NEW_123" {
		t.Fatalf("channels.telegram.token = %q, want new value", got)
	}
}
