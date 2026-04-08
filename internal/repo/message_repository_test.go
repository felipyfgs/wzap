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
