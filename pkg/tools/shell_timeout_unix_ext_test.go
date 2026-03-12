package tools

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	if err != nil && err != syscall.EPERM {
		return false
	}

	data, readErr := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if readErr != nil {
		return false
	}
	raw := string(data)
	end := strings.LastIndex(raw, ")")
	if end == -1 || end+2 >= len(raw) {
		return true
	}
	fields := strings.Fields(raw[end+2:])
	if len(fields) == 0 {
		return true
	}
	state := fields[0]
	return state != "Z"
}
