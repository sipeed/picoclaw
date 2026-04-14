package wecom

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestDispatchIncoming_UsesActualChatIDAndStoresReqIDRoute(t *testing.T) {
	t.Parallel()

	messageBus := bus.NewMessageBus()
	ch := newTestWeComChannel(t, messageBus)

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		return wecomTestAck(nil), nil
	}

	msg := wecomIncomingMessage{
		MsgID:    "msg-1",
		ChatID:   "chat-1",
		ChatType: "direct",
		MsgType:  "text",
		Text: &struct {
			Content string `json:"content"`
		}{Content: "hello"},
	}
	msg.From.UserID = "user-1"

	if err := ch.dispatchIncoming("req-1", msg); err != nil {
		t.Fatalf("dispatchIncoming() error = %v", err)
	}

	select {
	case inbound := <-messageBus.InboundChan():
		if inbound.ChatID != "chat-1" {
			t.Fatalf("inbound ChatID = %q, want chat-1", inbound.ChatID)
		}
		if inbound.MessageID != "msg-1" {
			t.Fatalf("inbound MessageID = %q, want msg-1", inbound.MessageID)
		}
		if inbound.Peer.ID != "chat-1" {
			t.Fatalf("inbound Peer.ID = %q, want chat-1", inbound.Peer.ID)
		}
		if inbound.Metadata["req_id"] != "req-1" {
			t.Fatalf("inbound req_id = %q, want req-1", inbound.Metadata["req_id"])
		}
	default:
		t.Fatal("expected inbound message to be published")
	}

	turn, ok := ch.getTurn("chat-1")
	if !ok {
		t.Fatal("expected queued turn for chat-1")
	}
	if turn.ReqID != "req-1" {
		t.Fatalf("turn.ReqID = %q, want req-1", turn.ReqID)
	}

	route, ok := ch.routes.Get("chat-1")
	if !ok {
		t.Fatal("expected persisted route for chat-1")
	}
	if route.ReqID != "req-1" || route.ChatType != 1 {
		t.Fatalf("route = %+v", route)
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 opening command, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdRespondMsg {
		t.Fatalf("opening command = %q, want %q", commands[0].Cmd, wecomCmdRespondMsg)
	}
	if commands[0].Headers.ReqID != "req-1" {
		t.Fatalf("opening req_id = %q, want req-1", commands[0].Headers.ReqID)
	}
}

func TestNewChannel_DoesNotRegisterMessageSplitLimit(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	if got := ch.MaxMessageLength(); got != 0 {
		t.Fatalf("MaxMessageLength() = %d, want 0", got)
	}
}

func TestBeginStream_UpdateAndFinalize(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)
	ch.queueTurn("chat-1", wecomTurn{
		ReqID:     "req-1",
		ChatID:    "chat-1",
		ChatType:  1,
		StreamID:  "stream-1",
		CreatedAt: time.Now(),
	})

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		return wecomTestAck(nil), nil
	}

	streamer, err := ch.BeginStream(context.Background(), "chat-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if err := streamer.Update(context.Background(), "draft"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := streamer.Finalize(context.Background(), "final"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	for i, wantFinish := range []bool{false, true} {
		if commands[i].Cmd != wecomCmdRespondMsg {
			t.Fatalf("command[%d].Cmd = %q, want %q", i, commands[i].Cmd, wecomCmdRespondMsg)
		}
		body, ok := commands[i].Body.(wecomRespondMsgBody)
		if !ok {
			t.Fatalf("command[%d] body type = %T", i, commands[i].Body)
		}
		if body.Stream == nil {
			t.Fatalf("command[%d] missing stream body", i)
		}
		if body.Stream.ID != "stream-1" {
			t.Fatalf("command[%d] stream id = %q, want stream-1", i, body.Stream.ID)
		}
		if body.Stream.Finish != wantFinish {
			t.Fatalf("command[%d] finish = %v, want %v", i, body.Stream.Finish, wantFinish)
		}
	}
	if body := commands[0].Body.(wecomRespondMsgBody); body.Stream.Content != "draft" {
		t.Fatalf("update content = %q, want draft", body.Stream.Content)
	}
	if body := commands[1].Body.(wecomRespondMsgBody); body.Stream.Content != "final" {
		t.Fatalf("final content = %q, want final", body.Stream.Content)
	}
	if _, ok := ch.getTurn("chat-1"); ok {
		t.Fatal("expected turn to be consumed after Finalize")
	}
}

