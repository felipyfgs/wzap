package storage_test

import (
	"testing"
	"time"

	"wzap/internal/storage"
)

func TestMediaObjectKey(t *testing.T) {
	ts := time.Date(2026, 4, 14, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		params storage.MediaKeyParams
		want   string
	}{
		{
			name: "incoming image DM — senderJID == chatJID",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "5511999999999@s.whatsapp.net",
				SenderJID: "5511999999999@s.whatsapp.net",
				FromMe:    false,
				MessageID: "msg-001",
				MimeType:  "image/jpeg",
				Timestamp: ts,
			},
			want: "sess-abc/5511999999999@s.whatsapp.net/5511999999999@s.whatsapp.net/incoming/2026-04-14/image/msg-001.jpg",
		},
		{
			name: "outgoing audio DM — senderJID empty falls back to chatJID",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "5511999999999@s.whatsapp.net",
				SenderJID: "",
				FromMe:    true,
				MessageID: "msg-002",
				MimeType:  "audio/ogg; codecs=opus",
				Timestamp: ts,
			},
			want: "sess-abc/5511999999999@s.whatsapp.net/5511999999999@s.whatsapp.net/outgoing/2026-04-14/audio/msg-002.opus",
		},
		{
			name: "document pdf DM",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "5511999999999@s.whatsapp.net",
				SenderJID: "5511999999999@s.whatsapp.net",
				FromMe:    false,
				MessageID: "msg-003",
				MimeType:  "application/pdf",
				Timestamp: ts,
			},
			want: "sess-abc/5511999999999@s.whatsapp.net/5511999999999@s.whatsapp.net/incoming/2026-04-14/document/msg-003.pdf",
		},
		{
			name: "video mp4 group — sender individualizado",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "120363999999999999@g.us",
				SenderJID: "5511888888888@s.whatsapp.net",
				FromMe:    true,
				MessageID: "msg-004",
				MimeType:  "video/mp4",
				Timestamp: ts,
			},
			want: "sess-abc/120363999999999999@g.us/5511888888888@s.whatsapp.net/outgoing/2026-04-14/video/msg-004.mp4",
		},
		{
			name: "empty mime",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "5511999999999@s.whatsapp.net",
				SenderJID: "5511999999999@s.whatsapp.net",
				FromMe:    false,
				MessageID: "msg-005",
				MimeType:  "",
				Timestamp: ts,
			},
			want: "sess-abc/5511999999999@s.whatsapp.net/5511999999999@s.whatsapp.net/incoming/2026-04-14/other/msg-005",
		},
		{
			name: "empty chatJID",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "",
				SenderJID: "",
				FromMe:    false,
				MessageID: "msg-006",
				MimeType:  "image/png",
				Timestamp: ts,
			},
			want: "sess-abc/_unknown/_unknown/incoming/2026-04-14/image/msg-006.png",
		},
		{
			name: "JID with colon sanitized",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "lid:123456@lid",
				SenderJID: "lid:123456@lid",
				FromMe:    false,
				MessageID: "msg-007",
				MimeType:  "image/webp",
				Timestamp: ts,
			},
			want: "sess-abc/lid_123456@lid/lid_123456@lid/incoming/2026-04-14/image/msg-007.webp",
		},
		{
			name: "sticker webp",
			params: storage.MediaKeyParams{
				SessionID: "sess-abc",
				ChatJID:   "5511999999999@s.whatsapp.net",
				SenderJID: "5511999999999@s.whatsapp.net",
				FromMe:    false,
				MessageID: "msg-008",
				MimeType:  "image/webp",
				Timestamp: ts,
			},
			want: "sess-abc/5511999999999@s.whatsapp.net/5511999999999@s.whatsapp.net/incoming/2026-04-14/image/msg-008.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.MediaObjectKey(tt.params)
			if got != tt.want {
				t.Errorf("MediaObjectKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
