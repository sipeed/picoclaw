package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Severity levels for security findings.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// TrustLevel for skill sources.
type TrustLevel string

const (
	TrustBuiltin      TrustLevel = "builtin"
	TrustTrusted      TrustLevel = "trusted"
	TrustCommunity    TrustLevel = "community"
	TrustAgentCreated TrustLevel = "agent-created"
)

// Verdict is the overall scan result.
type Verdict string

const (
	VerdictSafe      Verdict = "safe"
	VerdictCaution   Verdict = "caution"
	VerdictDangerous Verdict = "dangerous"
)

// Finding represents a single security issue detected.
type Finding struct {
	PatternID   string
	Severity    Severity
	Category    string
	File        string
	Line        int
	Match       string
	Description string
}

// ScanResult is the output of scanning a skill.
type ScanResult struct {
	SkillName string
	Source    TrustLevel
	Verdict   Verdict
	Findings  []Finding
	ScannedAt time.Time
	Summary   string
}

// threatPattern is a pre-compiled regex threat pattern.
type threatPattern struct {
	id          string
	re          *regexp.Regexp
	severity    Severity
	category    string
	description string
}

// Structural limits.
const (
	maxFileCount    = 50
	maxTotalSizeKB  = 1024
	maxSingleFileKB = 256
)

// scannableExtensions are file types we check for threat patterns.
var scannableExtensions = map[string]bool{
	".md": true, ".txt": true, ".py": true, ".sh": true, ".bash": true,
	".js": true, ".ts": true, ".rb": true, ".yaml": true, ".yml": true,
	".json": true, ".toml": true, ".cfg": true, ".ini": true, ".conf": true,
	".html": true, ".css": true, ".xml": true, ".go": true,
}

// suspiciousBinaryExtensions that should never be in a skill.
var suspiciousBinaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".dat": true, ".com": true, ".msi": true, ".dmg": true, ".app": true,
	".deb": true, ".rpm": true,
}

// invisibleUnicodeChars that may indicate hidden content.
var invisibleUnicodeChars = []rune{
	'\u200B', // zero-width space
	'\u200C', // zero-width non-joiner
	'\u200D', // zero-width joiner
	'\u200E', // left-to-right mark
	'\u200F', // right-to-left mark
	'\u2060', // word joiner
	'\u2061', // function application
	'\u2062', // invisible times
	'\u2063', // invisible separator
	'\u2064', // invisible plus
	'\uFEFF', // zero-width no-break space (BOM)
	'\u00AD', // soft hyphen
	'\u034F', // combining grapheme joiner
	'\u061C', // arabic letter mark
	'\u180E', // mongolian vowel separator
	'\u202A', // left-to-right embedding
	'\u202B', // right-to-left embedding
}

// guardThreatPatterns holds all compiled patterns. Initialized once at package load.
var guardThreatPatterns []threatPattern

