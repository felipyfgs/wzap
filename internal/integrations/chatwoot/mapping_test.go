package chatwoot

import (
	"context"
	"errors"
	"testing"
)

func TestBackfillCloudRefs_ReturnsUnavailableWithoutDatabaseURI(t *testing.T) {
	svc := &Service{
		repo:    &mockRepo{cfg: &Config{SessionID: "sess", InboxType: "cloud", AccountID: 1, InboxID: 1}},
		msgRepo: &mockMsgRepo{},
	}

	_, err := svc.BackfillCloudRefs(context.Background(), "sess")
	if !errors.Is(err, ErrBackfillUnavailable) {
		t.Fatalf("expected ErrBackfillUnavailable, got %v", err)
	}
}

func TestResolveMessageBySourceID_ReturnsNotFoundWhenDatabaseURIEmpty(t *testing.T) {
	cfg := &Config{SessionID: "sess", AccountID: 1, DatabaseURI: ""}

	ref, ok, err := ResolveMessageBySourceID(context.Background(), cfg, "WAID:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false when database_uri is empty")
	}
	if ref != nil {
		t.Fatalf("expected nil ref, got %+v", ref)
	}
}

func TestResolveAndPersistMessageRef_NoDatabaseURI(t *testing.T) {
	svc := &Service{
		repo:    &mockRepo{cfg: &Config{SessionID: "sess", AccountID: 1}},
		msgRepo: &mockMsgRepo{},
	}

	cfg := &Config{SessionID: "sess", AccountID: 1}
	ref, ok := svc.resolveAndPersistMessageRef(context.Background(), cfg, "abc")
	if ok {
		t.Fatal("expected ok=false when database_uri is empty")
	}
	if ref != nil {
		t.Fatalf("expected nil ref, got %+v", ref)
	}
}
