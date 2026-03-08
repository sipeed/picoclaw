package shell

import (
	"runtime"
	"testing"
)

func TestBuildSanitizedEnv_WindowsCaseInsensitive(t *testing.T) {
	// Skip on non-Windows
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// On Windows the OS typically stores "Path" not "PATH".
	// Verify that mixed-case inherited vars still pass the allowlist.
	t.Setenv("Path", `C:\Windows\system32`)

	env := BuildSanitizedEnv(nil, nil, nil, nil)

	v := getEnvValue(env, "PATH")
	if v == "" {
		t.Fatal("expected PATH to be present when OS provides 'Path'")
	}
	if v != `C:\Windows\system32` {
		t.Errorf("PATH = %q, want %q", v, `C:\Windows\system32`)
	}

	// Also verify lookup with original casing works.
	v2 := getEnvValue(env, "Path")
	if v2 == "" {
		t.Fatal("expected Get('Path') to resolve via case-insensitive lookup")
	}
}

func TestBuildSanitizedEnv_WindowsEnvSetCaseInsensitive(t *testing.T) {
	// Skip on non-Windows
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	env := BuildSanitizedEnv(nil, nil, map[string]string{
		"path": `C:\custom\bin`,
	}, nil)

	v := getEnvValue(env, "PATH")
	if v == "" {
		t.Fatal("expected PATH to be set via lowercase 'path' envSet key")
	}
	if v != `C:\custom\bin` {
		t.Errorf("PATH = %q, want %q", v, `C:\custom\bin`)
	}
}

func TestBuildSanitizedEnv_WindowsExtraAllowlistCaseInsensitive(t *testing.T) {
	// Skip on non-Windows
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	t.Setenv("my_custom_var", "hello")

	env := BuildSanitizedEnv(nil, []string{"MY_CUSTOM_VAR"}, nil, nil)

	v := getEnvValue(env, "my_custom_var")
	if v == "" {
		t.Fatal("expected my_custom_var to be found via case-insensitive allowlist")
	}
}
