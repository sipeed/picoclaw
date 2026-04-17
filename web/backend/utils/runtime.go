package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// GetPicoclawHome returns the picoclaw home directory.
// Priority: $PICOCLAW_HOME > ~/.picoclaw
func GetPicoclawHome() string {
	return config.GetHome()
}

// GetDefaultConfigPath returns the default path to the picoclaw config file.
func GetDefaultConfigPath() string {
	if configPath := os.Getenv(config.EnvConfig); configPath != "" {
		return configPath
	}
	return filepath.Join(GetPicoclawHome(), "config.json")
}

// FindPicoclawBinary locates the picoclaw executable.
// Search order:
//  1. PICOCLAW_BINARY environment variable (explicit override)
//  2. Same directory as the current executable
//  3. Falls back to "picoclaw" and relies on $PATH
func FindPicoclawBinary() string {
	binaryName := "picoclaw"
	if runtime.GOOS == "windows" {
		binaryName = "picoclaw.exe"
	}

	if p := os.Getenv(config.EnvBinary); p != "" {
		if info, _ := os.Stat(p); info != nil && !info.IsDir() {
			return p
		}
	}

	if exe, err := os.Executable(); err == nil {
		logger.Debugf("Trying to find picoclaw binary in %s", exe)
		candidate := filepath.Join(filepath.Dir(exe), binaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return "picoclaw"
}

func appendUniqueIP(addrs []string, seen map[string]struct{}, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return addrs
	}
	if _, ok := seen[value]; ok {
		return addrs
	}
	seen[value] = struct{}{}
	return append(addrs, value)
}

// GetLocalIPv4s returns all non-loopback local IPv4 addresses.
func GetLocalIPv4s() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	results := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP == nil || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			results = appendUniqueIP(results, seen, ip4.String())
		}
	}
	return results
}

func isDisplayGlobalIPv6(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.To4() != nil {
		return false
	}
	ip = ip.To16()
	if ip == nil {
		return false
	}
	// Only show IPv6 global unicast addresses in 2000::/3.
	return ip[0]&0xe0 == 0x20
}

// GetGlobalIPv6s returns all IPv6 global unicast addresses.
func GetGlobalIPv6s() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	results := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP == nil {
			continue
		}
		ip := ipnet.IP
		if !isDisplayGlobalIPv6(ip) {
			continue
		}
		results = appendUniqueIP(results, seen, ip.String())
	}
	return results
}

// GetLocalIPv4 returns the first non-loopback local IPv4 address.
func GetLocalIPv4() string {
	addrs := GetLocalIPv4s()
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// GetLocalIPv6 returns the first IPv6 global unicast address.
func GetLocalIPv6() string {
	addrs := GetGlobalIPv6s()
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// GetLocalIP returns a non-loopback local IPv4 address for backward compatibility.
func GetLocalIP() string {
	return GetLocalIPv4()
}

// OpenBrowser automatically opens the given URL in the default browser.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
