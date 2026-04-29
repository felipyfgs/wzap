package wautil

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestExtractForwarding_Nil(t *testing.T) {
	isFwd, score := ExtractForwarding(nil)
	if isFwd || score != 0 {
		t.Fatalf("expected (false,0) for nil msg, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_PlainText(t *testing.T) {
	msg := &waE2E.Message{Conversation: proto.String("oi")}
	isFwd, score := ExtractForwarding(msg)
	if isFwd || score != 0 {
		t.Fatalf("expected (false,0) for plain Conversation, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_ExtendedText(t *testing.T) {
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("oi"),
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(3),
			},
		},
	}
	isFwd, score := ExtractForwarding(msg)
	if !isFwd || score != 3 {
		t.Fatalf("expected (true,3), got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_ImageMessage(t *testing.T) {
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption: proto.String("foto"),
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(5),
			},
		},
	}
	isFwd, score := ExtractForwarding(msg)
	if !isFwd || score != 5 {
		t.Fatalf("expected (true,5), got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_NotForwarded(t *testing.T) {
	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			Caption: proto.String("video"),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID: proto.String("reply-target"),
			},
		},
	}
	isFwd, score := ExtractForwarding(msg)
	if isFwd || score != 0 {
		t.Fatalf("expected (false,0) for reply-only ContextInfo, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_EphemeralWrapper(t *testing.T) {
	inner := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("oi"),
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(2),
			},
		},
	}
	msg := &waE2E.Message{
		EphemeralMessage: &waE2E.FutureProofMessage{Message: inner},
	}
	isFwd, score := ExtractForwarding(msg)
	if !isFwd || score != 2 {
		t.Fatalf("expected (true,2) inside ephemeral wrapper, got (%v,%d)", isFwd, score)
	}
}

func TestExtractForwarding_ViewOnceV2Wrapper(t *testing.T) {
	inner := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(8),
			},
		},
	}
	msg := &waE2E.Message{
		ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: inner},
	}
	isFwd, score := ExtractForwarding(msg)
	if !isFwd || score != 8 {
		t.Fatalf("expected (true,8) inside viewOnceV2 wrapper, got (%v,%d)", isFwd, score)
	}
}
