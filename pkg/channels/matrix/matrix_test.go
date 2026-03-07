package matrix

import (
	"context"
	"testing"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func TestMatrixLocalpartMentionRegexp(t *testing.T) {
	re := localpartMentionRegexp("picoclaw")

	cases := []struct {
		text string
		want bool
	}{
		{text: "@picoclaw hello", want: true},
		{text: "hi @picoclaw:matrix.org", want: true},
		{text: "欢迎一下picoclaw小龙虾", want: false}, // historical false-positive case in PR #356
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
				Body: "欢迎一下picoclaw小龙虾",
			},
			want: false,
		},
	}

	for _, tc := range cases {
		if got := ch.isBotMentioned(&tc.msg); got != tc.want {
			t.Fatalf("%s: got=%v want=%v", tc.name, got, tc.want)
		}
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
