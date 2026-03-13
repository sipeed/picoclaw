package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"jane/pkg/config"
)

func TestHandleListSkills(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Join(workspace, "skills", "workspace-skill"), 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "skills", "workspace-skill", "SKILL.md"),
		[]byte("---\nname: workspace-skill\ndescription: Workspace skill\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(workspace skill) error = %v", err)
	}

	globalSkillDir := filepath.Join(globalConfigDir(), "skills", "global-skill")
	if err := os.MkdirAll(globalSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(global skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(globalSkillDir, "SKILL.md"),
		[]byte("---\nname: global-skill\ndescription: Global skill\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(global skill) error = %v", err)
	}

	builtinRoot := filepath.Join(t.TempDir(), "builtin-skills")
	oldBuiltin := os.Getenv("PICOCLAW_BUILTIN_SKILLS")
	if err := os.Setenv("PICOCLAW_BUILTIN_SKILLS", builtinRoot); err != nil {
		t.Fatalf("Setenv(PICOCLAW_BUILTIN_SKILLS) error = %v", err)
	}
	defer func() {
		if oldBuiltin == "" {
			_ = os.Unsetenv("PICOCLAW_BUILTIN_SKILLS")
		} else {
			_ = os.Setenv("PICOCLAW_BUILTIN_SKILLS", oldBuiltin)
		}
	}()

	builtinSkillDir := filepath.Join(builtinRoot, "builtin-skill")
	if err := os.MkdirAll(builtinSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(builtin skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(builtinSkillDir, "SKILL.md"),
		[]byte("---\nname: builtin-skill\ndescription: Builtin skill\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(builtin skill) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSupportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Skills) != 3 {
		t.Fatalf("skills count = %d, want 3", len(resp.Skills))
	}

	gotSkills := make(map[string]string, len(resp.Skills))
	for _, skill := range resp.Skills {
		gotSkills[skill.Name] = skill.Source
	}
	if gotSkills["workspace-skill"] != "workspace" {
		t.Fatalf("workspace-skill source = %q, want workspace", gotSkills["workspace-skill"])
	}
	if gotSkills["global-skill"] != "global" {
		t.Fatalf("global-skill source = %q, want global", gotSkills["global-skill"])
	}
	if gotSkills["builtin-skill"] != "builtin" {
		t.Fatalf("builtin-skill source = %q, want builtin", gotSkills["builtin-skill"])
	}
}

func TestHandleGetSkill(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	skillDir := filepath.Join(workspace, "skills", "viewer-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte(
			"---\nname: viewer-skill\ndescription: Viewable skill\n---\n# Viewer Skill\n\nThis is visible content.\n",
		),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/viewer-skill", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Name != "viewer-skill" || resp.Source != "workspace" || resp.Description != "Viewable skill" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Content != "# Viewer Skill\n\nThis is visible content.\n" {
		t.Fatalf("content = %q", resp.Content)
	}
}

func TestHandleGetSkillUsesResolvedPath(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	skillDir := filepath.Join(workspace, "skills", "folder-name")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: display-name\ndescription: Mismatched path skill\n---\n# Display Name\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/display-name", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Name != "display-name" {
		t.Fatalf("resp.Name = %q, want display-name", resp.Name)
	}
	if resp.Content != "# Display Name\n" {
		t.Fatalf("content = %q", resp.Content)
	}
}

func TestHandleImportSkill(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "Plain Skill.md")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	_, err = io.WriteString(part, "# Plain Skill\n\nUse this skill to test imports.\n")
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	skillFile := filepath.Join(workspace, "skills", "plain-skill", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	expected := "---\nname: plain-skill\ndescription: Plain Skill\n---\n\n# Plain Skill\n\nUse this skill to test imports.\n"
	if string(content) != expected {
		t.Fatalf("saved skill content mismatch:\n%s", string(content))
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", rec2.Code, http.StatusOK, rec2.Body.String())
	}
	var listResp skillSupportResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal list response error = %v", err)
	}
	found := false
	for _, skill := range listResp.Skills {
		if skill.Name == "plain-skill" && skill.Source == "workspace" && skill.Description == "Plain Skill" {
			found = true
		}
	}
	if !found {
		t.Fatalf("plain-skill should be listed after import, got %#v", listResp.Skills)
	}
}

func TestHandleDeleteSkill(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	skillDir := filepath.Join(workspace, "skills", "delete-me")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: delete-me\ndescription: delete me\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/skills/delete-me", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("skill directory should be removed, stat err=%v", err)
	}
}
