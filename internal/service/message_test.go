package service

import (
	"testing"

	"wzap/internal/dto"
)

func TestBuildContextInfo_Nil(t *testing.T) {
	if buildContextInfo(nil, nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestBuildContextInfo_EmptyMessageID(t *testing.T) {
	r := &dto.ReplyContext{MessageID: "", Participant: "5511@s.whatsapp.net"}
	if buildContextInfo(r, nil) != nil {
		t.Error("expected nil when MessageID is empty")
	}
}

func TestBuildContextInfo_Full(t *testing.T) {
	r := &dto.ReplyContext{
		MessageID:    "abc123",
		Participant:  "5511@s.whatsapp.net",
		MentionedJID: []string{"5522@s.whatsapp.net"},
	}
	ci := buildContextInfo(r, nil)
	if ci == nil {
		t.Fatal("expected non-nil ContextInfo")
	}
	if ci.GetStanzaID() != "abc123" {
		t.Errorf("unexpected StanzaID: %s", ci.GetStanzaID())
	}
	if ci.GetParticipant() != "5511@s.whatsapp.net" {
		t.Errorf("unexpected Participant: %s", ci.GetParticipant())
	}
	if len(ci.MentionedJID) != 1 || ci.MentionedJID[0] != "5522@s.whatsapp.net" {
		t.Error("unexpected MentionedJID")
	}
}

func TestBuildContextInfo_MentionedJIDsOnly(t *testing.T) {
	ci := buildContextInfo(nil, []string{"5511@s.whatsapp.net"})
	if ci == nil {
		t.Fatal("expected non-nil ContextInfo for MentionedJIDs only")
	}
	if len(ci.MentionedJID) != 1 || ci.MentionedJID[0] != "5511@s.whatsapp.net" {
		t.Errorf("unexpected MentionedJID: %v", ci.MentionedJID)
	}
}

func TestBuildSendOpts_Empty(t *testing.T) {
	opts := buildSendOpts("")
	if opts != nil {
		t.Error("expected nil for empty customID")
	}
}

func TestBuildSendOpts_WithID(t *testing.T) {
	opts := buildSendOpts("custom-id-123")
	if len(opts) != 1 {
		t.Fatalf("expected 1 opt, got %d", len(opts))
	}
	if opts[0].ID != "custom-id-123" {
		t.Errorf("unexpected ID: %s", opts[0].ID)
	}
}

func TestParseJID_Phone(t *testing.T) {
	jid, err := parseJID("5511999999999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jid.IsEmpty() {
		t.Error("expected non-empty JID for phone number")
	}
}

func TestParseJID_Full(t *testing.T) {
	jid, err := parseJID("5511999999999@s.whatsapp.net")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jid.IsEmpty() {
		t.Error("expected non-empty JID for full JID string")
	}
}

func TestParseJID_GroupJID(t *testing.T) {
	jid, err := parseJID("123456789@g.us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jid.IsEmpty() {
		t.Error("expected non-empty JID for group JID")
	}
}
