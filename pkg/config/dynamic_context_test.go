package config

import (
	"encoding/json"
	"testing"
)

func TestDynamicContext_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.ValidateDynamicContext(); err != nil {
		t.Fatalf("ValidateDynamicContext() error = %v", err)
	}

	resolved := cfg.Agents.Defaults.ResolveDynamicContext()
	if resolved.Time != DynamicContextTimeMinute {
		t.Fatalf("default time = %q, want %q", resolved.Time, DynamicContextTimeMinute)
	}
	if resolved.Position != DynamicContextPositionTail {
		t.Fatalf("default position = %q, want %q", resolved.Position, DynamicContextPositionTail)
	}
}

// A config that predates the block must resolve to the tail-placement default,
// not to a zero value.
func TestDynamicContext_OmittedBlockResolvesToDefaults(t *testing.T) {
	var defaults AgentDefaults
	resolved := defaults.ResolveDynamicContext()

	if resolved.Time != DynamicContextTimeMinute || resolved.Position != DynamicContextPositionTail {
		t.Fatalf("resolved = %+v, want minute/tail", resolved)
	}
}

func TestDynamicContext_ParseAndResolve(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"dynamic_context": {
					"time": "HOUR",
					"position": " system "
				}
			}
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := cfg.ValidateDynamicContext(); err != nil {
		t.Fatalf("ValidateDynamicContext() error = %v", err)
	}

	resolved := cfg.Agents.Defaults.ResolveDynamicContext()
	if resolved.Time != DynamicContextTimeHour {
		t.Fatalf("time = %q, want %q", resolved.Time, DynamicContextTimeHour)
	}
	if resolved.Position != DynamicContextPositionSystem {
		t.Fatalf("position = %q, want %q", resolved.Position, DynamicContextPositionSystem)
	}
}

func TestDynamicContext_RejectsUnsupportedValues(t *testing.T) {
	tests := []struct {
		name  string
		block DynamicContextConfig
	}{
		{"bad time", DynamicContextConfig{Time: "second"}},
		{"bad position", DynamicContextConfig{Position: "middle"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Agents.Defaults.DynamicContext = tt.block
			if err := cfg.ValidateDynamicContext(); err == nil {
				t.Fatalf("ValidateDynamicContext() error = nil, want an error for %+v", tt.block)
			}
		})
	}
}

func TestDynamicContext_AcceptsEveryTimePrecision(t *testing.T) {
	for _, precision := range []DynamicContextTime{
		DynamicContextTimeMinute,
		DynamicContextTimeHour,
		DynamicContextTimeOff,
	} {
		cfg := DefaultConfig()
		cfg.Agents.Defaults.DynamicContext.Time = precision
		if err := cfg.ValidateDynamicContext(); err != nil {
			t.Fatalf("ValidateDynamicContext() error = %v for time %q", err, precision)
		}
	}
}
