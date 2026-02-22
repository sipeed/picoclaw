package channels

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestEmailChannel_sanitizeHeaderValue(t *testing.T) {

	tests := []struct {
		name string
		s    string
		want string
	}{
		{name: "empty", s: "", want: ""},
		{name: "simple", s: "test", want: "test"},
		{name: "crlf", s: "test\r\n", want: "test"},
		{name: "lf", s: "test\n", want: "test"},
		{name: "cr", s: "test\r", want: "test"},
		{name: "crlf", s: "test\r\n", want: "test"},
		{name: "lfcr", s: "test\n\r", want: "test"},
		{name: "crlfcr", s: "test\r\n\r", want: "test"},
		{name: "lfcrlf", s: "test\n\r\n", want: "test"},
		{name: "crlfcrlf", s: "test\r\n\r\n", want: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeHeaderValue(tt.s); got != tt.want {
				t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestEmailChannel_parseFilenameFromDisposition(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{name: "inline", s: "inline; filename=\"pico.png\"", want: "pico.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFilenameFromDisposition(tt.s); got != tt.want {
				t.Errorf("parseFilenameFromDisposition(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}
func TestEmailChannel_decodeRFC2047Filename(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{name: "normal", s: "正常.png", want: "正常.png"},
		{name: "GB2312-Quoted-Printable", s: "=?GB2312?Q?gb2312=B2=E2=CA=D4=B2=E2=CA=D4.png?=", want: "gb2312测试测试.png"},
		{name: "GBK-Base64", s: "=?GBK?B?yfqzybLiytTNvMasLnBuZw==?=", want: "生成测试图片.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeRFC2047Filename(tt.s); got != tt.want {
				t.Errorf("decodeRFC2047Filename(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestEmailChannel_extractEmailBodyAndAttachments(t *testing.T) {
	c := &EmailChannel{
		config: config.EmailConfig{
			AttachmentDir: t.TempDir(),
		},
	}
	t.Run("nil message", func(t *testing.T) {
		content, paths := c.extractEmailBodyAndAttachments(nil)
		assert.Empty(t, content)
		assert.Nil(t, paths)
	})

	t.Run("plain text body", func(t *testing.T) {
		mimeBytes := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: Test\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello world")
		section := &imap.BodySectionName{}
		msg := &imap.Message{
			Uid:      1,
			Envelope: &imap.Envelope{Subject: "Test"},
			Body:     map[*imap.BodySectionName]imap.Literal{section: bytes.NewReader(mimeBytes)},
		}
		content, paths := c.extractEmailBodyAndAttachments(msg)
		assert.Contains(t, content, "Subject: Test")
		assert.Contains(t, content, "Hello world")
		assert.Empty(t, paths)
	})

	t.Run("html body", func(t *testing.T) {
		mimeBytes := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: Test\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<html><body><h1>Hello world</h1></body></html>")
		section := &imap.BodySectionName{}
		msg := &imap.Message{
			Uid:      1,
			Envelope: &imap.Envelope{Subject: "Test"},
			Body:     map[*imap.BodySectionName]imap.Literal{section: bytes.NewReader(mimeBytes)},
		}
		content, paths := c.extractEmailBodyAndAttachments(msg)
		assert.Contains(t, content, "Subject: Test")
		assert.Contains(t, content, "Hello world")
		assert.Empty(t, paths)
	})

	attachmentwithText := []byte(
		"From: a@b.com\r\n" +
			"To: c@d.com\r\n" +
			"Subject: test-file-and-body\r\n" +
			"Content-Type: multipart/mixed; boundary=\"outer\"\r\n" +
			"MIME-Version: 1.0\r\n" +
			"\r\n" +
			"--outer\r\n" +
			"Content-Type: multipart/alternative; boundary=\"alt\"\r\n" +
			"\r\n" +
			"--alt\r\n" +
			"Content-Type: text/plain; charset=GBK\r\n" +
			"Content-Transfer-Encoding: 7bit\r\n" +
			"\r\n" +
			"this body\r\n" +
			"--alt\r\n" +
			"Content-Type: text/html; charset=GBK\r\n" +
			"Content-Transfer-Encoding: 7bit\r\n" +
			"\r\n" +
			"<div>this body</div>\r\n" +
			"--alt--\r\n" +
			"\r\n" +
			"--outer\r\n" +
			"Content-Type: text/plain; name=test.txt\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"Content-Disposition: attachment; filename=\"test.txt\"\r\n" +
			"\r\n" +
			"VGVzdC0xMTEx\r\n" +
			"--outer--\r\n")
	t.Run("attachment and text body", func(t *testing.T) {
		mimeBytes := attachmentwithText
		section := &imap.BodySectionName{}
		msg := &imap.Message{
			Uid:      1,
			Envelope: &imap.Envelope{Subject: "Test"},
			Body:     map[*imap.BodySectionName]imap.Literal{section: bytes.NewReader(mimeBytes)},
		}
		content, paths := c.extractEmailBodyAndAttachments(msg)
		assert.Contains(t, content, "this body")
		assert.NotEmpty(t, paths)
		assert.Equal(t, 1, len(paths))
		assert.Contains(t, paths[0], filepath.Base(paths[0]))
	})

	newLimitClient := &EmailChannel{
		config: config.EmailConfig{
			BodyPartMaxBytes:   1,
			AttachmentDir:      t.TempDir(),
			AttachmentMaxBytes: 1024,
		},
	}
	t.Run("body part max bytes", func(t *testing.T) {

		mimeBytes := attachmentwithText
		section := &imap.BodySectionName{}
		msg := &imap.Message{
			Uid:      1,
			Envelope: &imap.Envelope{Subject: "Test"},
			Body:     map[*imap.BodySectionName]imap.Literal{section: bytes.NewReader(mimeBytes)},
		}
		content, paths := newLimitClient.extractEmailBodyAndAttachments(msg)
		assert.Contains(t, content, "[body part exceeds size limit (max 1 bytes), you can check body_part_max_bytes in config]")
		assert.NotEmpty(t, paths)
		assert.Equal(t, 1, len(paths))
		assert.Contains(t, paths[0], filepath.Base(paths[0]))
	})
}

func TestEmailChannel_extractTextFromHTML(t *testing.T) {

	tests := []struct {
		name string
		s    string
		want string
	}{
		{name: "with-script-and-style", s: `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>Minimal</title>
  <style>
    body { margin: 0; font-family: sans-serif; }
    .box { padding: 1rem; background: #eee; }
  </style>
</head>
<body>
  <div class="box">Hello</div>
  <script>
    document.querySelector('.box').onclick = function() {
      this.textContent = 'Clicked';
    };
  </script>
</body>
</html>`, want: "Minimal\n \n\n  Hello"},
	}
	c := &EmailChannel{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.extractTextFromHTML(tt.s); got != tt.want {
				t.Errorf("extractTextFromHTML(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func Test_saveAttachmentToLocal(t *testing.T) {
	tmpDir := t.TempDir()
	c := &EmailChannel{
		config: config.EmailConfig{
			AttachmentDir: tmpDir,
		},
	}
	content := "Hello world"
	body := bytes.NewReader([]byte(content))
	// test attachment max bytes
	c.config.AttachmentMaxBytes = len([]byte(content))
	path := c.saveAttachmentToLocal(1, 1, "test.txt", body)
	assert.NotEmpty(t, path)
	assert.Equal(t, path, filepath.Join(tmpDir, "1_1_test.txt"))

	// test greater than attachment max bytes
	c.config.AttachmentMaxBytes = len([]byte(content)) - 1
	body = bytes.NewReader([]byte(content))
	path = c.saveAttachmentToLocal(1, 1, "test.txt", body)
	assert.Empty(t, path)
}

func TestEmailChannel_checkNewEmails(t *testing.T) {
	// check if the current runtime is go1.25.xx
	if !strings.HasPrefix(runtime.Version(), "go1.25") {
		//  github.com/bytedance/mockey v1.4.4 is supported in go1.25.xx
		t.Skip("skipping test in non-go1.25.xx environment")
		return
	}

	// mock connect to return mockClient
	mockey.PatchConvey("checkNewEmails", t, func() {
		// --------------- mock start ---------------
		c := &EmailChannel{
			BaseChannel: &BaseChannel{
				bus: bus.NewMessageBus(),
			},
			config: config.EmailConfig{
				Enabled:   true,
				AllowFrom: config.FlexibleStringSlice{},
			},
			lastUID: 20,
		}
		mockClient := &client.Client{}
		c.imapClient = mockClient
		// mock login and select to return mockClient
		mockey.Mock(mockey.GetMethod(mockClient, "Login")).To(func(imapClient *client.Client, username, password string) error {
			return nil
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "Select")).To(func(imapClient *client.Client, mailbox string, readonly bool) (*imap.MailboxStatus, error) {
			return &imap.MailboxStatus{
				UidNext: 20,
			}, nil
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidSearch")).To(func(imapClient *client.Client, criteria *imap.SearchCriteria) ([]uint32, error) {
			return []uint32{21}, nil
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidFetch")).To(func(imapClient *client.Client, seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			mimeBytes := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: Test\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello world")
			section := &imap.BodySectionName{}
			msg := &imap.Message{
				Uid:      1,
				Envelope: &imap.Envelope{Subject: "Test"},
				Body:     map[*imap.BodySectionName]imap.Literal{section: bytes.NewReader(mimeBytes)},
			}
			ch <- msg
			close(ch)
			return nil
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "State")).To(func(*client.Client) imap.ConnState {
			return imap.SelectedState
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidStore")).To(func(imapClient *client.Client, seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error {
			return nil
		}).Build()
		// --------------- mock end ---------------
		ctx := context.Background()
		c.CheckNewEmails(context.Background())
		timeoutCtx, _ := context.WithTimeout(ctx, time.Second)
		messge, ok := c.bus.ConsumeInbound(timeoutCtx)
		assert.True(t, ok)
		assert.True(t, strings.Contains(messge.Content, "Hello world"))
	})

}

func TestEmailChannel_runIdleLoop(t *testing.T) {
	// check if the current runtime is go1.25.xx
	if !strings.HasPrefix(runtime.Version(), "go1.25") {
		//  github.com/bytedance/mockey v1.4.4 is supported in go1.25.xx
		t.Skip("skipping test in non-go1.25.xx environment")
		return
	}

	// mock connect to return mockClient
	mockey.PatchConvey("runIdleLoop", t, func() {
		// --------------- mock start ---------------
		var hasCheckEmail = false
		c := &EmailChannel{
			BaseChannel: &BaseChannel{
				bus: bus.NewMessageBus(),
			},
			config: config.EmailConfig{
				Enabled:   true,
				AllowFrom: config.FlexibleStringSlice{},
			},
			lastUID: 20,
		}
		mockClient := &client.Client{}
		c.imapClient = mockClient
		// mock login and select to return mockClient
		mockey.Mock(mockey.GetMethod(c, "CheckNewEmails")).To(func(*EmailChannel, context.Context) {
			hasCheckEmail = true
		}).Build()

		mockey.Mock(mockey.GetMethod(mockClient, "State")).To(func(*client.Client) imap.ConnState {
			return imap.SelectedState
		}).Build()
		triggerChannel := make(chan struct{}, 1)
		mockey.Mock(mockey.GetMethod(mockClient, "Idle")).To(func(self *client.Client, stop <-chan struct{}, opts *client.IdleOptions) error {
			<-triggerChannel
			self.Updates <- &client.StatusUpdate{}
			return nil
		}).Build()
		// --------------- mock end ---------------
		ctx := context.Background()
		go c.runIdleLoop(ctx, 2*time.Second)
		assert.False(t, hasCheckEmail)
		// sent update sigal
		triggerChannel <- struct{}{}
		time.Sleep(time.Second)
		assert.True(t, hasCheckEmail)

	})

}
