package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CmdFunc is the signature for a built-in shell command.
// It receives the arguments (after the command name) and the working directory.
type CmdFunc func(args []string, cwd string) string

// BuiltinCmds maps command names to their Go implementations.
// These run cross-platform without external dependencies.
var BuiltinCmds = map[string]CmdFunc{
	"ls":    cmdLs,
	"dir":   cmdLs,
	"cat":   cmdCat,
	"type":  cmdCat,
	"head":  cmdHead,
	"tail":  cmdTail,
	"grep":  cmdGrep,
	"wc":    cmdWc,
	"find":  cmdFind,
	"pwd":   cmdPwd,
	"echo":  cmdEcho,
	"stat":  cmdStat,
	"diff":  cmdDiff,
	"tree":  cmdTree,
	"touch": cmdTouch,
	"mkdir": cmdMkdir,
	"cp":    cmdCp,
	"mv":    cmdMv,
}

// DevToolPassthrough lists commands that pass through to the system shell.
var DevToolPassthrough = map[string]bool{
	"go": true, "git": true, "node": true, "python": true, "python3": true,
	"npm": true, "npx": true, "cargo": true, "make": true,
	"jq": true, "rg": true, "ag": true, "ack": true, "fd": true,
}

// ---------------------------------------------------------------------------
// ls / dir
// ---------------------------------------------------------------------------

