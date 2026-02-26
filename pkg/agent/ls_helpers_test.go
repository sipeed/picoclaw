package agent

import "testing"

func TestIsLsCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ls", true},
		{"ls -la", true},
		{"ls /tmp", true},
		{"lsof", false},
		{"echo ls", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isLsCommand(tt.cmd); got != tt.want {
			t.Errorf("isLsCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestHasLongFlag(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ls", false},
		{"ls -l", true},
		{"ls -la", true},
		{"ls -al", true},
		{"ls --color", false},
		{"ls -a /tmp", false},
		{"ls -l --color /tmp", true},
	}
	for _, tt := range tests {
		if got := hasLongFlag(tt.cmd); got != tt.want {
			t.Errorf("hasLongFlag(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsPermString(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"drwxr-xr-x", true},
		{"-rw-r--r--", true},
		{"lrwxrwxrwx", true},
		{"-rwsr-xr-x", true},  // setuid
		{"drwxrwxrwt", true},  // sticky
		{"hello world", false}, // wrong length/chars
		{"----------", true},
		{"xrwxrwxrwx", false}, // invalid first char
		{"", false},
	}
	for _, tt := range tests {
		if got := isPermString(tt.s); got != tt.want {
			t.Errorf("isPermString(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}
