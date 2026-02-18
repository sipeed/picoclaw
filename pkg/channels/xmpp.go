package channels

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/upload"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type XMPPChannel struct {
	*BaseChannel
	config     config.XMPPConfig
	session    *xmpp.Session
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client

	uploadMu  sync.Mutex
	uploadJID jid.JID

	lastMsgMu    sync.Mutex
	lastFromBare string
	lastContent  string
	lastTime     time.Time
}

func NewXMPPChannel(cfg config.XMPPConfig, messageBus *bus.MessageBus) (*XMPPChannel, error) {
	if cfg.JID == "" {
		return nil, fmt.Errorf("xmpp jid is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("xmpp password is required")
	}

	base := NewBaseChannel("xmpp", cfg, messageBus, cfg.AllowFrom)

	return &XMPPChannel{
		BaseChannel: base,
		config:      cfg,
		httpClient:  newHTTPClient(),
	}, nil
}

func (c *XMPPChannel) Start(ctx context.Context) error {
	if c.IsRunning() {
		return nil
	}

	j, err := jid.Parse(c.config.JID)
	if err != nil {
		return fmt.Errorf("invalid xmpp jid: %w", err)
	}

	logger.InfoCF("xmpp", "Connecting XMPP client", map[string]interface{}{
		"jid": j.String(),
	})

	session, err := xmpp.DialClientSession(
		ctx,
		j,
		xmpp.StartTLS(&tls.Config{
			ServerName: j.Domain().String(),
			MinVersion: tls.VersionTLS12,
		}),
		xmpp.SASL("", c.config.Password, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		return fmt.Errorf("failed to establish XMPP session: %w", err)
	}

	handler := mux.MessageHandlerFunc(func(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
		return c.handleIncomingMessage(msg, t)
	})

	m := mux.New(
		stanza.NSClient,
		mux.Message(stanza.ChatMessage, xml.Name{}, handler),
		mux.Message(stanza.MessageType(""), xml.Name{}, handler),
	)

	c.session = session
	c.ctx, c.cancel = context.WithCancel(context.Background())

	if err := c.sendInitialPresence(); err != nil {
		logger.WarnCF("xmpp", "Failed to send initial presence", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.setRunning(true)

	go func() {
		err := session.Serve(m)
		if err != nil {
			logger.ErrorCF("xmpp", "XMPP session ended with error", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			logger.InfoC("xmpp", "XMPP session ended")
		}
		c.setRunning(false)
	}()

	return nil
}

func (c *XMPPChannel) Stop(ctx context.Context) error {
	if !c.IsRunning() {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}
	if c.session != nil {
		if err := c.session.Close(); err != nil {
			logger.WarnCF("xmpp", "Error closing XMPP session", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("xmpp", "XMPP channel stopped")
	return nil
}

func (c *XMPPChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() || c.session == nil {
		return fmt.Errorf("xmpp channel not running")
	}

	to, err := jid.Parse(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid xmpp chat id: %w", err)
	}

	content := strings.TrimSpace(msg.Content)

	var mediaLinks []string
	var inlineParts []string
	if len(msg.Media) > 0 {
		uploadJID, discoverErr := c.discoverUploadService(ctx)
		if discoverErr != nil {
			logger.ErrorCF("xmpp", "Failed to discover XEP-0363 upload service", map[string]interface{}{
				"error": discoverErr.Error(),
			})
		}

		for _, path := range msg.Media {
			if discoverErr == nil {
				link, err := c.uploadFile(ctx, uploadJID, path)
				if err == nil && link != "" {
					mediaLinks = append(mediaLinks, link)
					continue
				}
				if err != nil {
					logger.ErrorCF("xmpp", "Failed to upload media file", map[string]interface{}{
						"path":  path,
						"error": err.Error(),
					})
				}
			}

			inlineContent, err := c.readSmallFile(path, 32*1024)
			if err != nil {
				logger.ErrorCF("xmpp", "Failed to inline media file", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
				continue
			}

			inlineParts = append(inlineParts, fmt.Sprintf("文件 %s:\n%s", filepath.Base(path), inlineContent))
		}
	}

	if len(inlineParts) > 0 {
		inlineText := strings.Join(inlineParts, "\n\n")
		if content == "" {
			content = inlineText
		} else {
			content = content + "\n\n" + inlineText
		}
	}

	if len(mediaLinks) > 0 {
		body := mediaLinks[0]
		desc := content
		return c.sendChatMessage(to, body, desc, mediaLinks)
	}

	if content == "" {
		return nil
	}

	return c.sendChatMessage(to, content, "", nil)
}

func (c *XMPPChannel) sendInitialPresence() error {
	return c.sendStanza(stanza.Presence{}.Wrap(nil))
}

func (c *XMPPChannel) sendChatMessage(to jid.JID, body string, desc string, mediaLinks []string) error {
	bodyElem := xmlstream.Wrap(
		xmlstream.Token(xml.CharData([]byte(body))),
		xml.StartElement{Name: xml.Name{Local: "body"}},
	)

	children := []xml.TokenReader{bodyElem}

	for _, link := range mediaLinks {
		urlElem := xmlstream.Wrap(
			xmlstream.Token(xml.CharData([]byte(link))),
			xml.StartElement{Name: xml.Name{Local: "url"}},
		)

		var inner xml.TokenReader = urlElem

		if desc != "" {
			descElem := xmlstream.Wrap(
				xmlstream.Token(xml.CharData([]byte(desc))),
				xml.StartElement{Name: xml.Name{Local: "desc"}},
			)
			inner = xmlstream.MultiReader(urlElem, descElem)
		}

		xElem := xmlstream.Wrap(
			inner,
			xml.StartElement{
				Name: xml.Name{Local: "x"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "xmlns"}, Value: "jabber:x:oob"},
				},
			},
		)

		children = append(children, xElem)
	}

	var payload xml.TokenReader
	if len(children) == 1 {
		payload = children[0]
	} else {
		payload = xmlstream.MultiReader(children...)
	}

	st := stanza.Message{
		Type: stanza.ChatMessage,
		To:   to,
	}

	return c.sendStanza(st.Wrap(payload))
}

func (c *XMPPChannel) sendStanza(r xml.TokenReader) error {
	if c.session == nil {
		return fmt.Errorf("xmpp session not initialized")
	}

	w := c.session.TokenWriter()
	defer w.Close()

	if _, err := xmlstream.Copy(w, r); err != nil {
		return err
	}

	type flusher interface {
		Flush() error
	}

	if f, ok := w.(flusher); ok {
		return f.Flush()
	}

	return nil
}

func (c *XMPPChannel) handleIncomingMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	var payload struct {
		Body string `xml:"body"`
	}

	d := xml.NewTokenDecoder(t)
	if err := d.Decode(&payload); err != nil {
		return err
	}

	content := strings.TrimSpace(payload.Body)
	if content == "" {
		return nil
	}

	fromBare := msg.From.Bare().String()
	chatID := msg.From.String()

	c.lastMsgMu.Lock()
	if fromBare == c.lastFromBare && content == c.lastContent && time.Since(c.lastTime) < 2*time.Second {
		c.lastMsgMu.Unlock()
		return nil
	}
	c.lastFromBare = fromBare
	c.lastContent = content
	c.lastTime = time.Now()
	c.lastMsgMu.Unlock()

	logger.DebugCF("xmpp", "Received message", map[string]interface{}{
		"from":      chatID,
		"from_bare": fromBare,
		"preview":   utils.Truncate(content, 80),
	})

	c.HandleMessage(fromBare, chatID, content, nil, map[string]string{
		"stanza_type": "message",
		"from_full":   chatID,
	})

	return nil
}

func (c *XMPPChannel) discoverUploadService(ctx context.Context) (jid.JID, error) {
	c.uploadMu.Lock()
	defer c.uploadMu.Unlock()

	if !c.uploadJID.Equal(jid.JID{}) {
		return c.uploadJID, nil
	}

	if c.config.UploadDomain != "" {
		j, err := jid.Parse(c.config.UploadDomain)
		if err == nil {
			c.uploadJID = j
			return c.uploadJID, nil
		}
	}

	if c.session == nil {
		return jid.JID{}, fmt.Errorf("xmpp session not initialized")
	}

	userJID, err := jid.Parse(c.config.JID)
	if err != nil {
		return jid.JID{}, fmt.Errorf("invalid xmpp jid in config: %w", err)
	}
	domain := userJID.Domain()
	info, err := disco.GetInfo(ctx, "", domain, c.session)
	if err == nil {
		for _, f := range info.Features {
			if f.Var == upload.NS {
				c.uploadJID = domain
				logger.InfoCF("xmpp", "Discovered HTTP upload support on server domain", map[string]interface{}{
					"jid": domain.String(),
				})
				return c.uploadJID, nil
			}
		}
	}

	var found jid.JID
	err = disco.WalkItem(ctx, items.Item{JID: domain}, c.session, func(level int, item items.Item, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		info, err := disco.GetInfo(ctx, "", item.JID, c.session)
		if err != nil {
			return nil
		}

		for _, f := range info.Features {
			if f.Var == upload.NS {
				found = item.JID
				return fmt.Errorf("found")
			}
		}
		return nil
	})
	if err != nil && err.Error() != "found" {
		return jid.JID{}, fmt.Errorf("service discovery failed: %w", err)
	}

	if found.Equal(jid.JID{}) {
		return jid.JID{}, fmt.Errorf("no XEP-0363 upload service found via disco")
	}

	c.uploadJID = found
	logger.InfoCF("xmpp", "Discovered HTTP upload service via disco", map[string]interface{}{
		"jid": found.String(),
	})
	return c.uploadJID, nil
}

func (c *XMPPChannel) uploadFile(ctx context.Context, uploadJID jid.JID, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	size := info.Size()
	if size <= 0 {
		return "", fmt.Errorf("file is empty")
	}

	buffer := make([]byte, 512)
	n, _ := f.Read(buffer)
	if _, err := f.Seek(0, 0); err != nil {
		return "", fmt.Errorf("seek file: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	file := upload.File{
		Name: filepath.Base(path),
		Size: int(size),
		Type: contentType,
	}

	slot, err := upload.GetSlot(ctx, file, uploadJID, c.session)
	if err != nil {
		return "", fmt.Errorf("get upload slot: %w", err)
	}

	logger.InfoCF("xmpp", "XEP-0363 upload slot acquired", map[string]interface{}{
		"put_url": func() string {
			if slot.PutURL != nil {
				return slot.PutURL.String()
			}
			return ""
		}(),
		"get_url": func() string {
			if slot.GetURL != nil {
				return slot.GetURL.String()
			}
			return ""
		}(),
		"headers":  slot.Header,
		"mime":     contentType,
		"size":     size,
		"filename": filepath.Base(path),
	})

	req, err := slot.Put(ctx, f)
	if err != nil {
		return "", fmt.Errorf("build put request: %w", err)
	}

	req.ContentLength = size
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http put: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		bodyText := strings.TrimSpace(string(snippet))
		if bodyText != "" {
			return "", fmt.Errorf("upload failed with status %s: %s", resp.Status, bodyText)
		}
		return "", fmt.Errorf("upload failed with status %s", resp.Status)
	}

	if slot.GetURL == nil {
		return "", fmt.Errorf("no download URL returned for upload slot")
	}

	return slot.GetURL.String(), nil
}

func (c *XMPPChannel) readSmallFile(path string, maxSize int64) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	if info.Size() <= 0 {
		return "", fmt.Errorf("file is empty")
	}
	if info.Size() > maxSize {
		return "", fmt.Errorf("file too large to inline")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func newHTTPClient() *http.Client {
	dialer := &net.Dialer{}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err == nil {
				return conn, nil
			}

			if !strings.Contains(err.Error(), "no such host") {
				return nil, err
			}

			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, err
			}

			parts := strings.SplitN(host, ".", 2)
			if len(parts) != 2 || parts[1] == "" {
				return nil, err
			}

			fallbackHost := parts[1]
			return dialer.DialContext(ctx, network, net.JoinHostPort(fallbackHost, port))
		},
	}

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}
}
