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
