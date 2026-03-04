package shell

import "testing"

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

		{[]string{"cp", "a", "b"}, RiskMedium},
		{[]string{"mv", "a", "b"}, RiskMedium},
		{[]string{"python3", "-c", "print(1)"}, RiskMedium},
		{[]string{"git", "status"}, RiskMedium},
		{[]string{"curl", "https://example.com"}, RiskMedium},

		{[]string{"rm", "file.txt"}, RiskHigh},
		{[]string{"chmod", "755", "script.sh"}, RiskHigh},
		{[]string{"docker", "ps"}, RiskHigh},
		{[]string{"ssh", "user@host"}, RiskHigh},

		{[]string{"sudo", "ls"}, RiskCritical},
		{[]string{"dd", "if=/dev/zero", "of=/dev/sda"}, RiskCritical},
		{[]string{"shutdown", "-h", "now"}, RiskCritical},
		{[]string{"eval", "echo hi"}, RiskCritical},
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
		{"curl -d data", []string{"curl", "-d", "data", "https://example.com"}, RiskHigh},
		{"curl --data data", []string{"curl", "--data", "data", "url"}, RiskHigh},
		{"curl -X DELETE", []string{"curl", "-X", "DELETE", "url"}, RiskHigh},
		{"curl --request POST", []string{"curl", "--request", "POST", "url"}, RiskHigh},

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
		"rm":   "low",
		"curl": "critical",
	}

	got := ClassifyCommand([]string{"rm", "-rf", "/"}, overrides)
	if got != RiskLow {
		t.Errorf("override rm to low: got %s", got)
	}

	got = ClassifyCommand([]string{"curl", "https://example.com"}, overrides)
	if got != RiskCritical {
		t.Errorf("override curl to critical: got %s", got)
	}

	got = ClassifyCommand([]string{"ls"}, overrides)
	if got != RiskLow {
		t.Errorf("ls (no override) should be low: got %s", got)
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
		{[]string{"-r", "-f"}, []string{"-r", "-f"}},
		{[]string{"--force"}, []string{"--force"}},
		{[]string{"-f"}, []string{"-f"}},
		{[]string{"push"}, []string{"push"}},
		{[]string{"-rf", "dir", "--verbose"}, []string{"-r", "-f", "dir", "--verbose"}},
	}

	for _, tt := range tests {
		got := normalizeFlags(tt.input)
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
	// elevates to critical and built-in is checked first.
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
