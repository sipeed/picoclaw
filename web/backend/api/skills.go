package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type skillSupportResponse struct {
	Skills []skills.SkillInfo `json:"skills"`
}

type skillDetailResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

var (
	skillNameSanitizer       = regexp.MustCompile(`[^a-z0-9-]+`)
	importedSkillFrontmatter = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---(?:\r\n|\n|\r)*`)
	skillFrontmatterStripper = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---(?:\r\n|\n|\r)*`)
)

func (h *Handler) registerSkillRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/skills", h.handleListSkills)
	mux.HandleFunc("GET /api/skills/{name}", h.handleGetSkill)
	mux.HandleFunc("POST /api/skills/import", h.handleImportSkill)
	mux.HandleFunc("DELETE /api/skills/{name}", h.handleDeleteSkill)
}

func (h *Handler) handleListSkills(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	loader := newSkillsLoader(cfg.WorkspacePath())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skillSupportResponse{
		Skills: loader.ListSkills(),
	})
}

func (h *Handler) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	loader := newSkillsLoader(cfg.WorkspacePath())
	name := r.PathValue("name")
	allSkills := loader.ListSkills()

	for _, skill := range allSkills {
		if skill.Name != name {
			continue
		}

		content, err := loadSkillContent(skill.Path)
		if err != nil {
			http.Error(w, "Skill content not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skillDetailResponse{
			Name:        skill.Name,
			Path:        skill.Path,
			Source:      skill.Source,
			Description: skill.Description,
			Content:     content,
		})
		return
	}

	http.Error(w, "Skill not found", http.StatusNotFound)
}

func (h *Handler) handleImportSkill(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	err = r.ParseMultipartForm(2 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid multipart form: %v", err), http.StatusBadRequest)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer uploadedFile.Close()

	content, err := io.ReadAll(io.LimitReader(uploadedFile, (1<<20)+1))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusBadRequest)
		return
	}
	if len(content) > 1<<20 {
		http.Error(w, "file exceeds 1MB limit", http.StatusBadRequest)
		return
	}

	skillName, err := normalizeImportedSkillName(fileHeader.Filename, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	content = normalizeImportedSkillContent(content, skillName)

	workspace := cfg.WorkspacePath()
	skillDir := filepath.Join(workspace, "skills", skillName)
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillDir); err == nil {
		http.Error(w, "skill already exists", http.StatusConflict)
		return
	}

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create skill directory: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(skillFile, content, 0o644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save skill: %v", err), http.StatusInternalServerError)
		return
	}

	loader := newSkillsLoader(workspace)
	for _, skill := range loader.ListSkills() {
		if skill.Path == skillFile || (skill.Name == skillName && skill.Source == "workspace") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(skill)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name": skillName,
		"path": skillFile,
	})
}

func (h *Handler) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	loader := newSkillsLoader(cfg.WorkspacePath())
	name := r.PathValue("name")
	for _, skill := range loader.ListSkills() {
		if skill.Name != name {
			continue
		}
		if skill.Source != "workspace" {
			http.Error(w, "only workspace skills can be deleted", http.StatusBadRequest)
			return
		}
		if err := os.RemoveAll(filepath.Dir(skill.Path)); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete skill: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Skill not found", http.StatusNotFound)
}

func newSkillsLoader(workspace string) *skills.SkillsLoader {
	return skills.NewSkillsLoader(
		workspace,
		filepath.Join(globalConfigDir(), "skills"),
		builtinSkillsDir(),
	)
}

func normalizeImportedSkillName(filename string, content []byte) (string, error) {
	rawContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	rawContent = strings.ReplaceAll(rawContent, "\r", "\n")
	metadata, _ := extractImportedSkillMetadata(rawContent)

	raw := strings.TrimSpace(metadata["name"])
	if raw == "" {
		raw = strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	}
	raw = strings.ToLower(raw)
	raw = strings.ReplaceAll(raw, "_", "-")
	raw = strings.ReplaceAll(raw, " ", "-")
	raw = skillNameSanitizer.ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-")
	raw = strings.Join(strings.FieldsFunc(raw, func(r rune) bool { return r == '-' }), "-")

	if raw == "" {
		return "", fmt.Errorf("skill name is required in frontmatter or filename")
	}
	if len(raw) > 64 {
		return "", fmt.Errorf("skill name exceeds 64 characters")
	}
	matched, err := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*$`, raw)
	if err != nil || !matched {
		return "", fmt.Errorf("skill name must be alphanumeric with hyphens")
	}
	return raw, nil
}

func normalizeImportedSkillContent(content []byte, skillName string) []byte {
	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	metadata, body := extractImportedSkillMetadata(raw)
	description := strings.TrimSpace(metadata["description"])
	if description == "" {
		description = inferImportedSkillDescription(body)
	}
	if description == "" {
		description = "Imported skill"
	}
	if len(description) > 1024 {
		description = strings.TrimSpace(description[:1024])
	}

	body = strings.TrimLeft(body, "\n")
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: ")
	builder.WriteString(skillName)
	builder.WriteString("\n")
	builder.WriteString("description: ")
	builder.WriteString(description)
	builder.WriteString("\n")
	builder.WriteString("---\n\n")
	builder.WriteString(body)
	if !strings.HasSuffix(builder.String(), "\n") {
		builder.WriteString("\n")
	}
	return []byte(builder.String())
}

func extractImportedSkillMetadata(raw string) (map[string]string, string) {
	matches := importedSkillFrontmatter.FindStringSubmatch(raw)
	if len(matches) != 2 {
		return map[string]string{}, raw
	}
	meta := parseImportedSkillYAML(matches[1])
	body := importedSkillFrontmatter.ReplaceAllString(raw, "")
	return meta, body
}

func parseImportedSkillYAML(frontmatter string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return result
}

func inferImportedSkillDescription(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#-*0123456789. ")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func loadSkillContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return skillFrontmatterStripper.ReplaceAllString(string(content), ""), nil
}

func globalConfigDir() string {
	return config.GetHome()
}

func builtinSkillsDir() string {
	if path := os.Getenv(config.EnvBuiltinSkills); path != "" {
		return path
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(wd, "skills")
}
