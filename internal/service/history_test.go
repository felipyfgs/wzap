package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
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

func TestHistoryServiceFiltersStubAndProtocolMessages(t *testing.T) {
	messageRepo := &messageRepoStub{}
	chatRepo := &chatRepoStub{}
	pool := async.NewPool("filter-test", 1, 10)
	defer pool.Shutdown(context.Background())

	historySvc := service.NewHistoryService(messageRepo, chatRepo, pool)

	stubType := waWeb.WebMessageInfo_GROUP_PARTICIPANT_ADD
	historySvc.PersistHistorySync("session-1", &events.HistorySync{
		Data: &waHistorySync.HistorySync{
			SyncType:   waHistorySync.HistorySync_INITIAL_BOOTSTRAP.Enum(),
			ChunkOrder: proto.Uint32(1),
			Conversations: []*waHistorySync.Conversation{
				{
					ID:   proto.String("5511999999999@s.whatsapp.net"),
					Name: proto.String("Test"),
					Messages: []*waHistorySync.HistorySyncMsg{
						{
							Message: &waWeb.WebMessageInfo{
								Key: &waCommon.MessageKey{
									RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
									ID:        proto.String("stub-msg-1"),
									FromMe:    proto.Bool(false),
								},
								MessageTimestamp: proto.Uint64(1713000000),
								MessageStubType:  &stubType,
							},
						},
						{
							Message: &waWeb.WebMessageInfo{
								Key: &waCommon.MessageKey{
									RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
									ID:        proto.String("proto-msg-1"),
									FromMe:    proto.Bool(true),
								},
								MessageTimestamp: proto.Uint64(1713000001),
								Message: &waE2E.Message{
									ProtocolMessage: &waE2E.ProtocolMessage{
										Type: waE2E.ProtocolMessage_HISTORY_SYNC_NOTIFICATION.Enum(),
									},
								},
							},
						},
						{
							Message: &waWeb.WebMessageInfo{
								Key: &waCommon.MessageKey{
									RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
									ID:        proto.String("skd-only-msg-1"),
									FromMe:    proto.Bool(false),
								},
								MessageTimestamp: proto.Uint64(1713000003),
								Message: &waE2E.Message{
									SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
										GroupID:                             proto.String("group-id"),
										AxolotlSenderKeyDistributionMessage: []byte{1, 2, 3},
									},
								},
							},
						},
						{
							Message: &waWeb.WebMessageInfo{
								Key: &waCommon.MessageKey{
									RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
									ID:        proto.String("skd-with-text-msg-1"),
									FromMe:    proto.Bool(false),
								},
								MessageTimestamp: proto.Uint64(1713000004),
								Message: &waE2E.Message{
									SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
										GroupID: proto.String("group-id"),
									},
									Conversation: proto.String("real text with skd"),
								},
							},
						},
						{
							Message: &waWeb.WebMessageInfo{
								Key: &waCommon.MessageKey{
									RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
									ID:        proto.String("real-msg-1"),
									FromMe:    proto.Bool(false),
								},
								MessageTimestamp: proto.Uint64(1713000002),
								Message: &waE2E.Message{
									Conversation: proto.String("real message"),
								},
							},
						},
					},
				},
			},
		},
	})

	waitUntil(t, 2*time.Second, func() bool {
		messageRepo.mu.Lock()
		count := len(messageRepo.messages)
		messageRepo.mu.Unlock()
		return count >= 2
	})

	time.Sleep(100 * time.Millisecond)

	messageRepo.mu.Lock()
	defer messageRepo.mu.Unlock()

	if len(messageRepo.messages) != 2 {
		t.Fatalf("expected 2 messages (stub, protocol, and standalone skd filtered), got %d", len(messageRepo.messages))
	}

	ids := make(map[string]string)
	for _, m := range messageRepo.messages {
		ids[m.ID] = m.MsgType
	}

	if ids["skd-only-msg-1"] != "" {
		t.Fatal("standalone senderKeyDistributionMessage should have been filtered")
	}
	if ids["skd-with-text-msg-1"] != "text" {
		t.Fatal("senderKeyDistribution with real content should be persisted as text")
	}
	if ids["real-msg-1"] != "text" {
		t.Fatal("real message should be persisted as text")
	}
}

