package tools

import (
	"context"
	"strings"
	"testing"
)

// Ensure imports are used.
var (
	_ = context.Background
	_ = strings.Contains
)

func TestValidateToolArgs(t *testing.T) {
	baseSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		"required": []string{"name"},
	}

	tests := []struct {
		name    string
		schema  map[string]any
		args    map[string]any
		wantErr string // empty means no error expected
	}{
		{
			name:   "valid args all required present",
			schema: baseSchema,
			args:   map[string]any{"name": "alice", "age": float64(30)},
		},
		{
			name:    "missing required field",
			schema:  baseSchema,
			args:    map[string]any{"age": float64(30)},
			wantErr: "missing required property \"name\"",
		},
		{
			name:    "wrong type string field gets number",
			schema:  baseSchema,
			args:    map[string]any{"name": float64(42)},
			wantErr: "expected string",
		},
		{
			name:    "nil args with required fields",
			schema:  baseSchema,
			args:    nil,
			wantErr: "missing required property \"name\"",
		},
		{
			name: "nil args no required fields",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			args: nil,
		},
		{
			name: "empty args no required fields",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			args: map[string]any{},
		},
		{
			name:   "optional field correct type",
			schema: baseSchema,
			args:   map[string]any{"name": "bob", "age": float64(25)},
		},
		{
			name:    "optional field wrong type",
			schema:  baseSchema,
			args:    map[string]any{"name": "bob", "age": "twenty"},
			wantErr: "expected integer",
		},
		{
			name:   "integer as float64 no fractional part",
			schema: baseSchema,
			args:   map[string]any{"name": "carol", "age": float64(42)},
		},
		{
			name:    "actual float for integer field",
			schema:  baseSchema,
			args:    map[string]any{"name": "dave", "age": float64(42.5)},
			wantErr: "expected integer, got float64 with fractional part",
		},
		{
			name: "number type accepts float",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"score": map[string]any{"type": "number"},
				},
			},
			args: map[string]any{"score": float64(3.14)},
		},
		{
			name: "number type accepts integer",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"score": map[string]any{"type": "number"},
				},
			},
			args: map[string]any{"score": float64(10)},
		},
		{
			name: "boolean type valid",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"flag": map[string]any{"type": "boolean"},
				},
			},
			args: map[string]any{"flag": true},
		},
		{
			name: "boolean type wrong",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"flag": map[string]any{"type": "boolean"},
				},
			},
			args:    map[string]any{"flag": "true"},
			wantErr: "expected boolean",
		},
		{
			name: "required as []any from MCP deserialization",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cmd": map[string]any{"type": "string"},
				},
				"required": []any{"cmd"},
			},
			args:    map[string]any{},
			wantErr: "missing required property \"cmd\"",
		},
		{
			name: "enum valid value []any",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []any{"red", "green", "blue"}},
				},
			},
			args: map[string]any{"color": "red"},
		},
		{
			name: "enum invalid value []any",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []any{"red", "green", "blue"}},
				},
			},
			args:    map[string]any{"color": "yellow"},
			wantErr: "not in enum",
		},
		{
			name: "enum valid value []string",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []string{"red", "green", "blue"}},
				},
			},
			args: map[string]any{"color": "green"},
		},
		{
			name: "enum invalid value []string",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []string{"red", "green", "blue"}},
				},
			},
			args:    map[string]any{"color": "yellow"},
			wantErr: "not in enum",
		},
		{
			name:    "extra unexpected property rejected",
			schema:  baseSchema,
			args:    map[string]any{"name": "eve", "hobby": "chess"},
			wantErr: "unexpected property \"hobby\"",
		},
		{
			name: "extra property allowed with additionalProperties true",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"additionalProperties": true,
			},
			args: map[string]any{"name": "eve", "hobby": "chess"},
		},
		{
			name: "nested object valid",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"address": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]any{"type": "string"},
						},
						"required": []string{"city"},
					},
				},
			},
			args: map[string]any{
				"address": map[string]any{"city": "Berlin"},
			},
		},
		{
			name: "nested object wrong type",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"address": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]any{"type": "string"},
						},
					},
				},
			},
			args:    map[string]any{"address": "not an object"},
			wantErr: "expected object",
		},
		{
			name: "array with valid element types",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
			},
			args: map[string]any{"tags": []any{"a", "b", "c"}},
		},
		{
			name: "array with wrong element types",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
			},
			args:    map[string]any{"tags": []any{"a", float64(2)}},
			wantErr: "expected string",
		},
		{
			name: "schema with no properties key accepts any args",
			schema: map[string]any{
				"type": "object",
			},
			args: map[string]any{"anything": "goes"},
		},
		{
			name:   "empty schema accepts anything",
			schema: map[string]any{},
			args:   map[string]any{"foo": "bar"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateToolArgs(tc.schema, tc.args)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateToolArgs_RegistryIntegration(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockRegistryTool{
		name: "read_file",
		desc: "reads a file",
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		result: SilentResult("file contents"),
	})

	// Valid args — should succeed
	result := r.Execute(context.Background(), "read_file", map[string]any{"path": "/tmp/x"})
	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}

	// Missing required field — should fail with validation error
	result = r.Execute(context.Background(), "read_file", map[string]any{})
	if !result.IsError {
		t.Error("expected validation error for missing required field")
	}
	if !strings.Contains(result.ForLLM, "missing required p") {
		t.Errorf("expected 'missing required p...' in error, got %q", result.ForLLM)
	}
	if result.Err == nil {
		t.Error("expected Err to be set via WithError")
	}

	// Wrong type — should fail with validation error
	result = r.Execute(context.Background(), "read_file", map[string]any{"path": 123.0})
	if !result.IsError {
		t.Error("expected validation error for wrong type")
	}
	if !strings.Contains(result.ForLLM, "expected string") {
		t.Errorf("expected 'expected string' in error, got %q", result.ForLLM)
	}

	// Extra property — should fail with validation error
	result = r.Execute(context.Background(), "read_file", map[string]any{"path": "/x", "__inject": true})
	if !result.IsError {
		t.Error("expected validation error for extra property")
	}
	if !strings.Contains(result.ForLLM, "unexpected prop") {
		t.Errorf("expected 'unexpected prop...' in error, got %q", result.ForLLM)
	}
}

