package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
)

// I2CTool provides I2C bus interaction for reading sensors and controlling peripherals.
type I2CTool struct{}

func NewI2CTool() *I2CTool {
	return &I2CTool{}
}

func (t *I2CTool) Name() string {
	return "i2c"
}

func (t *I2CTool) Description() string {
	return "Interact with I2C bus devices for reading sensors and controlling peripherals. Actions: detect (list buses), scan (find devices on a bus), read (read bytes from device), write (send bytes to device). Linux only."
}

func (t *I2CTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"detect", "scan", "read", "write"},
				"description": "Action to perform: detect (list available I2C buses), scan (find devices on a bus), read (read bytes from a device), write (send bytes to a device)",
			},
			"bus": map[string]interface{}{
				"type":        "string",
				"description": "I2C bus number (e.g. \"1\" for /dev/i2c-1). Required for scan/read/write.",
			},
			"address": map[string]interface{}{
				"type":        "integer",
				"description": "7-bit I2C device address (0x03-0x77). Required for read/write.",
			},
			"register": map[string]interface{}{
				"type":        "integer",
				"description": "Register address to read from or write to. If set, sends register byte before read/write.",
			},
			"data": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "integer"},
				"description": "Bytes to write (0-255 each). Required for write action.",
			},
			"length": map[string]interface{}{
				"type":        "integer",
				"description": "Number of bytes to read (1-256). Default: 1. Used with read action.",
			},
			"confirm": map[string]interface{}{
				"type":        "boolean",
				"description": "Must be true for write operations. Safety guard to prevent accidental writes.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *I2CTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if runtime.GOOS != "linux" {
		return ErrorResult("I2C is only supported on Linux. This tool requires /dev/i2c-* device files.")
	}

	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "detect":
		return t.detect()
	case "scan":
		return t.scan(args)
	case "read":
		return t.readDevice(args)
	case "write":
		return t.writeDevice(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s (valid: detect, scan, read, write)", action))
	}
}

// detect lists available I2C buses by globbing /dev/i2c-*
func (t *I2CTool) detect() *ToolResult {
	matches, err := filepath.Glob("/dev/i2c-*")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to scan for I2C buses: %v", err))
	}

	if len(matches) == 0 {
		return SilentResult("No I2C buses found. You may need to:\n1. Load the i2c-dev module: modprobe i2c-dev\n2. Check that I2C is enabled in device tree\n3. Configure pinmux for your board (see hardware skill)")
	}

	type busInfo struct {
		Path string `json:"path"`
		Bus  string `json:"bus"`
	}

	buses := make([]busInfo, 0, len(matches))
	re := regexp.MustCompile(`/dev/i2c-(\d+)`)
	for _, m := range matches {
		if sub := re.FindStringSubmatch(m); sub != nil {
			buses = append(buses, busInfo{Path: m, Bus: sub[1]})
		}
	}

	result, _ := json.MarshalIndent(buses, "", "  ")
	return SilentResult(fmt.Sprintf("Found %d I2C bus(es):\n%s", len(buses), string(result)))
}

// isValidBusID checks that a bus identifier is a simple number (prevents path injection)
func isValidBusID(id string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, id)
	return matched
}

// parseI2CAddress extracts and validates an I2C address from args
func parseI2CAddress(args map[string]interface{}) (int, *ToolResult) {
	addrFloat, ok := args["address"].(float64)
	if !ok {
		return 0, ErrorResult("address is required (e.g. 0x38 for AHT20)")
	}
	addr := int(addrFloat)
	if addr < 0x03 || addr > 0x77 {
		return 0, ErrorResult("address must be in valid 7-bit range (0x03-0x77)")
	}
	return addr, nil
}

// parseI2CBus extracts and validates an I2C bus from args
func parseI2CBus(args map[string]interface{}) (string, *ToolResult) {
	bus, ok := args["bus"].(string)
	if !ok || bus == "" {
		return "", ErrorResult("bus is required (e.g. \"1\" for /dev/i2c-1)")
	}
	if !isValidBusID(bus) {
		return "", ErrorResult("invalid bus identifier: must be a number (e.g. \"1\")")
	}
	return bus, nil
}
