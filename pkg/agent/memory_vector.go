package agent

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	defaultMemoryVectorDimensions      = 256
	defaultMemoryVectorTopK            = 6
	defaultMemoryVectorMinScore        = 0.15
	defaultMemoryVectorMaxContextChars = 1800
	defaultMemoryVectorRecentDailyDays = 14
	memoryVectorChunkChars             = 280
)

// MemoryVectorSettings controls semantic memory indexing and retrieval behavior.
type MemoryVectorSettings struct {
	Enabled         bool
	Dimensions      int
	TopK            int
	MinScore        float64
	MaxContextChars int
	RecentDailyDays int
}

type MemoryVectorHit struct {
	Source string
	Text   string
	Score  float64
}

type memoryVectorDocument struct {
	ID     string    `json:"id"`
	Source string    `json:"source"`
	Text   string    `json:"text"`
	Vector []float32 `json:"vector"`
}

type memoryVectorIndex struct {
	Version     int                    `json:"version"`
	BuiltAt     string                 `json:"built_at"`
	Fingerprint string                 `json:"fingerprint"`
	Dimensions  int                    `json:"dimensions"`
	Documents   []memoryVectorDocument `json:"documents"`
}

type memoryVectorSourceFile struct {
	Path    string
	RelPath string
	Size    int64
	ModUnix int64
}

type memoryVectorStore struct {
	memoryDir  string
	memoryFile string
	indexPath  string

	mu       sync.Mutex
	settings MemoryVectorSettings
	cache    *memoryVectorIndex
}

func defaultMemoryVectorSettings() MemoryVectorSettings {
	return MemoryVectorSettings{
		Enabled:         true,
		Dimensions:      defaultMemoryVectorDimensions,
		TopK:            defaultMemoryVectorTopK,
		MinScore:        defaultMemoryVectorMinScore,
		MaxContextChars: defaultMemoryVectorMaxContextChars,
		RecentDailyDays: defaultMemoryVectorRecentDailyDays,
	}
}

func normalizeMemoryVectorSettings(settings MemoryVectorSettings) MemoryVectorSettings {
	if settings.Dimensions <= 0 {
		settings.Dimensions = defaultMemoryVectorDimensions
	}
	if settings.TopK <= 0 {
		settings.TopK = defaultMemoryVectorTopK
	}
	if settings.MinScore < 0 || settings.MinScore >= 1 {
		settings.MinScore = defaultMemoryVectorMinScore
	}
	if settings.MaxContextChars <= 0 {
		settings.MaxContextChars = defaultMemoryVectorMaxContextChars
	}
	if settings.RecentDailyDays <= 0 {
		settings.RecentDailyDays = defaultMemoryVectorRecentDailyDays
	}
	return settings
}

func newMemoryVectorStore(memoryDir, memoryFile string, settings MemoryVectorSettings) *memoryVectorStore {
	settings = normalizeMemoryVectorSettings(settings)
	return &memoryVectorStore{
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
		indexPath:  filepath.Join(memoryDir, "vector", "index.json"),
		settings:   settings,
	}
}

func (vs *memoryVectorStore) SetSettings(settings MemoryVectorSettings) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	normalized := normalizeMemoryVectorSettings(settings)
	if vs.settings != normalized {
		vs.settings = normalized
		vs.cache = nil
	}
}

func (vs *memoryVectorStore) Rebuild() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.rebuildLocked()
}

