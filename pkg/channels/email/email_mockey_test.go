//go:build mockey

package email

// Tests in this file use github.com/bytedance/mockey and require -gcflags="all=-N -l" to run.
// Run with: go test -tags=mockey -gcflags="all=-N -l" ./pkg/channels/email/...
// Without -tags=mockey they are not compiled; without -gcflags they may fail due to Mockey.

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

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
		msgBus := bus.NewMessageBus()
		c, err := NewEmailChannel(config.EmailConfig{
			Enabled:   true,
			AllowFrom: config.FlexibleStringSlice{},
		}, msgBus)
		if err != nil {
			t.Fatal(err)
		}
		c.lastUID = 20
		mockClient := &client.Client{}
		c.imapClient = mockClient
		// mock login and select to return mockClient
		mockey.Mock(mockey.GetMethod(mockClient, "Login")).
			To(func(imapClient *client.Client, username, password string) error {
				return nil
			}).
			Build()
		mockey.Mock(mockey.GetMethod(mockClient, "Select")).
			To(func(imapClient *client.Client, mailbox string, readonly bool) (*imap.MailboxStatus, error) {
				return &imap.MailboxStatus{
					UidNext: 20,
				}, nil
			}).
			Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidSearch")).
			To(func(imapClient *client.Client, criteria *imap.SearchCriteria) ([]uint32, error) {
				return []uint32{21}, nil
			}).
			Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidFetch")).To(
			func(imapClient *client.Client, seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
				mimeBytes := []byte(
					"From: a@b.com\r\nTo: c@d.com\r\nSubject: Test\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello world",
				)
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
		mockClient.SetState(imap.SelectedState, &imap.MailboxStatus{
			UidNext: 20,
		})
		mockey.Mock(mockey.GetMethod(mockClient, "State")).To(func(*client.Client) imap.ConnState {
			return imap.SelectedState
		}).Build()
		mockey.Mock(mockey.GetMethod(mockClient, "UidStore")).To(
			func(imapClient *client.Client, seqset *imap.SeqSet, item imap.StoreItem, value any,
				ch chan *imap.Message,
			) error {
				return nil
			}).Build()
		// --------------- mock end ---------------
		ctx := context.Background()
		c.CheckNewEmails(context.Background())
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		messge, ok := msgBus.ConsumeInbound(timeoutCtx)
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
		hasCheckEmail := false
		msgBus := bus.NewMessageBus()
		c, err := NewEmailChannel(config.EmailConfig{
			Enabled:   true,
			AllowFrom: config.FlexibleStringSlice{},
		}, msgBus)
		if err != nil {
			t.Fatal(err)
		}
		c.lastUID = 20
		mockClient := &client.Client{}
		c.imapClient = mockClient
		mockey.Mock(mockey.GetMethod(c, "CheckNewEmails")).To(func(*EmailChannel, context.Context) {
			hasCheckEmail = true
		}).Build()

		mockey.Mock(mockey.GetMethod(mockClient, "State")).To(func(*client.Client) imap.ConnState {
			return imap.SelectedState
		}).Build()
		triggerChannel := make(chan struct{}, 1)
		mockey.Mock(mockey.GetMethod(mockClient, "Idle")).
			To(func(self *client.Client, stop <-chan struct{}, opts *client.IdleOptions) error {
				<-triggerChannel
				self.Updates <- &client.StatusUpdate{}
				return nil
			}).
			Build()
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

func TestEmailChannel_lifecycleCheck(t *testing.T) {
	// check if the current runtime is go1.25.xx
	if !strings.HasPrefix(runtime.Version(), "go1.25") {
		//  github.com/bytedance/mockey v1.4.4 is supported in go1.25.xx
		t.Skip("skipping test in non-go1.25.xx environment")
		return
	}

	mockey.PatchConvey("lifecycle test", t, func() {
		// --------------- mock start ---------------
		msgBus := bus.NewMessageBus()
		c, err := NewEmailChannel(config.EmailConfig{
			Enabled:       true,
			CheckInterval: 1,
			ForcedPolling: true,
			IMAPServer:    "imap.example.com",
			Username:      "testuser",
			Password:      "testpassword",
		}, msgBus)
		if err != nil {
			t.Fatal(err)
		}
		mockey.Mock(mockey.GetMethod(c, "connect")).To(func(*EmailChannel) error {
			return nil
		}).Build()
		mockey.Mock(mockey.GetMethod(c, "CheckNewEmails")).To(func(*EmailChannel, context.Context) {
			time.Sleep(1 * time.Second)
		}).Build()
		// --------------- mock end ---------------
		ctx := context.Background()
		err = c.Start(ctx)
		assert.NoError(t, err)
		wg := sync.WaitGroup{}
		wg.Add(1)
		var stopDone time.Time
		stopStart := time.Now()
		go func() {
			defer wg.Done()
			c.Stop(ctx)
			stopDone = time.Now()
		}()
		// wait for checkNewEmails to finish
		assert.True(t, c.IsRunning())
		wg.Wait()
		elapsed := stopDone.Sub(stopStart)
		// stop exit normally
		assert.False(t, c.IsRunning())
		// If Stop() did not wait for checkLoop, it would return in milliseconds.
		assert.GreaterOrEqual(t, elapsed, 1*time.Second, "Stop() must wait for checkLoop (lifecycle compliance)")
	})
}
