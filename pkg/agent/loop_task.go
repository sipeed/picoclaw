package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// activeTask tracks a running agent task for live status and intervention.

type activeTask struct {
	Description string

	Result string // LLM response summary for completion notification

	Iteration int

	MaxIter int

	StartedAt time.Time

	cancel context.CancelFunc

	interrupt chan string // buffered 1, for user message injection

	toolLog []toolLogEntry

	lastError *toolLogEntry // sticky: most recent error, persists across iterations

	projectDir string // detected from exec cd target (authoritative)

	fileCommonDir string // LCP of file paths relative to workspace (fallback)

	streamedChunks bool // true after onChunk fires at least once

	messageContent string // last content sent by the message tool (for inclusion in completion)

	mu sync.Mutex
}

// toolLogEntry records a single tool call for the live terminal view.

type toolLogEntry struct {
	Name string

	ArgsSnip string // first ~80 chars of args

	Result string // "✓ 4.9s" or "✗ 3.2s"

	ErrDetail string // non-empty on error — e.g. "Exit code: exit status 1"
}

// maxToolLogEntries limits the sliding window of tool log entries

// kept in memory and displayed in status messages.

const maxToolLogEntries = 5

// Task reminder constants and helpers.

const (
	taskReminderMaxChars = 500

	blockerMaxChars = 200
)

func shouldInjectReminder(iteration, interval int) bool {
	if interval <= 0 {
		return false
	}

	return iteration > 1 && iteration%interval == 0
}

func buildTaskReminder(userMessage string, lastBlocker string) providers.Message {
	truncatedTask := utils.Truncate(userMessage, taskReminderMaxChars)

	var content string

	if lastBlocker != "" {
		truncatedBlocker := utils.Truncate(lastBlocker, blockerMaxChars)

		content = fmt.Sprintf(
			"[TASK REMINDER]\nOriginal task:\n---\n%s\n---\nLast blocker:\n---\n%s\n---\nFix the blocker if essential, or find an alternative. If all steps are complete, move on.",
			truncatedTask,
			truncatedBlocker,
		)
	} else {
		content = fmt.Sprintf(
			"[TASK REMINDER]\nOriginal task:\n---\n%s\n---\nIf all steps of the original task are complete, move on. Otherwise, continue with the next step.",
			truncatedTask,
		)
	}

	return providers.Message{
		Role: "user",

		Content: content,
	}
}

// cdPrefixPattern matches "cd /some/path && " at the start of a shell command.

// Group 1 captures the target directory path.

var cdPrefixPattern = regexp.MustCompile(`^cd\s+(\S+)\s*&&\s*`)

// optFlagPattern matches option flags like --verbose, -v, --timeout=60, -q.

// Only standalone flags are removed; flags whose value is the next positional

// argument (e.g. "-A 20") are kept because removing them would lose context.

var optFlagPattern = regexp.MustCompile(`\s+--?\w[\w-]*(=\S*)?`)

// extractExecProjectDir extracts the basename of an exec cd target.

// Returns "" if the command has no cd prefix.

func extractExecProjectDir(args map[string]any) string {
	cmd, _ := args["command"].(string)

	if cmd == "" {
		return ""
	}

	m := cdPrefixPattern.FindStringSubmatch(cmd)

	if len(m) < 2 {
		return ""
	}

	cdPath := strings.TrimRight(m[1], "/\\")

	if idx := strings.LastIndex(cdPath, "/"); idx >= 0 {
		return cdPath[idx+1:]
	}

	if idx := strings.LastIndex(cdPath, "\\"); idx >= 0 {
		return cdPath[idx+1:]
	}

	return cdPath
}

// fileParentRelDir returns the parent directory of a file path, relative to

// workspace. Returns "" if the path is not under workspace or has no parent.