func (vs *memoryVectorStore) Search(query string, topK int, minScore float64) ([]MemoryVectorHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.settings.Enabled {
		return nil, nil
	}
	if topK <= 0 {
		topK = vs.settings.TopK
	}
	if minScore < 0 {
		minScore = 0
	}

	if err := vs.ensureIndexLocked(); err != nil {
		return nil, err
	}
	if vs.cache == nil || len(vs.cache.Documents) == 0 {
		return nil, nil
	}

	queryVec := embedHashedText(query, vs.settings.Dimensions)
	queryTerms := uniqueTokenSet(tokenizeForEmbedding(query))
	if len(queryVec) == 0 {
		return nil, nil
	}

	hits := make([]MemoryVectorHit, 0, minInt(topK, len(vs.cache.Documents)))
	for _, doc := range vs.cache.Documents {
		vectorScore := cosineSimilarity(queryVec, doc.Vector)
		keywordScore := lexicalSimilarity(queryTerms, tokenizeForEmbedding(doc.Text))
		// Blend semantic and lexical signals to improve recall on terse notes and identifiers.
		score := 0.8*vectorScore + 0.2*keywordScore
		if score < minScore {
			continue
		}
		hits = append(hits, MemoryVectorHit{
			Source: doc.Source,
			Text:   doc.Text,
			Score:  score,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Source < hits[j].Source
		}
		return hits[i].Score > hits[j].Score
	})

	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

func (vs *memoryVectorStore) GetBySource(source string) (MemoryVectorHit, bool, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return MemoryVectorHit{}, false, nil
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.settings.Enabled {
		return MemoryVectorHit{}, false, nil
	}

	if err := vs.ensureIndexLocked(); err != nil {
		return MemoryVectorHit{}, false, err
	}
	if vs.cache == nil || len(vs.cache.Documents) == 0 {
		return MemoryVectorHit{}, false, nil
	}

	for _, doc := range vs.cache.Documents {
		if doc.Source != source {
			continue
		}
		return MemoryVectorHit{
			Source: doc.Source,
			Text:   doc.Text,
			Score:  1,
		}, true, nil
	}

	return MemoryVectorHit{}, false, nil
}

func (vs *memoryVectorStore) ensureIndexLocked() error {
	sources, fingerprint, err := vs.collectSourceFilesLocked(time.Now())
	if err != nil {
		return err
	}

	if vs.cache != nil &&
		vs.cache.Fingerprint == fingerprint &&
		vs.cache.Dimensions == vs.settings.Dimensions {
		return nil
	}

	if disk, loadErr := vs.loadIndexLocked(); loadErr == nil && disk != nil {
		if disk.Fingerprint == fingerprint && disk.Dimensions == vs.settings.Dimensions {
			vs.cache = disk
			return nil
		}
	}

	return vs.rebuildFromSourcesLocked(sources, fingerprint)
}

func (vs *memoryVectorStore) rebuildLocked() error {
	sources, fingerprint, err := vs.collectSourceFilesLocked(time.Now())
	if err != nil {
		return err
	}
	return vs.rebuildFromSourcesLocked(sources, fingerprint)
}

func (vs *memoryVectorStore) rebuildFromSourcesLocked(
	sources []memoryVectorSourceFile,
	fingerprint string,
) error {
	docs, err := vs.buildDocumentsLocked(sources)
	if err != nil {
		return err
	}

	index := &memoryVectorIndex{
		Version:     1,
		BuiltAt:     time.Now().Format(time.RFC3339),
		Fingerprint: fingerprint,
		Dimensions:  vs.settings.Dimensions,
		Documents:   docs,
	}
	if err := vs.saveIndexLocked(index); err != nil {
		return err
	}
	vs.cache = index
	return nil
}

func (vs *memoryVectorStore) collectSourceFilesLocked(now time.Time) ([]memoryVectorSourceFile, string, error) {
	sources := make([]memoryVectorSourceFile, 0, vs.settings.RecentDailyDays+1)

	if info, err := os.Stat(vs.memoryFile); err == nil && !info.IsDir() {
		sources = append(sources, memoryVectorSourceFile{
			Path:    vs.memoryFile,
			RelPath: "MEMORY.md",
			Size:    info.Size(),
			ModUnix: info.ModTime().Unix(),
		})
	} else if err != nil && !os.IsNotExist(err) {
		return nil, "", err
	}

	for i := 0; i < vs.settings.RecentDailyDays; i++ {
		day := now.AddDate(0, 0, -i).Format("20060102")
		candidate := filepath.Join(vs.memoryDir, day[:6], day+".md")

		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", err
		}
		if info.IsDir() {
			continue
		}

		rel, relErr := filepath.Rel(vs.memoryDir, candidate)
		if relErr != nil {
			rel = filepath.Base(candidate)
		}
		sources = append(sources, memoryVectorSourceFile{
			Path:    candidate,
			RelPath: filepath.ToSlash(rel),
			Size:    info.Size(),
			ModUnix: info.ModTime().Unix(),
		})
	}

	fingerprint := buildSourceFingerprint(sources, vs.settings.Dimensions)
	return sources, fingerprint, nil
}

