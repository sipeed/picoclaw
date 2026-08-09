package weixin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

func TestDownloadRemoteMediaToTemp_BlocksPrivateRedirect(t *testing.T) {
	t.Parallel()

	privateHit := false
	private := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateHit = true
		_, _ = w.Write([]byte("SECRET"))
	}))
	t.Cleanup(private.Close)

	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, private.URL+"/secret", http.StatusFound)
	}))
	t.Cleanup(public.Close)

	client, err := utils.CreateSafeHTTPClient(utils.SafeHTTPClientOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateSafeHTTPClient: %v", err)
	}

	ch := &WeixinChannel{
		mediaClient: client,
		api:         &ApiClient{HttpClient: &http.Client{}},
	}

	_, _, _, err = ch.downloadRemoteMediaToTemp(context.Background(), public.URL, "file.bin")
	if err == nil {
		t.Fatal("expected downloadRemoteMediaToTemp to reject redirect to private host")
	}
	if privateHit {
		t.Fatal("private target was reached via redirect")
	}
	if !strings.Contains(err.Error(), "private or local") &&
		!strings.Contains(err.Error(), "blocked private") {
		t.Fatalf("unexpected error: %v", err)
	}
}
