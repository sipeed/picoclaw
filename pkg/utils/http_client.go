package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CreateHTTPClient creates an HTTP client with optional proxy support.
// If proxyURL is empty, it uses the system environment proxy settings.
// Supported proxy schemes: http, https, socks5, socks5h.
func CreateHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 15 * time.Second,
		},
	}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		scheme := strings.ToLower(proxy.Scheme)
		switch scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, fmt.Errorf(
				"unsupported proxy scheme %q (supported: http, https, socks5, socks5h)",
				proxy.Scheme,
			)
		}
		if proxy.Host == "" {
			return nil, fmt.Errorf("invalid proxy URL: missing host")
		}
		client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxy)
	} else {
		client.Transport.(*http.Transport).Proxy = http.ProxyFromEnvironment
	}

	return client, nil
}
