package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestLoadEnvFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expected  map[string]string
		expectErr bool
	}{
		{
			name: "basic env file",
			content: `API_KEY=secret123
DATABASE_URL=postgres://localhost/db
PORT=8080`,
			expected: map[string]string{
				"API_KEY":      "secret123",
				"DATABASE_URL": "postgres://localhost/db",
				"PORT":         "8080",
			},
			expectErr: false,
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
API_KEY=secret123

# Another comment
DATABASE_URL=postgres://localhost/db

PORT=8080`,
			expected: map[string]string{
				"API_KEY":      "secret123",
				"DATABASE_URL": "postgres://localhost/db",
				"PORT":         "8080",
			},
			expectErr: false,
		},
		{
			name: "with quoted values",
			content: `API_KEY="secret with spaces"
NAME='single quoted'
PLAIN=no-quotes`,
			expected: map[string]string{
				"API_KEY": "secret with spaces",
				"NAME":    "single quoted",
				"PLAIN":   "no-quotes",
			},
			expectErr: false,
		},
		{
			name: "with spaces around equals",
			content: `API_KEY = secret123
DATABASE_URL= postgres://localhost/db
PORT =8080`,
			expected: map[string]string{
				"API_KEY":      "secret123",
				"DATABASE_URL": "postgres://localhost/db",
				"PORT":         "8080",
			},
			expectErr: false,
		},
		{
			name:      "invalid format - no equals",
			content:   `INVALID_LINE`,
			expectErr: true,
		},
		{
			name:      "empty file",
			content:   ``,
			expected:  map[string]string{},
			expectErr: false,
		},
		{
			name: "only comments",
			content: `# Comment 1
# Comment 2`,
			expected:  map[string]string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".env")

			if err := os.WriteFile(envFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := loadEnvFile(envFile)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d variables, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Expected key %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %s: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLoadEnvFileNotFound(t *testing.T) {
	_, err := loadEnvFile("/nonexistent/file.env")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestEnvFilePriority(t *testing.T) {
	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	envContent := `API_KEY=from_file
DATABASE_URL=from_file
SHARED_VAR=from_file`

	if err := os.WriteFile(envFile, []byte(envContent), 0o644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Load envFile
	envVars, err := loadEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to load env file: %v", err)
	}

	// Verify envFile variables
	if envVars["API_KEY"] != "from_file" {
		t.Errorf("Expected API_KEY=from_file, got %s", envVars["API_KEY"])
	}

	// Simulate config.Env overriding envFile
	configEnv := map[string]string{
		"SHARED_VAR": "from_config",
		"NEW_VAR":    "from_config",
	}

	// Merge: envFile first, then config overrides
	merged := make(map[string]string)
	for k, v := range envVars {
		merged[k] = v
	}
	for k, v := range configEnv {
		merged[k] = v
	}

	// Verify priority: config.Env should override envFile
	if merged["SHARED_VAR"] != "from_config" {
		t.Errorf(
			"Expected SHARED_VAR=from_config (config should override file), got %s",
			merged["SHARED_VAR"],
		)
	}
	if merged["API_KEY"] != "from_file" {
		t.Errorf("Expected API_KEY=from_file, got %s", merged["API_KEY"])
	}
	if merged["NEW_VAR"] != "from_config" {
		t.Errorf("Expected NEW_VAR=from_config, got %s", merged["NEW_VAR"])
	}
}

func TestLoadFromMCPConfig_EmptyWorkspaceWithRelativeEnvFile(t *testing.T) {
	mgr := NewManager()

	mcpCfg := config.MCPConfig{
		Enabled: true,
		Servers: map[string]config.MCPServerConfig{
			"test-server": {
				Enabled: true,
				Command: "echo",
				Args:    []string{"ok"},
				EnvFile: ".env",
			},
		},
	}

	err := mgr.LoadFromMCPConfig(context.Background(), mcpCfg, "")
	if err == nil {
		t.Fatal("expected error for relative env_file with empty workspace path, got nil")
	}

	if !strings.Contains(err.Error(), "workspace path is empty") {
		t.Fatalf("expected workspace path validation error, got: %v", err)
	}
}

func TestNewManager_InitialState(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("expected manager instance, got nil")
	}
	if len(mgr.GetServers()) != 0 {
		t.Fatalf("expected no servers on new manager, got %d", len(mgr.GetServers()))
	}
}

func TestLoadFromMCPConfig_DisabledOrEmptyServers(t *testing.T) {
	mgr := NewManager()

	err := mgr.LoadFromMCPConfig(context.Background(), config.MCPConfig{Enabled: false}, "/tmp")
	if err != nil {
		t.Fatalf("expected nil error when MCP disabled, got: %v", err)
	}

	err = mgr.LoadFromMCPConfig(context.Background(), config.MCPConfig{Enabled: true}, "/tmp")
	if err != nil {
		t.Fatalf("expected nil error when no servers configured, got: %v", err)
	}
}

func TestGetServers_ReturnsCopy(t *testing.T) {
	mgr := NewManager()
	mgr.servers["s1"] = &ServerConnection{Name: "s1"}

	servers := mgr.GetServers()
	delete(servers, "s1")

	if _, ok := mgr.GetServer("s1"); !ok {
		t.Fatal("expected internal manager state to remain unchanged")
	}
}

func TestGetAllTools_FiltersEmptyTools(t *testing.T) {
	mgr := NewManager()
	mgr.servers["empty"] = &ServerConnection{Name: "empty", Tools: nil}
	mgr.servers["with-tools"] = &ServerConnection{Name: "with-tools", Tools: []*sdkmcp.Tool{{}}}

	all := mgr.GetAllTools()
	if _, ok := all["empty"]; ok {
		t.Fatal("expected server without tools to be excluded")
	}
	if _, ok := all["with-tools"]; !ok {
		t.Fatal("expected server with tools to be included")
	}
}

func TestCallTool_ErrorsForClosedOrMissingServer(t *testing.T) {
	t.Run("manager closed", func(t *testing.T) {
		mgr := NewManager()
		mgr.closed = true

		_, err := mgr.CallTool(context.Background(), "s1", "tool", nil)
		if err == nil || !strings.Contains(err.Error(), "manager is closed") {
			t.Fatalf("expected manager closed error, got: %v", err)
		}
	})

	t.Run("server missing", func(t *testing.T) {
		mgr := NewManager()

		_, err := mgr.CallTool(context.Background(), "missing", "tool", nil)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected server not found error, got: %v", err)
		}
	})
}

func TestClose_IdempotentOnEmptyManager(t *testing.T) {
	mgr := NewManager()

	if err := mgr.Close(); err != nil {
		t.Fatalf("first close should succeed, got: %v", err)
	}
	if err := mgr.Close(); err != nil {
		t.Fatalf("second close should be idempotent, got: %v", err)
	}
}
