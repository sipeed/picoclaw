package hardwaretools

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseSerialConfig(t *testing.T) {
	port := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		port = "COM3"
	}

	cfg, errResult := parseSerialConfig(map[string]any{
		"port":      port,
		"baud":      float64(9600),
		"data_bits": float64(7),
		"parity":    "even",
		"stop_bits": float64(2),
	})
	if errResult != nil {
		t.Fatalf("parseSerialConfig() unexpected error = %v", errResult.ForLLM)
	}

	wantPort := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		wantPort = `\\.\COM3`
	}
	if cfg.Port != wantPort || cfg.Baud != 9600 || cfg.DataBits != 7 || cfg.Parity != "even" || cfg.StopBits != 2 {
		t.Fatalf("parseSerialConfig() = %#v", cfg)
	}
}

func TestParseSerialConfigRejectsInvalidParity(t *testing.T) {
	port := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		port = "COM3"
	}

	_, errResult := parseSerialConfig(map[string]any{
		"port":   port,
		"parity": "mark",
	})
	if errResult == nil {
		t.Fatal("expected invalid parity to fail")
	}
}

func TestParseSerialConfigRejectsUnsupportedUnixBaud(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix baud validation only applies on Unix platforms")
	}

	_, errResult := parseSerialConfig(map[string]any{
		"port": "/dev/ttyUSB0",
		"baud": float64(460800),
	})
	if errResult == nil {
		t.Fatal("expected unsupported Unix baud rate to fail")
	}
}

func TestParseSerialWritePayloadRejectsFractionalBytes(t *testing.T) {
	_, errResult := parseSerialWritePayload(map[string]any{
		"data": []any{65.9},
	})
	if errResult == nil {
		t.Fatal("expected fractional byte value to fail")
	}
}

func TestValidateSerialBaud(t *testing.T) {
	tests := []struct {
		name    string
		baud    int
		wantErr bool
	}{
		{name: "default-supported", baud: 115200},
		{name: "max-unix-supported", baud: 230400},
		{name: "too-low", baud: 49, wantErr: true},
		{name: "too-high", baud: 4000001, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSerialBaud(tt.baud)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSerialBaud(%d) error = %v, wantErr %v", tt.baud, err, tt.wantErr)
			}
		})
	}
}

func TestSerialReadCanceledBeforeOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	port := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		port = "COM3"
	}

	_, err := serialRead(
		ctx,
		serialConfig{Port: port, Baud: 115200, DataBits: 8, Parity: "none", StopBits: 1},
		1,
		time.Second,
	)
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("serialRead() error = %v, want context canceled", err)
	}
}

func TestSerialWriteCanceledBeforeOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	port := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		port = "COM3"
	}

	_, err := serialWrite(
		ctx,
		serialConfig{Port: port, Baud: 115200, DataBits: 8, Parity: "none", StopBits: 1},
		[]byte("AT"),
		time.Second,
	)
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("serialWrite() error = %v, want context canceled", err)
	}
}

func TestParseSerialConfigRejectsUnsafePortPaths(t *testing.T) {
	tests := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		`C:\temp\device.txt`,
		`\\.\C:\temp\device.txt`,
	}

	for _, port := range tests {
		t.Run(strings.ReplaceAll(port, "/", "_"), func(t *testing.T) {
			_, errResult := parseSerialConfig(map[string]any{
				"port": port,
			})
			if errResult == nil {
				t.Fatalf("expected unsafe port %q to be rejected", port)
			}
		})
	}
}

func TestNormalizeUnixSerialPath(t *testing.T) {
	tests := []struct {
		port string
		want string
	}{
		{port: "ttyUSB0", want: "/dev/ttyUSB0"},
		{port: "/dev/ttyACM0", want: "/dev/ttyACM0"},
		{port: "/dev/cu.usbserial-0001", want: "/dev/cu.usbserial-0001"},
	}

	for _, tt := range tests {
		got, err := normalizeUnixSerialPath(tt.port)
		if err != nil {
			t.Fatalf("normalizeUnixSerialPath(%q) unexpected error = %v", tt.port, err)
		}
		if got != tt.want {
			t.Fatalf("normalizeUnixSerialPath(%q) = %q, want %q", tt.port, got, tt.want)
		}
	}
}

func TestNormalizeUnixSerialPathRejectsInvalidNames(t *testing.T) {
	tests := []string{
		"",
		"ttyUSB0/../../passwd",
		"/dev/../../etc/passwd",
		"/tmp/ttyUSB0",
		"ttyUSB",
		"COM3",
	}

	for _, port := range tests {
		t.Run(strings.ReplaceAll(port, "/", "_"), func(t *testing.T) {
			if _, err := normalizeUnixSerialPath(port); err == nil {
				t.Fatalf("expected %q to be rejected", port)
			}
		})
	}
}

func TestNormalizeWindowsSerialPath(t *testing.T) {
	tests := []struct {
		port string
		want string
	}{
		{port: "COM3", want: `\\.\COM3`},
		{port: "com12", want: `\\.\COM12`},
		{port: `\\.\COM7`, want: `\\.\COM7`},
	}

	for _, tt := range tests {
		got, err := normalizeWindowsSerialPath(tt.port)
		if err != nil {
			t.Fatalf("normalizeWindowsSerialPath(%q) unexpected error = %v", tt.port, err)
		}
		if got != tt.want {
			t.Fatalf("normalizeWindowsSerialPath(%q) = %q, want %q", tt.port, got, tt.want)
		}
	}
}

func TestNormalizeWindowsSerialPathRejectsInvalidNames(t *testing.T) {
	tests := []string{
		"",
		"COM0",
		"COM",
		"/dev/ttyUSB0",
		`C:\temp\device.txt`,
		`\\.\C:\temp\device.txt`,
		`\\server\share\COM3`,
	}

	for _, port := range tests {
		t.Run(strings.ReplaceAll(strings.ReplaceAll(port, `\`, "_"), "/", "_"), func(t *testing.T) {
			if _, err := normalizeWindowsSerialPath(port); err == nil {
				t.Fatalf("expected %q to be rejected", port)
			}
		})
	}
}

func TestParseSerialTimeout(t *testing.T) {
	timeout, errResult := parseSerialTimeout(map[string]any{
		"timeout_ms": float64(2500),
	})
	if errResult != nil {
		t.Fatalf("parseSerialTimeout() unexpected error = %v", errResult.ForLLM)
	}
	if timeout != 2500*time.Millisecond {
		t.Fatalf("timeout = %v, want 2500ms", timeout)
	}
}

func TestParseSerialWritePayloadSupportsText(t *testing.T) {
	data, errResult := parseSerialWritePayload(map[string]any{
		"text": "AT\r\n",
	})
	if errResult != nil {
		t.Fatalf("parseSerialWritePayload() unexpected error = %v", errResult.ForLLM)
	}
	if string(data) != "AT\r\n" {
		t.Fatalf("payload = %q, want %q", string(data), "AT\r\n")
	}
}

func TestParseSerialWritePayloadRejectsOutOfRangeByte(t *testing.T) {
	_, errResult := parseSerialWritePayload(map[string]any{
		"data": []any{float64(256)},
	})
	if errResult == nil {
		t.Fatal("expected payload validation failure")
	}
}