func fileParentRelDir(filePath, workspace string) string {
	ws := strings.TrimRight(workspace, "/\\")

	if ws == "" {
		return ""
	}

	rest := strings.TrimPrefix(filePath, ws)

	if rest == filePath {
		return "" // not under workspace
	}

	rest = strings.TrimLeft(rest, "/\\")

	// Remove the filename — keep only the directory part

	if idx := strings.LastIndexAny(rest, "/\\"); idx >= 0 {
		return rest[:idx]
	}

	return "" // file is directly under workspace, no meaningful dir
}

// commonDirPrefix computes the longest common directory prefix of two

// slash-separated paths. Returns "" if there is no common component.

func commonDirPrefix(a, b string) string {
	partsA := strings.Split(a, "/")

	partsB := strings.Split(b, "/")

	n := len(partsA)

	if len(partsB) < n {
		n = len(partsB)
	}

	common := 0

	for i := 0; i < n; i++ {
		if partsA[i] != partsB[i] {
			break
		}

		common = i + 1
	}

	if common == 0 {
		return ""
	}

	return strings.Join(partsA[:common], "/")
}

// displayProjectDir returns the project directory name for status display.

// Prefers the authoritative exec-based projectDir; falls back to the

// basename of the file-based common directory.

func displayProjectDir(task *activeTask) string {
	if task.projectDir != "" {
		return task.projectDir
	}

	if task.fileCommonDir != "" {
		dir := task.fileCommonDir

		if idx := strings.LastIndex(dir, "/"); idx >= 0 {
			return dir[idx+1:]
		}

		return dir
	}

	return ""
}

// buildArgsSnippet produces a human-friendly snippet for the tool log.

// For exec: extracts the command and strips the leading "cd <workspace> && ".

// For file tools: extracts the path and strips the workspace prefix.

// Falls back to raw JSON truncation.

func buildArgsSnippet(toolName string, args map[string]any, workspace string) string {
	switch toolName {
	case "exec":

		cmd, _ := args["command"].(string)

		if cmd == "" {
			break
		}

		cmd = cdPrefixPattern.ReplaceAllString(cmd, "")

		cmd = optFlagPattern.ReplaceAllString(cmd, "")

		return utils.Truncate(cmd, 80)

	case "read_file", "write_file", "edit_file", "append_file", "list_dir":

		path, _ := args["path"].(string)

		if path == "" {
			break
		}

		if workspace != "" {
			path = strings.TrimPrefix(path, workspace)

			path = strings.TrimPrefix(path, "/")
		}

		// Prioritize filename: if path is too long, show "…/filename"

		const maxPath = 60

		if runes := []rune(path); len(runes) > maxPath {
			// Find last slash to extract filename

			if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
				filename := path[lastSlash:] // includes "/"

				dirBudget := maxPath - len([]rune(filename)) - 1 // 1 for "…"

				if dirBudget > 0 {
					dir := []rune(path[:lastSlash])

					if len(dir) > dirBudget {
						dir = dir[:dirBudget]
					}

					path = string(dir) + "\u2026" + filename
				} else {
					path = "\u2026" + filename
				}
			} else {
				path = utils.Truncate(path, maxPath)
			}
		}

		return path
	}

	// Default: raw JSON truncated

	argsJSON, _ := json.Marshal(args)

	return utils.Truncate(string(argsJSON), 80)
}

// maxEntryLineWidth is the max rune count for a single-line log entry.

// Telegram chat bubbles on mobile are roughly 40-45 chars wide.

const maxEntryLineWidth = 42

// isFileToolEntry returns true if the entry name contains a file-operation tool.

func isFileToolEntry(name string) bool {
	for _, t := range []string{"read_file", "write_file", "edit_file", "append_file", "list_dir"} {
		if strings.Contains(name, t) {
			return true
		}
	}

	return false
}

// formatCompactEntry formats a finished tool log entry as a fixed single line.

// The result marker (✓/✗) is always shown at the end regardless of truncation.

// File tools omit duration (always near-instant); paths truncate from the

// start so the filename is always visible.

