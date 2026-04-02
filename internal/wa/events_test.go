package wa

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
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