func TestSend_StreamFailureFallsBackToActualChatID(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)
	ch.queueTurn("chat-1", wecomTurn{
		ReqID:     "req-1",
		ChatID:    "chat-1",
		ChatType:  1,
		StreamID:  "stream-1",
		CreatedAt: time.Now(),
	})
	ch.queueTurn("chat-1", wecomTurn{
		ReqID:     "req-2",
		ChatID:    "chat-1",
		ChatType:  1,
		StreamID:  "stream-2",
		CreatedAt: time.Now(),
	})
	if err := ch.routes.Put("chat-1", "req-2", 1, time.Hour); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		if len(commands) == 1 && cmd.Cmd == wecomCmdRespondMsg {
			return wecomEnvelope{}, errors.New("stream send failed")
		}
		return wecomTestAck(nil), nil
	}

	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "wecom",
		ChatID:  "chat-1",
		Content: "hello",
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdRespondMsg || commands[0].Headers.ReqID != "req-1" {
		t.Fatalf("first command = %+v", commands[0])
	}
	if commands[1].Cmd != wecomCmdSendMsg {
		t.Fatalf("second command = %q, want %q", commands[1].Cmd, wecomCmdSendMsg)
	}
	body, ok := commands[1].Body.(wecomSendMsgBody)
	if !ok {
		t.Fatalf("unexpected send body type %T", commands[1].Body)
	}
	if body.ChatID != "chat-1" {
		t.Fatalf("send chatid = %q, want chat-1", body.ChatID)
	}
	if body.ChatType != 1 {
		t.Fatalf("send chat_type = %d, want 1", body.ChatType)
	}

	nextTurn, ok := ch.getTurn("chat-1")
	if !ok {
		t.Fatal("expected second turn to remain queued")
	}
	if nextTurn.ReqID != "req-2" {
		t.Fatalf("next queued req_id = %q, want req-2", nextTurn.ReqID)
	}
}

func TestSend_DoesNotSplitStreamReply(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)
	ch.queueTurn("chat-1", wecomTurn{
		ReqID:     "req-1",
		ChatID:    "chat-1",
		ChatType:  1,
		StreamID:  "stream-1",
		CreatedAt: time.Now(),
	})

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		return wecomTestAck(nil), nil
	}

	content := strings.Repeat("\u4e2d", 30000)
	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "wecom",
		ChatID:  "chat-1",
		Content: content,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 stream command, got %d", len(commands))
	}
	body, ok := commands[0].Body.(wecomRespondMsgBody)
	if !ok {
		t.Fatalf("unexpected body type %T", commands[0].Body)
	}
	if body.Stream == nil || !body.Stream.Finish {
		t.Fatalf("stream body = %+v", body.Stream)
	}
	if body.Stream.Content != content {
		t.Fatalf("stream content length = %d, want %d", len(body.Stream.Content), len(content))
	}
}

