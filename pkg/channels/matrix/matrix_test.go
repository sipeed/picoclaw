package matrix

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestMatrixLocalpartMentionRegexp(t *testing.T) {
	re := localpartMentionRegexp("picoclaw")

	cases := []struct {
		text string
		want bool
	}{
		{text: "@picoclaw hello", want: true},
		{text: "hi @picoclaw:matrix.org", want: true},
		{
			text: "\u6b22\u8fce\u4e00\u4e0bpicoclaw\u5c0f\u9f99\u867e",
			want: false, // historical false-positive case in PR #356
		},
		{text: "mail test@example.com", want: false},
	}

	for _, tc := range cases {
		if got := re.MatchString(tc.text); got != tc.want {
			t.Fatalf("text=%q match=%v want=%v", tc.text, got, tc.want)
		}
	}
}

func TestStripUserMention(t *testing.T) {
	userID := id.UserID("@picoclaw:matrix.org")

	cases := []struct {
		in   string
		want string
	}{
		{in: "@picoclaw:matrix.org hello", want: "hello"},
		{in: "@picoclaw, hello", want: "hello"},
		{in: "no mention here", want: "no mention here"},
	}

	for _, tc := range cases {
		if got := stripUserMention(tc.in, userID); got != tc.want {
			t.Fatalf("stripUserMention(%q)=%q want=%q", tc.in, got, tc.want)
		}
	}
}

func TestIsBotMentioned(t *testing.T) {
	ch := &MatrixChannel{
		client: &mautrix.Client{
			UserID: id.UserID("@picoclaw:matrix.org"),
		},
	}

	cases := []struct {
		name string
		msg  event.MessageEventContent
		want bool
	}{
		{
			name: "mentions field",
			msg: event.MessageEventContent{
				Body: "hello",
				Mentions: &event.Mentions{
					UserIDs: []id.UserID{id.UserID("@picoclaw:matrix.org")},
				},
			},
			want: true,
		},
		{
			name: "full user id in body",
			msg: event.MessageEventContent{
				Body: "@picoclaw:matrix.org hello",
			},
			want: true,
		},
		{
			name: "localpart with at sign",
			msg: event.MessageEventContent{
				Body: "@picoclaw hello",
			},
			want: true,
		},
		{
			name: "localpart without at sign should not match",
			msg: event.MessageEventContent{
				Body: "\u6b22\u8fce\u4e00\u4e0bpicoclaw\u5c0f\u9f99\u867e",
			},
			want: false,
		},
		{
			name: "formatted mention href matrix.to plain",
			msg: event.MessageEventContent{
				Body:          "hello bot",
				FormattedBody: `<a href="https://matrix.to/#/@picoclaw:matrix.org">PicoClaw</a> hello`,
			},
			want: true,
		},
		{
			name: "formatted mention href matrix.to encoded",
			msg: event.MessageEventContent{
				Body:          "hello bot",
				FormattedBody: `<a href="https://matrix.to/#/%40picoclaw%3Amatrix.org">PicoClaw</a> hello`,
			},
			want: true,
		},
	}

	for _, tc := range cases {
		if got := ch.isBotMentioned(&tc.msg); got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.name, got, tc.want)
		}
	}
}

func TestRoomKindCache_ExpiresEntries(t *testing.T) {
	cache := newRoomKindCache(4, 5*time.Second)
	now := time.Unix(100, 0)
	cache.set("!room:matrix.org", true, now)

	if got, ok := cache.get("!room:matrix.org", now.Add(2*time.Second)); !ok || !got {
		t.Fatalf("expected cached group room before ttl, got ok=%v group=%v", ok, got)
	}

	if _, ok := cache.get("!room:matrix.org", now.Add(6*time.Second)); ok {
		t.Fatal("expected cache miss after ttl expiry")
	}
}

func TestRoomKindCache_EvictsOldestWhenFull(t *testing.T) {
	cache := newRoomKindCache(2, time.Minute)
	now := time.Unix(200, 0)

	cache.set("!room1:matrix.org", false, now)
	cache.set("!room2:matrix.org", false, now.Add(1*time.Second))
	cache.set("!room3:matrix.org", true, now.Add(2*time.Second))

	if _, ok := cache.get("!room1:matrix.org", now.Add(2*time.Second)); ok {
		t.Fatal("expected oldest cache entry to be evicted")
	}
	if got, ok := cache.get("!room2:matrix.org", now.Add(2*time.Second)); !ok || got {
		t.Fatalf("expected room2 to remain and be direct, got ok=%v group=%v", ok, got)
	}
	if got, ok := cache.get("!room3:matrix.org", now.Add(2*time.Second)); !ok || !got {
		t.Fatalf("expected room3 to remain and be group, got ok=%v group=%v", ok, got)
	}
}

