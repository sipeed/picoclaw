package utils

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoRequestWithRetry(t *testing.T) {
	retryDelayUnit = time.Millisecond
	t.Cleanup(func() { retryDelayUnit = time.Second })

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
			name: "fail-all-attempts",
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

			t.Cleanup(func() {
				server.Close()
			})

			client := &http.Client{Timeout: 5 * time.Second}
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := DoRequestWithRetry(client, req)

			if tc.wantSuccess {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			} else {
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				resp.Body.Close()
			}

			assert.Equal(t, tc.wantAttempts, attempts)
		})
	}
}

func TestDoRequestWithRetry_RetryAfter429Honored(t *testing.T) {
	retryDelayUnit = 10 * time.Millisecond
	t.Cleanup(func() { retryDelayUnit = time.Second })

	attempts := 0
	var firstAttemptAt time.Time
	var secondAttemptAt time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			firstAttemptAt = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if attempts == 2 {
			secondAttemptAt = time.Now()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := DoRequestWithRetry(client, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	require.Equal(t, 2, attempts)

	assert.GreaterOrEqual(t, secondAttemptAt.Sub(firstAttemptAt), 900*time.Millisecond)
}

func TestDoRequestWithRetry_RetryAfter429InvalidFallsBack(t *testing.T) {
	retryDelayUnit = 50 * time.Millisecond
	t.Cleanup(func() { retryDelayUnit = time.Second })

	attempts := 0
	var firstAttemptAt time.Time
	var secondAttemptAt time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			firstAttemptAt = time.Now()
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if attempts == 2 {
			secondAttemptAt = time.Now()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := DoRequestWithRetry(client, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	require.Equal(t, 2, attempts)

	assert.GreaterOrEqual(t, secondAttemptAt.Sub(firstAttemptAt), 45*time.Millisecond)
	assert.Less(t, secondAttemptAt.Sub(firstAttemptAt), 500*time.Millisecond)
}

func TestDoRequestWithRetry_ContextCancel(t *testing.T) {
	// Use a long retry delay so cancellation always hits during sleepWithCtx.
	retryDelayUnit = 10 * time.Second
	t.Cleanup(func() { retryDelayUnit = time.Second })

	bodyClosed := false
	firstRoundTripDone := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 30 * time.Second
	client.Transport = &bodyCloseTracker{
		rt:      client.Transport,
		onClose: func() { bodyClosed = true },
		// Signal after the first round-trip response is fully constructed on the client side.
		onRoundTrip: func() {
			select {
			case firstRoundTripDone <- struct{}{}:
			default:
			}
		},
		trackURL: server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel the context after the first round-trip completes on the client side.
	// This ensures client.Do has returned a valid resp (with body) and the retry
	// loop is about to enter sleepWithCtx, where the cancel will be detected.
	go func() {
		<-firstRoundTripDone
		cancel()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := DoRequestWithRetry(client, req)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err, "expected error from context cancellation")
	assert.Nil(t, resp, "expected nil response when context is canceled")
	assert.True(t, bodyClosed, "expected resp.Body to be closed on context cancellation")
}

// bodyCloseTracker wraps an http.RoundTripper and records when response bodies are closed.
type bodyCloseTracker struct {
	rt          http.RoundTripper
	onClose     func()
	onRoundTrip func() // called after each successful round-trip
	trackURL    string
}

func (t *bodyCloseTracker) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if strings.HasPrefix(req.URL.String(), t.trackURL) {
		resp.Body = &closeNotifier{ReadCloser: resp.Body, onClose: t.onClose}
		if t.onRoundTrip != nil {
			t.onRoundTrip()
		}
	}
	return resp, nil
}

// closeNotifier wraps an io.ReadCloser to detect Close calls.
type closeNotifier struct {
	io.ReadCloser
	onClose func()
}

func (c *closeNotifier) Close() error {
	c.onClose()
	return c.ReadCloser.Close()
}

func TestDoRequestWithRetry_Delay(t *testing.T) {
	retryDelayUnit = time.Millisecond
	t.Cleanup(func() { retryDelayUnit = time.Second })

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
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := DoRequestWithRetry(client, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	assert.GreaterOrEqual(t, delays[2], time.Millisecond)
}

func TestRetryDelayForAttempt_DateRetryAfterUsesResponseDateHeader(t *testing.T) {
	maxRetrySleepDuration = time.Minute
	t.Cleanup(func() { maxRetrySleepDuration = time.Minute })

	serverDate := time.Date(2000, 1, 2, 15, 4, 5, 0, time.UTC)
	retryAfterAt := serverDate.Add(10 * time.Second)
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Retry-After": []string{retryAfterAt.Format(http.TimeFormat)},
			"Date":        []string{serverDate.Format(http.TimeFormat)},
		},
	}

	assert.Equal(t, 10*time.Second, retryDelayForAttempt(resp, 0))
}

func TestRetryDelayForAttempt_DateRetryAfterInvalidOrMissingDateFallsBackSafely(t *testing.T) {
	maxRetrySleepDuration = 30 * time.Second
	t.Cleanup(func() { maxRetrySleepDuration = time.Minute })

	retryAfterAt := time.Now().UTC().Add(3 * time.Second).Format(http.TimeFormat)
	testcases := []struct {
		name   string
		header http.Header
	}{
		{
			name: "invalid-date-header",
			header: http.Header{
				"Retry-After": []string{retryAfterAt},
				"Date":        []string{"invalid-date"},
			},
		},
		{
			name: "missing-date-header",
			header: http.Header{
				"Retry-After": []string{retryAfterAt},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     tc.header,
			}

			delay := retryDelayForAttempt(resp, 0)
			assert.Greater(t, delay, time.Duration(0))
			assert.GreaterOrEqual(t, delay, 1500*time.Millisecond)
			assert.LessOrEqual(t, delay, 5*time.Second)
		})
	}
}

func TestRetryDelayForAttempt_RetryAfterIsCapped(t *testing.T) {
	maxRetrySleepDuration = 2 * time.Second
	t.Cleanup(func() { maxRetrySleepDuration = time.Minute })

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Retry-After": []string{"999999"},
		},
	}

	assert.Equal(t, 2*time.Second, retryDelayForAttempt(resp, 0))
}

func TestRetryDelayForAttempt_RetryAfterNumericOverflowStillCaps(t *testing.T) {
	maxRetrySleepDuration = 2 * time.Second
	t.Cleanup(func() { maxRetrySleepDuration = time.Minute })

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Retry-After": []string{"9223372036854775807"},
		},
	}

	assert.Equal(t, 2*time.Second, retryDelayForAttempt(resp, 0))
}