func TestSend_DoesNotSplitActivePush(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		return wecomTestAck(nil), nil
	}

	content := strings.Repeat("a", 30000)
	if _, err := ch.Send(context.Background(), bus.OutboundMessage{
		Channel: "wecom",
		ChatID:  "chat-1",
		Content: content,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 send command, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdSendMsg {
		t.Fatalf("command = %q, want %q", commands[0].Cmd, wecomCmdSendMsg)
	}
	body, ok := commands[0].Body.(wecomSendMsgBody)
	if !ok {
		t.Fatalf("unexpected body type %T", commands[0].Body)
	}
	if body.Markdown == nil || body.Markdown.Content != content {
		t.Fatalf("markdown content length = %d, want %d", len(body.Markdown.Content), len(content))
	}
}

func TestSendMedia_SendsActiveImage(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	imageData := wecomTestJPEGData(t)
	imagePath := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(imagePath, imageData, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ref, err := store.Store(imagePath, media.MediaMeta{
		Filename:      "photo.jpg",
		ContentType:   "image/jpeg",
		Source:        "test",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, "scope-1")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		switch cmd.Cmd {
		case wecomCmdUploadMediaInit:
			return wecomTestAck(wecomUploadMediaInitResponse{UploadID: "upload-1"}), nil
		case wecomCmdUploadMediaEnd:
			return wecomTestAck(wecomUploadMediaFinishResponse{
				Type:    "image",
				MediaID: "media-1",
			}), nil
		default:
			return wecomTestAck(nil), nil
		}
	}

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		Channel: "wecom",
		ChatID:  "chat-1",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "image",
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	if len(commands) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdUploadMediaInit {
		t.Fatalf("first command = %q, want %q", commands[0].Cmd, wecomCmdUploadMediaInit)
	}
	initBody, ok := commands[0].Body.(wecomUploadMediaInitBody)
	if !ok {
		t.Fatalf("unexpected init body type %T", commands[0].Body)
	}
	if initBody.Type != "image" || initBody.Filename != "photo.jpg" || initBody.TotalChunks != 1 {
		t.Fatalf("init body = %+v", initBody)
	}
	if commands[1].Cmd != wecomCmdUploadMediaChunk {
		t.Fatalf("second command = %q, want %q", commands[1].Cmd, wecomCmdUploadMediaChunk)
	}
	chunkBody, ok := commands[1].Body.(wecomUploadMediaChunkBody)
	if !ok {
		t.Fatalf("unexpected chunk body type %T", commands[1].Body)
	}
	if chunkBody.UploadID != "upload-1" || chunkBody.ChunkIndex != 0 || chunkBody.Base64Data == "" {
		t.Fatalf("chunk body = %+v", chunkBody)
	}
	if commands[2].Cmd != wecomCmdUploadMediaEnd {
		t.Fatalf("third command = %q, want %q", commands[2].Cmd, wecomCmdUploadMediaEnd)
	}
	if commands[3].Cmd != wecomCmdSendMsg {
		t.Fatalf("fourth command = %q, want %q", commands[3].Cmd, wecomCmdSendMsg)
	}

	body, ok := commands[3].Body.(wecomSendMsgBody)
	if !ok {
		t.Fatalf("unexpected send body type %T", commands[3].Body)
	}
	if body.MsgType != "image" || body.Image == nil {
		t.Fatalf("send body = %+v", body)
	}
	if body.ChatID != "chat-1" {
		t.Fatalf("send chatid = %q, want chat-1", body.ChatID)
	}
	if body.Image.MediaID != "media-1" {
		t.Fatalf("image media_id = %q, want media-1", body.Image.MediaID)
	}
}

func TestSendMedia_UsesTurnImageAndFinishesStream(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	imageData := wecomTestJPEGData(t)
	imagePath := filepath.Join(t.TempDir(), "reply.jpg")
	if err := os.WriteFile(imagePath, imageData, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ref, err := store.Store(imagePath, media.MediaMeta{
		Filename:      "reply.jpg",
		ContentType:   "image/jpeg",
		Source:        "test",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, "scope-2")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	ch.queueTurn("chat-1", wecomTurn{
		ReqID:     "req-1",
		ChatID:    "chat-1",
		ChatType:  1,
		StreamID:  "stream-1",
		CreatedAt: time.Now(),
	})
	putErr := ch.routes.Put("chat-1", "req-1", 1, time.Hour)
	if putErr != nil {
		t.Fatalf("Put() error = %v", putErr)
	}

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		switch cmd.Cmd {
		case wecomCmdUploadMediaInit:
			return wecomTestAck(wecomUploadMediaInitResponse{UploadID: "upload-2"}), nil
		case wecomCmdUploadMediaEnd:
			return wecomTestAck(wecomUploadMediaFinishResponse{
				Type:    "image",
				MediaID: "media-2",
			}), nil
		default:
			return wecomTestAck(nil), nil
		}
	}

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		Channel: "wecom",
		ChatID:  "chat-1",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "image",
			Filename:    "reply.jpg",
			ContentType: "image/jpeg",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	if len(commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdUploadMediaInit {
		t.Fatalf("first command = %+v", commands[0])
	}
	if commands[1].Cmd != wecomCmdUploadMediaChunk {
		t.Fatalf("second command = %+v", commands[1])
	}
	if commands[2].Cmd != wecomCmdUploadMediaEnd {
		t.Fatalf("third command = %+v", commands[2])
	}
	if commands[3].Cmd != wecomCmdRespondMsg || commands[3].Headers.ReqID != "req-1" {
		t.Fatalf("fourth command = %+v", commands[3])
	}
	if commands[4].Cmd != wecomCmdRespondMsg || commands[4].Headers.ReqID != "req-1" {
		t.Fatalf("fifth command = %+v", commands[4])
	}

	imageBody, ok := commands[3].Body.(wecomRespondMsgBody)
	if !ok {
		t.Fatalf("unexpected image body type %T", commands[3].Body)
	}
	if imageBody.MsgType != "image" || imageBody.Image == nil {
		t.Fatalf("image body = %+v", imageBody)
	}
	if imageBody.Image.MediaID != "media-2" {
		t.Fatalf("image media_id = %q, want media-2", imageBody.Image.MediaID)
	}

	streamBody, ok := commands[4].Body.(wecomRespondMsgBody)
	if !ok {
		t.Fatalf("unexpected finish body type %T", commands[4].Body)
	}
	if streamBody.MsgType != "stream" || streamBody.Stream == nil || !streamBody.Stream.Finish {
		t.Fatalf("finish body = %+v", streamBody)
	}

	if _, ok := ch.getTurn("chat-1"); ok {
		t.Fatal("expected turn to be removed after media send")
	}
}

func TestSendMedia_SendsActiveFile(t *testing.T) {
	t.Parallel()

	ch := newTestWeComChannel(t, bus.NewMessageBus())
	ch.SetRunning(true)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	filePath := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(filePath, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ref, err := store.Store(filePath, media.MediaMeta{
		Filename:      "report.pdf",
		ContentType:   "application/pdf",
		Source:        "test",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, "scope-3")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	var commands []wecomCommand
	ch.commandSend = func(cmd wecomCommand, _ time.Duration) (wecomEnvelope, error) {
		commands = append(commands, cmd)
		switch cmd.Cmd {
		case wecomCmdUploadMediaInit:
			return wecomTestAck(wecomUploadMediaInitResponse{UploadID: "upload-3"}), nil
		case wecomCmdUploadMediaEnd:
			return wecomTestAck(wecomUploadMediaFinishResponse{
				Type:    "file",
				MediaID: "media-3",
			}), nil
		default:
			return wecomTestAck(nil), nil
		}
	}

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		Channel: "wecom",
		ChatID:  "chat-2",
		Parts: []bus.MediaPart{{
			Ref:         ref,
			Type:        "file",
			Filename:    "report.pdf",
			ContentType: "application/pdf",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	if len(commands) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(commands))
	}
	if commands[0].Cmd != wecomCmdUploadMediaInit {
		t.Fatalf("first command = %q, want %q", commands[0].Cmd, wecomCmdUploadMediaInit)
	}
	initBody, ok := commands[0].Body.(wecomUploadMediaInitBody)
	if !ok {
		t.Fatalf("unexpected init body type %T", commands[0].Body)
	}
	if initBody.Type != "file" || initBody.Filename != "report.pdf" {
		t.Fatalf("init body = %+v", initBody)
	}
	if commands[1].Cmd != wecomCmdUploadMediaChunk {
		t.Fatalf("second command = %q, want %q", commands[1].Cmd, wecomCmdUploadMediaChunk)
	}
	if commands[2].Cmd != wecomCmdUploadMediaEnd {
		t.Fatalf("third command = %q, want %q", commands[2].Cmd, wecomCmdUploadMediaEnd)
	}
	if commands[3].Cmd != wecomCmdSendMsg {
		t.Fatalf("fourth command = %q, want %q", commands[3].Cmd, wecomCmdSendMsg)
	}

	body, ok := commands[3].Body.(wecomSendMsgBody)
	if !ok {
		t.Fatalf("unexpected body type %T", commands[3].Body)
	}
	if body.MsgType != "file" || body.File == nil {
		t.Fatalf("body = %+v", body)
	}
	if body.File.MediaID != "media-3" {
		t.Fatalf("file media_id = %q, want media-3", body.File.MediaID)
	}
}

func newTestWeComChannel(t *testing.T, messageBus *bus.MessageBus) *WeComChannel {
	t.Helper()

	cfg := config.WeComConfig{BotID: "bot-1"}
	cfg.SetSecret("secret-1")
	ch, err := NewChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("NewChannel() error = %v", err)
	}
	ch.ctx = context.Background()
	ch.routes = newReqIDStore(filepath.Join(t.TempDir(), "reqids.json"))
	return ch
}

func wecomTestJPEGData(t *testing.T) []byte {
	t.Helper()

	const jpegBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////" +
		"//////////////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////" +
		"//////////////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAb/xAAVEQEBAAAAAAAAAAAAAAAAAAAABf/aAAwDAQACEAMQAAAB6A//xAAVEAEBAAAAAAAAAAAAAAAAAAAAEf/aAAgBAQABBQJf/8QAFBEBAAAAAAAAAAAAAAAAAAAAEP/aAAgBAwEBPwF//8QAFBEBAAAAAAAAAAAAAAAAAAAAEP/aAAgBAgEBPwF//8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQAGPwJf/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQABPyFf/9k="

	return decodeTestBase64(t, jpegBase64)
}

func TestDecodeWeComUploadFinish_AcceptsNumericCreatedAt(t *testing.T) {
	t.Parallel()

	resp, err := decodeWeComEnvelopeBody[wecomUploadMediaFinishResponse](wecomEnvelope{
		Body: json.RawMessage(`{"type":"file","media_id":"media-1","created_at":1380000000}`),
	})
	if err != nil {
		t.Fatalf("decodeWeComEnvelopeBody() error = %v", err)
	}
	if resp.Type != "file" || resp.MediaID != "media-1" {
		t.Fatalf("response = %+v", resp)
	}
	if string(resp.CreatedAt) != "1380000000" {
		t.Fatalf("created_at = %s, want 1380000000", string(resp.CreatedAt))
	}
}

func wecomTestAck(body any) wecomEnvelope {
	var raw []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		raw = encoded
	}
	return wecomEnvelope{
		ErrCode: 0,
		ErrMsg:  "ok",
		Body:    raw,
	}
}
