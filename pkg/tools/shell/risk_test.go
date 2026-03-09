package shell

import (
	"runtime"
	"testing"
)

// Windows-specific command and arg modifier tests are in risk_windows_test.go
// (guarded by //go:build windows).

func TestClassifyCommand_BaseTable(t *testing.T) {
	tests := []struct {
		args  []string
		level RiskLevel
	}{
		{[]string{"ls", "-la"}, RiskLow},
		{[]string{"cat", "file.txt"}, RiskLow},
		{[]string{"grep", "-r", "pattern", "."}, RiskLow},
		{[]string{"find", ".", "-name", "*.go"}, RiskLow},
		{[]string{"wc", "-l"}, RiskLow},
		{[]string{"echo", "hello"}, RiskLow},
		{[]string{"jq", ".field", "data.json"}, RiskLow},
		{[]string{"ping", "localhost"}, RiskLow},
		{[]string{"dig", "example.com"}, RiskLow},
		{[]string{"ss", "-tulpn"}, RiskLow},
		{[]string{"bc"}, RiskLow},
		{[]string{"nproc"}, RiskLow},

		{[]string{"cp", "a", "b"}, RiskMedium},
		{[]string{"mv", "a", "b"}, RiskMedium},
		{[]string{"python3", "-c", "print(1)"}, RiskMedium},
		{[]string{"git", "status"}, RiskMedium},
		{[]string{"curl", "https://example.com"}, RiskMedium},
		{[]string{"openssl", "version"}, RiskMedium},
		{[]string{"crontab", "-l"}, RiskMedium},
		{[]string{"vim", "file.txt"}, RiskMedium},

		{[]string{"rm", "file.txt"}, RiskHigh},
		{[]string{"chmod", "755", "script.sh"}, RiskHigh},
		{[]string{"docker", "ps"}, RiskHigh},
		{[]string{"ssh", "user@host"}, RiskHigh},
		{[]string{"nc", "-l", "4444"}, RiskHigh},
		{[]string{"socat", "TCP:host:80", "STDOUT"}, RiskHigh},
		{[]string{"useradd", "testuser"}, RiskHigh},
		{[]string{"passwd", "testuser"}, RiskHigh},
		{[]string{"shred", "file"}, RiskHigh},

		{[]string{"sudo", "ls"}, RiskCritical},
		{[]string{"dd", "if=/dev/zero", "of=/dev/sda"}, RiskCritical},
		{[]string{"shutdown", "-h", "now"}, RiskCritical},
		{[]string{"eval", "echo hi"}, RiskCritical},
		{[]string{"chattr", "+i", "file"}, RiskCritical},
		{[]string{"visudo"}, RiskCritical},
		{[]string{"ip6tables", "-L"}, RiskCritical},
	}

	for _, tt := range tests {
		got := ClassifyCommand(tt.args, nil)
		if got != tt.level {
			t.Errorf("ClassifyCommand(%v) = %s, want %s", tt.args, got, tt.level)
		}
	}
}

