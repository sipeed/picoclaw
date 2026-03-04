package shell

import (
	"testing"
)

func TestBuildSanitizedEnv_WindowsCaseInsensitive(t *testing.T) {
	// On Windows the OS typically stores "Path" not "PATH".
	// Verify that mixed-case inherited vars still pass the allowlist.
	t.Setenv("Path", `C:\Windows\system32`)

	env := BuildSanitizedEnv(nil, nil)

	v := env.Get("PATH")
	if !v.IsSet() {
		t.Fatal("expected PATH to be present when OS provides 'Path'")
	}
	if v.Str != `C:\Windows\system32` {
		t.Errorf("PATH = %q, want %q", v.Str, `C:\Windows\system32`)
	}

	// Also verify lookup with original casing works.
	v2 := env.Get("Path")
	if !v2.IsSet() {
		t.Fatal("expected Get('Path') to resolve via case-insensitive lookup")
	}
}

func TestBuildSanitizedEnv_WindowsEnvSetCaseInsensitive(t *testing.T) {
	env := BuildSanitizedEnv(nil, map[string]string{
		"path": `C:\custom\bin`,
	})

	v := env.Get("PATH")
	if !v.IsSet() {
		t.Fatal("expected PATH to be set via lowercase 'path' envSet key")
	}
	if v.Str != `C:\custom\bin` {
		t.Errorf("PATH = %q, want %q", v.Str, `C:\custom\bin`)
	}
}

func TestBuildSanitizedEnv_WindowsExtraAllowlistCaseInsensitive(t *testing.T) {
	t.Setenv("my_custom_var", "hello")

	env := BuildSanitizedEnv([]string{"MY_CUSTOM_VAR"}, nil)

	v := env.Get("my_custom_var")
	if !v.IsSet() {
		t.Fatal("expected my_custom_var to be found via case-insensitive allowlist")
	}
}