func buildSourceFingerprint(sources []memoryVectorSourceFile, dims int) string {
	h := sha1.New()
	fmt.Fprintf(h, "dims=%d\n", dims)
	for _, src := range sources {
		fmt.Fprintf(h, "%s|%d|%d\n", src.RelPath, src.Size, src.ModUnix)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (vs *memoryVectorStore) buildDocumentsLocked(sources []memoryVectorSourceFile) ([]memoryVectorDocument, error) {
	docs := make([]memoryVectorDocument, 0, len(sources)*8)

	for _, src := range sources {
		data, err := os.ReadFile(src.Path)
		if err != nil {
			return nil, err
		}
		content := string(data)
		if strings.TrimSpace(content) == "" {
			continue
		}

		if src.RelPath == "MEMORY.md" {
			sections := parseMemorySections(content)
			for _, section := range memorySectionOrder {
				for _, entry := range sections[section] {
					text := compactWhitespace(entry)
					if text == "" {
						continue
					}
					payload := section + ": " + text
					docs = append(docs, memoryVectorDocument{
						ID:     buildMemoryVectorID(src.RelPath, section, text),
						Source: fmt.Sprintf("%s#%s", src.RelPath, section),
						Text:   payload,
						Vector: embedHashedText(payload, vs.settings.Dimensions),
					})
				}
			}
			continue
		}

		chunks := chunkMarkdownForVectors(content, memoryVectorChunkChars)
		for idx, chunk := range chunks {
			if strings.TrimSpace(chunk) == "" {
				continue
			}
			docs = append(docs, memoryVectorDocument{
				ID:     buildMemoryVectorID(src.RelPath, fmt.Sprintf("%d", idx+1), chunk),
				Source: fmt.Sprintf("%s#%d", src.RelPath, idx+1),
				Text:   chunk,
				Vector: embedHashedText(chunk, vs.settings.Dimensions),
			})
		}
	}

	return docs, nil
}

func chunkMarkdownForVectors(content string, maxChars int) []string {
	if maxChars <= 0 {
		maxChars = memoryVectorChunkChars
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, 8)
	var current strings.Builder
	currentHeading := ""

	flush := func() {
		chunk := compactWhitespace(current.String())
		if chunk != "" {
			out = append(out, chunk)
		}
		current.Reset()
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || line == "---" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			currentHeading = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}

		line = strings.TrimSpace(strings.TrimLeft(line, "-*+"))
		if line == "" {
			continue
		}
		if currentHeading != "" {
			line = currentHeading + ": " + line
		}

		if current.Len() == 0 {
			current.WriteString(line)
		} else {
			if current.Len()+1+len(line) > maxChars {
				flush()
				current.WriteString(line)
			} else {
				current.WriteString(" ")
				current.WriteString(line)
			}
		}
	}
	flush()

	return out
}

func (vs *memoryVectorStore) loadIndexLocked() (*memoryVectorIndex, error) {
	data, err := os.ReadFile(vs.indexPath)
	if err != nil {
		return nil, err
	}
	var index memoryVectorIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	return &index, nil
}

func (vs *memoryVectorStore) saveIndexLocked(index *memoryVectorIndex) error {
	if err := os.MkdirAll(filepath.Dir(vs.indexPath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vs.indexPath, payload, 0o644)
}

func buildMemoryVectorID(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		if part == "" {
			continue
		}
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func embedHashedText(text string, dims int) []float32 {
	if dims <= 0 {
		return nil
	}

	tokens := tokenizeForEmbedding(text)
	if len(tokens) == 0 {
		return nil
	}

	tf := make(map[string]int, len(tokens))
	for _, token := range tokens {
		tf[token]++
	}

	vec := make([]float64, dims)
	for token, count := range tf {
		h := fnv.New32a()
		_, _ = h.Write([]byte(token))
		sum := h.Sum32()

		index := int(sum % uint32(dims))
		sign := 1.0
		if (sum>>31)&1 == 1 {
			sign = -1
		}

		weight := 1.0 + math.Log(float64(count))
		vec[index] += sign * weight
	}

	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return nil
	}
	norm = math.Sqrt(norm)

	out := make([]float32, dims)
	for i, v := range vec {
		out[i] = float32(v / norm)
	}
	return out
}

func tokenizeForEmbedding(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}

	tokens := make([]string, 0, 32)
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, string(current))
		current = current[:0]
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()

	if len(tokens) > 0 {
		expanded := make([]string, 0, len(tokens)*3)
		for _, token := range tokens {
			expanded = append(expanded, token)
			runes := []rune(token)
			if len(runes) < 4 {
				continue
			}
			for i := 0; i+3 <= len(runes); i++ {
				expanded = append(expanded, string(runes[i:i+3]))
			}
		}
		return expanded
	}

	// For very short non-word strings, fall back to rune-level tokens.
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		tokens = append(tokens, string(r))
	}
	return tokens
}

func cosineSimilarity(a, b []float32) float64 {
	n := minInt(len(a), len(b))
	if n == 0 {
		return 0
	}

	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := 0; i < n; i++ {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / math.Sqrt(normA*normB)
}

func compactWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func uniqueTokenSet(tokens []string) map[string]struct{} {
	if len(tokens) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func lexicalSimilarity(queryTokens map[string]struct{}, docTokens []string) float64 {
	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0
	}

	docSet := uniqueTokenSet(docTokens)
	if len(docSet) == 0 {
		return 0
	}

	overlap := 0
	for token := range queryTokens {
		if _, ok := docSet[token]; ok {
			overlap++
		}
	}
	if overlap == 0 {
		return 0
	}

	denom := len(queryTokens)
	if denom == 0 {
		return 0
	}
	return float64(overlap) / float64(denom)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
