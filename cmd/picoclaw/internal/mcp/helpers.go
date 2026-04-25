package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	picomcp "github.com/sipeed/picoclaw/pkg/mcp"
)

type probeResult struct {
	ToolCount int
}

var (
	editorCommand = exec.Command
	serverProbe   = defaultServerProbe

	mcpConfigSchemaOnce sync.Once
	mcpConfigSchema     *jsonschema.Resolved
	errMcpConfigSchema  error
)

const mcpConfigSchemaJSON = `{
  "type": "object",
  "properties": {
    "tools": {
      "type": "object",
      "properties": {
        "mcp": {
          "type": "object",
          "properties": {
            "enabled": { "type": "boolean" },
            "discovery": { "type": "object", "additionalProperties": true },
            "max_inline_text_chars": { "type": "integer" },
            "servers": {
              "type": "object",
              "additionalProperties": {
                "type": "object",
                "properties": {
                  "enabled": { "type": "boolean" },
                  "deferred": { "type": "boolean" },
                  "command": { "type": "string" },
                  "args": {
                    "type": "array",
                    "items": { "type": "string" }
                  },
                  "env": {
                    "type": "object",
                    "additionalProperties": { "type": "string" }
                  },
                  "env_file": { "type": "string" },
                  "type": {
                    "type": "string",
                    "enum": ["stdio", "http", "sse"]
                  },
                  "url": { "type": "string" },
                  "headers": {
                    "type": "object",
                    "additionalProperties": { "type": "string" }
                  }
                },
                "required": ["enabled"],
                "anyOf": [
                  { "required": ["command"] },
                  { "required": ["url"] }
                ],
                "additionalProperties": false
              }
            }
          },
          "required": ["enabled"],
          "additionalProperties": true
        }
      },
      "required": ["mcp"],
      "additionalProperties": true
    }
  },
  "required": ["tools"],
  "additionalProperties": true
}`

func loadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(internal.GetConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

func saveValidatedConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := validateConfigDocument(data); err != nil {
		return err
	}

	if err := config.SaveConfig(internal.GetConfigPath(), cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func validateConfigDocument(data []byte) error {
	var instance map[string]any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("failed to decode serialized config: %w", err)
	}

	schema, err := loadMCPConfigSchema()
	if err != nil {
		return fmt.Errorf("failed to load MCP config schema: %w", err)
	}

	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

func loadMCPConfigSchema() (*jsonschema.Resolved, error) {
	mcpConfigSchemaOnce.Do(func() {
		var schema jsonschema.Schema
		if err := json.Unmarshal([]byte(mcpConfigSchemaJSON), &schema); err != nil {
			errMcpConfigSchema = err
			return
		}
		mcpConfigSchema, errMcpConfigSchema = schema.Resolve(nil)
	})

	return mcpConfigSchema, errMcpConfigSchema
}

func inferTransportType(server config.MCPServerConfig) string {
	switch server.Type {
	case "stdio", "http", "sse":
		return server.Type
	}
	if server.URL != "" {
		return "sse"
	}
	if server.Command != "" {
		return "stdio"
	}
	return "unknown"
}

func renderServerTarget(server config.MCPServerConfig) string {
	transport := inferTransportType(server)
	if transport == "http" || transport == "sse" {
		if server.URL == "" {
			return "<missing url>"
		}
		return server.URL
	}

	parts := append([]string{server.Command}, server.Args...)
	rendered := strings.TrimSpace(strings.Join(parts, " "))
	if rendered == "" {
		return "<missing command>"
	}
	return rendered
}

func sortedServerNames(servers map[string]config.MCPServerConfig) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func parseEnvAssignments(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	env := make(map[string]string, len(values))
	for _, entry := range values {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			return nil, fmt.Errorf("invalid env assignment %q: expected KEY=value", entry)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid env assignment %q: key cannot be empty", entry)
		}
		env[key] = value
	}

	return env, nil
}

func parseHeaderAssignments(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	headers := make(map[string]string, len(values))
	for _, entry := range values {
		key, value, found := strings.Cut(entry, ":")
		if !found {
			key, value, found = strings.Cut(entry, "=")
		}
		if !found {
			return nil, fmt.Errorf("invalid header %q: expected 'Name: Value' or 'Name=Value'", entry)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid header %q: name cannot be empty", entry)
		}
		headers[key] = value
	}

	return headers, nil
}

func looksLikeRemoteURL(target string) bool {
	parsedURL, err := url.ParseRequestURI(target)
	if err != nil {
		return false
	}
	if parsedURL.Host == "" {
		return false
	}
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func isLocalCommandPath(command string) bool {
	if command == "" {
		return false
	}
	if looksLikeRemoteURL(command) {
		return false
	}
	return filepath.IsAbs(command) ||
		filepath.VolumeName(command) != "" ||
		strings.HasPrefix(command, "."+string(os.PathSeparator)) ||
		strings.HasPrefix(command, ".."+string(os.PathSeparator)) ||
		command == "." ||
		command == ".." ||
		strings.ContainsRune(command, os.PathSeparator)
}

func expandHomePath(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func validateLocalCommandPath(command string) error {
	if !isLocalCommandPath(command) {
		return nil
	}

	path := expandHomePath(command)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("local command %q does not exist", command)
		}
		return fmt.Errorf("failed to stat local command %q: %w", command, err)
	}
	if info.IsDir() {
		return fmt.Errorf("local command %q is a directory", command)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return fmt.Errorf("local command %q is not executable", command)
	}
	return nil
}

func defaultServerProbe(
	ctx context.Context,
	name string,
	server config.MCPServerConfig,
	workspacePath string,
) (probeResult, error) {
	mgr := picomcp.NewManager()
	defer func() { _ = mgr.Close() }()

	server.Enabled = true
	mcpCfg := config.MCPConfig{
		ToolConfig: config.ToolConfig{Enabled: true},
		Servers: map[string]config.MCPServerConfig{
			name: server,
		},
	}

	if err := mgr.LoadFromMCPConfig(ctx, mcpCfg, workspacePath); err != nil {
		return probeResult{}, err
	}

	conn, ok := mgr.GetServer(name)
	if !ok {
		return probeResult{}, fmt.Errorf("server %q did not register a connection", name)
	}

	return probeResult{ToolCount: len(conn.Tools)}, nil
}

func confirmOverwrite(r io.Reader, w io.Writer, name string) (bool, error) {
	if _, err := fmt.Fprintf(w, "MCP server %q already exists. Overwrite? [y/N]: ", name); err != nil {
		return false, err
	}

	var answer string
	if _, err := fmt.Fscanln(r, &answer); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}