func TestMatrixMediaTempDir(t *testing.T) {
	dir, err := matrixMediaTempDir()
	if err != nil {
		t.Fatalf("matrixMediaTempDir failed: %v", err)
	}
	if filepath.Base(dir) != media.TempDirName {
		t.Fatalf("unexpected media dir base: %q", filepath.Base(dir))
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("media dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory, got mode=%v", info.Mode())
	}
}

func TestMatrixMediaExt(t *testing.T) {
	if got := matrixMediaExt("photo.png", "", "image"); got != ".png" {
		t.Fatalf("filename extension mismatch: got=%q", got)
	}
	if got := matrixMediaExt("", "image/webp", "image"); got != ".webp" {
		t.Fatalf("content-type extension mismatch: got=%q", got)
	}
	if got := matrixMediaExt("", "", "image"); got != ".jpg" {
		t.Fatalf("default image extension mismatch: got=%q", got)
	}
	if got := matrixMediaExt("", "", "audio"); got != ".ogg" {
		t.Fatalf("default audio extension mismatch: got=%q", got)
	}
	if got := matrixMediaExt("", "", "video"); got != ".mp4" {
		t.Fatalf("default video extension mismatch: got=%q", got)
	}
	if got := matrixMediaExt("", "", "file"); got != ".bin" {
		t.Fatalf("default file extension mismatch: got=%q", got)
	}
}

