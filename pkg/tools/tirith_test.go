package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewExecTool_DefaultTirithDisabled(t *testing.T) {
	tool, err := NewExecTool("", false)
	require.NoError(t, err)
	require.False(t, tool.tirithConfig.Enabled)
	require.Equal(t, "tirith", tool.tirithConfig.BinPath)
	require.Equal(t, 5, tool.tirithConfig.Timeout)
	require.True(t, tool.tirithConfig.FailOpen)
}

func TestTirithGuard_DisabledDoesNotResolve(t *testing.T) {
	cfg := TirithConfig{
		Enabled:  false,
		BinPath:  filepath.Join(t.TempDir(), "missing-tirith"),
		Timeout:  5,
		FailOpen: false,
	}
	require.Empty(t, tirithGuard(context.Background(), "echo hello", "", cfg))
}

func TestTirithGuard_MissingBinaryFailOpenAndFailClosed(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-tirith")

	failOpen := tirithGuard(context.Background(), "echo hello", "", TirithConfig{
		Enabled:  true,
		BinPath:  missing,
		Timeout:  5,
		FailOpen: true,
	})
	require.Empty(t, failOpen)

	failClosed := tirithGuard(context.Background(), "echo hello", "", TirithConfig{
		Enabled:  true,
		BinPath:  missing,
		Timeout:  5,
		FailOpen: false,
	})
	require.Contains(t, failClosed, "fail-closed")
}

func TestResolveTirithPath_NormalizationAndValidation(t *testing.T) {
	require.Equal(t, "tirith", normalizeTirithBin(""))
	require.Equal(t, "tirith", normalizeTirithBin("   "))

	dir := t.TempDir()
	_, _, err := resolveTirithPath(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "directory")

	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	homeBin := filepath.Join(home, "tirith")
	writeExecutableFile(t, homeBin, fakeTirithScript(`printf '{"findings":[]}'`+"\nexit 0\n"))
	resolved, cacheKey, err := resolveTirithPath("~/tirith")
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(homeBin), resolved)
	require.Equal(t, filepath.Clean(homeBin), cacheKey)

	if runtime.GOOS != "windows" {
		notExecutable := filepath.Join(t.TempDir(), "tirith")
		require.NoError(t, os.WriteFile(notExecutable, []byte("#!/bin/sh\nexit 0\n"), 0o644))
		_, _, err = resolveTirithPath(notExecutable)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not executable")
	}
}

func TestResolveTirithPath_PathLookup(t *testing.T) {
	skipWindowsProcessTest(t)

	dir := t.TempDir()
	bin := filepath.Join(dir, "tirith")
	writeExecutableFile(t, bin, fakeTirithScript(`printf '{"findings":[]}'`+"\nexit 0\n"))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	resolved, cacheKey, err := resolveTirithPath("tirith")
	require.NoError(t, err)
	require.Equal(t, bin, resolved)
	require.Equal(t, "tirith", cacheKey)
}

func TestResolveTirithPath_ExplicitRelativePathDoesNotUsePath(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWD))
	})
	require.NoError(t, os.Chdir(dir))

	local := filepath.Join(dir, "tirith")
	writeExecutableFile(t, local, fakeTirithScript(`printf '{"findings":[]}'`+"\nexit 0\n"))

	pathDir := t.TempDir()
	pathBin := filepath.Join(pathDir, "tirith")
	writeExecutableFile(t, pathBin, fakeTirithScript("exit 1\n"))
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	resolved, cacheKey, err := resolveTirithPath("./tirith")
	require.NoError(t, err)
	expected, err := filepath.Abs("tirith")
	require.NoError(t, err)
	require.Equal(t, expected, resolved)
	require.Equal(t, expected, cacheKey)
}

func TestTirithGuard_DoesNotCacheFailedResolution(t *testing.T) {
	skipWindowsProcessTest(t)

	bin := filepath.Join(t.TempDir(), "tirith")
	cfg := TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: true}

	require.Empty(t, tirithGuard(context.Background(), "echo first", "", cfg))

	writeExecutableFile(t, bin, fakeTirithScript(`printf '{"findings":[{"severity":"HIGH","title":"later install"}]}'`+"\nexit 1\n"))
	got := tirithGuard(context.Background(), "echo second", "", cfg)
	require.Contains(t, got, "Command blocked by Tirith")
	require.Contains(t, got, "later install")
}

func TestTirithGuard_Argv(t *testing.T) {
	skipWindowsProcessTest(t)

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript(`
printf '%s\n' "$@" > "$TIRITH_ARGS_FILE"
printf '{"findings":[]}'
exit 0
`))
	t.Setenv("TIRITH_ARGS_FILE", argsFile)

	got := tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: false})
	require.Empty(t, got)

	argsData, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	args := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	require.Equal(t, []string{
		"check",
		"--format",
		"json",
		"--non-interactive",
		"--no-daemon",
		"--shell",
		"posix",
		"--",
		"echo hello",
	}, args)
}

