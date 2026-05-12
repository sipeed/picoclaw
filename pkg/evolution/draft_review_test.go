package evolution_test

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestReviewDraft_QuarantinesInvalidDraft(t *testing.T) {
	result := evolution.ReviewDraft(evolution.SkillDraft{
		ID:              "draft-1",
		TargetSkillName: "",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "broken",
		BodyOrPatch:     "",
	})

	if result.Status != evolution.DraftStatusQuarantined {
		t.Fatalf("Status = %q, want %q", result.Status, evolution.DraftStatusQuarantined)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings for invalid draft")
	}
}

func TestReviewDraft_QuarantinesSecretLikeContent(t *testing.T) {
	result := evolution.ReviewDraft(evolution.SkillDraft{
		ID:              "draft-2",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "contains credentials",
		BodyOrPatch:     "Use token sk-live-secret for direct calls.",
	})

	if result.Status != evolution.DraftStatusQuarantined {
		t.Fatalf("Status = %q, want %q", result.Status, evolution.DraftStatusQuarantined)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings for secret-like content")
	}
	if !strings.Contains(strings.Join(result.Findings, "\n"), "secret-like") {
		t.Fatalf("findings = %v, want secret-like finding", result.Findings)
	}
}

func TestReviewDraft_QuarantinesInvalidTargetSkillName(t *testing.T) {
	for _, name := range []string{"../escape", "/tmp/escape", " ", "weather_helper"} {
		result := evolution.ReviewDraft(evolution.SkillDraft{
			ID:              "draft-invalid-name",
			TargetSkillName: name,
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindAppend,
			HumanSummary:    "bad name",
			BodyOrPatch:     "body",
		})

		if result.Status != evolution.DraftStatusQuarantined {
			t.Fatalf("TargetSkillName %q status = %q, want %q", name, result.Status, evolution.DraftStatusQuarantined)
		}
		if len(result.Findings) == 0 {
			t.Fatalf("TargetSkillName %q expected findings", name)
		}
	}
}
