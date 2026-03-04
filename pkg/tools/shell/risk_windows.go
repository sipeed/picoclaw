//go:build windows

package shell

func init() {
	// Windows-specific command risk entries.
	// On Windows, both Unix commands (from the shared table) and these
	// Windows-native commands are available — WSL, Git Bash, and MSYS2
	// expose Unix tooling alongside native Windows executables.

	windowsCommands := map[string]RiskLevel{
		// Low — read-only, informational
		"dir":        RiskLow,
		"where":      RiskLow,
		"ver":        RiskLow,
		"set":        RiskLow,
		"systeminfo": RiskLow,
		"tasklist":   RiskLow,
		"findstr":    RiskLow,
		"assoc":      RiskLow,
		"ftype":      RiskLow,
		"path":       RiskLow,
		"vol":        RiskLow,
		"chcp":       RiskLow,

		// Medium — file modification, utilities
		"copy":     RiskMedium,
		"xcopy":    RiskMedium,
		"robocopy": RiskMedium,
		"move":     RiskMedium,
		"ren":      RiskMedium,
		"rename":   RiskMedium,
		"md":       RiskMedium,
		"compact":  RiskMedium,
		"attrib":   RiskMedium,
		"certutil": RiskMedium,
		"clip":     RiskMedium,
		"mklink":   RiskMedium,

		// High — destructive, system-modifying
		"del":      RiskHigh,
		"erase":    RiskHigh,
		"rd":       RiskHigh,
		"taskkill": RiskHigh,
		"icacls":   RiskHigh,
		"cacls":    RiskHigh,
		"takeown":  RiskHigh,

		// Critical — privilege escalation, registry, system config
		"runas":    RiskCritical,
		"reg":      RiskCritical,
		"regedit":  RiskCritical,
		"bcdedit":  RiskCritical,
		"bcdboot":  RiskCritical,
		"net":      RiskCritical,
		"sc":       RiskCritical,
		"netsh":    RiskCritical,
		"schtasks": RiskCritical,
		"at":       RiskCritical,
		"wmic":     RiskCritical,
		"msiexec":  RiskCritical,
		"dism":     RiskCritical,
		"sfc":      RiskCritical,
		"format":   RiskCritical,
		"diskpart": RiskCritical,

		// Critical — shell wrappers (cmd.exe) and script hosts
		"powershell": RiskCritical, // Windows PowerShell 5.1 (Windows-only; pwsh is cross-platform)
		"cmd":        RiskCritical,
		"cmd.exe":    RiskCritical,
		"cscript":    RiskCritical,
		"wscript":    RiskCritical,
		"mshta":      RiskCritical,
	}

	for k, v := range windowsCommands {
		commandRiskTable[k] = v
	}

	// Windows-specific argument modifiers.
	windowsArgModifiers := map[string][]ArgModifier{
		"robocopy": {
			{Args: []string{"/MIR"}, Level: RiskHigh},   // mirror = deletes extras in destination
			{Args: []string{"/PURGE"}, Level: RiskHigh}, // delete dest files not in source
		},
		"certutil": {
			{Args: []string{"-urlcache"}, Level: RiskHigh},  // download files from URL
			{Args: []string{"-decode"}, Level: RiskHigh},    // decode Base64 (malware delivery)
			{Args: []string{"-decodehex"}, Level: RiskHigh}, // decode hex (malware delivery)
		},
		"del": {
			{Args: []string{"/s"}, Level: RiskCritical}, // recursive delete
			{Args: []string{"/q"}, Level: RiskCritical}, // quiet (no confirmation)
		},
		"rd": {
			{Args: []string{"/s"}, Level: RiskCritical}, // recursive delete
		},
		"taskkill": {
			{Args: []string{"/f"}, Level: RiskCritical},  // force kill
			{Args: []string{"/im"}, Level: RiskCritical}, // kill by image name (bulk)
		},
	}

	for k, v := range windowsArgModifiers {
		argumentModifiers[k] = append(argumentModifiers[k], v...)
	}
}
