package config

import (
	"net/http"
	"strings"
)

// HTTPUserAgent is the default User-Agent for outbound HTTP requests (e.g. "PicoClaw/0.2.4").
func HTTPUserAgent() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		v = "dev"
	}
	return "PicoClaw/" + v
}

// userAgentTransport wraps an http.RoundTripper and sets User-Agent to HTTPUserAgent when
// the request does not already specify one.
type userAgentTransport struct {
	base http.RoundTripper
	ua   string
}

// WrapTransportUserAgent wraps base so requests without User-Agent receive HTTPUserAgent().
// If base is nil, http.DefaultTransport is used.
func WrapTransportUserAgent(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &userAgentTransport{base: base, ua: HTTPUserAgent()}
}

// UnwrapUserAgent returns the inner RoundTripper if rt was produced by WrapTransportUserAgent; otherwise rt.
func UnwrapUserAgent(rt http.RoundTripper) http.RoundTripper {
	if t, ok := rt.(*userAgentTransport); ok {
		return t.base
	}
	return rt
}

// RoundTrip implements http.RoundTripper.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") != "" {
		return t.base.RoundTrip(req)
	}
	r2 := req.Clone(req.Context())
	r2.Header.Set("User-Agent", t.ua)
	return t.base.RoundTrip(r2)
}
