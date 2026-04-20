//go:build windows

package isolation

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const disableMaxPrivilege = 0x1

// windowsProcessResources holds native handles that must live for the lifetime
// of an isolated child process.
type windowsProcessResources struct {
	job   windows.Handle
	token windows.Token
}

var (
	windowsProcessResourcesByPID sync.Map
	windowsPendingResources      sync.Map
	advapi32                     = windows.NewLazySystemDLL("advapi32.dll")
	procCreateRestrictedToken    = advapi32.NewProc("CreateRestrictedToken")
)

func applyPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	if !isolation.Enabled || cmd == nil {
		return nil
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	rules := BuildWindowsAccessRules(root, isolation.ExposePaths)
	logger.InfoCF("isolation", "windows isolation process constraints",
		map[string]any{
			"root":    root,
			"command": cmd.Path,
			"rules":   formatWindowsAccessRules(rules),
			"note":    "Windows currently enforces restricted token, low integrity, and job object limits; expose_paths filesystem remapping is rejected during preflight",
		})
	// Create the restricted token before the process starts so CreateProcess uses
	// the reduced privilege set from the first instruction.
	restrictedToken, err := createRestrictedPrimaryToken()
	if err != nil {
		return fmt.Errorf("create restricted primary token: %w", err)
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_BREAKAWAY_FROM_JOB
	cmd.SysProcAttr.Token = syscall.Token(restrictedToken)
	windowsPendingResources.Store(cmd, windowsProcessResources{token: restrictedToken})
	return nil
}

func postStartPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	if !isolation.Enabled || cmd == nil || cmd.Process == nil {
		return nil
	}
	resourcesAny, _ := windowsPendingResources.LoadAndDelete(cmd)
	resources, _ := resourcesAny.(windowsProcessResources)
	// Job objects can only be attached after the process exists, so the Windows
	// backend finishes isolation in this post-start hook.
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		if resources.token != 0 {
			_ = resources.token.Close()
		}
		return fmt.Errorf("create windows job object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		if resources.token != 0 {
			_ = resources.token.Close()
		}
		return fmt.Errorf("set windows job object info: %w", err)
	}

	proc, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		if resources.token != 0 {
			_ = resources.token.Close()
		}
		return fmt.Errorf("open process for job assignment: %w", err)
	}

	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(proc)
		_ = windows.CloseHandle(job)
		if resources.token != 0 {
			_ = resources.token.Close()
		}
		return fmt.Errorf("assign process to job object: %w", err)
	}

	if resources.token != 0 {
		_ = resources.token.Close()
	}
	resources.job = job
	windowsProcessResourcesByPID.Store(cmd.Process.Pid, resources)
	go reapWindowsProcessResources(cmd.Process.Pid, proc, job)
	return nil
}

func cleanupPendingPlatformResources(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	resourcesAny, ok := windowsPendingResources.LoadAndDelete(cmd)
	if !ok {
		return
	}
	resources, _ := resourcesAny.(windowsProcessResources)
	if resources.token != 0 {
		_ = resources.token.Close()
	}
}

func reapWindowsProcessResources(pid int, proc windows.Handle, job windows.Handle) {
	_, _ = windows.WaitForSingleObject(proc, windows.INFINITE)
	_ = windows.CloseHandle(proc)
	_ = windows.CloseHandle(job)
	windowsProcessResourcesByPID.Delete(pid)
}

// createRestrictedPrimaryToken duplicates the current process token, removes
// maximum privileges, and lowers integrity before it is assigned to a child.
func createRestrictedPrimaryToken() (windows.Token, error) {
	var current windows.Token
	if err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT,
		&current,
	); err != nil {
		return 0, err
	}
	defer current.Close()

	var restricted windows.Token
	r1, _, e1 := procCreateRestrictedToken.Call(
		uintptr(current),
		uintptr(disableMaxPrivilege),
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&restricted)),
	)
	if r1 == 0 {
		if e1 != nil && e1 != syscall.Errno(0) {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	if err := setTokenLowIntegrity(restricted); err != nil {
		_ = restricted.Close()
		return 0, err
	}
	return restricted, nil
}

// setTokenLowIntegrity lowers the token integrity level so writes to higher
// integrity locations are blocked by the OS.
func setTokenLowIntegrity(token windows.Token) error {
	lowSID, err := windows.CreateWellKnownSid(windows.WinLowLabelSid)
	if err != nil {
		return fmt.Errorf("create low integrity sid: %w", err)
	}
	tml := windows.Tokenmandatorylabel{
		Label: windows.SIDAndAttributes{
			Sid:        lowSID,
			Attributes: windows.SE_GROUP_INTEGRITY,
		},
	}
	if err := windows.SetTokenInformation(
		token,
		windows.TokenIntegrityLevel,
		(*byte)(unsafe.Pointer(&tml)),
		tml.Size(),
	); err != nil {
		return fmt.Errorf("set token low integrity: %w", err)
	}
	return nil
}

// formatWindowsAccessRules reshapes the internal rules for structured logging.
func formatWindowsAccessRules(rules []AccessRule) []map[string]string {
	formatted := make([]map[string]string, 0, len(rules))
	for _, rule := range rules {
		formatted = append(formatted, map[string]string{
			"path": rule.Path,
			"mode": rule.Mode,
		})
	}
	return formatted
}