func TestClassifyCommand_ArgumentModifiers(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		level RiskLevel
	}{
		{"git push", []string{"git", "push", "origin", "main"}, RiskHigh},
		{"git push --force", []string{"git", "push", "--force", "origin"}, RiskCritical},
		{"git push -f", []string{"git", "push", "-f"}, RiskCritical},
		{"git -f push (reordered)", []string{"git", "-f", "push"}, RiskCritical},
		{"git push -x -f (extra flags)", []string{"git", "push", "-x", "-f"}, RiskCritical},
		{"git reset --hard", []string{"git", "reset", "--hard", "HEAD~1"}, RiskHigh},
		{"git clean -f", []string{"git", "clean", "-f"}, RiskHigh},
		{"git clean -fd", []string{"git", "clean", "-fd"}, RiskHigh},
		{"git clean -d -f (reordered)", []string{"git", "clean", "-d", "-f"}, RiskHigh},

		{"curl GET (default)", []string{"curl", "https://example.com"}, RiskMedium},
		{"curl POST", []string{"curl", "-X", "POST", "https://example.com"}, RiskHigh},
		{"curl post lowercase", []string{"curl", "-X", "post", "https://example.com"}, RiskHigh},
		{"curl -d data", []string{"curl", "-d", "data", "https://example.com"}, RiskHigh},
		{"curl --data data", []string{"curl", "--data", "data", "url"}, RiskHigh},
		{"curl --data=payload", []string{"curl", "--data=payload", "url"}, RiskHigh},
		{"curl -X DELETE", []string{"curl", "-X", "DELETE", "url"}, RiskHigh},
		{"curl -XDELETE", []string{"curl", "-XDELETE", "url"}, RiskHigh},
		{"curl -X=DELETE", []string{"curl", "-X=DELETE", "url"}, RiskHigh},
		{"curl --request POST", []string{"curl", "--request", "POST", "url"}, RiskHigh},
		{"curl --request=post lowercase", []string{"curl", "--request=post", "url"}, RiskHigh},
		{"curl --request=POST", []string{"curl", "--request=POST", "url"}, RiskHigh},
		{"curl -dDATA", []string{"curl", "-dDATA", "https://example.com"}, RiskHigh},
		{"curl -d=DATA", []string{"curl", "-d=DATA", "https://example.com"}, RiskHigh},

		{"rm file (no flags)", []string{"rm", "file.txt"}, RiskHigh},
		{"rm -rf", []string{"rm", "-rf", "/"}, RiskCritical},
		{"rm -fr", []string{"rm", "-fr", "/"}, RiskCritical},
		{"rm -r -f (separate)", []string{"rm", "-r", "-f", "dir"}, RiskCritical},
		{"rm -f -r (reordered)", []string{"rm", "-f", "-r", "dir"}, RiskCritical},
		{"rm -f -x -r (extra flags between)", []string{"rm", "-f", "-x", "-r", "dir"}, RiskCritical},

		{"kill (no signal)", []string{"kill", "1234"}, RiskHigh},
		{"kill -9", []string{"kill", "-9", "1234"}, RiskCritical},

		{"npm install (local)", []string{"npm", "install", "lodash"}, RiskMedium},
		{"npm install -g", []string{"npm", "install", "-g", "lodash"}, RiskHigh},
		{"npm publish", []string{"npm", "publish"}, RiskHigh},

		{"docker ps (no modifier)", []string{"docker", "ps"}, RiskHigh},
		{"docker run", []string{"docker", "run", "ubuntu"}, RiskHigh},
		{"docker exec", []string{"docker", "exec", "-it", "cnt", "bash"}, RiskHigh},

		{"apt install", []string{"apt", "install", "vim"}, RiskHigh},
		{"apt purge", []string{"apt", "purge", "vim"}, RiskCritical},

		{"find -delete", []string{"find", ".", "-name", "*.tmp", "-delete"}, RiskHigh},
		{"find -exec", []string{"find", ".", "-exec", "rm", "{}", ";"}, RiskHigh},
		{"sed -i", []string{"sed", "-i", "s/old/new/g", "file"}, RiskMedium},
		{"rsync --delete", []string{"rsync", "-a", "--delete", "src/", "dst/"}, RiskCritical},
		{"crontab -r", []string{"crontab", "-r"}, RiskHigh},
		{"ssh -R (reverse tunnel)", []string{"ssh", "-R", "8080:localhost:80", "host"}, RiskHigh},
		{"ssh -L (local tunnel)", []string{"ssh", "-L", "8080:remotehost:80", "host"}, RiskHigh},
		{"tar --to-command", []string{"tar", "xf", "archive.tar", "--to-command", "sh"}, RiskCritical},
		{"docker run --privileged", []string{"docker", "run", "--privileged", "ubuntu"}, RiskCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.args, nil)
			if got != tt.level {
				t.Errorf("ClassifyCommand(%v) = %s, want %s", tt.args, got, tt.level)
			}
		})
	}
}

func TestClassifyCommand_Overrides(t *testing.T) {
	overrides := map[string]string{
		"rm":   "medium",
		"curl": "critical",
	}

	// Override sets the BASE level, but modifiers still elevate.
	// rm is overridden to medium, but rm -rf triggers the built-in
	// modifier that elevates to critical.
	got := ClassifyCommand([]string{"rm", "-rf", "/"}, overrides)
	if got != RiskCritical {
		t.Errorf("override rm to medium + rm -rf modifier should be critical: got %s", got)
	}

	// Plain rm (no -rf) stays at the overridden level.
	got = ClassifyCommand([]string{"rm", "file.txt"}, overrides)
	if got != RiskMedium {
		t.Errorf("override rm to medium (no modifier match): got %s", got)
	}

	// Override elevates curl to critical unconditionally.
	got = ClassifyCommand([]string{"curl", "https://example.com"}, overrides)
	if got != RiskCritical {
		t.Errorf("override curl to critical: got %s", got)
	}

	// No override for ls — uses table as before.
	got = ClassifyCommand([]string{"ls"}, overrides)
	if got != RiskLow {
		t.Errorf("ls (no override) should be low: got %s", got)
	}
}