func init() {
	raw := []struct {
		id, pattern, category, description string
		severity                           Severity
	}{
		// --- Exfiltration ---
		{"exfil-curl-secret", `(?i)curl\s.*\$[A-Z_]+`, "exfiltration", "curl with environment variable (potential secret exfil)", SeverityCritical},
		{"exfil-wget-secret", `(?i)wget\s.*\$[A-Z_]+`, "exfiltration", "wget with environment variable (potential secret exfil)", SeverityCritical},
		{"exfil-env-dump", `(?i)(printenv|env\s*>|set\s*>|export\s+-p)\s*[|>]`, "exfiltration", "environment variable dump to file/pipe", SeverityHigh},
		{"exfil-dns-tunnel", `(?i)\$[A-Z_]+\.[a-z]+\.(com|net|org|io)`, "exfiltration", "DNS tunneling pattern (secret in subdomain)", SeverityCritical},
		{"exfil-base64-env", `(?i)(echo|printf)\s+\$[A-Z_]+\s*\|\s*base64`, "exfiltration", "base64 encoding of environment variable", SeverityHigh},
		{"exfil-markdown-img", `(?i)!\[.*\]\(https?://.*\$`, "exfiltration", "markdown image with variable (potential exfil via URL)", SeverityHigh},
		{"exfil-nc-data", `(?i)nc\s+.*\s+.*<\s*[/~]`, "exfiltration", "netcat sending file contents", SeverityCritical},
		{"exfil-ssh-key", `(?i)(cat|less|more|head|tail)\s+.*\.(ssh|aws|gnupg)/`, "exfiltration", "reading sensitive credential directories", SeverityCritical},

		// --- Prompt Injection ---
		{"inject-ignore-prev", `(?i)ignore\s+(all\s+)?previous\s+instructions`, "injection", "prompt injection: ignore previous instructions", SeverityCritical},
		{"inject-new-role", `(?i)you\s+are\s+now\s+(a|an)\s+`, "injection", "prompt injection: role reassignment", SeverityHigh},
		{"inject-system-override", `(?i)(system\s*prompt|system\s*message)\s*[:=]`, "injection", "prompt injection: system prompt override", SeverityCritical},
		{"inject-act-as", `(?i)act\s+as\s+(if|though)?\s*(you|a|an)\s+`, "injection", "prompt injection: act-as directive", SeverityHigh},
		{"inject-disregard", `(?i)disregard\s+(all\s+)?(previous|prior|above)`, "injection", "prompt injection: disregard instructions", SeverityCritical},
		{"inject-jailbreak", `(?i)(DAN|do\s+anything\s+now|jailbreak)`, "injection", "prompt injection: known jailbreak pattern", SeverityCritical},
		{"inject-restriction-bypass", `(?i)(bypass|circumvent|override)\s+(safety|restriction|filter|guard)`, "injection", "prompt injection: safety bypass attempt", SeverityHigh},
		{"inject-html-hidden", `<!--\s*(system|instruction|ignore|override)`, "injection", "HTML comment with hidden instructions", SeverityHigh},

		// --- Destructive Operations ---
		{"destruct-rm-rf-root", `rm\s+-[a-z]*r[a-z]*f[a-z]*\s+/\s*$`, "destructive", "rm -rf / (wipe filesystem root)", SeverityCritical},
		{"destruct-rm-rf-slash", `rm\s+-[a-z]*r[a-z]*f[a-z]*\s+/[a-z]`, "destructive", "rm -rf on system directory", SeverityHigh},
		{"destruct-mkfs", `(?i)mkfs\s`, "destructive", "filesystem format command", SeverityCritical},
		{"destruct-dd-zero", `dd\s+if=/dev/(zero|random)`, "destructive", "dd from zero/random device (disk wipe)", SeverityCritical},
		{"destruct-chmod-777-root", `chmod\s+-[Rr]\s+777\s+/`, "destructive", "recursive chmod 777 on root", SeverityHigh},
		{"destruct-truncate-boot", `>\s*/boot/`, "destructive", "truncating boot files", SeverityCritical},

		// --- Persistence ---
		{"persist-crontab", `(?i)crontab\s+-[el]`, "persistence", "crontab modification", SeverityHigh},
		{"persist-bashrc", `(?i)(>>|>)\s*~/?\.(bashrc|bash_profile|zshrc|profile)`, "persistence", "shell rc file modification", SeverityCritical},
		{"persist-ssh-keys", `(?i)(>>|>)\s*~/?\.ssh/authorized_keys`, "persistence", "SSH authorized_keys modification", SeverityCritical},
		{"persist-sudoers", `(?i)(>>|>)\s*/etc/sudoers`, "persistence", "sudoers file modification", SeverityCritical},
		{"persist-launchd", `(?i)(launchctl\s+load|LaunchAgents|LaunchDaemons)`, "persistence", "macOS launchd persistence", SeverityHigh},
		{"persist-systemd", `(?i)(systemctl\s+enable|\.service\s*$)`, "persistence", "systemd service persistence", SeverityHigh},

		// --- Reverse Shells ---
		{"revshell-nc", `(?i)nc\s+-[a-z]*l[a-z]*\s+-p?\s*\d+`, "reverse_shell", "netcat listener (potential reverse shell)", SeverityCritical},
		{"revshell-bash-tcp", `bash\s+-i\s+>&\s*/dev/tcp/`, "reverse_shell", "bash reverse shell via /dev/tcp", SeverityCritical},
		{"revshell-socat", `(?i)socat\s+.*exec`, "reverse_shell", "socat exec (potential reverse shell)", SeverityHigh},
		{"revshell-python", `(?i)python[23]?\s+-c\s+.*socket.*connect`, "reverse_shell", "python reverse shell", SeverityCritical},
		{"revshell-ngrok", `(?i)ngrok\s+(http|tcp)`, "reverse_shell", "ngrok tunnel (potential C2 channel)", SeverityMedium},

		// --- Obfuscation ---
		{"obfusc-base64-exec", `(?i)base64\s+(-d|--decode)\s*\|\s*(bash|sh|python|perl)`, "obfuscation", "base64 decode piped to interpreter", SeverityCritical},
		{"obfusc-eval", `(?i)\beval\s*\(`, "obfuscation", "eval() call (potential code injection)", SeverityMedium},
		{"obfusc-exec", `(?i)\bexec\s*\(`, "obfuscation", "exec() call (potential code injection)", SeverityMedium},
		{"obfusc-hex-decode", `(?i)(\\x[0-9a-f]{2}){4,}`, "obfuscation", "hex-encoded string (potential hidden payload)", SeverityMedium},
		{"obfusc-curl-pipe-sh", `(?i)curl\s+.*\|\s*(bash|sh)`, "obfuscation", "curl piped to shell (remote code execution)", SeverityCritical},

		// --- Hardcoded Secrets ---
		{"secret-openai-key", `sk-[a-zA-Z0-9]{20,}`, "hardcoded_secret", "potential OpenAI API key", SeverityHigh},
		{"secret-anthropic-key", `sk-ant-[a-zA-Z0-9]{20,}`, "hardcoded_secret", "potential Anthropic API key", SeverityHigh},
		{"secret-github-token", `ghp_[a-zA-Z0-9]{36,}`, "hardcoded_secret", "potential GitHub personal access token", SeverityHigh},
		{"secret-aws-key", `AKIA[A-Z0-9]{16}`, "hardcoded_secret", "potential AWS access key", SeverityHigh},
		{"secret-generic-apikey", `(?i)(api[_-]?key|api[_-]?secret|api[_-]?token)\s*[=:]\s*["'][a-zA-Z0-9]{16,}["']`, "hardcoded_secret", "potential hardcoded API key/secret", SeverityMedium},
	}

	guardThreatPatterns = make([]threatPattern, 0, len(raw))
	for _, r := range raw {
		re, err := regexp.Compile(r.pattern)
		if err != nil {
			// Programming error — panic at init time so it's caught immediately.
			panic(fmt.Sprintf("skills/guard: bad pattern %q: %v", r.id, err))
		}
		guardThreatPatterns = append(guardThreatPatterns, threatPattern{
			id:          r.id,
			re:          re,
			severity:    r.severity,
			category:    r.category,
			description: r.description,
		})
	}
}

