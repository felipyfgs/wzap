package repo_test

import (
	"context"
	"testing"
	"time"

	"wzap/internal/model"
	"wzap/internal/repo"
)

func TestMessageRepositorySavePreservesBestDataAcrossHistoryAndLive(t *testing.T) {
	db := openTestDB(t)
	sessionID := insertTestSession(t, db, "message-repo")
	repository := repo.NewMessageRepository(db.Pool)

	chunkOrder := 3
	messageOrder := int64(11)
	historyTimestamp := time.Unix(1712000000, 0).UTC()
	liveTimestamp := historyTimestamp.Add(2 * time.Minute)

	historyMessage := &model.Message{
		ID:                  "same-id",
		SessionID:           sessionID,
		ChatJID:             "5511888888888@s.whatsapp.net",
		SenderJID:           "5511888888888@s.whatsapp.net",
		FromMe:              false,
		MsgType:             "text",
		Body:                "texto histórico",
		Source:              "history_sync",
		SourceSyncType:      "INITIAL_BOOTSTRAP",
		HistoryChunkOrder:   &chunkOrder,
		HistoryMessageOrder: &messageOrder,
		Raw:                 map[string]any{"origin": "history"},
		Timestamp:           historyTimestamp,
	}
	if err := repository.Save(context.Background(), historyMessage); err != nil {
		t.Fatalf("failed to save history message: %v", err)
	}

	liveMessage := &model.Message{
		ID:        "same-id",
		SessionID: sessionID,
		ChatJID:   "5511888888888@s.whatsapp.net",
		SenderJID: "5511888888888@s.whatsapp.net",
		FromMe:    false,
		MsgType:   "text",
		Body:      "",
		MediaType: "image/jpeg",
		MediaURL:  "https://cdn.example/media.jpg",
		Source:    "live",
		Raw:       map[string]any{"origin": "live"},
		Timestamp: liveTimestamp,
	}
	if err := repository.Save(context.Background(), liveMessage); err != nil {
		t.Fatalf("failed to save live message: %v", err)
	}

	stored, err := repository.FindByID(context.Background(), sessionID, "same-id")
	if err != nil {
		t.Fatalf("failed to load stored message: %v", err)
	}
	if stored.Source != "live" {
		t.Fatalf("expected source live after reconciliation, got %q", stored.Source)
	}
	if stored.SourceSyncType != "INITIAL_BOOTSTRAP" {
		t.Fatalf("expected sync type to be preserved, got %q", stored.SourceSyncType)
	}
	if stored.HistoryChunkOrder == nil || *stored.HistoryChunkOrder != chunkOrder {
		t.Fatalf("expected chunk order %d, got %v", chunkOrder, stored.HistoryChunkOrder)
	}
	if stored.HistoryMessageOrder == nil || *stored.HistoryMessageOrder != messageOrder {
		t.Fatalf("expected message order %d, got %v", messageOrder, stored.HistoryMessageOrder)
	}
	if stored.Body != "texto histórico" {
		t.Fatalf("expected history body to be preserved, got %q", stored.Body)
	}
	if stored.MediaType != "image/jpeg" {
		t.Fatalf("expected media type from live message, got %q", stored.MediaType)
	}
	if stored.MediaURL != "https://cdn.example/media.jpg" {
		t.Fatalf("expected media url from live message, got %q", stored.MediaURL)
	}
	if !stored.Timestamp.Equal(historyTimestamp) {
		t.Fatalf("expected earliest timestamp %v, got %v", historyTimestamp, stored.Timestamp)
	}

	messages, err := repository.FindByChat(context.Background(), sessionID, "5511888888888@s.whatsapp.net", 10, 0)
	if err != nil {
		t.Fatalf("failed to list messages by chat: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 canonical message after deduplication, got %d", len(messages))
	}
}

