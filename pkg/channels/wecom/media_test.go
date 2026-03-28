package wecom

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	basechannels "github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestStoreRemoteMedia_DetectsJPEGContentTypeFromBody(t *testing.T) {
	t.Parallel()

	const jpegBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////" +
		"//////////////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////" +
		"//////////////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAb/xAAVEQEBAAAAAAAAAAAAAAAAAAAABf/aAAwDAQACEAMQAAAB6A//xAAVEAEBAAAAAAAAAAAAAAAAAAAAEf/aAAgBAQABBQJf/8QAFBEBAAAAAAAAAAAAAAAAAAAAEP/aAAgBAwEBPwF//8QAFBEBAAAAAAAAAAAAAAAAAAAAEP/aAAgBAgEBPwF//8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQAGPwJf/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQABPyFf/9k="

	jpegData := decodeTestBase64(t, jpegBase64)
	store := media.NewFileMediaStore()
	ch := &WeComChannel{
		BaseChannel: basechannels.NewBaseChannel("wecom", nil, nil, nil),
		mediaClient: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
					Body:       io.NopCloser(bytes.NewReader(jpegData)),
				}, nil
			}),
		},
	}
	ch.SetMediaStore(store)

	ref, err := ch.storeRemoteMedia(context.Background(), "test-scope", "msg-1", "https://wecom.example/media", "", "")
	if err != nil {
		t.Fatalf("storeRemoteMedia returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.ReleaseAll("test-scope")
	})

	_, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("resolve media ref: %v", err)
	}
	if meta.ContentType != "image/jpeg" {
		t.Fatalf("expected image/jpeg content type, got %q", meta.ContentType)
	}
	if !strings.HasSuffix(meta.Filename, ".jpg") && !strings.HasSuffix(meta.Filename, ".jpeg") {
		t.Fatalf("expected jpeg filename, got %q", meta.Filename)
	}
}

func TestDetectWeComMediaMetadata_UsesFallbackExtensionWhenBodyUnknown(t *testing.T) {
	t.Parallel()

	filename, contentType := detectWeComMediaMetadata([]byte("not a real image"), "msg-2.pdf", "", "", "")
	if filename != "msg-2.pdf" {
		t.Fatalf("expected fallback filename to be preserved, got %q", filename)
	}
	if contentType != "application/pdf" {
		t.Fatalf("expected application/pdf from fallback extension, got %q", contentType)
	}
}

func TestStoreRemoteMedia_PreservesSuffixFromURL(t *testing.T) {
	t.Parallel()

	docxLikeData := []byte("PK\x03\x04fake office payload")
	store := media.NewFileMediaStore()
	ch := &WeComChannel{
		BaseChannel: basechannels.NewBaseChannel("wecom", nil, nil, nil),
		mediaClient: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
					Body:       io.NopCloser(bytes.NewReader(docxLikeData)),
				}, nil
			}),
		},
	}
	ch.SetMediaStore(store)

	ref, err := ch.storeRemoteMedia(
		context.Background(),
		"test-scope",
		"msg-docx",
		"https://wecom.example/media/report.docx?signature=1",
		"",
		".bin",
	)
	if err != nil {
		t.Fatalf("storeRemoteMedia returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.ReleaseAll("test-scope")
	})

	localPath, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("resolve media ref: %v", err)
	}
	if !strings.HasSuffix(meta.Filename, ".docx") {
		t.Fatalf("expected docx filename, got %q", meta.Filename)
	}
	if !strings.HasSuffix(strings.ToLower(localPath), ".docx") {
		t.Fatalf("expected docx temp path, got %q", localPath)
	}
}

func TestStoreRemoteMedia_PreservesSuffixFromContentDisposition(t *testing.T) {
	t.Parallel()

	pptxLikeData := []byte("PK\x03\x04fake office payload")
	store := media.NewFileMediaStore()
	ch := &WeComChannel{
		BaseChannel: basechannels.NewBaseChannel("wecom", nil, nil, nil),
		mediaClient: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":        []string{"application/octet-stream"},
						"Content-Disposition": []string{`attachment; filename="slides.pptx"`},
					},
					Body: io.NopCloser(bytes.NewReader(pptxLikeData)),
				}, nil
			}),
		},
	}
	ch.SetMediaStore(store)

	ref, err := ch.storeRemoteMedia(
		context.Background(),
		"test-scope",
		"msg-pptx",
		"https://wecom.example/media/download",
		"",
		".bin",
	)
	if err != nil {
		t.Fatalf("storeRemoteMedia returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.ReleaseAll("test-scope")
	})

	localPath, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("resolve media ref: %v", err)
	}
	if !strings.HasSuffix(meta.Filename, ".pptx") {
		t.Fatalf("expected pptx filename, got %q", meta.Filename)
	}
	if !strings.HasSuffix(strings.ToLower(localPath), ".pptx") {
		t.Fatalf("expected pptx temp path, got %q", localPath)
	}
}

func decodeTestBase64(t *testing.T, value string) []byte {
	t.Helper()

	data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(value)))
	if err != nil {
		t.Fatalf("decode base64 fixture: %v", err)
	}
	return data
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
