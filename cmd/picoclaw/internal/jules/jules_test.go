package jules

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJulesCommands(t *testing.T) {
	os.Setenv("JULES_API_KEY", "test-api-key")
	defer os.Unsetenv("JULES_API_KEY")

	var lastReq *http.Request
	var lastBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = r
		var err error
		lastBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer ts.Close()

	originalBaseURL := julesBaseURL
	julesBaseURL = ts.URL
	defer func() { julesBaseURL = originalBaseURL }()

	t.Run("session create", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "create", "--prompt", "test prompt", "--source", "sources/test", "--title", "test title", "--branch", "main"})

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.Equal(t, "POST", lastReq.Method)
		assert.Equal(t, "/sessions", lastReq.URL.Path)
		assert.Equal(t, "test-api-key", lastReq.Header.Get("x-goog-api-key"))

		expectedBody := `{"prompt":"test prompt","sourceContext":{"githubRepoContext":{"startingBranch":"main"},"source":"sources/test"},"title":"test title"}`
		// unmarshal and marshal again to compare json ignoring key order
		assert.JSONEq(t, expectedBody, string(lastBody))
		assert.Contains(t, buf.String(), `"status": "ok"`)
	})

	t.Run("session list", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "list"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "GET", lastReq.Method)
		assert.Equal(t, "/sessions", lastReq.URL.Path)
		assert.Equal(t, "test-api-key", lastReq.Header.Get("x-goog-api-key"))
	})

	t.Run("session get", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "get", "123"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "GET", lastReq.Method)
		assert.Equal(t, "/sessions/123", lastReq.URL.Path)
	})

	t.Run("session delete", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "delete", "123"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "DELETE", lastReq.Method)
		assert.Equal(t, "/sessions/123", lastReq.URL.Path)
	})

	t.Run("session message", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "message", "123", "--message", "hello jules"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "POST", lastReq.Method)
		assert.Equal(t, "/sessions/123:sendMessage", lastReq.URL.Path)
		assert.JSONEq(t, `{"prompt":"hello jules"}`, string(lastBody))
	})

	t.Run("session approve", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"session", "approve", "123"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "POST", lastReq.Method)
		assert.Equal(t, "/sessions/123:approvePlan", lastReq.URL.Path)
		assert.JSONEq(t, `{}`, string(lastBody))
	})

	t.Run("activity list", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"activity", "list", "123"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "GET", lastReq.Method)
		assert.Equal(t, "/sessions/123/activities", lastReq.URL.Path)
	})

	t.Run("activity get", func(t *testing.T) {
		cmd := NewJulesCommand()
		cmd.SetArgs([]string{"activity", "get", "123", "act1"})

		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		assert.Equal(t, "GET", lastReq.Method)
		assert.Equal(t, "/sessions/123/activities/act1", lastReq.URL.Path)
	})
}
