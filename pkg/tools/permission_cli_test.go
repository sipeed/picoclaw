package tools

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIPermissionFunc(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{name: "approve y", input: "y\n", wantOK: true},
		{name: "approve yes", input: "yes\n", wantOK: true},
		{name: "approve Y", input: "Y\n", wantOK: true},
		{name: "approve YES", input: "YES\n", wantOK: true},
		{name: "deny n", input: "n\n", wantOK: false},
		{name: "deny empty", input: "\n", wantOK: false},
		{name: "deny other", input: "maybe\n", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			var output bytes.Buffer
			fn := NewCLIPermissionFunc(reader, &output)

			got, err := fn(context.Background(), "/Volumes/Code/project")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantOK {
				t.Errorf("got %v, want %v", got, tt.wantOK)
			}
			if !strings.Contains(output.String(), "/Volumes/Code/project") {
				t.Errorf("output should mention the path, got: %s", output.String())
			}
		})
	}
}
