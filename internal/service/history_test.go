package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"wzap/internal/async"
	"wzap/internal/model"
	"wzap/internal/service"
)

type messageRepoStub struct {
	mu       sync.Mutex
	messages []*model.Message
}

func (r *messageRepoStub) Save(_ context.Context, msg *model.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := *msg
	r.messages = append(r.messages, &clone)
	return nil
}

type chatRepoStub struct {
	mu    sync.Mutex
	chats []*model.ChatUpsert
}

func (r *chatRepoStub) Upsert(_ context.Context, chat *model.ChatUpsert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := *chat
	r.chats = append(r.chats, &clone)
	return nil
}

func TestHistoryServicePersistHistorySyncHandlesMultipleConversationsAndChunkMetadata(t *testing.T) {
	messageRepo := &messageRepoStub{}
	chatRepo := &chatRepoStub{}
	pool := async.NewPool("history-test", 1, 10)
	defer pool.Shutdown(context.Background())

	historySvc := service.NewHistoryService(messageRepo, chatRepo, pool)

	historySvc.PersistHistorySync("session-1", newHistorySyncEvent(2,
		newConversation("5511999999999@lid", "5511999999999@s.whatsapp.net", "5511999999999@lid", "Cliente 1", "msg-2", 1713000010, false),
		newConversation("120363040000000000@g.us", "", "", "Grupo 1", "msg-3", 1713000020, true),
	))
	historySvc.PersistHistorySync("session-1", newHistorySyncEvent(1,
		newConversation("5511888888888@lid", "5511888888888@s.whatsapp.net", "5511888888888@lid", "Cliente 2", "msg-1", 1713000000, false),
	))

	waitUntil(t, 2*time.Second, func() bool {
		chatRepo.mu.Lock()
		chatCount := len(chatRepo.chats)
		chatRepo.mu.Unlock()
		messageRepo.mu.Lock()
		messageCount := len(messageRepo.messages)
		messageRepo.mu.Unlock()
		return chatCount == 3 && messageCount == 3
	})

	chatRepo.mu.Lock()
	defer chatRepo.mu.Unlock()
	messageRepo.mu.Lock()
	defer messageRepo.mu.Unlock()

	if len(chatRepo.chats) != 3 {
		t.Fatalf("expected 3 chat upserts, got %d", len(chatRepo.chats))
	}
	if len(messageRepo.messages) != 3 {
		t.Fatalf("expected 3 message upserts, got %d", len(messageRepo.messages))
	}

	firstChat := chatRepo.chats[0]
	if firstChat.Source != "history_sync" {
		t.Fatalf("expected chat source history_sync, got %q", firstChat.Source)
	}
	if firstChat.HistoryChunkOrder == nil || *firstChat.HistoryChunkOrder != 2 {
		t.Fatalf("expected first chunk order 2, got %v", firstChat.HistoryChunkOrder)
	}
	if firstChat.ChatJID != "5511999999999@s.whatsapp.net" {
		t.Fatalf("expected PN chat jid to be canonicalized, got %q", firstChat.ChatJID)
	}
	if firstChat.PnJID == nil || *firstChat.PnJID != "5511999999999@s.whatsapp.net" {
		t.Fatalf("expected PN alias to be registered, got %v", firstChat.PnJID)
	}
	if firstChat.LidJID == nil || *firstChat.LidJID != "5511999999999@lid" {
		t.Fatalf("expected LID alias to be registered, got %v", firstChat.LidJID)
	}

	groupMessage := messageRepo.messages[1]
	if groupMessage.HistoryChunkOrder == nil || *groupMessage.HistoryChunkOrder != 2 {
		t.Fatalf("expected group message chunk order 2, got %v", groupMessage.HistoryChunkOrder)
	}
	if groupMessage.SourceSyncType != "INITIAL_BOOTSTRAP" {
		t.Fatalf("expected sync type INITIAL_BOOTSTRAP, got %q", groupMessage.SourceSyncType)
	}
	if groupMessage.ChatJID != "120363040000000000@g.us" {
		t.Fatalf("expected group chat jid, got %q", groupMessage.ChatJID)
	}

	olderMessage := messageRepo.messages[2]
	if olderMessage.HistoryChunkOrder == nil || *olderMessage.HistoryChunkOrder != 1 {
		t.Fatalf("expected older chunk order 1, got %v", olderMessage.HistoryChunkOrder)
	}
	if olderMessage.ChatJID != "5511888888888@s.whatsapp.net" {
		t.Fatalf("expected older message PN chat jid, got %q", olderMessage.ChatJID)
	}
}

func newHistorySyncEvent(chunkOrder uint32, conversations ...*waHistorySync.Conversation) *events.HistorySync {
	return &events.HistorySync{
		Data: &waHistorySync.HistorySync{
			SyncType:      waHistorySync.HistorySync_INITIAL_BOOTSTRAP.Enum(),
			ChunkOrder:    proto.Uint32(chunkOrder),
			Progress:      proto.Uint32(100),
			Conversations: conversations,
			Pushnames: []*waHistorySync.Pushname{
				{ID: proto.String("5511999999999@s.whatsapp.net"), Pushname: proto.String("Cliente 1")},
				{ID: proto.String("5511888888888@s.whatsapp.net"), Pushname: proto.String("Cliente 2")},
			},
			PhoneNumberToLidMappings: []*waHistorySync.PhoneNumberToLIDMapping{
				{PnJID: proto.String("5511999999999@s.whatsapp.net"), LidJID: proto.String("5511999999999@lid")},
				{PnJID: proto.String("5511888888888@s.whatsapp.net"), LidJID: proto.String("5511888888888@lid")},
			},
		},
	}
}

func newConversation(chatJID, pnJID, lidJID, displayName, messageID string, timestamp uint64, isGroup bool) *waHistorySync.Conversation {
	name := displayName
	archived := false
	pinned := uint32(1)
	readOnly := isGroup
	unreadCount := uint32(3)
	unreadMentions := uint32(1)

	remoteJID := chatJID
	participant := ""
	if isGroup {
		participant = "5511777777777@s.whatsapp.net"
	}

	return &waHistorySync.Conversation{
		ID:                    proto.String(chatJID),
		DisplayName:           proto.String(displayName),
		Name:                  proto.String(name),
		PnJID:                 stringOrNil(pnJID),
		LidJID:                stringOrNil(lidJID),
		Archived:              proto.Bool(archived),
		Pinned:                proto.Uint32(pinned),
		ReadOnly:              proto.Bool(readOnly),
		UnreadCount:           proto.Uint32(unreadCount),
		UnreadMentionCount:    proto.Uint32(unreadMentions),
		LastMsgTimestamp:      proto.Uint64(timestamp),
		ConversationTimestamp: proto.Uint64(timestamp),
		Messages: []*waHistorySync.HistorySyncMsg{
			{
				Message: &waWeb.WebMessageInfo{
					Key: &waCommon.MessageKey{
						RemoteJID:   proto.String(remoteJID),
						ID:          proto.String(messageID),
						FromMe:      proto.Bool(false),
						Participant: stringOrNil(participant),
					},
					MessageTimestamp: proto.Uint64(timestamp),
					Message: &waE2E.Message{
						Conversation: proto.String(displayName + " body"),
					},
				},
				MsgOrderID: proto.Uint64(timestamp),
			},
		},
	}
}

func stringOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return proto.String(value)
}

func waitUntil(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for async persistence")
}
