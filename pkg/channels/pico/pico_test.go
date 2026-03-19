package pico

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jane/pkg/config"
)

func TestPicoChannel_Authenticate(t *testing.T) {
	tests := []struct {
		name            string
		token           string
		allowTokenQuery bool
		setupRequest    func(*http.Request)
		want            bool
	}{
		{
			name:            "empty token config",
			token:           "",
			allowTokenQuery: false,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer valid-token")
			},
			want: false,
		},
		{
			name:            "valid authorization header",
			token:           "valid-token",
			allowTokenQuery: false,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer valid-token")
			},
			want: true,
		},
		{
			name:            "invalid authorization header",
			token:           "valid-token",
			allowTokenQuery: false,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid-token")
			},
			want: false,
		},
		{
			name:            "missing authorization header prefix",
			token:           "valid-token",
			allowTokenQuery: false,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "valid-token")
			},
			want: false,
		},
		{
			name:            "valid query parameter token - allowed",
			token:           "valid-token",
			allowTokenQuery: true,
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("token", "valid-token")
				r.URL.RawQuery = q.Encode()
			},
			want: true,
		},
		{
			name:            "invalid query parameter token - allowed",
			token:           "valid-token",
			allowTokenQuery: true,
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("token", "invalid-token")
				r.URL.RawQuery = q.Encode()
			},
			want: false,
		},
		{
			name:            "valid query parameter token - not allowed",
			token:           "valid-token",
			allowTokenQuery: false,
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("token", "valid-token")
				r.URL.RawQuery = q.Encode()
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PicoChannel{
				config: config.PicoConfig{
					Token:           tt.token,
					AllowTokenQuery: tt.allowTokenQuery,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			tt.setupRequest(req)

			if got := c.authenticate(req); got != tt.want {
				t.Errorf("authenticate() = %v, want %v", got, tt.want)
			}
		})
	}
}
