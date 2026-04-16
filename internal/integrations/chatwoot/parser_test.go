package chatwoot

import (
	"encoding/json"
	"testing"

	"wzap/internal/model"
)

func TestParseMessagePayload(t *testing.T) {
	envelope := model.EventEnvelope{
		Event:     "Message",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	info := waMessageInfo{
		Chat:     "5511999999999@s.whatsapp.net",
		Sender:   "5511999999999@s.whatsapp.net",
		IsFromMe: false,
		IsGroup:  false,
		ID:       "msg-id-123",
		PushName: "Test User",
	}

	msgPayload := waMessagePayload{
		Info:    info,
		Message: map[string]any{"conversation": "Hello World"},
	}

	envelope.Data, _ = json.Marshal(msgPayload)
	payload, _ := json.Marshal(envelope)

	data, err := parseMessagePayload(payload)
	if err != nil {
		t.Fatalf("parseMessagePayload returned error: %v", err)
	}

	if data.Info.Chat != "5511999999999@s.whatsapp.net" {
		t.Errorf("expected Chat = 5511999999999@s.whatsapp.net, got %s", data.Info.Chat)
	}
	if data.Info.ID != "msg-id-123" {
		t.Errorf("expected ID = msg-id-123, got %s", data.Info.ID)
	}
	if data.Info.IsFromMe {
		t.Error("expected IsFromMe = false")
	}
	if data.Info.PushName != "Test User" {
		t.Errorf("expected PushName = Test User, got %s", data.Info.PushName)
	}
}

func TestParseMessagePayload_GroupMessage(t *testing.T) {
	envelope := model.EventEnvelope{
		Event:     "Message",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	info := waMessageInfo{
		Chat:     "5511999999999-123456789@g.us",
		Sender:   "5511888888888@s.whatsapp.net",
		IsFromMe: false,
		IsGroup:  true,
		ID:       "msg-id-456",
		PushName: "Group Participant",
	}

	msgPayload := waMessagePayload{
		Info:    info,
		Message: map[string]any{"conversation": "Group message"},
	}

	envelope.Data, _ = json.Marshal(msgPayload)
	payload, _ := json.Marshal(envelope)

	data, err := parseMessagePayload(payload)
	if err != nil {
		t.Fatalf("parseMessagePayload returned error: %v", err)
	}

	if !data.Info.IsGroup {
		t.Error("expected IsGroup = true")
	}
	if data.Info.Chat != "5511999999999-123456789@g.us" {
		t.Errorf("expected Chat = group JID, got %s", data.Info.Chat)
	}
}

func TestParseReceiptPayload(t *testing.T) {
	envelope := model.EventEnvelope{
		Event:     "Receipt",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	receiptData := waReceiptPayload{
		Type:       "read",
		MessageIDs: []string{"msg-1", "msg-2", "msg-3"},
	}

	envelope.Data, _ = json.Marshal(receiptData)
	payload, _ := json.Marshal(envelope)

	data, err := parseReceiptPayload(payload)
	if err != nil {
		t.Fatalf("parseReceiptPayload returned error: %v", err)
	}

	if data.Type != "read" {
		t.Errorf("expected Type = read, got %s", data.Type)
	}
	if len(data.MessageIDs) != 3 {
		t.Errorf("expected 3 MessageIDs, got %d", len(data.MessageIDs))
	}
}

func TestParseDeletePayload(t *testing.T) {
	envelope := model.EventEnvelope{
		Event:     "DeleteForMe",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	deleteData := waDeletePayload{
		MessageID: "msg-to-delete-123",
		Chat:      "5511999999999@s.whatsapp.net",
	}

	envelope.Data, _ = json.Marshal(deleteData)
	payload, _ := json.Marshal(envelope)

	data, err := parseDeletePayload(payload)
	if err != nil {
		t.Fatalf("parseDeletePayload returned error: %v", err)
	}

	if data.MessageID != "msg-to-delete-123" {
		t.Errorf("expected MessageID = msg-to-delete-123, got %s", data.MessageID)
	}
}

func TestConvertWAToCWMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold conversion",
			input:    "This is *bold* text",
			expected: "This is **bold** text",
		},
		{
			name:     "italic conversion",
			input:    "This is _italic_ text",
			expected: "This is *italic* text",
		},
		{
			name:     "strikethrough conversion",
			input:    "This is ~strikethrough~ text",
			expected: "This is ~~strikethrough~~ text",
		},
		{
			name:     "multiple formats",
			input:    "*bold* and _italic_ and ~strike~",
			expected: "**bold** and *italic* and ~~strike~~",
		},
		{
			name:     "no formatting",
			input:    "Plain text without formatting",
			expected: "Plain text without formatting",
		},
		{
			name:     "multiline with formatting",
			input:    "Line 1 with *bold*\nLine 2 with _italic_",
			expected: "Line 1 with **bold**\nLine 2 with *italic*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertWAToCWMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("convertWAToCWMarkdown(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertCWToWAMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold conversion",
			input:    "This is **bold** text",
			expected: "This is *bold* text",
		},
		{
			name:     "italic conversion",
			input:    "This is *italic* text",
			expected: "This is _italic_ text",
		},
		{
			name:     "strikethrough conversion",
			input:    "This is ~~strikethrough~~ text",
			expected: "This is ~strikethrough~ text",
		},
		{
			name:     "multiple formats",
			input:    "**bold** and *italic* and ~~strike~~",
			expected: "*bold* and _italic_ and ~strike~",
		},
		{
			name:     "no formatting",
			input:    "Plain text without formatting",
			expected: "Plain text without formatting",
		},
		{
			name:     "italic multi-word",
			input:    "This is *italic text* here",
			expected: "This is _italic text_ here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertCWToWAMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("convertCWToWAMarkdown(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnoreJID(t *testing.T) {
	tests := []struct {
		name         string
		chatJID      string
		ignoreGroups bool
		ignoreJIDs   []string
		expected     bool
	}{
		{
			name:         "no ignore - individual chat",
			chatJID:      "5511999999999@s.whatsapp.net",
			ignoreGroups: false,
			ignoreJIDs:   nil,
			expected:     false,
		},
		{
			name:         "ignore groups - group chat",
			chatJID:      "5511999999999-123456@g.us",
			ignoreGroups: true,
			ignoreJIDs:   nil,
			expected:     true,
		},
		{
			name:         "ignore specific JID",
			chatJID:      "5511888888888@s.whatsapp.net",
			ignoreGroups: false,
			ignoreJIDs:   []string{"5511888888888@s.whatsapp.net"},
			expected:     true,
		},
		{
			name:         "ignore all groups via ignoreJIDs",
			chatJID:      "5511999999999-123456@g.us",
			ignoreGroups: false,
			ignoreJIDs:   []string{"@g.us"},
			expected:     true,
		},
		{
			name:         "ignore all contacts via ignoreJIDs",
			chatJID:      "5511999999999@s.whatsapp.net",
			ignoreGroups: false,
			ignoreJIDs:   []string{"@s.whatsapp.net"},
			expected:     true,
		},
		{
			name:         "no match in ignoreJIDs",
			chatJID:      "5511999999999@s.whatsapp.net",
			ignoreGroups: false,
			ignoreJIDs:   []string{"5511888888888@s.whatsapp.net"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnoreJID(tt.chatJID, tt.ignoreGroups, tt.ignoreJIDs)
			if result != tt.expected {
				t.Errorf("shouldIgnoreJID(%q, %v, %v) = %v, expected %v", tt.chatJID, tt.ignoreGroups, tt.ignoreJIDs, result, tt.expected)
			}
		})
	}
}