func formatCompactEntry(entry toolLogEntry) string {
	result := entry.Result

	if result == "" {
		result = "\u23F3" // ⏳
	}

	// File tools: strip duration, keep only marker (✓/✗/⏳)

	isFile := isFileToolEntry(entry.Name)

	if isFile {
		if r := []rune(result); len(r) > 0 {
			result = string(r[0:1]) // just the symbol
		}
	}

	// Budget for ArgsSnip: total - name - " " - " " - result

	nameLen := utf8.RuneCountInString(entry.Name)

	resultLen := utf8.RuneCountInString(result)

	argsBudget := maxEntryLineWidth - nameLen - 1 - 1 - resultLen

	args := entry.ArgsSnip

	if args != "" && argsBudget > 3 {
		argsRunes := []rune(args)

		if len(argsRunes) > argsBudget {
			// Paths: truncate from the start, keeping the filename visible

			if strings.Contains(args, "/") {
				args = "\u2026" + string(argsRunes[len(argsRunes)-argsBudget+1:])
			} else {
				args = string(argsRunes[:argsBudget-1]) + "\u2026"
			}
		}

		var sb strings.Builder

		sb.Grow(len(entry.Name) + 1 + len(args) + 1 + len(result))

		sb.WriteString(entry.Name)

		sb.WriteByte(' ')

		sb.WriteString(args)

		sb.WriteByte(' ')

		sb.WriteString(result)

		return sb.String()
	}

	// No room for args or args empty

	var sb strings.Builder

	sb.Grow(len(entry.Name) + 1 + len(result))

	sb.WriteString(entry.Name)

	sb.WriteByte(' ')

	sb.WriteString(result)

	return sb.String()
}

// formatLatestEntry formats the latest entry command without its result marker.

// Since the result goes on the next line, the full width is available for the command.

func formatLatestEntry(entry toolLogEntry) string {
	nameLen := utf8.RuneCountInString(entry.Name)

	argsBudget := maxEntryLineWidth - nameLen - 1 // name + space + args (no result)

	args := entry.ArgsSnip

	if args != "" && argsBudget > 3 {
		argsRunes := []rune(args)

		if len(argsRunes) > argsBudget {
			if strings.Contains(args, "/") {
				args = "\u2026" + string(argsRunes[len(argsRunes)-argsBudget+1:])
			} else {
				args = string(argsRunes[:argsBudget-1]) + "\u2026"
			}
		}

		var sb strings.Builder

		sb.Grow(len(entry.Name) + 1 + len(args))

		sb.WriteString(entry.Name)

		sb.WriteByte(' ')

		sb.WriteString(args)

		return sb.String()
	}

	return entry.Name
}

// compressRepeats reduces runs of 3+ identical non-alphanumeric, non-space

// characters to just 2. e.g. "======" → "==", "---" → "--".

func compressRepeats(s string) string {
	runes := []rune(s)

	if len(runes) < 3 {
		return s
	}

	var sb strings.Builder

	sb.Grow(len(s))

	i := 0

	for i < len(runes) {
		r := runes[i]

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			j := i + 1

			for j < len(runes) && runes[j] == r {
				j++
			}

			if j-i >= 3 {
				sb.WriteRune(r)

				sb.WriteRune(r)

				i = j

				continue
			}
		}

		sb.WriteRune(r)

		i++
	}

	return sb.String()
}

// Display layout constants.

const (
	displayPastEntries = 4 // number of compact 1-line past entries

	displayErrorLines = 5 // content lines inside the error code block

	statusSeparator = "\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n"

	streamingDisplayLines = 17 // line count matching buildRichStatus output

)

// buildRichStatus builds a fixed-height terminal-like status display.

//

// Layout (always the same number of lines):

//

//	🔄 Task in progress (N/M)          header

//	📁 workspace-path                   header

//	━━━━━━━━━━                          separator

//	[N] compact-past-1 ✓ Xs            past (1 line each)

//	[N] compact-past-2 ✗ Xs            past

//	[N] compact-past-3 ✓ Xs            past

//	[N] compact-past-4 ✓ Xs            past

//	[N] latest-command                  latest (no result, wider args)

