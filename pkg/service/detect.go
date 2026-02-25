package service

import (
	"os"
	"strings"
)

func detectWSL() bool {
	return detectWSLWith(os.Getenv, os.ReadFile)
}

func detectWSLWith(getenv func(string) string, readFile func(string) ([]byte, error)) bool {
	for _, key := range []string{"WSL_DISTRO_NAME", "WSL_INTEROP"} {
		if strings.TrimSpace(getenv(key)) != "" {
			return true
		}
	}
	b, err := readFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}