type expiredDownloaderStub struct{}

func (d *expiredDownloaderStub) DownloadMediaByPath(_ context.Context, _, _ string, _, _, _ []byte, _ int, _ string) ([]byte, error) {
	return nil, whatsmeow.ErrMediaDownloadFailedWith403
}

type noopStorageStub struct{}

func (s *noopStorageStub) Upload(_ context.Context, _, _, _, _ string, _ bool, _ []byte, _ string, _ time.Time) (string, error) {
	return "", nil
}

type retryRecorderStub struct {
	mu    sync.Mutex
	times []time.Time
}

func (r *retryRecorderStub) RequestMediaRetry(_ context.Context, _, _, _, _ string, _ bool, _ string, _ time.Time, _, _, _ []byte, _ int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.times = append(r.times, time.Now())
	return nil
}

func TestHistoryServiceMediaRetryRateLimit(t *testing.T) {
	messageRepo := &messageRepoStub{}
	chatRepo := &chatRepoStub{}
	pool := async.NewPool("ratelimit-test", 1, 10)
	defer pool.Shutdown(context.Background())

	retryRecorder := &retryRecorderStub{}
	historySvc := service.NewHistoryService(messageRepo, chatRepo, pool)
	historySvc.SetMediaDownloader(&expiredDownloaderStub{})
	historySvc.SetMediaStorage(&noopStorageStub{})
	historySvc.SetMediaRetryRequester(retryRecorder)

	const retryCount = 3
	msgs := make([]*waHistorySync.HistorySyncMsg, retryCount)
	for i := range msgs {
		msgID := fmt.Sprintf("media-msg-%d", i)
		msgs[i] = &waHistorySync.HistorySyncMsg{
			Message: &waWeb.WebMessageInfo{
				Key: &waCommon.MessageKey{
					RemoteJID: proto.String("5511999999999@s.whatsapp.net"),
					ID:        proto.String(msgID),
					FromMe:    proto.Bool(false),
				},
				MessageTimestamp: proto.Uint64(1713000000),
				Message: &waE2E.Message{
					ImageMessage: &waE2E.ImageMessage{
						Mimetype:      proto.String("image/jpeg"),
						DirectPath:    proto.String("/path/to/image"),
						FileEncSHA256: []byte{1},
						FileSHA256:    []byte{2},
						MediaKey:      []byte{3},
						FileLength:    proto.Uint64(1024),
					},
				},
			},
		}
	}

	historySvc.PersistHistorySync("session-1", &events.HistorySync{
		Data: &waHistorySync.HistorySync{
			SyncType:   waHistorySync.HistorySync_INITIAL_BOOTSTRAP.Enum(),
			ChunkOrder: proto.Uint32(1),
			Conversations: []*waHistorySync.Conversation{
				{
					ID:       proto.String("5511999999999@s.whatsapp.net"),
					Name:     proto.String("Test"),
					Messages: msgs,
				},
			},
		},
	})

	waitUntil(t, 30*time.Second, func() bool {
		retryRecorder.mu.Lock()
		count := len(retryRecorder.times)
		retryRecorder.mu.Unlock()
		return count >= retryCount
	})

	retryRecorder.mu.Lock()
	defer retryRecorder.mu.Unlock()

	if len(retryRecorder.times) != retryCount {
		t.Fatalf("expected %d retries, got %d", retryCount, len(retryRecorder.times))
	}
	for i := 1; i < len(retryRecorder.times); i++ {
		gap := retryRecorder.times[i].Sub(retryRecorder.times[i-1])
		if gap < 3*time.Second {
			t.Fatalf("gap between retry %d and %d was %v, expected >= 3s", i-1, i, gap)
		}
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
