package initcmd

import (
	"testing"
)

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://api.openai.com/v1", "openai"},
		{"https://api.anthropic.com/v1", "anthropic"},
		{"https://api.deepseek.com/v1", "deepseek"},
		{"https://generativelanguage.googleapis.com/v1beta", "gemini"},
		{"https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen"},
		{"https://open.bigmodel.cn/api/paas/v4", "zhipu"},
		{"https://api.moonshot.cn/v1", "moonshot"},
		{"https://openrouter.ai/api/v1", "openrouter"},
		{"https://api.groq.com/openai/v1", "groq"},
		{"http://localhost:11434/v1", "ollama"},
		{"https://api.mistral.ai/v1", "mistral"},
		{"https://api.cerebras.ai/v1", "cerebras"},
		{"https://integrate.api.nvidia.com/v1", "nvidia"},
		{"https://ark.cn-beijing.volces.com/api/v3", "volcengine"},
		{"https://some-custom-endpoint.com/v1", "openai"}, // default
	}

	for _, tt := range tests {
		got := detectProtocol(tt.url)
		if got != tt.expected {
			t.Errorf("detectProtocol(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}
