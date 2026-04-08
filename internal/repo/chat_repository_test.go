package repo_test

import (
	"context"
	"testing"
	"time"

	"wzap/internal/model"
	"wzap/internal/repo"
)

func TestChatRepositoryUpsertAndListBySession(t *testing.T) {
	db := openTestDB(t)
	sessionID := insertTestSession(t, db, "chat-repo")
	repository := repo.NewChatRepository(db.Pool)

	chatJID := "5511999999999@s.whatsapp.net"
	name := "Fulano"
	displayName := "Fulano Silva"
	chatType := "direct"
	archived := true
	unreadCount := 5
	chunkOrder := 2
	lastMessageAt := time.Unix(1712500000, 0).UTC()
	conversationTimestamp := time.Unix(1712490000, 0).UTC()
	pnJID := "5511999999999@s.whatsapp.net"
	lidJID := "5511999999999@lid"
	lastMessageID := "history-msg-1"
	syncType := "INITIAL_BOOTSTRAP"

	if err := repository.Upsert(context.Background(), &model.ChatUpsert{
		SessionID:             sessionID,
		ChatJID:               chatJID,
		Name:                  &name,
		DisplayName:           &displayName,
		ChatType:              &chatType,
		Archived:              &archived,
		UnreadCount:           &unreadCount,
		LastMessageID:         &lastMessageID,
		LastMessageAt:         &lastMessageAt,
		ConversationTimestamp: &conversationTimestamp,
		PnJID:                 &pnJID,
		LidJID:                &lidJID,
		Source:                "history_sync",
		SourceSyncType:        &syncType,
		HistoryChunkOrder:     &chunkOrder,
		Raw:                   map[string]any{"source": "history"},
	}); err != nil {
		t.Fatalf("failed to upsert history chat: %v", err)
	}

	liveMessageID := "live-msg-1"
	liveAt := time.Unix(1712600000, 0).UTC()
	if err := repository.Upsert(context.Background(), &model.ChatUpsert{
		SessionID:     sessionID,
		ChatJID:       chatJID,
		ChatType:      &chatType,
		LastMessageID: &liveMessageID,
		LastMessageAt: &liveAt,
		Source:        "live",
	}); err != nil {
		t.Fatalf("failed to upsert live chat: %v", err)
	}

	chat, err := repository.FindBySessionAndChat(context.Background(), sessionID, chatJID)
	if err != nil {
		t.Fatalf("failed to load chat: %v", err)
	}
	if chat.Name != name {
		t.Fatalf("expected name %q, got %q", name, chat.Name)
	}
	if chat.DisplayName != displayName {
		t.Fatalf("expected display name %q, got %q", displayName, chat.DisplayName)
	}
	if chat.Source != "live" {
		t.Fatalf("expected source live, got %q", chat.Source)
	}
	if chat.SourceSyncType != syncType {
		t.Fatalf("expected source sync type %q, got %q", syncType, chat.SourceSyncType)
	}
	if chat.LastMessageID != liveMessageID {
		t.Fatalf("expected last message ID %q, got %q", liveMessageID, chat.LastMessageID)
	}
	if chat.LastMessageAt == nil || !chat.LastMessageAt.Equal(liveAt) {
		t.Fatalf("expected last message timestamp %v, got %v", liveAt, chat.LastMessageAt)
	}
	if chat.HistoryChunkOrder == nil || *chat.HistoryChunkOrder != chunkOrder {
		t.Fatalf("expected chunk order %d, got %v", chunkOrder, chat.HistoryChunkOrder)
	}
	if chat.PnJID != pnJID || chat.LidJID != lidJID {
		t.Fatalf("expected pn/lid aliases to be preserved, got pn=%q lid=%q", chat.PnJID, chat.LidJID)
	}

	chats, err := repository.ListBySession(context.Background(), sessionID, 10, 0)
	if err != nil {
		t.Fatalf("failed to list chats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].ChatJID != chatJID {
		t.Fatalf("expected chat jid %q, got %q", chatJID, chats[0].ChatJID)
	}
}
