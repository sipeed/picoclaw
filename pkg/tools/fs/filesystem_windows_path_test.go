package fstools

import "testing"

func TestNormalizeRootRelPathForWindowsSeparator(t *testing.T) {
	got := normalizeRootRelPathForSeparator(`aaa\bbb\file.txt`, '\\')
	want := "aaa/bbb/file.txt"

	if got != want {
		t.Fatalf("normalizeRootRelPathForSeparator() = %q, want %q", got, want)
	}
}

func TestNormalizeRootRelPathForUnixSeparatorLeavesBackslashUnchanged(t *testing.T) {
	input := `aaa\bbb\file.txt`
	got := normalizeRootRelPathForSeparator(input, '/')

	if got != input {
		t.Fatalf("normalizeRootRelPathForSeparator() = %q, want %q", got, input)
	}
}
