package tailscale

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// tailscaleStatus is the subset of `tailscale status --json` we need.
type tailscaleStatus struct {
	Self struct {
		DNSName string `json:"DNSName"`
	} `json:"Self"`
}

// DetectHostname runs `tailscale status --json` and returns the machine's
// MagicDNS hostname (e.g. "machine.tailnet.ts.net"), with the trailing dot
// stripped.
func DetectHostname() (string, error) {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("tailscale status failed: %w", err)
	}

	hostname, err := ParseHostname(out)
	if err != nil {
		return "", err
	}
	return hostname, nil
}

// ParseHostname extracts the hostname from `tailscale status --json` output.
func ParseHostname(jsonData []byte) (string, error) {
	var status tailscaleStatus
	if err := json.Unmarshal(jsonData, &status); err != nil {
		return "", fmt.Errorf("failed to parse tailscale status: %w", err)
	}

	hostname := strings.TrimSuffix(status.Self.DNSName, ".")
	if hostname == "" {
		return "", fmt.Errorf("tailscale DNSName is empty")
	}
	return hostname, nil
}

// FetchCert runs `tailscale cert` to obtain a TLS certificate for the given
// hostname. Certificates are written to certDir. Returns paths to the cert
// and key files.
func FetchCert(hostname, certDir string) (certFile, keyFile string, err error) {
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return "", "", fmt.Errorf("failed to create cert dir: %w", err)
	}

	certFile = filepath.Join(certDir, hostname+".crt")
	keyFile = filepath.Join(certDir, hostname+".key")

	cmd := exec.Command("tailscale", "cert",
		"--cert-file="+certFile,
		"--key-file="+keyFile,
		hostname,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("tailscale cert failed: %w: %s", err, string(out))
	}

	return certFile, keyFile, nil
}