func TestClassifyCommand_OverrideLowers_ModifierStillElevates(t *testing.T) {
	// Scenario: user sets rm to low ("I want rm allowed"), but rm -rf
	// still hits the built-in modifier → critical.
	overrides := map[string]string{"rm": "low"}

	got := ClassifyCommand([]string{"rm", "file.txt"}, overrides)
	if got != RiskLow {
		t.Errorf("plain rm with override=low should be low: got %s", got)
	}

	got = ClassifyCommand([]string{"rm", "-rf", "/"}, overrides)
	if got != RiskCritical {
		t.Errorf("rm -rf should still be critical despite override=low: got %s", got)
	}
}

func TestClassifyCommand_UnknownCommand(t *testing.T) {
	got := ClassifyCommand([]string{"some_unknown_tool", "--flag"}, nil)
	if got != RiskMedium {
		t.Errorf("unknown command should default to medium, got %s", got)
	}
}

func TestClassifyCommand_FullPath(t *testing.T) {
	got := ClassifyCommand([]string{"/usr/bin/rm", "-rf", "/"}, nil)
	if got != RiskCritical {
		t.Errorf("/usr/bin/rm -rf should be critical, got %s", got)
	}

	got = ClassifyCommand([]string{"/bin/ls", "-la"}, nil)
	if got != RiskLow {
		t.Errorf("/bin/ls should be low, got %s", got)
	}
}

func TestClassifyCommand_DeepPath(t *testing.T) {
	// Forward-slash paths at various depths.
	got := ClassifyCommand([]string{"/usr/sbin/shutdown", "-h"}, nil)
	if got != RiskCritical {
		t.Errorf("/usr/sbin/shutdown should be critical, got %s", got)
	}

	got = ClassifyCommand([]string{"/usr/local/bin/sudo", "ls"}, nil)
	if got != RiskCritical {
		t.Errorf("/usr/local/bin/sudo should be critical, got %s", got)
	}

	// Bare command still works after the filepath.Base change.
	got = ClassifyCommand([]string{"dd", "if=/dev/zero"}, nil)
	if got != RiskCritical {
		t.Errorf("bare dd should be critical, got %s", got)
	}
}