func TestDownloadMedia_WritesResponseToTempFile(t *testing.T) {
	const wantBody = "matrix-media-payload"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/_matrix/client/v1/media/download/matrix.test/abc123") {
			t.Fatalf("unexpected download path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(wantBody))
	}))
	defer server.Close()

	client, err := mautrix.NewClient(server.URL, id.UserID("@picoclaw:matrix.test"), "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ch := &MatrixChannel{client: client}
	msg := &event.MessageEventContent{
		MsgType: event.MsgImage,
		Body:    "image.png",
		URL:     id.ContentURIString("mxc://matrix.test/abc123"),
		Info:    &event.FileInfo{MimeType: "image/png"},
	}

	path, err := ch.downloadMedia(context.Background(), msg, "image")
	if err != nil {
		t.Fatalf("downloadMedia: %v", err)
	}
	defer os.Remove(path)

	if ext := filepath.Ext(path); ext != ".png" {
		t.Fatalf("temp file extension=%q want=.png", ext)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != wantBody {
		t.Fatalf("file contents=%q want=%q", string(got), wantBody)
	}
}

func TestExtractInboundContent_ImageNoURLFallback(t *testing.T) {
	ch := &MatrixChannel{}
	msg := &event.MessageEventContent{
		MsgType: event.MsgImage,
		Body:    "test.png",
	}

	content, mediaRefs, ok := ch.extractInboundContent(context.Background(), msg, "matrix:room:event")
	if !ok {
		t.Fatal("expected ok for image fallback")
	}
	if content != "[image: test.png]" {
		t.Fatalf("unexpected content: %q", content)
	}
	if len(mediaRefs) != 0 {
		t.Fatalf("expected no media refs, got %d", len(mediaRefs))
	}
}

func TestExtractInboundContent_AudioNoURLFallback(t *testing.T) {
	ch := &MatrixChannel{}
	msg := &event.MessageEventContent{
		MsgType:  event.MsgAudio,
		FileName: "voice.ogg",
		Body:     "please transcribe",
	}

	content, mediaRefs, ok := ch.extractInboundContent(context.Background(), msg, "matrix:room:event")
	if !ok {
		t.Fatal("expected ok for audio fallback")
	}
	if content != "please transcribe\n[audio: voice.ogg]" {
		t.Fatalf("unexpected content: %q", content)
	}
	if len(mediaRefs) != 0 {
		t.Fatalf("expected no media refs, got %d", len(mediaRefs))
	}
}

func TestMatrixOutboundMsgType(t *testing.T) {
	cases := []struct {
		name        string
		partType    string
		filename    string
		contentType string
		want        event.MessageType
	}{
		{name: "explicit image", partType: "image", want: event.MsgImage},
		{name: "explicit audio", partType: "audio", want: event.MsgAudio},
		{name: "mime fallback video", contentType: "video/mp4", want: event.MsgVideo},
		{name: "extension fallback audio", filename: "voice.ogg", want: event.MsgAudio},
		{name: "unknown defaults file", filename: "report.txt", want: event.MsgFile},
	}

	for _, tc := range cases {
		if got := matrixOutboundMsgType(tc.partType, tc.filename, tc.contentType); got != tc.want {
			t.Fatalf("%s: got=%q want=%q", tc.name, got, tc.want)
		}
	}
}

func TestMatrixOutboundContent(t *testing.T) {
	content := matrixOutboundContent(
		"please review",
		"voice.ogg",
		event.MsgAudio,
		"audio/ogg",
		1234,
		id.ContentURIString("mxc://matrix.org/abc"),
	)
	if content.Body != "please review" {
		t.Fatalf("unexpected body: %q", content.Body)
	}
	if content.FileName != "voice.ogg" {
		t.Fatalf("unexpected filename: %q", content.FileName)
	}
	if content.Info == nil || content.Info.MimeType != "audio/ogg" {
		t.Fatalf("unexpected content type: %+v", content.Info)
	}
	if content.Info == nil || content.Info.Size != 1234 {
		t.Fatalf("unexpected size: %+v", content.Info)
	}

	noCaption := matrixOutboundContent(
		"",
		"image.png",
		event.MsgImage,
		"image/png",
		0,
		id.ContentURIString("mxc://matrix.org/def"),
	)
	if noCaption.Body != "image.png" {
		t.Fatalf("unexpected fallback body: %q", noCaption.Body)
	}
}

func TestMarkdownToHTML(t *testing.T) {
	cases := []struct {
		name     string
		md       string
		rendered string
	}{
		{
			name:     "paragraph",
			md:       "just **some** text with _custom_ formatting and `inline` code",
			rendered: "<p>just <strong>some</strong> text with <em>custom</em> formatting and <code>inline</code> code</p>",
		},
		{
			name:     "heading",
			md:       "### Title",
			rendered: `<h3>Title</h3>`,
		},
		{
			name:     "fenced code block",
			md:       "```\nfoo()\n```",
			rendered: "<pre><code>foo()\n</code></pre>",
		},
		{
			name: "loose list",
			md:   "- Item one\n\n- Item two\n",
			rendered: `<ul>
<li><p>Item one</p></li>

<li><p>Item two</p></li>
</ul>`,
		},
		{
			name: "tight list",
			md:   "- Alpha\n- Beta\n",
			rendered: `<ul>
<li>Alpha</li>
<li>Beta</li>
</ul>`,
		},
		{
			name: "list item with nested sublist",
			md:   "1. Steps overview:\n\n   - Step A\n   - Step B\n",
			rendered: `<ol>
<li><p>Steps overview:</p>

<ul>
<li>Step A</li>
<li>Step B</li>
</ul></li>
</ol>`,
		},
		{
			// Definition list syntax is not enabled; the term and definition are
			// rendered as a plain paragraph rather than <dl>/<dt>/<dd> elements.
			name:     "definition list syntax renders as plain paragraph",
			md:       "Term\n:   Definition of the term.\n",
			rendered: "<p>Term\n:   Definition of the term.</p>",
		},
		{
			name: "comprehensive document with headings, paragraphs, list, and code block",
			md:   "# Overview\n\nThis is a sample document designed to demonstrate various Markdown elements in a single block of text.\n\nThe first paragraph introduces the concept of structured data.\n\n## Details\n\nThe following is a list:\n\n*   First\n*   Second\n*   Third\n\nThe second paragraph focuses on details. Below is a generic code snippet:\n\n```python\ndef calculate_area(radius):\n    import math\n    return math.pi * (radius ** 2)\n```\n\nThis concludes the generic sample text.\n",
			rendered: `<h1>Overview</h1>

<p>This is a sample document designed to demonstrate various Markdown elements in a single block of text.</p>

<p>The first paragraph introduces the concept of structured data.</p>

<h2>Details</h2>

<p>The following is a list:</p>

<ul>
<li>First</li>
<li>Second</li>
<li>Third</li>
</ul>

<p>The second paragraph focuses on details. Below is a generic code snippet:</p>

<pre><code class="language-python">def calculate_area(radius):
    import math
    return math.pi * (radius ** 2)
</code></pre>

<p>This concludes the generic sample text.</p>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := markdownToHTML(tc.md); got != tc.rendered {
				t.Fatalf("markdownToHTML(%q)\n got: %q\nwant: %q", tc.md, got, tc.rendered)
			}
		})
	}
}

func TestMessageContent(t *testing.T) {
	richtext := &MatrixChannel{config: &config.MatrixSettings{MessageFormat: "richtext"}}
	plain := &MatrixChannel{config: &config.MatrixSettings{MessageFormat: "plain"}}
	defaultt := &MatrixChannel{config: &config.MatrixSettings{}}

	for _, c := range []*MatrixChannel{richtext, defaultt} {
		mc := c.messageContent("**hi**")
		if mc.Format != event.FormatHTML {
			t.Errorf("format %q: expected FormatHTML, got %q", c.config.MessageFormat, mc.Format)
		}
		if !strings.Contains(mc.FormattedBody, "<strong>hi</strong>") {
			t.Errorf("format %q: FormattedBody %q missing <strong>", c.config.MessageFormat, mc.FormattedBody)
		}
		if mc.Body != "**hi**" {
			t.Errorf("format %q: Body should remain plain, got %q", c.config.MessageFormat, mc.Body)
		}
	}

	mc := plain.messageContent("**hi**")
	if mc.Format != "" || mc.FormattedBody != "" {
		t.Errorf("plain: expected no formatting, got format=%q formattedBody=%q", mc.Format, mc.FormattedBody)
	}
}
