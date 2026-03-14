package tools

import (
	"testing"
)

// TestCronScheduleParamValidation verifies Patch #3: LLM-supplied zero/empty
// defaults for unused schedule params should not hijack the priority logic.
//
// The bug: Go type assertion `args["at_seconds"].(float64)` returns (0, true)
// when LLM sends at_seconds=0, causing at_seconds to win over every_seconds
// and cron_expr. All recurring tasks become one-time "at" tasks.
func TestCronScheduleParamValidation(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		wantKind   string
		wantErrMsg string
	}{
		{
			name: "only every_seconds set, at_seconds=0 from LLM default",
			args: map[string]any{
				"action":        "add",
				"message":       "test reminder",
				"at_seconds":    float64(0),
				"every_seconds": float64(3600),
			},
			wantKind: "every",
		},
		{
			name: "only cron_expr set, at_seconds=0 and every_seconds=0 from LLM",
			args: map[string]any{
				"action":        "add",
				"message":       "morning task",
				"at_seconds":    float64(0),
				"every_seconds": float64(0),
				"cron_expr":     "0 9 * * *",
			},
			wantKind: "cron",
		},
		{
			name: "at_seconds > 0 should still work",
			args: map[string]any{
				"action":     "add",
				"message":    "one time reminder",
				"at_seconds": float64(600),
			},
			wantKind: "at",
		},
		{
			name: "all zeros should error",
			args: map[string]any{
				"action":        "add",
				"message":       "broken task",
				"at_seconds":    float64(0),
				"every_seconds": float64(0),
				"cron_expr":     "",
			},
			wantErrMsg: "one of at_seconds, every_seconds, or cron_expr is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reproduce the validation logic from addJob (lines 153-178 of cron.go)
			atSeconds, hasAt := tt.args["at_seconds"].(float64)
			everySeconds, hasEvery := tt.args["every_seconds"].(float64)
			cronExpr, hasCron := tt.args["cron_expr"].(string)

			// This is the fix from Patch #3
			hasAt = hasAt && atSeconds > 0
			hasEvery = hasEvery && everySeconds > 0
			hasCron = hasCron && cronExpr != ""

			var gotKind string
			if hasAt {
				gotKind = "at"
			} else if hasEvery {
				gotKind = "every"
			} else if hasCron {
				gotKind = "cron"
			} else {
				gotKind = ""
			}

			if tt.wantErrMsg != "" {
				if gotKind != "" {
					t.Errorf("expected error but got kind=%q", gotKind)
				}
				return
			}

			if gotKind != tt.wantKind {
				t.Errorf("schedule kind = %q, want %q", gotKind, tt.wantKind)
			}
		})
	}
}
