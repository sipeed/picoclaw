package common

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// LoggingRoundTripper wraps an http.RoundTripper and logs HTTP requests and responses
// when PicoClaw is configured in DEBUG mode.
type LoggingRoundTripper struct {
	Proxied http.RoundTripper
}

func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if logger.GetLevel() <= logger.DEBUG {
		logger.DebugCF("http_client", "Req: "+req.Method+" "+req.URL.String(), nil)

		// Log Headers (redact sensitive)
		headers := make(map[string]string)
		for k, v := range req.Header {
			lowerK := strings.ToLower(k)
			if lowerK == "authorization" || lowerK == "api-key" || lowerK == "x-api-key" {
				headers[k] = "[REDACTED]"
			} else {
				headers[k] = strings.Join(v, ", ")
			}
		}
		logger.DebugCF("http_client", "Req Headers", map[string]any{"headers": headers})

		// Log Body
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				logger.DebugCF("http_client", "Req Body", map[string]any{"body": string(bodyBytes)})
			}
		}
	}

	res, err := lrt.Proxied.RoundTrip(req)

	if logger.GetLevel() <= logger.DEBUG && res != nil {
		logger.DebugCF("http_client", "Res Status: "+res.Status, nil)

		// Optional: Log Response Body
		if res.Body != nil {
			bodyBytes, err := io.ReadAll(res.Body)
			if err == nil {
				res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				// Truncate response body if too long to avoid flooding logs
				respStr := string(bodyBytes)
				if len(respStr) > 4000 {
					respStr = respStr[:4000] + "... [TRUNCATED]"
				}
				logger.DebugCF("http_client", "Res Body", map[string]any{"body": respStr})
			}
		}
	}

	return res, err
}