func TestTirithGuard_UsesLocalOnlyPolicyEnvironment(t *testing.T) {
	skipWindowsProcessTest(t)

	envFile := filepath.Join(t.TempDir(), "env.txt")
	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript(`
{
  printf 'policy_root=%s\n' "$TIRITH_POLICY_ROOT"
  printf 'server=%s\n' "${TIRITH_SERVER_URL-unset}"
  printf 'api=%s\n' "${TIRITH_API_KEY-unset}"
  printf 'bypass=%s\n' "${TIRITH-unset}"
  printf 'allow_http=%s\n' "${TIRITH_ALLOW_HTTP-unset}"
  printf 'gsb=%s\n' "${GOOGLE_SAFE_BROWSING_API_KEY-unset}"
  printf 'abusech=%s\n' "${ABUSECH_AUTH_KEY-unset}"
  printf 'path=%s\n' "${PATH-unset}"
  printf 'home=%s\n' "${HOME-unset}"
  printf '%s\n' '---policy---'
  cat "$TIRITH_POLICY_ROOT/.tirith/policy.yaml"
} > "$TIRITH_ENV_FILE"
printf '{"findings":[]}'
exit 0
`))
	t.Setenv("TIRITH_ENV_FILE", envFile)
	t.Setenv("TIRITH_POLICY_ROOT", "/should/not/be/used")
	t.Setenv("TIRITH_SERVER_URL", "https://policy.example.invalid")
	t.Setenv("TIRITH_API_KEY", "secret")
	t.Setenv("TIRITH", "0")
	t.Setenv("TIRITH_ALLOW_HTTP", "1")
	t.Setenv("GOOGLE_SAFE_BROWSING_API_KEY", "gsb-secret")
	t.Setenv("ABUSECH_AUTH_KEY", "abusech-secret")

	got := tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: false})
	require.Empty(t, got)

	data, err := os.ReadFile(envFile)
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "policy_root=")
	require.NotContains(t, text, "/should/not/be/used")
	require.Contains(t, text, "server=unset")
	require.Contains(t, text, "api=unset")
	require.Contains(t, text, "bypass=unset")
	require.Contains(t, text, "allow_http=unset")
	require.Contains(t, text, "gsb=unset")
	require.Contains(t, text, "abusech=unset")
	require.Contains(t, text, "path=")
	require.Contains(t, text, "home=")
	require.Contains(t, text, "allow_bypass_env: false")
	require.Contains(t, text, "allow_bypass_env_noninteractive: false")
	require.Contains(t, text, "auto_update_hours: 0")
	require.Contains(t, text, "osv_enabled: false")
	require.Contains(t, text, "deps_dev_enabled: false")
	require.Contains(t, text, "phishing_army_enabled: false")
}

func TestTirithGuard_UsesCommandWorkingDirectory(t *testing.T) {
	skipWindowsProcessTest(t)

	cwd := t.TempDir()
	pwdFile := filepath.Join(t.TempDir(), "pwd.txt")
	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript(`
pwd > "$TIRITH_PWD_FILE"
printf '{"findings":[]}'
exit 0
`))
	t.Setenv("TIRITH_PWD_FILE", pwdFile)

	got := tirithGuard(context.Background(), "echo hello", cwd, TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: false})
	require.Empty(t, got)

	data, err := os.ReadFile(pwdFile)
	require.NoError(t, err)
	expectedCWD, err := filepath.EvalSymlinks(cwd)
	require.NoError(t, err)
	require.Equal(t, expectedCWD, strings.TrimSpace(string(data)))
}

func TestTirithGuard_ExitCodes(t *testing.T) {
	skipWindowsProcessTest(t)

	tests := []struct {
		name      string
		exitCode  int
		failOpen  bool
		wantBlock bool
		wantText  string
	}{
		{name: "allow", exitCode: 0, failOpen: false},
		{name: "block", exitCode: 1, failOpen: false, wantBlock: true, wantText: "Pipe to shell"},
		{name: "warn", exitCode: 2, failOpen: false},
		{name: "warn ack defensive", exitCode: 3, failOpen: false},
		{name: "unknown fail open", exitCode: 9, failOpen: true},
		{name: "unknown fail closed", exitCode: 9, failOpen: false, wantBlock: true, wantText: "fail-closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "tirith")
			writeExecutableFile(t, bin, fakeTirithScript(`printf '{"findings":[{"severity":"HIGH","title":"Pipe to shell"}]}'`+"\nexit "+itoa(tt.exitCode)+"\n"))
			got := tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: tt.failOpen})
			if tt.wantBlock {
				require.NotEmpty(t, got)
				require.Contains(t, got, tt.wantText)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

func TestTirithGuard_InvalidJSONKeepsExitVerdict(t *testing.T) {
	skipWindowsProcessTest(t)

	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript("printf 'not json'\nexit 1\n"))

	got := tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: false})
	require.Contains(t, got, "Command blocked by Tirith")
	require.Contains(t, got, "details unavailable")
}