func cmdLs(args []string, cwd string) string {
	dir := cwd
	showAll := false
	longFmt := false

	for _, a := range args {
		switch {
		case a == "-a":
			showAll = true
		case a == "-l":
			longFmt = true
		case a == "-la" || a == "-al":
			showAll = true
			longFmt = true
		case !strings.HasPrefix(a, "-"):
			dir = ResolvePath(a, cwd)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("ls: %v", err)
	}

	var sb strings.Builder
	for _, e := range entries {
		name := e.Name()
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}
		if longFmt {
			info, _ := e.Info()
			if info != nil {
				mode := info.Mode().String()
				size := info.Size()
				mod := info.ModTime().Format("Jan 02 15:04")
				if e.IsDir() {
					name += "/"
				}
				fmt.Fprintf(&sb, "%s %8d %s %s\n", mode, size, mod, name)
			} else {
				fmt.Fprintf(&sb, "%s\n", name)
			}
		} else {
			if e.IsDir() {
				name += "/"
			}
			sb.WriteString(name + "\n")
		}
	}
	if sb.Len() == 0 {
		return "(empty directory)"
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// cat / type
// ---------------------------------------------------------------------------

func cmdCat(args []string, cwd string) string {
	if len(args) == 0 {
		return "cat: missing file operand"
	}
	var sb strings.Builder
	for _, f := range args {
		if strings.HasPrefix(f, "-") {
			continue
		}
		data, err := os.ReadFile(ResolvePath(f, cwd))
		if err != nil {
			fmt.Fprintf(&sb, "cat: %v\n", err)
			continue
		}
		sb.Write(data)
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// head
// ---------------------------------------------------------------------------

func cmdHead(args []string, cwd string) string {
	n := 10
	var file string
	for i := 0; i < len(args); i++ {
		if args[i] == "-n" && i+1 < len(args) {
			n, _ = strconv.Atoi(args[i+1])
			i++
		} else if !strings.HasPrefix(args[i], "-") {
			file = args[i]
		}
	}
	if file == "" {
		return "head: missing file"
	}
	data, err := os.ReadFile(ResolvePath(file, cwd))
	if err != nil {
		return fmt.Sprintf("head: %v", err)
	}
	lines := strings.SplitN(string(data), "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// tail
// ---------------------------------------------------------------------------

func cmdTail(args []string, cwd string) string {
	n := 10
	var file string
	for i := 0; i < len(args); i++ {
		if args[i] == "-n" && i+1 < len(args) {
			n, _ = strconv.Atoi(args[i+1])
			i++
		} else if !strings.HasPrefix(args[i], "-") {
			file = args[i]
		}
	}
	if file == "" {
		return "tail: missing file"
	}
	data, err := os.ReadFile(ResolvePath(file, cwd))
	if err != nil {
		return fmt.Sprintf("tail: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	return strings.Join(lines[start:], "\n")
}

// ---------------------------------------------------------------------------
// grep
// ---------------------------------------------------------------------------

func cmdGrep(args []string, cwd string) string {
	ignoreCase := false
	showLineNum := false
	recursive := false
	var pattern string
	var paths []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && pattern == "" {
			for _, ch := range a[1:] {
				switch ch {
				case 'i':
					ignoreCase = true
				case 'n':
					showLineNum = true
				case 'r', 'R':
					recursive = true
				}
			}
		} else if pattern == "" {
			pattern = a
		} else {
			paths = append(paths, a)
		}
	}

	if pattern == "" {
		return "grep: missing pattern"
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	pat := pattern
	if ignoreCase {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return fmt.Sprintf("grep: invalid pattern: %v", err)
	}

	var sb strings.Builder
	matchCount := 0
	maxMatches := 200

	var searchFile func(path string)
	searchFile = func(path string) {
		if matchCount >= maxMatches {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		if IsBinary(data) {
			return
		}
		relPath, _ := filepath.Rel(cwd, path)
		if relPath == "" {
			relPath = path
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if matchCount >= maxMatches {
				break
			}
			if re.MatchString(line) {
				matchCount++
				if showLineNum {
					fmt.Fprintf(&sb, "%s:%d:%s\n", relPath, i+1, line)
				} else {
					fmt.Fprintf(&sb, "%s:%s\n", relPath, line)
				}
			}
		}
	}

	skipDirs := map[string]bool{".git": true, "node_modules": true, "vendor": true, "__pycache__": true}

	for _, p := range paths {
		resolved := ResolvePath(p, cwd)
		info, err := os.Stat(resolved)
		if err != nil {
			fmt.Fprintf(&sb, "grep: %v\n", err)
			continue
		}
		if info.IsDir() {
			if !recursive {
				fmt.Fprintf(&sb, "grep: %s: is a directory\n", p)
				continue
			}
			_ = filepath.Walk(resolved, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if fi.IsDir() {
					if skipDirs[fi.Name()] || strings.HasPrefix(fi.Name(), ".") {
						return filepath.SkipDir
					}
					return nil
				}
				searchFile(path)
				return nil
			})
		} else {
			searchFile(resolved)
		}
	}

	if matchCount == 0 {
		return "(no matches)"
	}
	if matchCount >= maxMatches {
		fmt.Fprintf(&sb, "\n... (truncated at %d matches)\n", maxMatches)
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// wc
// ---------------------------------------------------------------------------

func cmdWc(args []string, cwd string) string {
	countLines := false
	countWords := false
	countBytes := false
	var files []string

	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			for _, ch := range a[1:] {
				switch ch {
				case 'l':
					countLines = true
				case 'w':
					countWords = true
				case 'c':
					countBytes = true
				}
			}
		} else {
			files = append(files, a)
		}
	}
	if !countLines && !countWords && !countBytes {
		countLines, countWords, countBytes = true, true, true
	}
	if len(files) == 0 {
		return "wc: missing file"
	}

	var sb strings.Builder
	totalL, totalW, totalB := 0, 0, 0

	for _, f := range files {
		data, err := os.ReadFile(ResolvePath(f, cwd))
		if err != nil {
			fmt.Fprintf(&sb, "wc: %v\n", err)
			continue
		}
		l := strings.Count(string(data), "\n")
		w := len(strings.Fields(string(data)))
		b := len(data)
		totalL += l
		totalW += w
		totalB += b

		var parts []string
		if countLines {
			parts = append(parts, fmt.Sprintf("%7d", l))
		}
		if countWords {
			parts = append(parts, fmt.Sprintf("%7d", w))
		}
		if countBytes {
			parts = append(parts, fmt.Sprintf("%7d", b))
		}
		fmt.Fprintf(&sb, "%s %s\n", strings.Join(parts, ""), f)
	}

	if len(files) > 1 {
		var parts []string
		if countLines {
			parts = append(parts, fmt.Sprintf("%7d", totalL))
		}
		if countWords {
			parts = append(parts, fmt.Sprintf("%7d", totalW))
		}
		if countBytes {
			parts = append(parts, fmt.Sprintf("%7d", totalB))
		}
		fmt.Fprintf(&sb, "%s total\n", strings.Join(parts, ""))
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// find
// ---------------------------------------------------------------------------

func cmdFind(args []string, cwd string) string {
	dir := cwd
	namePattern := ""
	typeFilter := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-name":
			if i+1 < len(args) {
				namePattern = args[i+1]
				i++
			}
		case "-type":
			if i+1 < len(args) {
				typeFilter = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && namePattern == "" {
				dir = ResolvePath(args[i], cwd)
			}
		}
	}

	skipDirs := map[string]bool{".git": true, "node_modules": true, "vendor": true}
	var sb strings.Builder
	count := 0
	maxResults := 200

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || count >= maxResults {
			return nil
		}
		name := info.Name()
		if info.IsDir() && skipDirs[name] {
			return filepath.SkipDir
		}
		if strings.HasPrefix(name, ".") && path != dir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if typeFilter == "f" && info.IsDir() {
			return nil
		}
		if typeFilter == "d" && !info.IsDir() {
			return nil
		}
		if namePattern != "" {
			matched, _ := filepath.Match(namePattern, name)
			if !matched {
				return nil
			}
		}
		rel, _ := filepath.Rel(cwd, path)
		if rel == "" {
			rel = path
		}
		sb.WriteString(rel + "\n")
		count++
		return nil
	})

	if count == 0 {
		return "(no matches)"
	}
	if count >= maxResults {
		fmt.Fprintf(&sb, "... (truncated at %d results)\n", maxResults)
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// pwd / echo / stat
// ---------------------------------------------------------------------------

func cmdPwd(_ []string, cwd string) string { return cwd }

func cmdEcho(args []string, _ string) string { return strings.Join(args, " ") }

func cmdStat(args []string, cwd string) string {
	if len(args) == 0 {
		return "stat: missing file"
	}
	var sb strings.Builder
	for _, f := range args {
		info, err := os.Stat(ResolvePath(f, cwd))
		if err != nil {
			fmt.Fprintf(&sb, "stat: %v\n", err)
			continue
		}
		fmt.Fprintf(&sb, "  File: %s\n", f)
		fmt.Fprintf(&sb, "  Size: %d bytes\n", info.Size())
		fmt.Fprintf(&sb, "  Mode: %s\n", info.Mode())
		fmt.Fprintf(&sb, "  Modified: %s\n", info.ModTime().Format(time.RFC3339))
		if info.IsDir() {
			sb.WriteString("  Type: directory\n")
		} else {
			sb.WriteString("  Type: regular file\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------

func cmdDiff(args []string, cwd string) string {
	if len(args) < 2 {
		return "diff: need two files"
	}
	data1, err := os.ReadFile(ResolvePath(args[0], cwd))
	if err != nil {
		return fmt.Sprintf("diff: %v", err)
	}
	data2, err := os.ReadFile(ResolvePath(args[1], cwd))
	if err != nil {
		return fmt.Sprintf("diff: %v", err)
	}

	lines1 := strings.Split(string(data1), "\n")
	lines2 := strings.Split(string(data2), "\n")

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", args[0], args[1])

	maxLen := len(lines1)
	if len(lines2) > maxLen {
		maxLen = len(lines2)
	}

	diffs := 0
	for i := 0; i < maxLen; i++ {
		var l1, l2 string
		if i < len(lines1) {
			l1 = lines1[i]
		}
		if i < len(lines2) {
			l2 = lines2[i]
		}
		if l1 != l2 {
			diffs++
			if diffs > 100 {
				sb.WriteString("... (too many differences)\n")
				break
			}
			fmt.Fprintf(&sb, "@@ line %d @@\n", i+1)
			if l1 != "" {
				fmt.Fprintf(&sb, "-%s\n", l1)
			}
			if l2 != "" {
				fmt.Fprintf(&sb, "+%s\n", l2)
			}
		}
	}

	if diffs == 0 {
		return "Files are identical"
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// tree
// ---------------------------------------------------------------------------

func cmdTree(args []string, cwd string) string {
	dir := cwd
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = ResolvePath(args[0], cwd)
	}

	skipDirs := map[string]bool{".git": true, "node_modules": true, "vendor": true, "__pycache__": true}
	var sb strings.Builder
	sb.WriteString(dir + "\n")
	count := 0
	maxEntries := 300

	var walk func(path, prefix string)
	walk = func(path, prefix string) {
		if count >= maxEntries {
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		var visible []os.DirEntry
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), ".") && !skipDirs[e.Name()] {
				visible = append(visible, e)
			}
		}
		sort.Slice(visible, func(i, j int) bool { return visible[i].Name() < visible[j].Name() })
		for i, e := range visible {
			if count >= maxEntries {
				sb.WriteString(prefix + "... (truncated)\n")
				return
			}
			count++
			connector := "鈹溾攢鈹€ "
			childPrefix := prefix + "鈹?  "
			if i == len(visible)-1 {
				connector = "鈹斺攢鈹€ "
				childPrefix = prefix + "    "
			}
			sb.WriteString(prefix + connector + e.Name())
			if e.IsDir() {
				sb.WriteString("/\n")
				walk(filepath.Join(path, e.Name()), childPrefix)
			} else {
				sb.WriteString("\n")
			}
		}
	}

	walk(dir, "")
	return sb.String()
}

// ---------------------------------------------------------------------------
// touch / mkdir / cp / mv
// ---------------------------------------------------------------------------

func cmdTouch(args []string, cwd string) string {
	if len(args) == 0 {
		return "touch: missing file"
	}
	for _, f := range args {
		if strings.HasPrefix(f, "-") {
			continue
		}
		p := ResolvePath(f, cwd)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, []byte{}, 0644); err != nil {
				return fmt.Sprintf("touch: %v", err)
			}
		} else {
			now := time.Now()
			_ = os.Chtimes(p, now, now)
		}
	}
	return fmt.Sprintf("touched %d file(s)", len(args))
}

func cmdMkdir(args []string, cwd string) string {
	if len(args) == 0 {
		return "mkdir: missing directory"
	}
	mkParents := false
	var dirs []string
	for _, a := range args {
		if a == "-p" {
			mkParents = true
		} else {
			dirs = append(dirs, a)
		}
	}
	for _, d := range dirs {
		p := ResolvePath(d, cwd)
		var err error
		if mkParents {
			err = os.MkdirAll(p, 0755)
		} else {
			err = os.Mkdir(p, 0755)
		}
		if err != nil {
			return fmt.Sprintf("mkdir: %v", err)
		}
	}
	return fmt.Sprintf("created %d dir(s)", len(dirs))
}

func cmdCp(args []string, cwd string) string {
	if len(args) < 2 {
		return "cp: need source and destination"
	}
	src := ResolvePath(args[0], cwd)
	dst := ResolvePath(args[1], cwd)

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Sprintf("cp: %v", err)
	}
	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Sprintf("cp: %v", err)
	}
	return fmt.Sprintf("copied %s -> %s", args[0], filepath.Base(dst))
}

func cmdMv(args []string, cwd string) string {
	if len(args) < 2 {
		return "mv: need source and destination"
	}
	src := ResolvePath(args[0], cwd)
	dst := ResolvePath(args[1], cwd)

	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Sprintf("mv: %v", err)
	}
	return fmt.Sprintf("moved %s -> %s", args[0], filepath.Base(dst))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ResolvePath resolves a path relative to cwd.
func ResolvePath(path, cwd string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

// IsBinary checks if the first 512 bytes contain null bytes.
func IsBinary(data []byte) bool {
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	for _, b := range check {
		if b == 0 {
			return true
		}
	}
	return false
}