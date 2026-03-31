package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPUserAgent_format(t *testing.T) {
	ua := HTTPUserAgent()
	if !strings.HasPrefix(ua, "PicoClaw/") {
		t.Fatalf("want PicoClaw/ prefix, got %q", ua)
	}
	suffix := strings.TrimPrefix(ua, "PicoClaw/")
	if strings.TrimSpace(suffix) == "" {
		t.Fatal("empty version suffix")
	}
}

func TestWrapTransportUserAgent_SetsDefaultUA(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: WrapTransportUserAgent(http.DefaultTransport)}
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	want := HTTPUserAgent()
	if gotUA != want {
		t.Fatalf("User-Agent = %q, want %q", gotUA, want)
	}
}

func TestWrapTransportUserAgent_PreservesExplicitUA(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: WrapTransportUserAgent(http.DefaultTransport)}
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "custom-agent/1")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if gotUA != "custom-agent/1" {
		t.Fatalf("User-Agent = %q, want custom-agent/1", gotUA)
	}
}

func TestWrapTransportUserAgent_DoesNotMutateOriginalRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	rt := WrapTransportUserAgent(http.DefaultTransport)
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("User-Agent") != "" {
		t.Fatal("expected empty User-Agent on original request")
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if req.Header.Get("User-Agent") != "" {
		t.Fatal("RoundTrip must not mutate the original request's headers")
	}
}

func TestUnwrapUserAgent(t *testing.T) {
	inner := http.DefaultTransport
	wrapped := WrapTransportUserAgent(inner)
	if UnwrapUserAgent(wrapped) != inner {
		t.Fatal("UnwrapUserAgent should return inner transport")
	}
	if UnwrapUserAgent(inner) != inner {
		t.Fatal("UnwrapUserAgent on non-wrapped should return same")
	}
}