// ScanSkill scans all files in a skill directory for security threats.
func ScanSkill(skillDir string, source TrustLevel) *ScanResult {
	result := &ScanResult{
		SkillName: filepath.Base(skillDir),
		Source:    source,
		Verdict:   VerdictSafe,
		ScannedAt: time.Now(),
	}

	// Structural checks.
	var fileCount int
	var totalSize int64
	_ = filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fileCount++
		info, err := d.Info()
		if err != nil {
			return nil
		}
		totalSize += info.Size()

		ext := strings.ToLower(filepath.Ext(path))

		// Check for suspicious binaries.
		if suspiciousBinaryExtensions[ext] {
			result.Findings = append(result.Findings, Finding{
				PatternID:   "struct-binary",
				Severity:    SeverityHigh,
				Category:    "structure",
				File:        guardRelPath(skillDir, path),
				Description: "suspicious binary file in skill",
			})
		}

		// Scan content of text files.
		if scannableExtensions[ext] {
			sizeKB := info.Size() / 1024
			if sizeKB > maxSingleFileKB {
				result.Findings = append(result.Findings, Finding{
					PatternID:   "struct-large-file",
					Severity:    SeverityMedium,
					Category:    "structure",
					File:        guardRelPath(skillDir, path),
					Description: fmt.Sprintf("file too large: %dKB > %dKB limit", sizeKB, maxSingleFileKB),
				})
			} else {
				findings := scanFile(path, guardRelPath(skillDir, path))
				result.Findings = append(result.Findings, findings...)
			}
		}
		return nil
	})

	if fileCount > maxFileCount {
		result.Findings = append(result.Findings, Finding{
			PatternID:   "struct-too-many-files",
			Severity:    SeverityMedium,
			Category:    "structure",
			Description: fmt.Sprintf("skill has %d files (limit: %d)", fileCount, maxFileCount),
		})
	}
	if totalSize/1024 > maxTotalSizeKB {
		result.Findings = append(result.Findings, Finding{
			PatternID:   "struct-total-size",
			Severity:    SeverityMedium,
			Category:    "structure",
			Description: fmt.Sprintf("total size %dKB exceeds %dKB limit", totalSize/1024, maxTotalSizeKB),
		})
	}

	result.Verdict = determineVerdict(result.Findings)
	result.Summary = buildSummary(result)
	return result
}