//	  ⏳                                latest result

//	                                    reserved

//	```                                 error fence

//	err-line / placeholder              error body (5 lines)

//	```                                 error fence

//	↩️ Reply to intervene               footer (background only)

func buildRichStatus(task *activeTask, isBackground bool, workspace string) string {
	task.mu.Lock()

	defer task.mu.Unlock()

	var sb strings.Builder

	// --- Header ---

	sb.WriteString("\U0001F504 Task in progress (")

	sb.WriteString(strconv.Itoa(task.Iteration))

	sb.WriteByte('/')

	sb.WriteString(strconv.Itoa(task.MaxIter))

	sb.WriteString(")\n")

	// Project directory: exec cd (authoritative) → file LCP → workspace basename

	sb.WriteString("\U0001F4C1 ")

	if dir := displayProjectDir(task); dir != "" {
		sb.WriteString(dir)
	} else if workspace != "" {
		project := strings.TrimRight(workspace, "/\\")

		if idx := strings.LastIndex(project, "/"); idx >= 0 {
			project = project[idx+1:]
		} else if idx := strings.LastIndex(project, "\\"); idx >= 0 {
			project = project[idx+1:]
		}

		sb.WriteString(project)
	}

	sb.WriteByte('\n')

	sb.WriteString(statusSeparator)

	// --- Task entries (displayPastEntries + 2 lines for latest) ---

	entries := task.toolLog

	if len(entries) > maxToolLogEntries {
		entries = entries[len(entries)-maxToolLogEntries:]
	}

	var pastEntries []toolLogEntry

	var latest *toolLogEntry

	if len(entries) > 0 {
		latest = &entries[len(entries)-1]

		if len(entries) > 1 {
			start := len(entries) - 1 - displayPastEntries

			if start < 0 {
				start = 0
			}

			pastEntries = entries[start : len(entries)-1]
		}
	}

	// Past entries: exactly displayPastEntries lines (pad if fewer)

	for i := 0; i < displayPastEntries; i++ {
		if i < len(pastEntries) {
			sb.WriteString(formatCompactEntry(pastEntries[i]))
		} else {
			sb.WriteString("\u2800")
		}

		sb.WriteByte('\n')
	}

	// Latest entry: command on one line, result on next

	if latest != nil {
		sb.WriteString(formatLatestEntry(*latest))

		sb.WriteByte('\n')

		sb.WriteString("  ")

		if latest.Result != "" {
			sb.WriteString(latest.Result)
		} else {
			sb.WriteString("\u23F3")
		}

		sb.WriteByte('\n')
	} else {
		sb.WriteString("\u23F3 waiting...\n")

		sb.WriteString("\u2800\n")
	}

	// Reserved (1 line)

	sb.WriteString("\u2800\n")

	// --- Error region (code fence, no separator) ---

	sb.WriteString("```\n")

	errEntry := task.lastError

	if errEntry != nil {
		sb.WriteString("\u274C ")

		sb.WriteString(formatCompactEntry(*errEntry))

		sb.WriteByte('\n')

		var detailLines []string

		if errEntry.ErrDetail != "" {
			detailLines = strings.Split(errEntry.ErrDetail, "\n")
		}

		for i := 0; i < displayErrorLines-1; i++ {
			if i < len(detailLines) {
				line := compressRepeats(detailLines[i])

				if runes := []rune(line); len(runes) > maxEntryLineWidth {
					line = string(runes[:maxEntryLineWidth-1]) + "\u2026"
				}

				sb.WriteString(line)
			} else {
				sb.WriteString("\u2800")
			}

			sb.WriteByte('\n')
		}
	} else {
		sb.WriteString("\u2714 No errors\n")

		for i := 0; i < displayErrorLines-1; i++ {
			sb.WriteString("\u2800\n")
		}
	}

	sb.WriteString("```\n")

	if isBackground {
		sb.WriteString("\u21A9\uFE0F Reply to intervene")
	}

	return sb.String()
}
