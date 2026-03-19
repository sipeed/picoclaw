package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestInjectImageDescriptions_ReplacesTag(t *testing.T) {
	content := "Check this out [image: photo]"
	result := injectImageDescriptions(content, []string{"a cat sitting on a table"})
	want := "Check this out [image: a cat sitting on a table]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_MultipleImages(t *testing.T) {
	content := "Here are two photos [image: photo] and [image: photo]"
	result := injectImageDescriptions(content, []string{"a dog", "a cat"})
	want := "Here are two photos [image: a dog] and [image: a cat]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_NoTag(t *testing.T) {
	content := "Hello"
	result := injectImageDescriptions(content, []string{"a sunset"})
	want := "Hello [image: a sunset]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_EmptyContent(t *testing.T) {
	result := injectImageDescriptions("", []string{"a chart"})
	want := "[image: a chart]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestDescribeImages_NoImageCandidates(t *testing.T) {
	// When no ImageCandidates are configured, messages should pass through unchanged.
	messages := []providers.Message{
		{
			Role:    "user",
			Content: "[image: photo]",
			Media:   []string{"data:image/jpeg;base64,/9j/4AAQ"},
		},
	}
	agent := &AgentInstance{
		ImageCandidates: nil,
	}

	// With no candidates, describeImagesInMessages should still work
	// but return "description unavailable" for each image.
	al := &AgentLoop{}
	result := al.describeImagesInMessages(t.Context(), messages, agent, "", "")

	// The data URL should be removed from Media
	if len(result[0].Media) != 0 {
		t.Errorf("expected empty media, got %v", result[0].Media)
	}
	// Content should have the placeholder
	if result[0].Content != "[image: description unavailable]" {
		t.Errorf("content = %q", result[0].Content)
	}
}

func TestDescribeImages_SkippedInPlanMode(t *testing.T) {
	// isPlanPreExecution should return true for interviewing/review
	if !isPlanPreExecution("interviewing") {
		t.Error("interviewing should be pre-execution")
	}
	if !isPlanPreExecution("review") {
		t.Error("review should be pre-execution")
	}
	if isPlanPreExecution("executing") {
		t.Error("executing should not be pre-execution")
	}
	if isPlanPreExecution("") {
		t.Error("empty should not be pre-execution")
	}
}

func TestResolveImageModel_FallsToPlanModel(t *testing.T) {
	// When ImageModel is empty, should fall back to PlanModel
	model := resolveImageModel(nil, &config.AgentDefaults{
		PlanModel: "openai/gpt-5.4",
	})
	if model != "openai/gpt-5.4" {
		t.Errorf("got %q, want %q", model, "openai/gpt-5.4")
	}

	// When ImageModel is set, should use it
	model = resolveImageModel(nil, &config.AgentDefaults{
		ImageModel: "openai/gpt-5.4-nano",
		PlanModel:  "openai/gpt-5.4",
	})
	if model != "openai/gpt-5.4-nano" {
		t.Errorf("got %q, want %q", model, "openai/gpt-5.4-nano")
	}
}
