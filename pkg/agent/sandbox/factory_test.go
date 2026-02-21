package sandbox

import "testing"

func TestNormalizeWorkspaceAccess(t *testing.T) {
	if got := normalizeWorkspaceAccess("ro"); got != "ro" {
		t.Fatalf("normalizeWorkspaceAccess(ro) = %q", got)
	}
	if got := normalizeWorkspaceAccess("RW"); got != "rw" {
		t.Fatalf("normalizeWorkspaceAccess(RW) = %q", got)
	}
	if got := normalizeWorkspaceAccess("invalid"); got != "none" {
		t.Fatalf("normalizeWorkspaceAccess(invalid) = %q", got)
	}
}
