package wa

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestExtractMessageContent_Nil(t *testing.T) {
	msgType, body, mediaType := extractMessageContent(nil)
	if msgType != "unknown" || body != "" || mediaType != "" {
		t.Errorf("unexpected: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func TestExtractMessageContent_Conversation(t *testing.T) {
	msg := &waE2E.Message{Conversation: proto.String("hello")}
	msgType, body, mediaType := extractMessageContent(msg)
	if msgType != "text" || body != "hello" || mediaType != "" {
		t.Errorf("unexpected: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func TestExtractMessageContent_ExtendedText(t *testing.T) {
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("extended hello"),
		},
	}
	msgType, body, mediaType := extractMessageContent(msg)
	if msgType != "text" || body != "extended hello" {
		t.Errorf("unexpected: type=%s body=%s", msgType, body)
	}
	_ = mediaType
}

func TestExtractMessageContent_Image(t *testing.T) {
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:  proto.String("photo caption"),
			Mimetype: proto.String("image/jpeg"),
		},
	}
	msgType, body, mediaType := extractMessageContent(msg)
	if msgType != "image" || body != "photo caption" || mediaType != "image/jpeg" {
		t.Errorf("unexpected: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func TestExtractMessageContent_Audio(t *testing.T) {
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			Mimetype: proto.String("audio/ogg"),
		},
	}
	msgType, body, mediaType := extractMessageContent(msg)
	if msgType != "audio" || body != "" || mediaType != "audio/ogg" {
		t.Errorf("unexpected: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func TestExtractMessageContent_Document(t *testing.T) {
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			FileName: proto.String("report.pdf"),
			Mimetype: proto.String("application/pdf"),
		},
	}
	msgType, body, mediaType := extractMessageContent(msg)
	if msgType != "document" || body != "report.pdf" || mediaType != "application/pdf" {
		t.Errorf("unexpected: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func TestExtractMessageContent_Poll(t *testing.T) {
	msg := &waE2E.Message{
		PollCreationMessage: &waE2E.PollCreationMessage{
			Name: proto.String("Which option?"),
		},
	}
	msgType, body, _ := extractMessageContent(msg)
	if msgType != "poll" || body != "Which option?" {
		t.Errorf("unexpected: type=%s body=%s", msgType, body)
	}
}

func TestExtractMessageContent_Reaction(t *testing.T) {
	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Text: proto.String("👍"),
		},
	}
	msgType, body, _ := extractMessageContent(msg)
	if msgType != "reaction" || body != "👍" {
		t.Errorf("unexpected: type=%s body=%s", msgType, body)
	}
}

func TestExtractMessageContent_PollUpdate(t *testing.T) {
	msg := &waE2E.Message{
		PollUpdateMessage: &waE2E.PollUpdateMessage{},
	}
	msgType, _, _ := extractMessageContent(msg)
	if msgType != "poll_update" {
		t.Errorf("unexpected: type=%s", msgType)
	}
}

func TestExtractMessageContent_ProtocolMessage(t *testing.T) {
	msgType, body, mediaType := extractMessageContent(&waE2E.Message{
		ProtocolMessage: &waE2E.ProtocolMessage{},
	})
	if msgType != "unknown" || body != "" || mediaType != "" {
		t.Errorf("protocol should be unknown: type=%s body=%s media=%s", msgType, body, mediaType)
	}
}

func newTestManager() *Manager {
	return &Manager{
		clients:      make(map[string]*whatsmeow.Client),
		sessionNames: make(map[string]string),
	}
}

func TestHandleEventFiltersSenderKeyDistributionStandalone(t *testing.T) {
	called := false
	mgr := newTestManager()
	mgr.OnMessageReceived = func(_, _, _, _ string, _ bool, _, _, _ string, _ int64, _ any) {
		called = true
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.JID{User: "5511999999999", Server: "s.whatsapp.net"},
				Sender: types.JID{User: "5511888888888", Server: "s.whatsapp.net"},
			},
			ID:        "skd-msg-1",
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{
			SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
				GroupID:                             proto.String("group-id"),
				AxolotlSenderKeyDistributionMessage: []byte{1, 2, 3},
			},
		},
	}

	mgr.handleEvent("session-1", evt)

	if called {
		t.Fatal("OnMessageReceived should not be called for standalone senderKeyDistributionMessage")
	}
}

func TestHandleEventAllowsSenderKeyDistributionWithRealContent(t *testing.T) {
	called := false
	mgr := newTestManager()
	mgr.OnMessageReceived = func(_, _, _, _ string, _ bool, msgType, _, _ string, _ int64, _ any) {
		called = true
		if msgType != "text" {
			t.Fatalf("expected msgType text, got %q", msgType)
		}
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.JID{User: "5511999999999", Server: "s.whatsapp.net"},
				Sender: types.JID{User: "5511888888888", Server: "s.whatsapp.net"},
			},
			ID:        "skd-real-msg-1",
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{
			SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
				GroupID: proto.String("group-id"),
			},
			Conversation: proto.String("real text content"),
		},
	}

	mgr.handleEvent("session-1", evt)

	if !called {
		t.Fatal("OnMessageReceived should be called for senderKeyDistribution with real content")
	}
}
