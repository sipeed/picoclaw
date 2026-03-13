package tools

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// SetLocalNetOnly restricts curl/wget to localhost and RFC 1918 private addresses.
func (t *ExecTool) SetLocalNetOnly(v bool) {
	t.localNetOnly = v
}

// isCurlOrWget reports whether command is a curl or wget invocation.

func isCurlOrWget(command string) bool {
	fields := strings.Fields(command)

	if len(fields) == 0 {
		return false
	}

	base := filepath.Base(fields[0])

	return base == "curl" || base == "wget"
}

// checkCurlLocalNet validates that all http/https URLs in a curl/wget command

// target localhost or RFC 1918 private addresses.

// Returns an error message string, or empty string if the command is allowed.

func checkCurlLocalNet(command string) string {
	for _, token := range strings.Fields(command) {
		token = strings.Trim(token, "\"'")

		if !strings.HasPrefix(token, "http://") && !strings.HasPrefix(token, "https://") {
			continue
		}

		u, err := url.Parse(token)
		if err != nil {
			continue
		}

		host := u.Hostname()

		if !isLocalHost(host) {
			return fmt.Sprintf(

				"Command blocked by safety guard "+

					"(curl/wget is restricted to localhost and private network; %q is a public address)",

				host,
			)
		}
	}

	return ""
}

// isLocalHost reports whether host is localhost or a loopback/RFC 1918 private IP.

// DNS resolution is intentionally avoided to prevent DNS rebinding attacks.

func isLocalHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	if ip == nil {
		return false
	}

	return ip.IsLoopback() || ip.IsPrivate()
}
