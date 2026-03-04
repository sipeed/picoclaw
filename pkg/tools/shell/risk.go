package shell

import "fmt"

// RiskLevel represents the potential danger of a shell command.
type RiskLevel int

const (
	RiskLow      RiskLevel = iota // Read-only, informational
	RiskMedium                    // File modification, network read
	RiskHigh                      // Destructive, system-modifying
	RiskCritical                  // Privilege escalation, always dangerous
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ParseRiskLevel converts a string to a RiskLevel.
// Returns an error if the string is unrecognized.
func ParseRiskLevel(s string) (RiskLevel, error) {
	switch s {
	case "low":
		return RiskLow, nil
	case "medium":
		return RiskMedium, nil
	case "high":
		return RiskHigh, nil
	case "critical":
		return RiskCritical, nil
	default:
		return RiskMedium, fmt.Errorf("unknown risk level %q, must be one of: low, medium, high, critical", s)
	}
}

// commandRiskTable maps base command names to their default risk level.
// Commands not in this table default to RiskMedium.
var commandRiskTable = map[string]RiskLevel{
	// Low — read-only, informational
	"ls":        RiskLow,
	"cat":       RiskLow,
	"head":      RiskLow,
	"tail":      RiskLow,
	"grep":      RiskLow,
	"egrep":     RiskLow,
	"fgrep":     RiskLow,
	"find":      RiskLow,
	"wc":        RiskLow,
	"echo":      RiskLow,
	"printf":    RiskLow,
	"pwd":       RiskLow,
	"whoami":    RiskLow,
	"id":        RiskLow,
	"date":      RiskLow,
	"uname":     RiskLow,
	"hostname":  RiskLow,
	"env":       RiskLow,
	"printenv":  RiskLow,
	"which":     RiskLow,
	"type":      RiskLow,
	"file":      RiskLow,
	"stat":      RiskLow,
	"readlink":  RiskLow,
	"realpath":  RiskLow,
	"basename":  RiskLow,
	"dirname":   RiskLow,
	"sort":      RiskLow,
	"uniq":      RiskLow,
	"cut":       RiskLow,
	"tr":        RiskLow,
	"awk":       RiskLow,
	"sed":       RiskLow,
	"diff":      RiskLow,
	"md5sum":    RiskLow,
	"sha256sum": RiskLow,
	"sha1sum":   RiskLow,
	"xxd":       RiskLow,
	"od":        RiskLow,
	"hexdump":   RiskLow,
	"strings":   RiskLow,
	"tee":       RiskLow,
	"xargs":     RiskLow,
	"true":      RiskLow,
	"false":     RiskLow,
	"test":      RiskLow,
	"[":         RiskLow,
	"seq":       RiskLow,
	"yes":       RiskLow,
	"sleep":     RiskLow,
	"du":        RiskLow,
	"df":        RiskLow,
	"free":      RiskLow,
	"top":       RiskLow,
	"ps":        RiskLow,
	"uptime":    RiskLow,
	"lsof":      RiskLow,
	"tree":      RiskLow,
	"less":      RiskLow,
	"more":      RiskLow,
	"jq":        RiskLow,
	"yq":        RiskLow,
	"column":    RiskLow,
	"fold":      RiskLow,
	"fmt":       RiskLow,
	"rev":       RiskLow,
	"tac":       RiskLow,
	"nl":        RiskLow,
	"comm":      RiskLow,
	"join":      RiskLow,
	"paste":     RiskLow,
	"expand":    RiskLow,
	"unexpand":  RiskLow,

	// Medium — file modification, network reads, build tools
	"cp":      RiskMedium,
	"mv":      RiskMedium,
	"mkdir":   RiskMedium,
	"touch":   RiskMedium,
	"ln":      RiskMedium,
	"tar":     RiskMedium,
	"zip":     RiskMedium,
	"unzip":   RiskMedium,
	"gzip":    RiskMedium,
	"gunzip":  RiskMedium,
	"bzip2":   RiskMedium,
	"xz":      RiskMedium,
	"curl":    RiskMedium,
	"wget":    RiskMedium,
	"git":     RiskMedium,
	"make":    RiskMedium,
	"go":      RiskMedium,
	"python":  RiskMedium,
	"python3": RiskMedium,
	"node":    RiskMedium,
	"npm":     RiskMedium,
	"npx":     RiskMedium,
	"yarn":    RiskMedium,
	"pnpm":    RiskMedium,
	"pip":     RiskMedium,
	"pip3":    RiskMedium,
	"cargo":   RiskMedium,
	"rustc":   RiskMedium,
	"gcc":     RiskMedium,
	"g++":     RiskMedium,
	"clang":   RiskMedium,
	"javac":   RiskMedium,
	"java":    RiskMedium,
	"ruby":    RiskMedium,
	"perl":    RiskMedium,
	"php":     RiskMedium,
	"patch":   RiskMedium,

	// High — destructive, system-modifying
	"rm":        RiskHigh,
	"rmdir":     RiskHigh,
	"chmod":     RiskHigh,
	"chown":     RiskHigh,
	"chgrp":     RiskHigh,
	"kill":      RiskHigh,
	"pkill":     RiskHigh,
	"killall":   RiskHigh,
	"ssh":       RiskHigh,
	"scp":       RiskHigh,
	"rsync":     RiskHigh,
	"docker":    RiskHigh,
	"kubectl":   RiskHigh,
	"systemctl": RiskHigh,
	"service":   RiskHigh,

	// Critical — privilege escalation, always dangerous
	"sudo":     RiskCritical,
	"su":       RiskCritical,
	"dd":       RiskCritical,
	"mkfs":     RiskCritical,
	"fdisk":    RiskCritical,
	"parted":   RiskCritical,
	"mount":    RiskCritical,
	"umount":   RiskCritical,
	"shutdown": RiskCritical,
	"reboot":   RiskCritical,
	"poweroff": RiskCritical,
	"halt":     RiskCritical,
	"init":     RiskCritical,
	"insmod":   RiskCritical,
	"rmmod":    RiskCritical,
	"modprobe": RiskCritical,
	"iptables": RiskCritical,
	"nft":      RiskCritical,
	"eval":     RiskCritical,
	"exec":     RiskCritical,
	"source":   RiskCritical,
	".":        RiskCritical,
	"format":   RiskCritical,
	"diskpart": RiskCritical,
}

// ArgModifier describes a condition that elevates a command's risk level.
// All tokens in Args must be present in the command (order-independent, after
// flag normalization).
type ArgModifier struct {
	Args  []string
	Level RiskLevel
}

// argumentModifiers maps command names to their argument-aware risk adjustments.
// Checked in order; first match wins.
//
// Patterns use individual flags (e.g., "-r", "-f") rather than combined forms
// ("-rf") because normalizeFlags splits combined flags before matching. This
// means "rm -rf", "rm -fr", "rm -r -f", and "rm -f -r" all match correctly.
var argumentModifiers = map[string][]ArgModifier{
	"git": {
		{Args: []string{"push", "--force"}, Level: RiskCritical},
		{Args: []string{"push", "-f"}, Level: RiskCritical},
		{Args: []string{"push"}, Level: RiskHigh},
		{Args: []string{"reset", "--hard"}, Level: RiskHigh},
		{Args: []string{"clean", "-f", "-d"}, Level: RiskHigh},
		{Args: []string{"clean", "-f"}, Level: RiskHigh},
	},
	"curl": {
		{Args: []string{"-X", "POST"}, Level: RiskHigh},
		{Args: []string{"-X", "PUT"}, Level: RiskHigh},
		{Args: []string{"-X", "DELETE"}, Level: RiskHigh},
		{Args: []string{"--request", "POST"}, Level: RiskHigh},
		{Args: []string{"--request", "PUT"}, Level: RiskHigh},
		{Args: []string{"--request", "DELETE"}, Level: RiskHigh},
		{Args: []string{"--data"}, Level: RiskHigh},
		{Args: []string{"-d"}, Level: RiskHigh},
		{Args: []string{"--upload-file"}, Level: RiskHigh},
		{Args: []string{"-T"}, Level: RiskHigh},
	},
	"wget": {
		{Args: []string{"--post-data"}, Level: RiskHigh},
		{Args: []string{"--post-file"}, Level: RiskHigh},
	},
	"npm": {
		{Args: []string{"install", "-g"}, Level: RiskHigh},
		{Args: []string{"install", "--global"}, Level: RiskHigh},
		{Args: []string{"publish"}, Level: RiskHigh},
	},
	"pip": {
		{Args: []string{"install", "--user"}, Level: RiskHigh},
	},
	"pip3": {
		{Args: []string{"install", "--user"}, Level: RiskHigh},
	},
	"docker": {
		{Args: []string{"run"}, Level: RiskHigh},
		{Args: []string{"exec"}, Level: RiskHigh},
		{Args: []string{"rm"}, Level: RiskHigh},
		{Args: []string{"rmi"}, Level: RiskHigh},
	},
	"apt": {
		{Args: []string{"install"}, Level: RiskHigh},
		{Args: []string{"remove"}, Level: RiskHigh},
		{Args: []string{"purge"}, Level: RiskCritical},
	},
	"apt-get": {
		{Args: []string{"install"}, Level: RiskHigh},
		{Args: []string{"remove"}, Level: RiskHigh},
		{Args: []string{"purge"}, Level: RiskCritical},
	},
	"yum": {
		{Args: []string{"install"}, Level: RiskHigh},
		{Args: []string{"remove"}, Level: RiskHigh},
	},
	"dnf": {
		{Args: []string{"install"}, Level: RiskHigh},
		{Args: []string{"remove"}, Level: RiskHigh},
	},
	"rm": {
		{Args: []string{"-r", "-f"}, Level: RiskCritical},
	},
	"kill": {
		{Args: []string{"-9"}, Level: RiskCritical},
		{Args: []string{"-KILL"}, Level: RiskCritical},
		{Args: []string{"-SIGKILL"}, Level: RiskCritical},
	},
}

// ClassifyCommand determines the risk level of a resolved command.
// args[0] is the command name (basename), args[1:] are the arguments.
// overrides allows per-command level overrides from config.
// extraModifiers are checked after built-in argumentModifiers.
// The highest matching level across all sources wins.
func ClassifyCommand(args []string, overrides map[string]string, extraModifiers ...map[string][]ArgModifier) RiskLevel {
	if len(args) == 0 {
		return RiskMedium
	}

	cmdName := baseCommand(args[0])

	if overrides != nil {
		if levelStr, ok := overrides[cmdName]; ok {
			level, err := ParseRiskLevel(levelStr)
			if err == nil {
				return level
			}
			// Invalid risk level in override: fall through to default classification.
			// The parse error is descriptive, but we can't log from here without
			// injecting a logger. Config-time validation catches user errors.
		}
	}

	level, known := commandRiskTable[cmdName]
	if !known {
		level = RiskMedium
	}

	// Normalize args: expand combined short flags like -rf → -r, -f
	// so that modifiers match regardless of how flags were grouped or ordered.
	normalizedArgs := normalizeFlags(args[1:])

	// Check built-in modifiers, then user-supplied. Keep the highest match.
	if elevated, ok := applyModifiers(normalizedArgs, cmdName, level, argumentModifiers); ok {
		level = elevated
	}
	for _, extra := range extraModifiers {
		if extra == nil {
			continue
		}
		if elevated, ok := applyModifiers(normalizedArgs, cmdName, level, extra); ok {
			level = elevated
		}
	}

	return level
}

// applyModifiers checks whether any modifier for cmdName matches the
// normalised args and would elevate the risk. Returns (newLevel, true)
// on first match, or (0, false) if nothing matched.
func applyModifiers(
	normalizedArgs []string,
	cmdName string,
	baseLevel RiskLevel,
	mods map[string][]ArgModifier,
) (RiskLevel, bool) {
	if entries, ok := mods[cmdName]; ok {
		for _, mod := range entries {
			if matchArgs(normalizedArgs, mod.Args) && mod.Level > baseLevel {
				return mod.Level, true
			}
		}
	}
	return 0, false
}

// IsAllowed returns true if the given risk level is at or below the threshold.
func IsAllowed(level, threshold RiskLevel) bool {
	return level <= threshold
}

// BlockedCommandError formats a structured error message for the LLM.
func BlockedCommandError(args []string, level, threshold RiskLevel, reason string) string {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
		if len(args) > 1 {
			end := len(args)
			if end > 5 {
				end = 5
			}
			for _, a := range args[1:end] {
				cmd += " " + a
			}
			if len(args) > 5 {
				cmd += " ..."
			}
		}
	}

	return fmt.Sprintf(
		"Command blocked by risk classifier: command=%q risk_level=%s threshold=%s reason=%s",
		cmd, level, threshold, reason,
	)
}

