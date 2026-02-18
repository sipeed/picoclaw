package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
)

// SPITool provides SPI bus interaction for high-speed peripheral communication.
type SPITool struct{}

func NewSPITool() *SPITool {
	return &SPITool{}
}

func (t *SPITool) Name() string {
	return "spi"
}

func (t *SPITool) Description() string {
	return "Interact with SPI bus devices for high-speed peripheral communication. Actions: list (find SPI devices), transfer (full-duplex send/receive), read (receive bytes). Linux only."
}

func (t *SPITool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "transfer", "read"},
				"description": "Action to perform: list (find available SPI devices), transfer (full-duplex send/receive), read (receive bytes by sending zeros)",
			},
			"device": map[string]interface{}{
				"type":        "string",
				"description": "SPI device identifier (e.g. \"2.0\" for /dev/spidev2.0). Required for transfer/read.",
			},
			"speed": map[string]interface{}{
				"type":        "integer",
				"description": "SPI clock speed in Hz. Default: 1000000 (1 MHz).",
			},
			"mode": map[string]interface{}{
				"type":        "integer",
				"description": "SPI mode (0-3). Default: 0. Mode sets CPOL and CPHA: 0=0,0 1=0,1 2=1,0 3=1,1.",
			},
			"bits": map[string]interface{}{
				"type":        "integer",
				"description": "Bits per word. Default: 8.",
			},
			"data": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "integer"},
				"description": "Bytes to send (0-255 each). Required for transfer action.",
			},
			"length": map[string]interface{}{
				"type":        "integer",
				"description": "Number of bytes to read (1-4096). Required for read action.",
			},
			"confirm": map[string]interface{}{
				"type":        "boolean",
				"description": "Must be true for transfer operations. Safety guard to prevent accidental writes.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *SPITool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if runtime.GOOS != "linux" {
		return ErrorResult("SPI is only supported on Linux. This tool requires /dev/spidev* device files.")
	}

	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "list":
		return t.list()
	case "transfer":
		return t.transfer(args)
	case "read":
		return t.readDevice(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s (valid: list, transfer, read)", action))
	}
}

// list finds available SPI devices by globbing /dev/spidev*
func (t *SPITool) list() *ToolResult {
	matches, err := filepath.Glob("/dev/spidev*")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to scan for SPI devices: %v", err))
	}

	if len(matches) == 0 {
		return SilentResult("No SPI devices found. You may need to:\n1. Enable SPI in device tree\n2. Configure pinmux for your board (see hardware skill)\n3. Check that spidev module is loaded")
	}

	type devInfo struct {
		Path   string `json:"path"`
		Device string `json:"device"`
	}

	devices := make([]devInfo, 0, len(matches))
	re := regexp.MustCompile(`/dev/spidev(\d+\.\d+)`)
	for _, m := range matches {
		if sub := re.FindStringSubmatch(m); sub != nil {
			devices = append(devices, devInfo{Path: m, Device: sub[1]})
		}
	}

	result, _ := json.MarshalIndent(devices, "", "  ")
	return SilentResult(fmt.Sprintf("Found %d SPI device(s):\n%s", len(devices), string(result)))
}

// parseSPIArgs extracts and validates common SPI parameters
func parseSPIArgs(args map[string]interface{}) (device string, speed uint32, mode uint8, bits uint8, errMsg string) {
	dev, ok := args["device"].(string)
	if !ok || dev == "" {
		return "", 0, 0, 0, "device is required (e.g. \"2.0\" for /dev/spidev2.0)"
	}
	matched, _ := regexp.MatchString(`^\d+\.\d+$`, dev)
	if !matched {
		return "", 0, 0, 0, "invalid device identifier: must be in format \"X.Y\" (e.g. \"2.0\")"
	}

	speed = 1000000 // default 1 MHz
	if s, ok := args["speed"].(float64); ok {
		if s < 1 || s > 125000000 {
			return "", 0, 0, 0, "speed must be between 1 Hz and 125 MHz"
		}
		speed = uint32(s)
	}

	mode = 0
	if m, ok := args["mode"].(float64); ok {
		if int(m) < 0 || int(m) > 3 {
			return "", 0, 0, 0, "mode must be 0-3"
		}
		mode = uint8(m)
	}

	bits = 8
	if b, ok := args["bits"].(float64); ok {
		if int(b) < 1 || int(b) > 32 {
			return "", 0, 0, 0, "bits must be between 1 and 32"
		}
		bits = uint8(b)
	}

	return dev, speed, mode, bits, ""
}