// scanFile checks a single file for threat patterns and invisible unicode.
func scanFile(path, rel string) []Finding {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	var findings []Finding

	// Check for invisible unicode.
	for lineNum, line := range lines {
		for _, r := range line {
			for _, inv := range invisibleUnicodeChars {
				if r == inv {
					findings = append(findings, Finding{
						PatternID:   "unicode-invisible",
						Severity:    SeverityHigh,
						Category:    "obfuscation",
						File:        rel,
						Line:        lineNum + 1,
						Match:       fmt.Sprintf("U+%04X", r),
						Description: "invisible unicode character (potential hidden content)",
					})
					break
				}
			}
		}
	}

	// Check threat patterns.
	for _, tp := range guardThreatPatterns {
		for lineNum, line := range lines {
			if tp.re.MatchString(line) {
				match := tp.re.FindString(line)
				if len(match) > 120 {
					match = match[:120] + "..."
				}
				findings = append(findings, Finding{
					PatternID:   tp.id,
					Severity:    tp.severity,
					Category:    tp.category,
					File:        rel,
					Line:        lineNum + 1,
					Match:       match,
					Description: tp.description,
				})
			}
		}
	}

	// Check for non-UTF8 content.
	if !utf8.Valid(data) {
		findings = append(findings, Finding{
			PatternID:   "encoding-invalid-utf8",
			Severity:    SeverityMedium,
			Category:    "obfuscation",
			File:        rel,
			Description: "file contains invalid UTF-8 (potential hidden content)",
		})
	}

	return findings
}

// determineVerdict picks the highest severity verdict from findings.
func determineVerdict(findings []Finding) Verdict {
	if len(findings) == 0 {
		return VerdictSafe
	}
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			return VerdictDangerous
		}
	}
	for _, f := range findings {
		if f.Severity == SeverityHigh {
			return VerdictCaution
		}
	}
	return VerdictCaution // medium findings still warrant caution
}

// installPolicy defines what to do for each (trust, verdict) pair.
// true = allow, false = block.
var installPolicy = map[TrustLevel]map[Verdict]bool{
	TrustBuiltin:      {VerdictSafe: true, VerdictCaution: true, VerdictDangerous: true},
	TrustTrusted:      {VerdictSafe: true, VerdictCaution: true, VerdictDangerous: false},
	TrustCommunity:    {VerdictSafe: true, VerdictCaution: false, VerdictDangerous: false},
	TrustAgentCreated: {VerdictSafe: true, VerdictCaution: true, VerdictDangerous: false},
}

// ShouldAllowInstall checks the scan result against the trust-based install policy.
func ShouldAllowInstall(result *ScanResult) (bool, string) {
	policy, ok := installPolicy[result.Source]
	if !ok {
		return false, fmt.Sprintf("unknown trust level: %s", result.Source)
	}
	allowed, ok := policy[result.Verdict]
	if !ok {
		return false, fmt.Sprintf("unknown verdict: %s", result.Verdict)
	}
	if !allowed {
		return false, fmt.Sprintf("policy blocks %s skills with %s verdict", result.Source, result.Verdict)
	}
	return true, ""
}

// FormatScanReport returns a human-readable multi-line report.
func FormatScanReport(result *ScanResult) string {
	if len(result.Findings) == 0 {
		return fmt.Sprintf("Skill %q: SAFE (no findings)", result.SkillName)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Skill %q: %s (%d finding(s))\n", result.SkillName, strings.ToUpper(string(result.Verdict)), len(result.Findings))
	for _, f := range result.Findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&sb, "  [%s] %s — %s", f.Severity, loc, f.Description)
		if f.Match != "" {
			fmt.Fprintf(&sb, " (%s)", f.Match)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildSummary creates a short summary string.
func buildSummary(result *ScanResult) string {
	if len(result.Findings) == 0 {
		return "no security issues found"
	}
	categories := map[string]int{}
	for _, f := range result.Findings {
		categories[f.Category]++
	}
	var parts []string
	for cat, count := range categories {
		parts = append(parts, fmt.Sprintf("%s(%d)", cat, count))
	}
	return fmt.Sprintf("%d finding(s): %s", len(result.Findings), strings.Join(parts, ", "))
}

// guardRelPath returns a relative path from base to target, or target if that fails.
func guardRelPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