func TestFindUnimportedHistory(t *testing.T) {
	db := openTestDB(t)
	sessionID := insertTestSession(t, db, "find-unimported")
	repository := repo.NewMessageRepository(db.Pool)

	now := time.Now()

	msg1 := &model.Message{
		ID:        "hist-msg-1",
		SessionID: sessionID,
		ChatJID:   "5511999990001@s.whatsapp.net",
		FromMe:    false,
		MsgType:   "text",
		Body:      "hello from history",
		Source:    "history_sync",
		Timestamp: now.Add(-2 * time.Hour),
	}
	msg2 := &model.Message{
		ID:        "hist-msg-2",
		SessionID: sessionID,
		ChatJID:   "5511999990001@s.whatsapp.net",
		FromMe:    false,
		MsgType:   "text",
		Body:      "another history msg",
		Source:    "history_sync",
		Timestamp: now.Add(-1 * time.Hour),
	}
	liveMsg := &model.Message{
		ID:        "live-msg-1",
		SessionID: sessionID,
		ChatJID:   "5511999990001@s.whatsapp.net",
		FromMe:    false,
		MsgType:   "text",
		Body:      "live message",
		Source:    "live",
		Timestamp: now,
	}

	for _, m := range []*model.Message{msg1, msg2, liveMsg} {
		if err := repository.Save(context.Background(), m); err != nil {
			t.Fatalf("failed to save message: %v", err)
		}
	}

	t.Cleanup(func() {
		for _, id := range []string{"hist-msg-1", "hist-msg-2", "live-msg-1"} {
			_, _ = db.Pool.Exec(context.Background(), `DELETE FROM wz_messages WHERE id = $1 AND session_id = $2`, id, sessionID)
		}
	})

	since := now.Add(-24 * time.Hour)
	msgs, err := repository.FindUnimportedHistory(context.Background(), sessionID, since, 100, 0)
	if err != nil {
		t.Fatalf("FindUnimportedHistory failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 unimported history messages, got %d", len(msgs))
	}
	if msgs[0].ID != "hist-msg-1" {
		t.Fatalf("expected first message to be hist-msg-1, got %s", msgs[0].ID)
	}
	if msgs[1].ID != "hist-msg-2" {
		t.Fatalf("expected second message to be hist-msg-2, got %s", msgs[1].ID)
	}

	if err := repository.MarkImportedToChatwoot(context.Background(), sessionID, "hist-msg-1"); err != nil {
		t.Fatalf("MarkImportedToChatwoot failed: %v", err)
	}

	msgs, err = repository.FindUnimportedHistory(context.Background(), sessionID, since, 100, 0)
	if err != nil {
		t.Fatalf("FindUnimportedHistory after mark failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 unimported history message after mark, got %d", len(msgs))
	}
	if msgs[0].ID != "hist-msg-2" {
		t.Fatalf("expected remaining message to be hist-msg-2, got %s", msgs[0].ID)
	}
}

func TestMarkImportedToChatwoot(t *testing.T) {
	db := openTestDB(t)
	sessionID := insertTestSession(t, db, "mark-imported")
	repository := repo.NewMessageRepository(db.Pool)

	msg := &model.Message{
		ID:        "import-mark-msg",
		SessionID: sessionID,
		ChatJID:   "5511999990001@s.whatsapp.net",
		FromMe:    false,
		MsgType:   "text",
		Body:      "test mark",
		Source:    "history_sync",
		Timestamp: time.Now(),
	}
	if err := repository.Save(context.Background(), msg); err != nil {
		t.Fatalf("failed to save message: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM wz_messages WHERE id = $1 AND session_id = $2`, "import-mark-msg", sessionID)
	})

	stored, err := repository.FindByID(context.Background(), sessionID, "import-mark-msg")
	if err != nil {
		t.Fatalf("failed to find message: %v", err)
	}
	if stored.ImportedToChatwootAt != nil {
		t.Fatalf("expected imported_to_chatwoot_at to be nil before mark, got %v", stored.ImportedToChatwootAt)
	}

	if err := repository.MarkImportedToChatwoot(context.Background(), sessionID, "import-mark-msg"); err != nil {
		t.Fatalf("MarkImportedToChatwoot failed: %v", err)
	}

	stored, err = repository.FindByID(context.Background(), sessionID, "import-mark-msg")
	if err != nil {
		t.Fatalf("failed to find message after mark: %v", err)
	}
	if stored.ImportedToChatwootAt == nil {
		t.Fatal("expected imported_to_chatwoot_at to be set after mark")
	}
}
