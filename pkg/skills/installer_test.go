package skills

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoRequestWithRetry(t *testing.T) {
	testcases := []struct {
		name           string
		serverBehavior func(*httptest.Server) int
		wantSuccess    bool
		wantAttempts   int
	}{
		{
			name: "success-on-first-attempt",
			serverBehavior: func(server *httptest.Server) int {
				return 0
			},
			wantSuccess:  true,
			wantAttempts: 1,
		},
		{
			name: "success-on-second-attempt",
			serverBehavior: func(server *httptest.Server) int {
				return 1
			},
			wantSuccess:  true,
			wantAttempts: 2,
		},
		{
			name: "success-on-third-attempt",
			serverBehavior: func(server *httptest.Server) int {
				return 2
			},
			wantSuccess:  true,
			wantAttempts: 3,
		},
		{
			name: "fail-all-attempts",
			serverBehavior: func(server *httptest.Server) int {
				return 4
			},
			wantSuccess:  false,
			wantAttempts: 3,
		},
		{
			name: "non-ok-status-code",
			serverBehavior: func(server *httptest.Server) int {
				return 4
			},
			wantSuccess:  false,
			wantAttempts: 3,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts <= tc.serverBehavior(nil) {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}))

			defer server.Close()

			client := &http.Client{Timeout: 5 * time.Second}
			req, err := http.NewRequest("GET", server.URL, nil)
			assert.NoError(t, err)

			resp, err := doRequestWithRetry(client, req)

			if tc.wantSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			} else {
				assert.NotNil(t, resp)
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				resp.Body.Close()
			}

			assert.Equal(t, tc.wantAttempts, attempts)
		})
	}
}

func TestDoRequestWithRetry_Delay(t *testing.T) {
	var start time.Time
	delays := []time.Duration{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(delays) == 0 {
			delays = append(delays, 0)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(delays) == 1 {
			start = time.Now()
			delays = append(delays, 0)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(delays) == 2 {
			elapsed := time.Since(start)
			delays = append(delays, elapsed)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(t, err)

	resp, err := doRequestWithRetry(client, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	assert.GreaterOrEqual(t, delays[2], time.Second)
}
