package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunIntegrationTestsScriptExecutesSuiteCommand(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}

	repoRoot := repoRootFromTestFile(t)
	suitesRoot := filepath.Join(repoRoot, "integration", "suites")
	suiteDir, err := os.MkdirTemp(suitesRoot, "runner-script-")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(suiteDir)
	})

	suiteName := filepath.Base(suiteDir)
	if err := os.WriteFile(
		filepath.Join(suiteDir, "suite.env"),
		[]byte("TEST_COMMAND='printf runner-ok'\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(suite.env) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(suiteDir, "docker-compose.yml"),
		[]byte("services:\n  fake-dependency:\n    image: busybox\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(docker-compose.yml) error = %v", err)
	}

	stubDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "docker.log")
	if err := os.WriteFile(filepath.Join(stubDir, "docker"), []byte(`#!/bin/sh
set -eu

log_file="${DOCKER_LOG:?}"
{
  printf '%s\n' '---'
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} >>"$log_file"

subcommand=""
for arg in "$@"; do
  case "$arg" in
    config|up|run|down)
      subcommand="$arg"
      ;;
  esac
done

case "$subcommand" in
  config)
    printf '%s\n' integration-runner fake-dependency
    ;;
  up)
    ;;
  run)
    printf '%s\n' runner-ok
    ;;
  down)
    ;;
  *)
    printf 'unexpected docker invocation: %s\n' "$*" >&2
    exit 1
    ;;
esac
`), 0o755); err != nil {
		t.Fatalf("WriteFile(docker stub) error = %v", err)
	}

	cmd := exec.Command(bashPath, filepath.Join(repoRoot, "scripts", "run-integration-tests.sh"), suiteName)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"DOCKER_LOG="+logPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run-integration-tests.sh error = %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(string(output), "runner-ok") {
		t.Fatalf("script output did not include runner output:\n%s", output)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(logPath) error = %v", err)
	}

	runArgs := findLoggedDockerInvocation(t, string(logData), "run")
	if strings.Contains(strings.Join(runArgs, "\n"), "\nsh\n-c\n") {
		t.Fatalf("docker compose run unexpectedly wrapped TEST_COMMAND with sh -c:\n%v", runArgs)
	}

	if !containsArg(runArgs, "integration-runner") {
		t.Fatalf("docker compose run args missing runner service:\n%v", runArgs)
	}
	if !containsArg(runArgs, "printf runner-ok") {
		t.Fatalf("docker compose run args missing suite command as a single argument:\n%v", runArgs)
	}
}

func repoRootFromTestFile(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Dir(wd)
}

func findLoggedDockerInvocation(t *testing.T, logData, subcommand string) []string {
	t.Helper()

	for _, block := range strings.Split(logData, "---\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		args := strings.Split(block, "\n")
		if containsArg(args, subcommand) {
			return args
		}
	}

	t.Fatalf("did not find docker %q invocation in log:\n%s", subcommand, logData)
	return nil
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