func TestBaseCommand_StripsExeExtensions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("extension stripping only applies on Windows")
	}
	tests := []struct {
		input string
		want  string
	}{
		{"cmd.exe", "cmd"},
		{"GIT.EXE", "git"},
		{"POWERSHELL.EXE", "powershell"},
		{"script.bat", "script"},
		{"helper.cmd", "helper"},
		{"run.COM", "run"},
		{"ls", "ls"},
		{"my.tool", "my.tool"},
		{"", "."},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := baseCommand(tt.input)
			if got != tt.want {
				t.Errorf("baseCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBaseCommand_PreservesOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test verifies non-Windows behavior")
	}
	tests := []struct {
		input string
		want  string
	}{
		{"cmd.exe", "cmd.exe"},  // NOT stripped
		{"GIT.EXE", "GIT.EXE"},  // NOT lowercased
		{"LS", "LS"},            // case preserved
		{"ls", "ls"},            // unchanged
		{"/usr/bin/git", "git"}, // filepath.Base still works
		{"", "."},               // filepath.Base edge case
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := baseCommand(tt.input)
			if got != tt.want {
				t.Errorf("baseCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClassifyCommand_WindowsExePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("extension stripping only applies on Windows")
	}
	tests := []struct {
		name string
		args []string
		want RiskLevel
	}{
		{"cmd.exe bare", []string{"cmd.exe"}, RiskCritical},
		{"CMD.EXE upper", []string{"CMD.EXE"}, RiskCritical},
		{"git.exe status", []string{"git.exe", "status"}, RiskMedium},
		{"git.exe push", []string{"git.exe", "push"}, RiskHigh},
		{"rm.exe -rf", []string{"rm.exe", "-rf", "/"}, RiskCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.args, nil)
			if got != tt.want {
				t.Errorf("ClassifyCommand(%v) = %s, want %s", tt.args, got, tt.want)
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		level     RiskLevel
		threshold RiskLevel
		allowed   bool
	}{
		{RiskLow, RiskMedium, true},
		{RiskMedium, RiskMedium, true},
		{RiskHigh, RiskMedium, false},
		{RiskCritical, RiskMedium, false},
		{RiskCritical, RiskCritical, true},
		{RiskLow, RiskLow, true},
		{RiskMedium, RiskLow, false},
	}

	for _, tt := range tests {
		got := IsAllowed(tt.level, tt.threshold)
		if got != tt.allowed {
			t.Errorf("IsAllowed(%s, %s) = %v, want %v", tt.level, tt.threshold, got, tt.allowed)
		}
	}
}

func TestParseRiskLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    RiskLevel
		wantErr bool
	}{
		{"low", RiskLow, false},
		{"medium", RiskMedium, false},
		{"high", RiskHigh, false},
		{"critical", RiskCritical, false},
		{"bogus", RiskMedium, true},
		{"", RiskMedium, true},
		{"invalid", RiskMedium, true},
	}

	for _, tt := range tests {
		got, err := ParseRiskLevel(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseRiskLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseRiskLevel(%q) = %s, want %s", tt.input, got, tt.want)
		}
		// For error cases, verify the default is RiskMedium
		if tt.wantErr && got != RiskMedium {
			t.Errorf("ParseRiskLevel(%q) = %s on error, should default to RiskMedium", tt.input, got)
		}
	}
}

func TestNormalizeFlags(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{[]string{"-rf"}, []string{"-r", "-f"}},
		{[]string{"-fr"}, []string{"-f", "-r"}},
		{[]string{"-XPOST"}, []string{"-XPOST"}},
		{[]string{"-X=POST"}, []string{"-X=POST"}},
		{[]string{"-dDATA"}, []string{"-dDATA"}},
		{[]string{"-d=DATA"}, []string{"-d=DATA"}},
		{[]string{"--request=POST"}, []string{"--request", "POST"}},
		{[]string{"--data=body"}, []string{"--data", "body"}},
		{[]string{"-r", "-f"}, []string{"-r", "-f"}},
		{[]string{"--force"}, []string{"--force"}},
		{[]string{"-f"}, []string{"-f"}},
		{[]string{"-9"}, []string{"-9"}},
		{[]string{"push"}, []string{"push"}},
		{[]string{"-rf", "dir", "--verbose"}, []string{"-r", "-f", "dir", "--verbose"}},
	}

	for _, tt := range tests {
		got := normalizeFlags("git", tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("normalizeFlags(%v) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("normalizeFlags(%v) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestNormalizeFlags_CurlAttachedShortValues(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{[]string{"-XPOST"}, []string{"-X", "POST"}},
		{[]string{"-Xpost"}, []string{"-X", "POST"}},
		{[]string{"-X=POST"}, []string{"-X", "POST"}},
		{[]string{"-dDATA"}, []string{"-d", "DATA"}},
		{[]string{"-d=DATA"}, []string{"-d", "DATA"}},
		{[]string{"-Tfile.txt"}, []string{"-T", "file.txt"}},
		{[]string{"--request=post"}, []string{"--request", "POST"}},
		{[]string{"-X", "post"}, []string{"-X", "POST"}},
	}

	for _, tt := range tests {
		got := normalizeFlags("curl", tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("normalizeFlags(curl, %v) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("normalizeFlags(curl, %v) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestClassifyCommandWithProfiles_CustomAttachedValue(t *testing.T) {
	profiles := map[string]FlagProfile{
		"http": {
			ShortAttachedValue: map[string]FlagValueTransform{
				"-m": FlagValueUpper,
			},
			SeparateValueFlags: map[string]FlagValueTransform{
				"-m": FlagValueUpper,
			},
		},
	}
	modifiers := map[string][]ArgModifier{
		"http": {
			{Args: []string{"-m", "POST"}, Level: RiskHigh},
		},
	}

	got := ClassifyCommandWithProfiles([]string{"http", "-mpost", "https://example.com"}, nil, profiles, modifiers)
	if got != RiskHigh {
		t.Fatalf("ClassifyCommandWithProfiles(custom attached value) = %s, want %s", got, RiskHigh)
	}
}

func TestMatchArgs_OrderIndependent(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		pattern []string
		match   bool
	}{
		{"exact match", []string{"push", "-f"}, []string{"push", "-f"}, true},
		{"reversed order", []string{"-f", "push"}, []string{"push", "-f"}, true},
		{"extra flags between", []string{"push", "-x", "-f"}, []string{"push", "-f"}, true},
		{"missing token", []string{"push", "-x"}, []string{"push", "-f"}, false},
		{"empty pattern", []string{"push"}, []string{}, true},
		{"empty args", []string{}, []string{"push"}, false},
		{"superset ok", []string{"push", "-f", "--verbose", "origin"}, []string{"push", "-f"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchArgs(tt.args, tt.pattern)
			if got != tt.match {
				t.Errorf("matchArgs(%v, %v) = %v, want %v", tt.args, tt.pattern, got, tt.match)
			}
		})
	}
}

func TestClassifyCommand_ExtraArgModifiers(t *testing.T) {
	extra := map[string][]ArgModifier{
		// Custom: "make deploy" should be critical
		"make": {
			{Args: []string{"deploy"}, Level: RiskCritical},
		},
		// Custom: "git push --mirror" should be critical (not in built-ins)
		"git": {
			{Args: []string{"push", "--mirror"}, Level: RiskCritical},
		},
	}

	tests := []struct {
		name  string
		args  []string
		level RiskLevel
	}{
		{"make deploy (extra)", []string{"make", "deploy"}, RiskCritical},
		{"make build (no extra)", []string{"make", "build"}, RiskMedium},
		{"git push --mirror (extra)", []string{"git", "push", "--mirror"}, RiskCritical},
		// Built-in still works: git push -f is critical even without extra
		{"git push -f (built-in)", []string{"git", "push", "-f"}, RiskCritical},
		// Extra doesn't override a higher built-in: git push --force is already critical
		{"git push --force (built-in wins)", []string{"git", "push", "--force"}, RiskCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.args, nil, extra)
			if got != tt.level {
				t.Errorf("ClassifyCommand(%v, nil, extra) = %s, want %s", tt.args, got, tt.level)
			}
		})
	}
}

func TestClassifyCommand_ExtraArgModifiers_NoOverrideBuiltIn(t *testing.T) {
	// Extra modifier tries to set "rm -rf" to medium, but built-in already
	// elevates to critical. Since we take the max across all matching
	// modifiers, the built-in critical wins.
	extra := map[string][]ArgModifier{
		"rm": {
			{Args: []string{"-r", "-f"}, Level: RiskMedium},
		},
	}

	got := ClassifyCommand([]string{"rm", "-rf", "/"}, nil, extra)
	if got != RiskCritical {
		t.Errorf("built-in should win over extra for rm -rf: got %s", got)
	}
}

func TestClassifyCommand_ShellWrappers(t *testing.T) {
	// Shell wrappers must be critical to prevent classifier bypass.
	// cmd and cmd.exe are tested in risk_windows_test.go.
	shells := []string{
		"sh",
		"bash",
		"zsh",
		"dash",
		"fish",
		"ksh",
		"csh",
		"tcsh",
		"pwsh",
	}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			got := ClassifyCommand([]string{sh, "-c", "echo hi"}, nil)
			if got != RiskCritical {
				t.Errorf("%s should be critical, got %s", sh, got)
			}
		})
	}
}

func TestClassifyCommand_ShellWrapperFullPath(t *testing.T) {
	// /bin/sh, /usr/bin/bash etc. should also be caught via baseCommand.
	got := ClassifyCommand([]string{"/bin/sh", "-c", "rm -rf /"}, nil)
	if got != RiskCritical {
		t.Errorf("/bin/sh should be critical, got %s", got)
	}

	got = ClassifyCommand([]string{"/usr/bin/bash", "-c", "sudo rm -rf /"}, nil)
	if got != RiskCritical {
		t.Errorf("/usr/bin/bash should be critical, got %s", got)
	}
}

func TestApplyModifiers_HighestMatchWins(t *testing.T) {
	// When multiple modifiers match, the highest level should win.
	// Scenario: git push matches both ["push"] → High and ["push", "-f"] → Critical
	args := normalizeFlags("git", []string{"push", "-f", "origin"})
	result := applyModifiers(args, "git", RiskMedium, argumentModifiers)
	if result != RiskCritical {
		t.Errorf("git push -f should resolve to critical (highest match), got %s", result)
	}

	// Only ["push"] matches → High
	args2 := normalizeFlags("git", []string{"push", "origin"})
	result2 := applyModifiers(args2, "git", RiskMedium, argumentModifiers)
	if result2 != RiskHigh {
		t.Errorf("git push (no -f) should resolve to high, got %s", result2)
	}

	// No modifier matches → base level unchanged
	args3 := normalizeFlags("git", []string{"status"})
	result3 := applyModifiers(args3, "git", RiskMedium, argumentModifiers)
	if result3 != RiskMedium {
		t.Errorf("git status should stay medium, got %s", result3)
	}
}
