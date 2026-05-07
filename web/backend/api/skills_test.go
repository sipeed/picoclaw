package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/config"
)

func setClawHubBaseURL(cfg *config.Config, baseURL string) {
	registryCfg, _ := cfg.Tools.Skills.Registries.Get("clawhub")
	registryCfg.BaseURL = baseURL
	cfg.Tools.Skills.Registries.Set("clawhub", registryCfg)
}

func setGithubBaseURL(cfg *config.Config, baseURL string) {
	registryCfg, ok := cfg.Tools.Skills.Registries.Get("github")
	if !ok {
		return
	}
	registryCfg.BaseURL = baseURL
	cfg.Tools.Skills.Registries.Set("github", registryCfg)
}

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
	gotOriginKinds := make(map[string]string, len(resp.Skills))
	for _, skill := range resp.Skills {
		gotSkills[skill.Name] = skill.Source
		gotOriginKinds[skill.Name] = skill.OriginKind
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
	if gotOriginKinds["workspace-skill"] != "builtin" {
		t.Fatalf("workspace-skill origin_kind = %q, want builtin", gotOriginKinds["workspace-skill"])
	}
	if gotOriginKinds["global-skill"] != "builtin" {
		t.Fatalf("global-skill origin_kind = %q, want builtin", gotOriginKinds["global-skill"])
	}
	if gotOriginKinds["builtin-skill"] != "builtin" {
		t.Fatalf("builtin-skill origin_kind = %q, want builtin", gotOriginKinds["builtin-skill"])
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
	if resp.OriginKind != "builtin" {
		t.Fatalf("resp.OriginKind = %q, want builtin", resp.OriginKind)
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
	metaContent, err := os.ReadFile(filepath.Join(workspace, "skills", "plain-skill", ".skill-origin.json"))
	if err != nil {
		t.Fatalf("ReadFile(origin metadata) error = %v", err)
	}
	var originMeta installedSkillOriginMeta
	if err := json.Unmarshal(metaContent, &originMeta); err != nil {
		t.Fatalf("Unmarshal(origin metadata) error = %v", err)
	}
	if originMeta.OriginKind != "manual" {
		t.Fatalf("originMeta.OriginKind = %q, want manual", originMeta.OriginKind)
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

func TestHandleImportSkillZip(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	zipContent := buildSkillZip(t, map[string]string{
		"Wrapped Skill/SKILL.md":       "---\nname: wrapped-skill\ndescription: Wrapped skill\n---\n# Wrapped Skill\n\nUse this skill from zip.\n",
		"Wrapped Skill/docs/README.md": "# Extra file\n",
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, createErr := writer.CreateFormFile("file", "Wrapped Skill.zip")
	if createErr != nil {
		t.Fatalf("CreateFormFile() error = %v", createErr)
	}
	if _, writeErr := part.Write(zipContent); writeErr != nil {
		t.Fatalf("Write(zipContent) error = %v", writeErr)
	}
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
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

	skillDir := filepath.Join(workspace, "skills", "wrapped-skill")
	skillFile := filepath.Join(skillDir, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	expected := "---\nname: wrapped-skill\ndescription: Wrapped skill\n---\n\n# Wrapped Skill\n\nUse this skill from zip.\n"
	if string(content) != expected {
		t.Fatalf("saved skill content mismatch:\n%s", string(content))
	}

	extraFile := filepath.Join(skillDir, "docs", "README.md")
	extraContent, err := os.ReadFile(extraFile)
	if err != nil {
		t.Fatalf("ReadFile(extra file) error = %v", err)
	}
	if string(extraContent) != "# Extra file\n" {
		t.Fatalf("extra file content = %q", string(extraContent))
	}
}

func TestHandleImportSkillZipRejectsArchiveWithoutSkill(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	zipContent := buildSkillZip(t, map[string]string{
		"README.md": "# Not a skill\n",
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "invalid.zip")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(zipContent); err != nil {
		t.Fatalf("Write(zipContent) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "skills", "invalid")); !os.IsNotExist(err) {
		t.Fatalf("invalid archive should not leave behind a skill dir, stat err=%v", err)
	}
}

func TestHandleImportSkillRollsBackOnOriginMetadataWriteFailure(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	previousPersist := persistSkillOriginMeta
	persistSkillOriginMeta = func(targetDir string, meta installedSkillOriginMeta) error {
		return errors.New("forced metadata failure")
	}
	defer func() {
		persistSkillOriginMeta = previousPersist
	}()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "Rollback Skill.md")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := io.WriteString(part, "# Rollback Skill\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	skillDir := filepath.Join(workspace, "skills", "rollback-skill")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("skill directory should be removed after metadata write failure, stat err=%v", err)
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

func TestHandleDeleteSkillPrefersWorkspaceMatch(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	homeDir := t.TempDir()
	t.Setenv(config.EnvHome, homeDir)
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	workspaceSkillDir := filepath.Join(workspace, "skills", "delete-me-workspace")
	if err := os.MkdirAll(workspaceSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspaceSkillDir, "SKILL.md"),
		[]byte("---\nname: delete-me\ndescription: workspace delete me\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(workspace) error = %v", err)
	}

	globalSkillDir := filepath.Join(homeDir, "skills", "delete-me-global")
	if err := os.MkdirAll(globalSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(global) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(globalSkillDir, "SKILL.md"),
		[]byte("---\nname: delete-me\ndescription: global delete me\n---\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(global) error = %v", err)
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
	if _, err := os.Stat(workspaceSkillDir); !os.IsNotExist(err) {
		t.Fatalf("workspace skill directory should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(globalSkillDir); err != nil {
		t.Fatalf("global skill directory should remain, stat err=%v", err)
	}
}

func TestHandleSearchSkills(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	if err := os.MkdirAll(filepath.Join(workspace, "skills", "github"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "skills", "github", "SKILL.md"),
		[]byte("---\nname: github\ndescription: Installed GitHub skill\n---\n# GitHub\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("q"); got != "github" {
			t.Fatalf("query = %q, want github", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"score":       0.95,
					"slug":        "github",
					"displayName": "GitHub",
					"summary":     "GitHub integration skill",
					"version":     "1.2.3",
				},
				{
					"score":       0.87,
					"slug":        "jira",
					"displayName": "Jira",
					"summary":     "Issue tracker skill",
					"version":     "0.9.0",
				},
			},
		})
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=github&limit=5", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Limit != 5 {
		t.Fatalf("limit = %d, want 5", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Fatalf("offset = %d, want 0", resp.Offset)
	}
	if resp.HasMore {
		t.Fatalf("has_more = true, want false")
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results count = %d, want 2", len(resp.Results))
	}
	if resp.Results[0].URL != server.URL+"/skills/github" {
		t.Fatalf("first result URL = %q, want %q", resp.Results[0].URL, server.URL+"/skills/github")
	}
	if !resp.Results[0].Installed || resp.Results[0].InstalledName != "github" {
		t.Fatalf("first result should be treated as occupying the workspace slug, got %#v", resp.Results[0])
	}
	if resp.Results[1].Installed {
		t.Fatalf("second result should not be installed, got %#v", resp.Results[1])
	}
}

func TestHandleSearchSkillsUsesGitHubResultVersionInURL(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/search/code" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"path":  "skills/pr-review/SKILL.md",
					"score": 10,
					"repository": map[string]any{
						"full_name":      "foo/bar",
						"name":           "bar",
						"description":    "Review pull requests",
						"default_branch": "master",
					},
				},
			},
		})
	}))
	defer server.Close()

	setGithubBaseURL(cfg, server.URL)
	clawHubRegistry, _ := cfg.Tools.Skills.Registries.Get("clawhub")
	clawHubRegistry.Enabled = false
	cfg.Tools.Skills.Registries.Set("clawhub", clawHubRegistry)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=pr+review&limit=5", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results count = %d, want 1", len(resp.Results))
	}
	if resp.Results[0].URL != server.URL+"/foo/bar/tree/master/skills/pr-review" {
		t.Fatalf("result URL = %q", resp.Results[0].URL)
	}
}

func TestHandleSearchSkillsGitHubRateLimitDegradesGracefully(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/search/code" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for 1.2.3.4"}`))
	}))
	defer server.Close()

	setGithubBaseURL(cfg, server.URL)
	clawHubRegistry, _ := cfg.Tools.Skills.Registries.Get("clawhub")
	clawHubRegistry.Enabled = false
	cfg.Tools.Skills.Registries.Set("clawhub", clawHubRegistry)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=pr+review&limit=5", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("results count = %d, want 0", len(resp.Results))
	}
}

func TestHandleSearchSkillsPagination(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit = %q, want 5", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"score":       0.99,
					"slug":        "skill-1",
					"displayName": "Skill 1",
					"summary":     "Summary 1",
					"version":     "1.0.0",
				},
				{
					"score":       0.98,
					"slug":        "skill-2",
					"displayName": "Skill 2",
					"summary":     "Summary 2",
					"version":     "1.0.0",
				},
				{
					"score":       0.97,
					"slug":        "skill-3",
					"displayName": "Skill 3",
					"summary":     "Summary 3",
					"version":     "1.0.0",
				},
				{
					"score":       0.96,
					"slug":        "skill-4",
					"displayName": "Skill 4",
					"summary":     "Summary 4",
					"version":     "1.0.0",
				},
			},
		})
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=github&limit=2&offset=2", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Limit != 2 {
		t.Fatalf("limit = %d, want 2", resp.Limit)
	}
	if resp.Offset != 2 {
		t.Fatalf("offset = %d, want 2", resp.Offset)
	}
	if resp.HasMore {
		t.Fatalf("has_more = true, want false")
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results count = %d, want 2", len(resp.Results))
	}
	if resp.Results[0].Slug != "skill-3" || resp.Results[1].Slug != "skill-4" {
		t.Fatalf("unexpected paged results: %#v", resp.Results)
	}
	if resp.NextOffset != 0 {
		t.Fatalf("next_offset = %d, want 0", resp.NextOffset)
	}
}

func TestHandleSearchSkillsClampsRegistryFanout(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("limit"); got != strconv.Itoa(maxRegistrySearchFanout) {
			t.Fatalf("limit = %q, want %d", got, maxRegistrySearchFanout)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"score":       0.99,
					"slug":        "skill-1",
					"displayName": "Skill 1",
					"summary":     "Summary 1",
					"version":     "1.0.0",
				},
			},
		})
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=github&limit=20&offset=100000", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("results count = %d, want 0", len(resp.Results))
	}
}

func TestHandleInstallSkill(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	zipContent := buildSkillZip(t, map[string]string{
		"SKILL.md": "---\nname: github\ndescription: GitHub registry skill\n---\n# GitHub\n\nUse this skill.\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/search":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"score":       0.95,
						"slug":        "github",
						"displayName": "GitHub",
						"summary":     "GitHub registry skill",
						"version":     "1.2.3",
					},
				},
			})
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			if got := r.URL.Query().Get("slug"); got != "github" {
				t.Fatalf("slug = %q, want github", got)
			}
			if got := r.URL.Query().Get("version"); got != "1.2.3" {
				t.Fatalf("version = %q, want 1.2.3", got)
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp installSkillResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Status != "ok" || resp.Version != "1.2.3" || resp.InstalledSkill == nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.InstalledSkill.OriginKind != "third_party" {
		t.Fatalf("resp.InstalledSkill.OriginKind = %q, want third_party", resp.InstalledSkill.OriginKind)
	}
	if resp.InstalledSkill.RegistryURL != server.URL+"/skills/github" {
		t.Fatalf(
			"resp.InstalledSkill.RegistryURL = %q, want %q",
			resp.InstalledSkill.RegistryURL,
			server.URL+"/skills/github",
		)
	}

	skillFile := filepath.Join(workspace, "skills", "github", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("installed skill file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "skills", "github", ".skill-origin.json")); err != nil {
		t.Fatalf("origin metadata missing: %v", err)
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/skills/github", nil)
	mux.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d, body=%s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}

	var detailResp skillDetailResponse
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("Unmarshal(detail response) error = %v", err)
	}
	if detailResp.RegistryURL != server.URL+"/skills/github" {
		t.Fatalf("detailResp.RegistryURL = %q, want %q", detailResp.RegistryURL, server.URL+"/skills/github")
	}

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=github&limit=5", nil)
	mux.ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d, body=%s", searchRec.Code, http.StatusOK, searchRec.Body.String())
	}

	var searchResp skillSearchResponse
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("Unmarshal(search response) error = %v", err)
	}
	if len(searchResp.Results) != 1 {
		t.Fatalf("search results count = %d, want 1", len(searchResp.Results))
	}
	if !searchResp.Results[0].Installed || searchResp.Results[0].InstalledName != "github" {
		t.Fatalf("search result should be treated as installed after registry install, got %#v", searchResp.Results[0])
	}
}

func TestHandleInstallSkillForcePreservesExistingSkillOnFailure(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	skillDir := filepath.Join(workspace, "skills", "github")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	oldContent := []byte("---\nname: github\ndescription: Existing skill\n---\n# Existing\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), oldContent, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			http.Error(w, "upstream download failed", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
		Force:    true,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}

	gotContent, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(gotContent, oldContent) {
		t.Fatalf("existing skill should remain unchanged, got:\n%s", string(gotContent))
	}
}

func TestHandleInstallSkillDefaultsRegistryToGitHub(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/repos/foo/bar":
			json.NewEncoder(w).Encode(map[string]any{"default_branch": "master"})
		case "/api/v3/repos/foo/bar/contents/.agents/skills/pr-review":
			assert.Equal(t, "ref=master", r.URL.RawQuery)
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"type":         "file",
					"name":         "SKILL.md",
					"download_url": server.URL + "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md",
				},
			})
		case "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md":
			_, _ = w.Write([]byte("---\nname: pr-review\ndescription: PR review skill\n---\n# PR Review\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	githubRegistry, ok := cfg.Tools.Skills.Registries.Get("github")
	if !ok {
		t.Fatalf("github registry missing from default config")
	}
	githubRegistry.BaseURL = server.URL
	cfg.Tools.Skills.Registries.Set("github", githubRegistry)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug: "foo/bar/.agents/skills/pr-review",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp installSkillResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Registry != "github" {
		t.Fatalf("resp.Registry = %q, want github", resp.Registry)
	}
}

func TestHandleInstallSkillTracksGitHubURLInstallsAsInstalled(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/repos/foo/bar":
			json.NewEncoder(w).Encode(map[string]any{"default_branch": "master"})
		case "/api/v3/repos/foo/bar/contents/.agents/skills/pr-review":
			assert.Equal(t, "ref=master", r.URL.RawQuery)
			json.NewEncoder(w).Encode([]map[string]any{{
				"type":         "file",
				"name":         "SKILL.md",
				"download_url": server.URL + "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md",
			}})
		case "/api/v3/search/code":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"path":  ".agents/skills/pr-review/SKILL.md",
					"score": 10,
					"repository": map[string]any{
						"full_name":      "foo/bar",
						"name":           "bar",
						"description":    "PR review skill",
						"default_branch": "master",
					},
				}},
			})
		case "/raw/foo/bar/master/.agents/skills/pr-review/SKILL.md":
			_, _ = w.Write([]byte("---\nname: pr-review\ndescription: PR review skill\n---\n# PR Review\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setGithubBaseURL(cfg, server.URL)
	clawHubRegistry, _ := cfg.Tools.Skills.Registries.Get("clawhub")
	clawHubRegistry.Enabled = false
	cfg.Tools.Skills.Registries.Set("clawhub", clawHubRegistry)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	installBody, err := json.Marshal(installSkillRequest{
		Slug: server.URL + "/foo/bar/tree/master/.agents/skills/pr-review",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	installRec := httptest.NewRecorder()
	installReq := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(installBody))
	installReq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(installRec, installReq)

	if installRec.Code != http.StatusOK {
		t.Fatalf("install status = %d, want %d, body=%s", installRec.Code, http.StatusOK, installRec.Body.String())
	}

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=pr+review&limit=5", nil)
	mux.ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d, body=%s", searchRec.Code, http.StatusOK, searchRec.Body.String())
	}

	var searchResp skillSearchResponse
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("Unmarshal(search response) error = %v", err)
	}
	if len(searchResp.Results) != 1 {
		t.Fatalf("search results count = %d, want 1", len(searchResp.Results))
	}
	if !searchResp.Results[0].Installed || searchResp.Results[0].InstalledName != "pr-review" {
		t.Fatalf("search result should be treated as installed after URL install, got %#v", searchResp.Results[0])
	}
}

func TestHandleSearchSkillsMarksDirectoryCollisionAsInstalled(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	skillDir := filepath.Join(workspace, "skills", "pr-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: pr-review\ndescription: Workspace PR review skill\n---\n# PR Review\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}
	if err := writeSkillOriginMeta(skillDir, installedSkillOriginMeta{
		Version:          1,
		OriginKind:       "third_party",
		Registry:         "github",
		Slug:             "foo/bar/.agents/skills/pr-review",
		RegistryURL:      "https://github.com/foo/bar/tree/master/.agents/skills/pr-review",
		InstalledVersion: "master",
		InstalledAt:      time.Now().UnixMilli(),
	}); err != nil {
		t.Fatalf("writeSkillOriginMeta() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/search":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{{
					"slug":        "pr-review",
					"displayName": "PR Review",
					"summary":     "ClawHub PR review skill",
					"version":     "1.2.3",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	githubRegistry, _ := cfg.Tools.Skills.Registries.Get("github")
	githubRegistry.Enabled = false
	cfg.Tools.Skills.Registries.Set("github", githubRegistry)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/skills/search?q=pr+review&limit=5", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp skillSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results count = %d, want 1", len(resp.Results))
	}
	if !resp.Results[0].Installed || resp.Results[0].InstalledName != "pr-review" {
		t.Fatalf("search result should be treated as installed when directory is occupied, got %#v", resp.Results[0])
	}
}

func TestHandleInstallSkillRollsBackOnOriginMetadataWriteFailure(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	zipContent := buildSkillZip(t, map[string]string{
		"SKILL.md": "---\nname: github\ndescription: GitHub registry skill\n---\n# GitHub\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	previousPersist := persistSkillOriginMeta
	persistSkillOriginMeta = func(targetDir string, meta installedSkillOriginMeta) error {
		return errors.New("forced metadata failure")
	}
	defer func() {
		persistSkillOriginMeta = previousPersist
	}()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	skillDir := filepath.Join(workspace, "skills", "github")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("skill directory should be removed after metadata write failure, stat err=%v", err)
	}
}

func TestHandleInstallSkillSerializesConcurrentRequests(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	zipContent := buildSkillZip(t, map[string]string{
		"SKILL.md": "---\nname: github\ndescription: GitHub registry skill\n---\n# GitHub\n",
	})

	downloadStarted := make(chan struct{}, 2)
	releaseFirstDownload := make(chan struct{})
	downloadCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			downloadCount++
			downloadStarted <- struct{}{}
			if downloadCount == 1 {
				<-releaseFirstDownload
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	type installResult struct {
		code int
		body string
	}
	results := make(chan installResult, 2)
	startInstall := func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		results <- installResult{
			code: rec.Code,
			body: rec.Body.String(),
		}
	}

	go startInstall()

	select {
	case <-downloadStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first install download to start")
	}

	go startInstall()

	select {
	case <-downloadStarted:
		t.Fatal("second install should not reach registry download before the first request completes")
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseFirstDownload)

	firstResult := <-results
	secondResult := <-results

	codes := map[int]int{
		firstResult.code:  1,
		secondResult.code: 1,
	}
	if codes[http.StatusOK] != 1 || codes[http.StatusConflict] != 1 {
		t.Fatalf(
			"unexpected install results: first=(%d, %q) second=(%d, %q)",
			firstResult.code,
			firstResult.body,
			secondResult.code,
			secondResult.body,
		)
	}
}

func TestHandleImportSkillWaitsForConcurrentInstall(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	zipContent := buildSkillZip(t, map[string]string{
		"SKILL.md": "---\nname: github\ndescription: GitHub registry skill\n---\n# GitHub\n",
	})

	downloadStarted := make(chan struct{}, 1)
	releaseDownload := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			downloadStarted <- struct{}{}
			<-releaseDownload
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	installBody, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	type result struct {
		code int
		body string
	}
	installResults := make(chan result, 1)
	importResults := make(chan result, 1)

	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(installBody))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		installResults <- result{code: rec.Code, body: rec.Body.String()}
	}()

	select {
	case <-downloadStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for install download to start")
	}

	var importBody bytes.Buffer
	writer := multipart.NewWriter(&importBody)
	part, err := writer.CreateFormFile("file", "github.md")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := io.WriteString(part, "# GitHub\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/skills/import", &importBody)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		mux.ServeHTTP(rec, req)
		importResults <- result{code: rec.Code, body: rec.Body.String()}
	}()

	select {
	case got := <-importResults:
		t.Fatalf("import should wait for the install lock, got early response (%d, %q)", got.code, got.body)
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseDownload)

	installResult := <-installResults
	importResult := <-importResults

	if installResult.code != http.StatusOK {
		t.Fatalf("install status = %d, want %d, body=%s", installResult.code, http.StatusOK, installResult.body)
	}
	if importResult.code != http.StatusConflict {
		t.Fatalf("import status = %d, want %d, body=%s", importResult.code, http.StatusConflict, importResult.body)
	}
}

func TestHandleInstallSkillRejectsInvalidArchive(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, loadErr := config.LoadConfig(configPath)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	workspace := filepath.Join(t.TempDir(), "workspace")
	cfg.Agents.Defaults.Workspace = workspace

	zipContent := buildSkillZip(t, map[string]string{
		"README.md": "# Not a skill\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills/github":
			json.NewEncoder(w).Encode(map[string]any{
				"slug":        "github",
				"displayName": "GitHub",
				"summary":     "GitHub registry skill",
				"latestVersion": map[string]any{
					"version": "1.2.3",
				},
				"moderation": map[string]any{
					"isMalwareBlocked": false,
					"isSuspicious":     false,
				},
			})
		case "/api/v1/download":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	setClawHubBaseURL(cfg, server.URL)
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(installSkillRequest{
		Slug:     "github",
		Registry: "clawhub",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}

	skillDir := filepath.Join(workspace, "skills", "github")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("invalid installed archive should be removed, stat err=%v", err)
	}
}

func buildSkillZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
		if _, err := io.WriteString(writer, content); err != nil {
			t.Fatalf("WriteString(%q) error = %v", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buf.Bytes()
}
