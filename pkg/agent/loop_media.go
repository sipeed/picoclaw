// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/mediacache"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// resolveMediaRefs resolves media:// refs in messages.
// Images are base64-encoded into the Media array for multimodal LLMs.
// Non-image files (documents, audio, video) have their local path injected
// into Content so the agent can access them via file tools like read_file.
// Returns a new slice; original messages are not mutated.
func resolveMediaRefs(messages []providers.Message, store media.MediaStore, maxSize int) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, len(messages))
	copy(result, messages)

	for i, m := range result {
		if len(m.Media) == 0 {
			continue
		}

		resolved := make([]string, 0, len(m.Media))
		var pathTags []string

		for _, ref := range m.Media {
			if !strings.HasPrefix(ref, "media://") {
				resolved = append(resolved, ref)
				continue
			}

			localPath, meta, err := store.ResolveWithMeta(ref)
			if err != nil {
				logger.WarnCF("agent", "Failed to resolve media ref", map[string]any{
					"ref":   ref,
					"error": err.Error(),
				})
				continue
			}

			info, err := os.Stat(localPath)
			if err != nil {
				logger.WarnCF("agent", "Failed to stat media file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}

			mime := detectMIME(localPath, meta)

			if strings.HasPrefix(mime, "image/") {
				dataURL := encodeImageToDataURL(localPath, mime, info, maxSize)
				if dataURL != "" {
					resolved = append(resolved, dataURL)
				}
				continue
			}

			pathTags = append(pathTags, buildPathTag(mime, localPath))
		}

		result[i].Media = resolved
		if len(pathTags) > 0 {
			result[i].Content = injectPathTags(result[i].Content, pathTags)
		}
	}

	return result
}

// detectMIME determines the MIME type from metadata or magic-bytes detection.
// Returns empty string if detection fails.
func detectMIME(localPath string, meta media.MediaMeta) string {
	if meta.ContentType != "" {
		return meta.ContentType
	}
	kind, err := filetype.MatchFile(localPath)
	if err != nil || kind == filetype.Unknown {
		return ""
	}
	return kind.MIME.Value
}

// encodeImageToDataURL base64-encodes an image file into a data URL.
// Returns empty string if the file exceeds maxSize or encoding fails.
func encodeImageToDataURL(localPath, mime string, info os.FileInfo, maxSize int) string {
	if info.Size() > int64(maxSize) {
		logger.WarnCF("agent", "Media file too large, skipping", map[string]any{
			"path":     localPath,
			"size":     info.Size(),
			"max_size": maxSize,
		})
		return ""
	}

	f, err := os.Open(localPath)
	if err != nil {
		logger.WarnCF("agent", "Failed to open media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	defer f.Close()

	prefix := "data:" + mime + ";base64,"
	encodedLen := base64.StdEncoding.EncodedLen(int(info.Size()))
	var buf bytes.Buffer
	buf.Grow(len(prefix) + encodedLen)
	buf.WriteString(prefix)

	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(encoder, f); err != nil {
		logger.WarnCF("agent", "Failed to encode media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	encoder.Close()

	return buf.String()
}

// buildPathTag creates a structured tag exposing the local file path.
// Tag type is derived from MIME: [audio:/path], [video:/path], or [file:/path].
func buildPathTag(mime, localPath string) string {
	switch {
	case strings.HasPrefix(mime, "audio/"):
		return "[audio:" + localPath + "]"
	case strings.HasPrefix(mime, "video/"):
		return "[video:" + localPath + "]"
	default:
		return "[file:" + localPath + "]"
	}
}

// injectPathTags replaces generic media tags in content with path-bearing versions,
// or appends if no matching generic tag is found.
func injectPathTags(content string, tags []string) string {
	for _, tag := range tags {
		var generic string
		switch {
		case strings.HasPrefix(tag, "[audio:"):
			generic = "[audio]"
		case strings.HasPrefix(tag, "[video:"):
			generic = "[video]"
		case strings.HasPrefix(tag, "[file:"):
			generic = "[file]"
		}

		if generic != "" && strings.Contains(content, generic) {
			content = strings.Replace(content, generic, tag, 1)
		} else if content == "" {
			content = tag
		} else {
			content += " " + tag
		}
	}
	return content
}

const imageDescriptionSystemPrompt = "Describe this image concisely but thoroughly. Focus on text content, visual elements, and any details relevant to understanding the image. If the image contains text, transcribe it. Output only the description."

// describeImagesInMessages replaces data:image/* entries in Message.Media
// with text descriptions generated by a vision model. The data URL is removed
// from Media and the "[image: photo]" tag in Content is replaced with the description.
// A braille spinner is shown via draft messages while processing.
// Returns a new slice; original messages are not mutated.
func (al *AgentLoop) describeImagesInMessages(
	ctx context.Context, messages []providers.Message, agent *AgentInstance,
	channel, chatID string,
) []providers.Message {
	// Count images to decide whether to show indicator
	var imageCount int
	for _, m := range messages {
		for _, ref := range m.Media {
			if strings.HasPrefix(ref, "data:image/") {
				imageCount++
			}
		}
	}
	if imageCount == 0 {
		return messages
	}

	result := make([]providers.Message, len(messages))
	copy(result, messages)

	// Start processing indicator
	label := "Processing image..."
	if imageCount > 1 {
		label = fmt.Sprintf("Processing %d images...", imageCount)
	}
	indicator := al.processingIndicator(ctx, channel, chatID, label)
	defer indicator.Stop()

	for i, m := range result {
		if len(m.Media) == 0 {
			continue
		}

		var descriptions []string
		var kept []string
		for _, mediaURL := range m.Media {
			if !strings.HasPrefix(mediaURL, "data:image/") {
				kept = append(kept, mediaURL)
				continue
			}

			desc := al.describeImage(ctx, mediaURL, m.Content, agent)
			descriptions = append(descriptions, desc)
		}

		if len(descriptions) == 0 {
			continue
		}

		result[i].Media = kept
		result[i].Content = injectImageDescriptions(result[i].Content, descriptions)
	}

	return result
}

// describeImage calls a vision model to describe a single image.
// Results are cached by content hash to avoid redundant API calls.
// Returns the description text, or a placeholder on error.
func (al *AgentLoop) describeImage(
	ctx context.Context, dataURL, userContext string, agent *AgentInstance,
) string {
	// Check cache first
	hash := mediacache.HashData([]byte(dataURL))
	if al.mediaCache != nil {
		if cached, ok := al.mediaCache.Get(hash, mediacache.TypeImageDesc); ok {
			logger.InfoCF("agent", "Image description (cached)", map[string]any{
				"hash":        hash,
				"description": cached,
			})
			return cached
		}
	}

	messages := []providers.Message{
		{Role: "system", Content: imageDescriptionSystemPrompt},
		{
			Role:    "user",
			Content: userContext,
			Media:   []string{dataURL},
		},
	}

	candidates := agent.ImageCandidates
	if len(candidates) == 0 {
		return "description unavailable"
	}

	var resp *providers.LLMResponse
	var err error

	opts := map[string]any{
		"max_tokens":  1024,
		"temperature": 0.3,
	}

	if len(candidates) > 1 && al.fallback != nil {
		fbResult, fbErr := al.fallback.Execute(ctx, candidates,
			func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
				p := al.resolveProvider(provider, model, agent.Provider)
				return p.Chat(ctx, messages, nil, model, opts)
			},
		)
		if fbErr != nil {
			err = fbErr
		} else {
			resp = fbResult.Response
		}
	} else {
		c := candidates[0]
		p := al.resolveProvider(c.Provider, c.Model, agent.Provider)
		resp, err = p.Chat(ctx, messages, nil, c.Model, opts)
	}

	if err != nil {
		logger.WarnCF("agent", "Image description failed", map[string]any{"error": err.Error()})
		return "description unavailable"
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return "description unavailable"
	}

	desc := strings.TrimSpace(resp.Content)

	logger.InfoCF("agent", "Image described", map[string]any{
		"hash":        hash,
		"description": desc,
	})

	// Store in cache
	if al.mediaCache != nil {
		if cErr := al.mediaCache.Put(hash, mediacache.TypeImageDesc, desc); cErr != nil {
			logger.WarnCF("agent", "Failed to cache image description", map[string]any{"error": cErr.Error()})
		}
	}

	return desc
}

// brailleSpinnerFrames is a set of braille characters that produce
// a smooth rotating animation when displayed sequentially.
var brailleSpinnerFrames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressIndicator manages a braille spinner with a dynamically updatable label.
// Call UpdateLabel to change the displayed text during processing.
type progressIndicator struct {
	label atomic.Value // string
	done  chan struct{}
}

// UpdateLabel changes the label shown alongside the spinner.
func (p *progressIndicator) UpdateLabel(label string) {
	p.label.Store(label)
}

// Stop terminates the spinner goroutine.
func (p *progressIndicator) Stop() {
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

// processingIndicator publishes draft status messages with a braille spinner
// animation to indicate active processing. It runs until Stop is called.
// Use UpdateLabel to change the displayed text during long operations.
func (al *AgentLoop) processingIndicator(ctx context.Context, channel, chatID, label string) *progressIndicator {
	p := &progressIndicator{done: make(chan struct{})}
	p.label.Store(label)

	if al.bus == nil || channel == "" || chatID == "" {
		return p
	}

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		frame := 0
		for {
			select {
			case <-p.done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				lbl, _ := p.label.Load().(string)
				content := fmt.Sprintf("%s %s", brailleSpinnerFrames[frame%len(brailleSpinnerFrames)], lbl)
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel:  channel,
					ChatID:   chatID,
					Content:  content,
					IsStatus: true,
				})
				frame++
			}
		}
	}()

	return p
}

// injectImageDescriptions replaces "[image: photo]" tags in content with
// "[image: <description>]" for each description provided.
func injectImageDescriptions(content string, descriptions []string) string {
	for _, desc := range descriptions {
		replacement := "[image: " + desc + "]"
		if strings.Contains(content, "[image: photo]") {
			content = strings.Replace(content, "[image: photo]", replacement, 1)
		} else if content == "" {
			content = replacement
		} else {
			content += " " + replacement
		}
	}
	return content
}

// maxPreviewRunes is the maximum number of runes to store as preview
// in the media cache for PDF OCR results.
const maxPreviewRunes = 500

// figureKeywords triggers --figure --figure_letter when found in the message.
//
//nolint:gosmopolitan // intentional CJK keywords for Japanese users
var figureKeywords = []string{
	"figure", "figures", "with images",
	"図版", "図付き", "画像付き", "図も",
}

// wantFigures returns true if the message content contains a figure keyword.
func wantFigures(content string) bool {
	lower := strings.ToLower(content)
	for _, kw := range figureKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// readingOrderKeywords maps message keywords to yomitoku --reading_order values.
//
//nolint:gosmopolitan // intentional CJK keywords for Japanese users
var readingOrderKeywords = []struct {
	keyword string
	order   string
}{
	{"right2left", "right2left"},
	{"top2bottom", "top2bottom"},
	{"left2right", "left2right"},
	{"縦書き", "right2left"},
	{"たてがき", "right2left"},
	{"vertical", "right2left"},
	{"横書き", "top2bottom"},
	{"よこがき", "top2bottom"},
	{"horizontal", "top2bottom"},
}

// detectReadingOrder returns the reading order specified in the message content.
// Falls back to the configured default, then "auto".
func detectReadingOrder(content string, cfgDefault string) string {
	lower := strings.ToLower(content)
	for _, kw := range readingOrderKeywords {
		if strings.Contains(lower, kw.keyword) {
			return kw.order
		}
	}
	if cfgDefault != "" {
		return cfgDefault
	}
	return "auto"
}

const pdfHintMessage = "PDF OCR in progress. " +
	"Tip: include \"figures\" or \"\u56f3\u7248\" to extract images. " +
	"Add \"\u7e26\u66f8\u304d\" or \"\u6a2a\u66f8\u304d\" to set reading order."

// processPDFsInMessages finds [file:/path.pdf] tags in messages and replaces
// them with [document: preview... (full: /path/to.md, N pages)] tags after
// running OCR. A braille spinner with page progress is shown during processing.
func (al *AgentLoop) processPDFsInMessages(
	ctx context.Context, messages []providers.Message, ocrCfg *config.OCRConfig,
	channel, chatID string,
) []providers.Message {
	result := make([]providers.Message, len(messages))
	copy(result, messages)

	for i, m := range result {
		if !strings.Contains(m.Content, "[file:") {
			continue
		}
		withFigures := wantFigures(m.Content)
		var cfgRO string
		if ocrCfg != nil {
			cfgRO = ocrCfg.ReadingOrder
		}
		readingOrder := detectReadingOrder(m.Content, cfgRO)
		result[i].Content = al.replacePDFTags(
			ctx, m.Content, ocrCfg, channel, chatID, withFigures, readingOrder,
		)
	}

	return result
}

// pdfTagPrefix is the file tag pattern for PDF files injected by resolveMediaRefs.
const pdfTagPrefix = "[file:"

// replacePDFTags finds [file:*.pdf] tags and replaces them with OCR results.
func (al *AgentLoop) replacePDFTags(
	ctx context.Context, content string, ocrCfg *config.OCRConfig,
	channel, chatID string, withFigures bool, readingOrder string,
) string {
	var out strings.Builder
	rest := content

	for {
		idx := strings.Index(rest, pdfTagPrefix)
		if idx < 0 {
			out.WriteString(rest)
			break
		}
		endRel := strings.Index(rest[idx:], "]")
		if endRel < 0 {
			out.WriteString(rest)
			break
		}
		end := idx + endRel + 1

		tag := rest[idx:end]
		path := tag[len(pdfTagPrefix) : len(tag)-1]

		out.WriteString(rest[:idx])

		if strings.HasSuffix(strings.ToLower(path), ".pdf") {
			out.WriteString(al.ocrPDF(ctx, path, ocrCfg, channel, chatID, withFigures, readingOrder))
		} else {
			out.WriteString(tag)
		}

		rest = rest[end:]
	}

	return out.String()
}

// ocrPDF runs OCR on a PDF file and returns a document tag with preview.
// Uses the media cache to avoid redundant OCR runs.
// When withFigures is true, --figure and --figure_letter flags are added.
// readingOrder is passed as --reading_order to yomitoku.
func (al *AgentLoop) ocrPDF(
	ctx context.Context, pdfPath string, ocrCfg *config.OCRConfig,
	channel, chatID string, withFigures bool, readingOrder string,
) string {
	// Hash the file content for cache lookup.
	// Include figure mode in the hash so both variants are cached separately.
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		logger.WarnCF("agent", "Failed to read PDF", map[string]any{"path": pdfPath, "error": err.Error()})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}
	hashInput := pdfData
	if withFigures {
		hashInput = append(hashInput, []byte(":figures")...)
	}
	if readingOrder != "" && readingOrder != "auto" {
		hashInput = append(hashInput, []byte(":ro="+readingOrder)...)
	}
	hash := mediacache.HashData(hashInput)

	// Check cache (both text extraction and OCR)
	if al.mediaCache != nil {
		if entry, ok := al.mediaCache.GetEntry(hash, mediacache.TypePDFText); ok {
			logger.DebugCF("agent", "PDF text cache hit", map[string]any{"hash": hash})
			return formatDocumentTag(entry.Result, entry.FilePath, entry.Pages)
		}
		if entry, ok := al.mediaCache.GetEntry(hash, mediacache.TypePDFOCR); ok {
			logger.DebugCF("agent", "PDF OCR cache hit", map[string]any{"hash": hash})
			return formatDocumentTag(entry.Result, entry.FilePath, entry.Pages)
		}
	}

	// Fast path: try pdftotext for PDFs with a text layer (skip if figures requested).
	// pdftotext is orders of magnitude faster than OCR.
	if !withFigures {
		if text, pages, ok := tryPdftotextExtract(ctx, pdfPath); ok {
			return al.savePdftotextResult(pdfPath, text, pages, hash)
		}
	}

	// Get page count for progress display (use pdfinfo, fall back to regex)
	totalPages := pdfinfoPageCount(ctx, pdfPath)
	if totalPages == 0 {
		totalPages = mediacache.PDFPageCount(pdfPath)
	}
	totalStr := mediacache.FormatPageCount(totalPages)

	// Send hint message (only if not already sent by Phase 1 waitForPDFFollowUp)
	if al.bus != nil && channel != "" && chatID != "" {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:         channel,
			ChatID:          chatID,
			Content:         pdfHintMessage,
			SkipPlaceholder: true,
			IsStatus:        true,
		})
	}

	modeLabel := "Processing PDF"
	if withFigures && readingOrder != "" && readingOrder != "auto" {
		modeLabel = fmt.Sprintf("Processing PDF (figures, %s)", readingOrder)
	} else if withFigures {
		modeLabel = "Processing PDF (with figures)"
	} else if readingOrder != "" && readingOrder != "auto" {
		modeLabel = fmt.Sprintf("Processing PDF (%s)", readingOrder)
	}
	indicator := al.processingIndicator(ctx, channel, chatID,
		fmt.Sprintf("%s (0/%s)...", modeLabel, totalStr))
	defer indicator.Stop()

	// Determine output directory for OCR results
	outputDir := al.ocrOutputDir()
	os.MkdirAll(outputDir, 0o755)

	// Build command
	timeout := time.Duration(ocrCfg.GetOCRTimeout()) * time.Second
	cmdCtx, cmdCancel := context.WithTimeout(ctx, timeout)
	defer cmdCancel()

	args := make([]string, 0, len(ocrCfg.Args)+8)
	args = append(args, ocrCfg.Args...)
	if withFigures {
		args = append(args, "--figure", "--figure_letter")
	}
	if readingOrder != "" && readingOrder != "auto" {
		args = append(args, "--reading_order", readingOrder)
	}
	args = append(args, pdfPath, "-o", outputDir)

	cmd := exec.CommandContext(cmdCtx, ocrCfg.Command, args...)

	// Set environment
	if len(ocrCfg.Env) > 0 {
		cmd.Env = append(os.Environ(), ocrEnvSlice(ocrCfg.Env)...)
	}

	// Pipe stderr for progress tracking
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.WarnCF("agent", "Failed to create stderr pipe", map[string]any{"error": err.Error()})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	logger.InfoCF("agent", "Starting PDF OCR", map[string]any{
		"path":          pdfPath,
		"pages":         totalStr,
		"cmd":           ocrCfg.Command,
		"reading_order": readingOrder,
	})

	if startErr := cmd.Start(); startErr != nil {
		logger.WarnCF("agent", "Failed to start OCR command", map[string]any{"error": startErr.Error()})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	// Track progress via stderr, keep last lines for error diagnosis
	page := 0
	var stderrTail []string
	const maxTailLines = 20
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "TextDetector __call__") {
			page++
			indicator.UpdateLabel(fmt.Sprintf("%s (%d/%s)...", modeLabel, page, totalStr))
		}
		stderrTail = append(stderrTail, line)
		if len(stderrTail) > maxTailLines {
			stderrTail = stderrTail[1:]
		}
	}

	if waitErr := cmd.Wait(); waitErr != nil {
		logger.WarnCF("agent", "OCR command failed", map[string]any{
			"path":   pdfPath,
			"error":  waitErr.Error(),
			"stderr": strings.Join(stderrTail, "\n"),
		})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	// Clean up page images (_pN.jpg) generated by yomitoku.
	// These are always created and cannot be suppressed via CLI options.
	// Keep: .md files, figures/ directory (referenced by markdown output).
	cleanupOCRPageImages(outputDir, pdfPath)

	// Find the output markdown file
	mdPath := findOCROutput(outputDir, pdfPath)
	if mdPath == "" {
		logger.WarnCF("agent", "OCR output not found", map[string]any{"output_dir": outputDir})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	// Read preview from first part of the markdown
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		logger.WarnCF("agent", "Failed to read OCR output", map[string]any{"path": mdPath, "error": err.Error()})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	preview := extractPreview(string(mdData), maxPreviewRunes)
	if totalPages == 0 {
		totalPages = page // use detected page count as fallback
	}

	// Store in cache
	if al.mediaCache != nil {
		_ = al.mediaCache.PutEntry(hash, mediacache.TypePDFOCR, mediacache.Entry{
			Result:   preview,
			FilePath: mdPath,
			Pages:    totalPages,
		})
	}

	logger.InfoCF("agent", "PDF OCR completed", map[string]any{
		"path":    pdfPath,
		"pages":   totalPages,
		"md_path": mdPath,
	})

	return formatDocumentTag(preview, mdPath, totalPages)
}

// formatDocumentTag creates the tag injected into message content.
func formatDocumentTag(preview, mdPath string, pages int) string {
	pagesStr := mediacache.FormatPageCount(pages)
	return fmt.Sprintf("[document: %s\n  full: %s (%s pages)\n  Use read_file to see the complete document.]",
		preview, mdPath, pagesStr)
}

// extractPreview returns the first maxRunes runes of text, appending "..." if truncated.
func extractPreview(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

// ocrOutputDir returns the directory for storing OCR markdown output files.
func (al *AgentLoop) ocrOutputDir() string {
	registry := al.GetRegistry()
	if agent := registry.GetDefaultAgent(); agent != nil {
		return filepath.Join(agent.Workspace, ".ocr_cache")
	}
	return filepath.Join(os.TempDir(), "picoclaw-ocr")
}

// findOCROutput locates the markdown file generated by yomitoku.
// yomitoku names output as <basename>.md or <basename>_combined.md in the output dir.
func findOCROutput(outputDir, pdfPath string) string {
	base := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))

	// Try common yomitoku output patterns
	candidates := []string{
		filepath.Join(outputDir, base+".md"),
		filepath.Join(outputDir, base+"_combined.md"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Fallback: find any .md file in the output directory
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return filepath.Join(outputDir, e.Name())
		}
	}

	return ""
}

// ocrEnvSlice converts a map to "KEY=VALUE" slice for exec.Cmd.Env.
func ocrEnvSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// cleanupOCRPageImages removes _pN.jpg files generated by yomitoku.
// These per-page images are always created by the CLI and cannot be suppressed.
// Only top-level _pN.jpg files matching the PDF basename are removed;
// the figures/ subdirectory and .md files are preserved.
func cleanupOCRPageImages(outputDir, pdfPath string) {
	base := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return
	}

	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Match pattern: <dirname>_<basename>_pN.jpg
		if !strings.HasSuffix(strings.ToLower(name), ".jpg") {
			continue
		}
		if !strings.Contains(name, base+"_p") {
			continue
		}
		if rmErr := os.Remove(filepath.Join(outputDir, name)); rmErr == nil {
			removed++
		}
	}
	if removed > 0 {
		logger.DebugCF("agent", "Cleaned up OCR page images", map[string]any{
			"dir":     outputDir,
			"removed": removed,
		})
	}
}
