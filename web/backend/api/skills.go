package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const defaultInstallSkillRegistry = "github"

type skillSupportResponse struct {
	Skills []skillSupportItem `json:"skills"`
}

type skillSupportItem struct {
	Name             string `json:"name"`
	Path             string `json:"path"`
	Source           string `json:"source"`
	Description      string `json:"description"`
	OriginKind       string `json:"origin_kind"`
	RegistryName     string `json:"registry_name,omitempty"`
	RegistryURL      string `json:"registry_url,omitempty"`
	InstalledVersion string `json:"installed_version,omitempty"`
	InstalledAt      int64  `json:"installed_at,omitempty"`
}

type skillDetailResponse struct {
	skillSupportItem
	Content string `json:"content"`
}

type skillSearchResultItem struct {
	Score         float64 `json:"score"`
	Slug          string  `json:"slug"`
	DisplayName   string  `json:"display_name"`
	Summary       string  `json:"summary"`
	Version       string  `json:"version"`
	RegistryName  string  `json:"registry_name"`
	URL           string  `json:"url,omitempty"`
	Installed     bool    `json:"installed"`
	InstalledName string  `json:"installed_name,omitempty"`
}

type skillSearchResponse struct {
	Results    []skillSearchResultItem `json:"results"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	NextOffset int                     `json:"next_offset,omitempty"`
	HasMore    bool                    `json:"has_more"`
}

type installSkillRequest struct {
	Slug     string `json:"slug"`
	Registry string `json:"registry"`
	Version  string `json:"version,omitempty"`
	Force    bool   `json:"force,omitempty"`
}

type installSkillResponse struct {
	Status         string            `json:"status"`
	Slug           string            `json:"slug"`
	Registry       string            `json:"registry"`
	Version        string            `json:"version"`
	Summary        string            `json:"summary,omitempty"`
	IsSuspicious   bool              `json:"is_suspicious,omitempty"`
	InstalledSkill *skillSupportItem `json:"skill,omitempty"`
}

type installedSkillOriginMeta struct {
	Version          int    `json:"version"`
	OriginKind       string `json:"origin_kind,omitempty"`
	Registry         string `json:"registry,omitempty"`
	Slug             string `json:"slug,omitempty"`
	RegistryURL      string `json:"registry_url,omitempty"`
	InstalledVersion string `json:"installed_version,omitempty"`
	InstalledAt      int64  `json:"installed_at"`
}

var (
	skillNameSanitizer       = regexp.MustCompile(`[^a-z0-9-]+`)
	importedSkillFrontmatter = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---(?:\r\n|\n|\r)*`)
	skillFrontmatterStripper = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---(?:\r\n|\n|\r)*`)
	persistSkillOriginMeta   = writeSkillOriginMeta
	workspaceSkillWriteMu    sync.Mutex
	errImportedSkillExists   = errors.New("skill already exists")
)

const (
	maxImportedSkillSize    = 1 << 20
	maxRegistrySearchFanout = 1000
)

func (h *Handler) registerSkillRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/skills", h.handleListSkills)
	mux.HandleFunc("GET /api/skills/{name}", h.handleGetSkill)
	mux.HandleFunc("GET /api/skills/search", h.handleSearchSkills)
	mux.HandleFunc("POST /api/skills/install", h.handleInstallSkill)
	mux.HandleFunc("POST /api/skills/import", h.handleImportSkill)
	mux.HandleFunc("DELETE /api/skills/{name}", h.handleDeleteSkill)
}

func (h *Handler) handleListSkills(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	items, err := buildSkillSupportItems(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build skill list: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skillSupportResponse{
		Skills: items,
	})
}

func (h *Handler) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	skillItems, err := buildSkillSupportItems(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build skill list: %v", err), http.StatusInternalServerError)
		return
	}
	name := r.PathValue("name")
	for _, skillItem := range skillItems {
		if skillItem.Name != name {
			continue
		}

		content, err := loadSkillContent(skillItem.Path)
		if err != nil {
			http.Error(w, "Skill content not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skillDetailResponse{
			skillSupportItem: skillItem,
			Content:          content,
		})
		return
	}

	http.Error(w, "Skill not found", http.StatusNotFound)
}

func (h *Handler) handleSearchSkills(w http.ResponseWriter, r *http.Request) {
	cfg, loadErr := config.LoadConfig(h.configPath)
	if loadErr != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", loadErr), http.StatusInternalServerError)
		return
	}
	if registryErr := ensureSkillRegistryToolEnabled(cfg, "find_skills"); registryErr != nil {
		http.Error(w, registryErr.Error(), http.StatusBadRequest)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	limit := 20
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil || parsedLimit < 1 || parsedLimit > 50 {
			http.Error(w, "limit must be between 1 and 50", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}
	offset := 0
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		parsedOffset, parseErr := strconv.Atoi(rawOffset)
		if parseErr != nil || parsedOffset < 0 {
			http.Error(w, "offset must be 0 or greater", http.StatusBadRequest)
			return
		}
		offset = parsedOffset
	}

	installedSkills, err := buildOccupiedWorkspaceSkillsByDirectory(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to inspect installed skills: %v", err), http.StatusInternalServerError)
		return
	}

	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skillSearchResponse{
			Results: []skillSearchResultItem{},
			Limit:   limit,
			Offset:  offset,
			HasMore: false,
		})
		return
	}

	registryMgr := newSkillsRegistryManager(cfg)
	searchLimit := offset + limit + 1
	if searchLimit > maxRegistrySearchFanout {
		searchLimit = maxRegistrySearchFanout
	}
	results, err := registryMgr.SearchAll(r.Context(), query, searchLimit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search skills: %v", err), http.StatusBadGateway)
		return
	}

	if offset > len(results) {
		offset = len(results)
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	pageResults := results[offset:end]
	response := make([]skillSearchResultItem, 0, len(pageResults))
	for _, result := range pageResults {
		installedSkill, installed := installedSkills[result.Slug]
		item := skillSearchResultItem{
			Score:        result.Score,
			Slug:         result.Slug,
			DisplayName:  result.DisplayName,
			Summary:      result.Summary,
			Version:      result.Version,
			RegistryName: result.RegistryName,
			URL:          registrySkillURL(cfg, result.RegistryName, result.Slug, result.Version),
			Installed:    installed,
		}
		if installed {
			item.InstalledName = installedSkill.Name
		}
		response = append(response, item)
	}

	w.Header().Set("Content-Type", "application/json")
	nextOffset := 0
	hasMore := len(results) > end
	if hasMore {
		nextOffset = end
	}
	json.NewEncoder(w).Encode(skillSearchResponse{
		Results:    response,
		Limit:      limit,
		Offset:     offset,
		NextOffset: nextOffset,
		HasMore:    hasMore,
	})
}

func (h *Handler) handleInstallSkill(w http.ResponseWriter, r *http.Request) {
	cfg, loadErr := config.LoadConfig(h.configPath)
	if loadErr != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", loadErr), http.StatusInternalServerError)
		return
	}
	if registryErr := ensureSkillRegistryToolEnabled(cfg, "install_skill"); registryErr != nil {
		http.Error(w, registryErr.Error(), http.StatusBadRequest)
		return
	}

	var req installSkillRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", decodeErr), http.StatusBadRequest)
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Registry = strings.TrimSpace(req.Registry)
	req.Version = strings.TrimSpace(req.Version)
	if req.Registry == "" {
		req.Registry = defaultInstallSkillRegistry
	}

	if validateErr := utils.ValidateSkillIdentifier(req.Registry); validateErr != nil {
		http.Error(
			w,
			fmt.Sprintf("invalid registry %q: error: %s", req.Registry, validateErr.Error()),
			http.StatusBadRequest,
		)
		return
	}

	registryMgr := newSkillsRegistryManager(cfg)
	registry := registryMgr.GetRegistry(req.Registry)
	if registry == nil {
		http.Error(w, fmt.Sprintf("registry %q not found", req.Registry), http.StatusBadRequest)
		return
	}
	dirName, err := registry.ResolveInstallDirName(req.Slug)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid slug %q: error: %s", req.Slug, err.Error()), http.StatusBadRequest)
		return
	}

	workspace := cfg.WorkspacePath()
	skillsRoot := filepath.Join(workspace, "skills")
	targetDir := filepath.Join(workspace, "skills", dirName)
	workspaceSkillWriteMu.Lock()
	defer workspaceSkillWriteMu.Unlock()

	targetExists := false
	if _, statErr := os.Stat(targetDir); statErr == nil {
		targetExists = true
	} else if !os.IsNotExist(statErr) {
		http.Error(w, fmt.Sprintf("Failed to inspect install target: %v", statErr), http.StatusInternalServerError)
		return
	}

	if !req.Force && targetExists {
		http.Error(w, fmt.Sprintf("skill %q already installed at %s", dirName, targetDir), http.StatusConflict)
		return
	}
	if mkdirErr := os.MkdirAll(skillsRoot, 0o755); mkdirErr != nil {
		http.Error(w, fmt.Sprintf("Failed to create skills directory: %v", mkdirErr), http.StatusInternalServerError)
		return
	}

	stagedWorkspaceRoot, stagedTargetDir, err := createStagedSkillInstall(skillsRoot, dirName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to prepare staged install: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(stagedWorkspaceRoot)

	result, err := registry.DownloadAndInstall(r.Context(), req.Slug, req.Version, stagedTargetDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to install skill: %v", err), http.StatusBadGateway)
		return
	}
	if result.IsMalwareBlocked {
		http.Error(
			w,
			fmt.Sprintf("skill %q is flagged as malicious and cannot be installed", req.Slug),
			http.StatusForbidden,
		)
		return
	}

	if findWorkspaceSkillInfoByDirectory(stagedWorkspaceRoot, dirName) == nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to install skill: registry archive for %q is not a valid skill", req.Slug),
			http.StatusBadGateway,
		)
		return
	}

	installedAt := time.Now().UnixMilli()
	normalizedSlug := skills.NormalizeInstallTargetForRegistry(cfg.Tools.Skills, registry.Name(), req.Slug)
	if err := persistSkillOriginMeta(stagedTargetDir, installedSkillOriginMeta{
		Version:          1,
		OriginKind:       "third_party",
		Registry:         registry.Name(),
		Slug:             normalizedSlug,
		RegistryURL:      registrySkillURL(cfg, registry.Name(), normalizedSlug, result.Version),
		InstalledVersion: result.Version,
		InstalledAt:      installedAt,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to persist skill metadata: %v", err), http.StatusInternalServerError)
		return
	}

	if err := commitStagedSkillInstall(
		stagedWorkspaceRoot,
		stagedTargetDir,
		targetDir,
		req.Force && targetExists,
	); err != nil {
		http.Error(w, fmt.Sprintf("Failed to activate installed skill: %v", err), http.StatusInternalServerError)
		return
	}

	validatedSkill := findWorkspaceSkillByDirectory(cfg, dirName)
	if validatedSkill == nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to install skill: activated archive for %q is not a valid skill", req.Slug),
			http.StatusBadGateway,
		)
		return
	}

	installedSkill := &skillSupportItem{
		Name:             validatedSkill.Name,
		Path:             validatedSkill.Path,
		Source:           validatedSkill.Source,
		Description:      validatedSkill.Description,
		OriginKind:       "third_party",
		RegistryName:     registry.Name(),
		RegistryURL:      registrySkillURL(cfg, registry.Name(), normalizedSlug, result.Version),
		InstalledVersion: result.Version,
		InstalledAt:      installedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installSkillResponse{
		Status:         "ok",
		Slug:           req.Slug,
		Registry:       registry.Name(),
		Version:        result.Version,
		Summary:        result.Summary,
		IsSuspicious:   result.IsSuspicious,
		InstalledSkill: installedSkill,
	})
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

	content, err := io.ReadAll(io.LimitReader(uploadedFile, maxImportedSkillSize+1))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusBadRequest)
		return
	}
	if len(content) > maxImportedSkillSize {
		http.Error(w, "file exceeds 1MB limit", http.StatusBadRequest)
		return
	}
	workspaceSkillWriteMu.Lock()
	defer workspaceSkillWriteMu.Unlock()

	importedSkill, statusCode, err := importUploadedSkill(cfg, fileHeader.Filename, content)
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(importedSkill)
}

func (h *Handler) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	loader := newSkillsLoader(cfg.WorkspacePath())
	name := r.PathValue("name")
	workspaceSkillWriteMu.Lock()
	defer workspaceSkillWriteMu.Unlock()

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

func newSkillsRegistryManager(cfg *config.Config) *skills.RegistryManager {
	return skills.NewRegistryManagerFromToolsConfig(cfg.Tools.Skills)
}

func ensureSkillRegistryToolEnabled(cfg *config.Config, toolName string) error {
	if !cfg.Tools.IsToolEnabled("skills") {
		return fmt.Errorf("tools.skills is disabled")
	}
	if !cfg.Tools.IsToolEnabled(toolName) {
		return fmt.Errorf("%s is disabled", toolName)
	}
	return nil
}

func buildSkillSupportItems(cfg *config.Config) ([]skillSupportItem, error) {
	rawSkills := newSkillsLoader(cfg.WorkspacePath()).ListSkills()
	items := make([]skillSupportItem, 0, len(rawSkills))
	for _, skill := range rawSkills {
		item, err := enrichSkillInfo(cfg, skill)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func buildWorkspaceSkillItemsByDirectory(cfg *config.Config) (map[string]skillSupportItem, error) {
	result := make(map[string]skillSupportItem)
	items, err := buildSkillSupportItems(cfg)
	if err != nil {
		return nil, err
	}
	for _, skill := range items {
		if skill.Source != "workspace" {
			continue
		}
		dir := filepath.Base(filepath.Dir(skill.Path))
		if dir == "" {
			continue
		}
		result[dir] = skill
	}
	return result, nil
}

func buildOccupiedWorkspaceSkillsByDirectory(cfg *config.Config) (map[string]skillSupportItem, error) {
	result := make(map[string]skillSupportItem)
	items, err := buildSkillSupportItems(cfg)
	if err != nil {
		return nil, err
	}
	for _, skill := range items {
		if skill.Source != "workspace" {
			continue
		}

		key := filepath.Base(filepath.Dir(skill.Path))
		if meta, err := readInstalledSkillOriginMeta(skill.Path); err == nil && meta != nil && meta.Slug != "" {
			key = skills.NormalizeInstallTargetForRegistry(cfg.Tools.Skills, meta.Registry, meta.Slug)
			if key == "" {
				key = meta.Slug
			}
		}
		if key == "" {
			continue
		}
		result[key] = skill
	}
	return result, nil
}

func findWorkspaceSkillByDirectory(cfg *config.Config, directory string) *skillSupportItem {
	items, err := buildWorkspaceSkillItemsByDirectory(cfg)
	if err != nil {
		return nil
	}
	skill, ok := items[directory]
	if !ok {
		return nil
	}
	return &skill
}

func findWorkspaceSkillInfoByDirectory(workspace, directory string) *skills.SkillInfo {
	loader := skills.NewSkillsLoader(workspace, "", "")
	for _, skill := range loader.ListSkills() {
		if skill.Source != "workspace" {
			continue
		}
		if filepath.Base(filepath.Dir(skill.Path)) != directory {
			continue
		}
		skillCopy := skill
		return &skillCopy
	}
	return nil
}

func createStagedSkillInstall(skillsRoot, slug string) (string, string, error) {
	stagedWorkspaceRoot, err := os.MkdirTemp(skillsRoot, "."+slug+"-install-*")
	if err != nil {
		return "", "", err
	}
	stagedTargetDir := filepath.Join(stagedWorkspaceRoot, "skills", slug)
	return stagedWorkspaceRoot, stagedTargetDir, nil
}

func commitStagedSkillInstall(stagedWorkspaceRoot, stagedTargetDir, targetDir string, replaceExisting bool) error {
	if !replaceExisting {
		return os.Rename(stagedTargetDir, targetDir)
	}

	backupDir, err := reserveTempDirPath(filepath.Dir(targetDir), "."+filepath.Base(targetDir)+"-backup-*")
	if err != nil {
		return err
	}

	if err := os.Rename(targetDir, backupDir); err != nil {
		return fmt.Errorf("failed to move existing skill aside: %w", err)
	}

	if err := os.Rename(stagedTargetDir, targetDir); err != nil {
		if rollbackErr := os.Rename(backupDir, targetDir); rollbackErr != nil {
			return fmt.Errorf("failed to activate replacement: %w (rollback failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("failed to activate replacement: %w", err)
	}

	_ = os.RemoveAll(backupDir)
	_ = os.RemoveAll(stagedWorkspaceRoot)
	return nil
}

func reserveTempDirPath(parent, pattern string) (string, error) {
	tempDir, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Remove(tempDir); err != nil {
		return "", err
	}
	return tempDir, nil
}

func enrichSkillInfo(cfg *config.Config, skill skills.SkillInfo) (skillSupportItem, error) {
	item := skillSupportItem{
		Name:        skill.Name,
		Path:        skill.Path,
		Source:      skill.Source,
		Description: skill.Description,
		OriginKind:  "builtin",
	}

	switch skill.Source {
	case "builtin":
		item.OriginKind = "builtin"
	case "global":
		item.OriginKind = "builtin"
	case "workspace":
		meta, err := readInstalledSkillOriginMeta(skill.Path)
		if err == nil && meta != nil {
			switch meta.OriginKind {
			case "manual":
				item.OriginKind = "manual"
				item.InstalledAt = meta.InstalledAt
			case "third_party":
				item.OriginKind = "third_party"
				item.RegistryName = meta.Registry
				item.RegistryURL = registrySkillURLFromMeta(cfg, meta)
				item.InstalledVersion = meta.InstalledVersion
				item.InstalledAt = meta.InstalledAt
			default:
				if meta.Registry != "" || meta.Slug != "" || meta.InstalledVersion != "" {
					item.OriginKind = "third_party"
					item.RegistryName = meta.Registry
					item.RegistryURL = registrySkillURLFromMeta(cfg, meta)
					item.InstalledVersion = meta.InstalledVersion
					item.InstalledAt = meta.InstalledAt
				} else {
					item.OriginKind = "builtin"
					item.InstalledAt = meta.InstalledAt
				}
			}
		} else {
			item.OriginKind = "builtin"
		}
	default:
		item.OriginKind = "builtin"
	}

	return item, nil
}

func readInstalledSkillOriginMeta(skillPath string) (*installedSkillOriginMeta, error) {
	metaPath := filepath.Join(filepath.Dir(skillPath), ".skill-origin.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta installedSkillOriginMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func writeSkillOriginMeta(targetDir string, meta installedSkillOriginMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(filepath.Join(targetDir, ".skill-origin.json"), data, 0o600)
}

func registrySkillURL(cfg *config.Config, registryName, slug, version string) string {
	if cfg == nil || registryName == "" || slug == "" {
		return ""
	}
	registry := skills.LookupRegistryFromToolsConfig(cfg.Tools.Skills, registryName)
	if registry == nil {
		return ""
	}
	return registry.SkillURL(slug, version)
}

func registrySkillURLFromMeta(cfg *config.Config, meta *installedSkillOriginMeta) string {
	if meta == nil || meta.Slug == "" {
		return ""
	}
	if meta.RegistryURL != "" {
		return meta.RegistryURL
	}
	if cfg == nil || meta.Registry == "" {
		return ""
	}
	return registrySkillURL(cfg, meta.Registry, meta.Slug, meta.InstalledVersion)
}

func normalizeImportedSkillName(filename string, content []byte) (string, error) {
	return normalizeImportedSkillNameWithHint(filename, "", content)
}

func normalizeImportedSkillNameWithHint(filename, directoryHint string, content []byte) (string, error) {
	rawContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	rawContent = strings.ReplaceAll(rawContent, "\r", "\n")
	metadata, _ := extractImportedSkillMetadata(rawContent)

	raw := strings.TrimSpace(metadata["name"])
	if raw == "" {
		raw = strings.TrimSpace(directoryHint)
	}
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

func importUploadedSkill(cfg *config.Config, filename string, content []byte) (*skillSupportItem, int, error) {
	if isImportedSkillArchive(filename, content) {
		return importUploadedSkillArchive(cfg, filename, content)
	}
	return importUploadedMarkdownSkill(cfg, filename, content)
}

func importUploadedMarkdownSkill(cfg *config.Config, filename string, content []byte) (*skillSupportItem, int, error) {
	skillName, err := normalizeImportedSkillName(filename, content)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	normalizedContent := normalizeImportedSkillContent(content, skillName)
	workspace := cfg.WorkspacePath()
	skillDir := filepath.Join(workspace, "skills", skillName)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	if err := ensureWorkspaceSkillDoesNotExist(skillDir); err != nil {
		return nil, statusCodeForImportedSkillWriteError(err), err
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to create skill directory: %v", err)
	}
	if err := fileutil.WriteFileAtomic(skillFile, normalizedContent, 0o644); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to save skill: %v", err)
	}

	return finalizeImportedSkill(cfg, skillDir, skillName, false)
}

func importUploadedSkillArchive(cfg *config.Config, filename string, content []byte) (*skillSupportItem, int, error) {
	tmpDir, tempDirErr := os.MkdirTemp("", "picoclaw-skill-import-*")
	if tempDirErr != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to create temp directory: %v", tempDirErr)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "import.zip")
	if writeErr := fileutil.WriteFileAtomic(archivePath, content, 0o600); writeErr != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to stage uploaded archive: %v", writeErr)
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if extractErr := utils.ExtractZipFile(archivePath, extractDir); extractErr != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid ZIP archive: %w", extractErr)
	}

	skillRoot, err := findImportedSkillRoot(extractDir)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	skillFile := filepath.Join(skillRoot, "SKILL.md")
	skillContent, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read SKILL.md from archive: %w", err)
	}

	directoryHint := ""
	if filepath.Clean(skillRoot) != filepath.Clean(extractDir) {
		directoryHint = filepath.Base(skillRoot)
	}
	skillName, err := normalizeImportedSkillNameWithHint(filename, directoryHint, skillContent)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	workspace := cfg.WorkspacePath()
	skillDir := filepath.Join(workspace, "skills", skillName)
	if err := ensureWorkspaceSkillDoesNotExist(skillDir); err != nil {
		return nil, statusCodeForImportedSkillWriteError(err), err
	}
	if err := copyImportedSkillTree(skillRoot, skillDir); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to save skill: %v", err)
	}

	normalizedContent := normalizeImportedSkillContent(skillContent, skillName)
	if err := fileutil.WriteFileAtomic(filepath.Join(skillDir, "SKILL.md"), normalizedContent, 0o644); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to normalize skill: %v", err)
	}

	return finalizeImportedSkill(cfg, skillDir, skillName, true)
}

func isImportedSkillArchive(filename string, content []byte) bool {
	if strings.EqualFold(filepath.Ext(filename), ".zip") {
		return true
	}
	return len(content) >= 4 && bytes.HasPrefix(content, []byte("PK\x03\x04"))
}

func ensureWorkspaceSkillDoesNotExist(skillDir string) error {
	if _, err := os.Stat(skillDir); err == nil {
		return errImportedSkillExists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect skill directory: %w", err)
	}
	return nil
}

func statusCodeForImportedSkillWriteError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, errImportedSkillExists) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func finalizeImportedSkill(
	cfg *config.Config,
	skillDir string,
	skillName string,
	requireValidatedSkill bool,
) (*skillSupportItem, int, error) {
	if err := persistSkillOriginMeta(skillDir, installedSkillOriginMeta{
		Version:     1,
		OriginKind:  "manual",
		InstalledAt: time.Now().UnixMilli(),
	}); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, http.StatusInternalServerError, fmt.Errorf("Failed to persist skill metadata: %v", err)
	}

	if importedSkill := findWorkspaceSkillByDirectory(cfg, skillName); importedSkill != nil {
		return importedSkill, http.StatusOK, nil
	}

	if requireValidatedSkill {
		_ = os.RemoveAll(skillDir)
		return nil, http.StatusBadRequest, fmt.Errorf("imported archive is not a valid skill")
	}

	return &skillSupportItem{
		Name:        skillName,
		Path:        filepath.Join(skillDir, "SKILL.md"),
		Source:      "workspace",
		Description: "Imported skill",
		OriginKind:  "manual",
	}, http.StatusOK, nil
}

func findImportedSkillRoot(extractDir string) (string, error) {
	skillFiles := make([]string, 0, 1)
	err := filepath.WalkDir(extractDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "SKILL.md" {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to inspect ZIP archive: %w", err)
	}

	switch len(skillFiles) {
	case 0:
		return "", fmt.Errorf("ZIP archive must contain a SKILL.md file")
	case 1:
		return filepath.Dir(skillFiles[0]), nil
	default:
		return "", fmt.Errorf("ZIP archive must contain exactly one SKILL.md file")
	}
}

func copyImportedSkillTree(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(destDir, 0o755)
		}

		destPath := filepath.Join(destDir, relPath)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("archive contains unsupported file %q", relPath)
		}
		return fileutil.CopyFile(path, destPath, info.Mode().Perm())
	})
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