func TestValidateToolArgs_RealSchemas(t *testing.T) {
	execSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command":     map[string]any{"type": "string"},
			"working_dir": map[string]any{"type": "string"},
		},
		"required": []string{"command"},
	}

	cronSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []any{"add", "list", "remove", "enable", "disable"},
			},
		},
		"required": []string{"action"},
	}

	webSearchSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"count": map[string]any{"type": "integer"},
		},
		"required": []string{"query"},
	}

	tests := []struct {
		name    string
		schema  map[string]any
		args    map[string]any
		wantErr string
	}{
		// ExecTool
		{
			name:   "exec valid args",
			schema: execSchema,
			args:   map[string]any{"command": "ls -la", "working_dir": "/tmp"},
		},
		{
			name:    "exec missing required command",
			schema:  execSchema,
			args:    map[string]any{"working_dir": "/tmp"},
			wantErr: "missing required property \"command\"",
		},
		{
			name:    "exec wrong type for command",
			schema:  execSchema,
			args:    map[string]any{"command": float64(123)},
			wantErr: "expected string",
		},
		{
			name:    "exec extra injected arg",
			schema:  execSchema,
			args:    map[string]any{"command": "ls", "malicious": "payload"},
			wantErr: "unexpected property \"malicious\"",
		},

		// CronTool
		{
			name:   "cron valid enum value",
			schema: cronSchema,
			args:   map[string]any{"action": "add"},
		},
		{
			name:    "cron invalid enum value",
			schema:  cronSchema,
			args:    map[string]any{"action": "destroy"},
			wantErr: "not in enum",
		},

		// WebSearchTool
		{
			name:   "websearch valid args",
			schema: webSearchSchema,
			args:   map[string]any{"query": "golang testing", "count": float64(10)},
		},
		{
			name:    "websearch missing required query",
			schema:  webSearchSchema,
			args:    map[string]any{"count": float64(5)},
			wantErr: "missing required property \"query\"",
		},
		{
			name:    "websearch wrong type for count",
			schema:  webSearchSchema,
			args:    map[string]any{"query": "test", "count": "ten"},
			wantErr: "expected integer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateToolArgs(tc.schema, tc.args)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