// baseCommand extracts the basename from a command path.
func baseCommand(cmd string) string {
	for i := len(cmd) - 1; i >= 0; i-- {
		if cmd[i] == '/' {
			return cmd[i+1:]
		}
	}
	return cmd
}

// normalizeFlags expands combined short flags (e.g., "-rf" → "-r", "-f")
// so that modifier matching works regardless of how flags are grouped.
// Long flags (--flag) and non-flag arguments are passed through unchanged.
func normalizeFlags(args []string) []string {
	result := make([]string, 0, len(args)*2)
	for _, a := range args {
		if len(a) > 2 && a[0] == '-' && a[1] != '-' {
			for _, ch := range a[1:] {
				result = append(result, "-"+string(ch))
			}
		} else {
			result = append(result, a)
		}
	}
	return result
}

// matchArgs checks if ALL pattern tokens are present in args (order-independent).
// "git push -x -f" matches pattern ["push", "-f"] because both tokens exist.
// "git -f push" also matches ["push", "-f"]. Order does not matter.
func matchArgs(args, pattern []string) bool {
	if len(pattern) == 0 {
		return true
	}
	argSet := make(map[string]int, len(args))
	for _, a := range args {
		argSet[a]++
	}
	for _, p := range pattern {
		if argSet[p] <= 0 {
			return false
		}
		argSet[p]--
	}
	return true
}