func TestTirithGuard_TimeoutUsesFailOpen(t *testing.T) {
	skipWindowsProcessTest(t)

	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript("sleep 2\nexit 0\n"))

	require.Empty(t, tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 1, FailOpen: true}))

	got := tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 1, FailOpen: false})
	require.Contains(t, got, "fail-closed")
	require.Contains(t, got, "timed out")
}

func TestTirithGuard_TimeoutKillsProcessTree(t *testing.T) {
	skipWindowsProcessTest(t)

	childPIDFile := filepath.Join(t.TempDir(), "child.pid")
	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript(`
sleep 30 &
echo $! > "$TIRITH_CHILD_PID_FILE"
wait
`))
	t.Setenv("TIRITH_CHILD_PID_FILE", childPIDFile)

	start := time.Now()
	require.Empty(t, tirithGuard(context.Background(), "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 1, FailOpen: true}))
	require.Less(t, time.Since(start), 5*time.Second)

	data, err := os.ReadFile(childPIDFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return !tirithTestProcessExists(pid)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestTirithGuard_ContextCanceledBlocksExecution(t *testing.T) {
	skipWindowsProcessTest(t)

	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript("sleep 2\nexit 0\n"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := tirithGuard(ctx, "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: true})
	require.Equal(t, "Tirith security scan canceled", got)
}

func TestTirithGuard_ParentDeadlineBlocksExecution(t *testing.T) {
	skipWindowsProcessTest(t)

	bin := filepath.Join(t.TempDir(), "tirith")
	writeExecutableFile(t, bin, fakeTirithScript("sleep 2\nexit 0\n"))

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	got := tirithGuard(ctx, "echo hello", "", TirithConfig{Enabled: true, BinPath: bin, Timeout: 5, FailOpen: true})
	require.Equal(t, "Tirith security scan canceled", got)
}

func TestCappedBuffer_DiscardReturnsFullWriteLength(t *testing.T) {
	var buf cappedBuffer
	buf.limit = 5

	n, err := buf.Write([]byte("abcdef"))
	require.NoError(t, err)
	require.Equal(t, 6, n)
	require.Equal(t, "abcde", buf.String())

	n, err = buf.Write([]byte("ghij"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "abcde", buf.String())
}

func TestTirithSummarize_SanitizesAndCapsFindings(t *testing.T) {
	findings := make([]map[string]string, 0, tirithMaxFindings+5)
	findings = append(findings, map[string]string{
		"severity": "\x1b[31mHIGH\x1b[0m",
		"title":    "bad\u202e\u200b title",
	})
	for i := 0; i < tirithMaxFindings+4; i++ {
		findings = append(findings, map[string]string{"severity": "LOW", "title": "extra"})
	}
	payload, err := json.Marshal(map[string]any{"findings": findings})
	require.NoError(t, err)

	got := tirithSummarize(payload)
	require.Contains(t, got, "[HIGH] bad title")
	require.NotContains(t, got, "\x1b")
	require.NotContains(t, got, "[31m")
	require.NotContains(t, got, "\u202e")
	require.NotContains(t, got, "\u200b")
	require.Contains(t, got, "...and 5 more")
	require.LessOrEqual(t, len([]rune(got)), tirithMaxSummaryRunes+3)
}

func TestTirithSummarize_CapsSummary(t *testing.T) {
	payload, err := json.Marshal(map[string]any{"summary": strings.Repeat("x", tirithMaxSummaryRunes+20)})
	require.NoError(t, err)

	got := tirithSummarize(payload)
	require.Len(t, []rune(got), tirithMaxSummaryRunes+3)
	require.True(t, strings.HasSuffix(got, "..."))
}

func TestTirithWarningCacheSuppressesRepeatedScannerFailureLogs(t *testing.T) {
	resetTirithWarningCacheForTest()

	var logs bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
	}()

	missing := filepath.Join(t.TempDir(), "missing-tirith")
	cfg := TirithConfig{Enabled: true, BinPath: missing, Timeout: 5, FailOpen: true}
	require.Empty(t, tirithGuard(context.Background(), "echo one", "", cfg))
	require.Empty(t, tirithGuard(context.Background(), "echo two", "", cfg))

	require.Equal(t, 1, strings.Count(logs.String(), "unavailable (fail-open)"))
}

func TestTirithSummarize_InvalidAndEmptyJSON(t *testing.T) {
	require.Contains(t, tirithSummarize([]byte("not json")), "details unavailable")
	require.Equal(t, "security issue detected", tirithSummarize([]byte(`{"findings":[]}`)))
	require.Equal(t, "security issue detected", tirithSummarize(nil))
}

func resetTirithWarningCacheForTest() {
	tirithWarningCache.Lock()
	defer tirithWarningCache.Unlock()
	tirithWarningCache.seen = make(map[string]struct{})
}

func writeExecutableFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(path, 0o755))
	}
}

func fakeTirithScript(body string) string {
	return "#!/bin/sh\n" + body
}

func skipWindowsProcessTest(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script Tirith fake is Unix-only")
	}
}

func tirithTestProcessExists(pid int) bool {
	return exec.Command("kill", "-0", itoa(pid)).Run() == nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	return string(digits[i:])
}
