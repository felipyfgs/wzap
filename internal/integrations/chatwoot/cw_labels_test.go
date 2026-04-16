package chatwoot

import (
	"context"
	"testing"
)

func TestAddLabelToContact_EmptyDBURI(t *testing.T) {
	err := addLabelToContact(context.Background(), "", "my-inbox", 1)
	if err != nil {
		t.Errorf("expected nil error for empty dbURI, got: %v", err)
	}
}

func TestAddLabelToContact_EmptyInboxName(t *testing.T) {
	err := addLabelToContact(context.Background(), "postgres://user:pass@localhost/cw", "", 1)
	if err != nil {
		t.Errorf("expected nil error for empty inboxName, got: %v", err)
	}
}

func TestAddLabelToContact_BothEmpty(t *testing.T) {
	err := addLabelToContact(context.Background(), "", "", 1)
	if err != nil {
		t.Errorf("expected nil error when both are empty, got: %v", err)
	}
}

func TestAddLabelToContact_WhitespaceInboxName(t *testing.T) {
	err := addLabelToContact(context.Background(), "", "  ", 1)
	if err != nil {
		t.Errorf("expected nil error for empty dbURI with whitespace inboxName, got: %v", err)
	}
}
