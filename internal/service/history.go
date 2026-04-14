package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/async"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/storage"
)

type messageHistoryRepository interface {
	Save(ctx context.Context, msg *model.Message) error
}

type chatHistoryRepository interface {
	Upsert(ctx context.Context, chat *model.ChatUpsert) error
}

type MediaDownloader interface {
	DownloadMediaByPath(ctx context.Context, sessionID, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, mediaType string) ([]byte, error)
}

type MediaStorage interface {
	Upload(ctx context.Context, sessionID, messageID, chatJID, senderJID string, fromMe bool, data []byte, mimeType string, timestamp time.Time) (string, error)
}

type MediaRetryRequester interface {
	RequestMediaRetry(ctx context.Context, sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, encFileHash, fileHash, mediaKey []byte, fileLength int) error
}

type jidAliasMaps struct {
	pnToLID       map[string]string
	lidToPN       map[string]string
	pushnameByJID map[string]string
}

const mediaRetryInterval = 3 * time.Second

type HistoryService struct {
	messageRepo    messageHistoryRepository
	chatRepo       chatHistoryRepository
	pool           *async.Pool
	downloader     MediaDownloader
	storage        MediaStorage
	retryRequester MediaRetryRequester
}

func NewHistoryService(messageRepo messageHistoryRepository, chatRepo chatHistoryRepository, pool *async.Pool) *HistoryService {
	return &HistoryService{messageRepo: messageRepo, chatRepo: chatRepo, pool: pool}
}

func (s *HistoryService) SetMediaDownloader(d MediaDownloader)         { s.downloader = d }
func (s *HistoryService) SetMediaStorage(ms MediaStorage)              { s.storage = ms }
func (s *HistoryService) SetMediaRetryRequester(r MediaRetryRequester) { s.retryRequester = r }

func NewMinioMediaStorage(m *storage.Minio) MediaStorage {
	return &minioMediaStorage{minio: m}
}

type minioMediaStorage struct {
	minio *storage.Minio
}

func (m *minioMediaStorage) Upload(ctx context.Context, sessionID, messageID, chatJID, senderJID string, fromMe bool, data []byte, mimeType string, timestamp time.Time) (string, error) {
	key := storage.MediaObjectKey(storage.MediaKeyParams{
		SessionID: sessionID,
		ChatJID:   chatJID,
		SenderJID: senderJID,
		FromMe:    fromMe,
		MessageID: messageID,
		MimeType:  mimeType,
		Timestamp: timestamp,
	})
	if err := m.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return "", fmt.Errorf("failed to upload media to S3: %w", err)
	}
	return key, nil
}

func (s *HistoryService) PersistMessage(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw any) {
	_ = s.pool.Submit(func(ctx context.Context) {
		msg := &model.Message{
			ID:        messageID,
			SessionID: sessionID,
			ChatJID:   chatJID,
			SenderJID: senderJID,
			FromMe:    fromMe,
			MsgType:   msgType,
			Body:      body,
			MediaType: mediaType,
			Source:    "live",
			Raw:       raw,
			Timestamp: time.Unix(timestamp, 0),
		}

		if err := s.messageRepo.Save(ctx, msg); err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Failed to persist message")
		}

		if s.chatRepo == nil || chatJID == "" {
			return
		}

		chat := buildLiveChatUpsert(sessionID, chatJID, messageID, timestamp)
		if err := s.chatRepo.Upsert(ctx, chat); err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("chat", chatJID).Msg("Failed to persist canonical chat from live message")
		}
	})
}

func (s *HistoryService) PersistHistorySync(sessionID string, syncEvent *events.HistorySync) {
	if syncEvent == nil || syncEvent.Data == nil {
		return
	}

	_ = s.pool.Submit(func(ctx context.Context) {
		aliases := buildJIDAliasMaps(syncEvent.Data)
		syncType := syncEvent.Data.GetSyncType().String()
		chunkOrder := int(syncEvent.Data.GetChunkOrder())

		for _, conversation := range syncEvent.Data.GetConversations() {
			chat := buildHistoryChatUpsert(sessionID, conversation, aliases, syncType, chunkOrder)
			if chat != nil && s.chatRepo != nil {
				if err := s.chatRepo.Upsert(ctx, chat); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("chat", chat.ChatJID).Msg("Failed to persist canonical chat from history sync")
				}
			}

			if s.messageRepo == nil {
				continue
			}

			for _, historyMessage := range conversation.GetMessages() {
				msg := buildHistoryMessage(sessionID, conversation, historyMessage, aliases, syncType, chunkOrder)
				if msg == nil {
					continue
				}

				if s.downloader != nil && s.storage != nil && msg.MediaURL == "" {
					info := historyMessage.GetMessage()
					if info != nil {
						protoMsg := info.GetMessage()
						directPath, encFileHash, fileHash, mediaKey, fileLength, hasMedia := extractMediaDownloadInfo(protoMsg)
						if hasMedia && directPath != "" {
							dlCtx, dlCancel := context.WithTimeout(ctx, 30*time.Second)
							data, err := s.downloader.DownloadMediaByPath(dlCtx, sessionID, directPath, encFileHash, fileHash, mediaKey, fileLength, msg.MediaType)
							dlCancel()
							if err != nil {
								if s.retryRequester != nil && isExpiredMediaError(err) {
									logger.Debug().Str("component", "service").Str("session", sessionID).Str("mid", msg.ID).Msg("History sync media retry: aguardando rate limit")
									time.Sleep(mediaRetryInterval)
									if retryErr := s.retryRequester.RequestMediaRetry(ctx, sessionID, msg.ID, msg.ChatJID, msg.SenderJID, msg.FromMe, msg.MediaType, msg.Timestamp, encFileHash, fileHash, mediaKey, fileLength); retryErr != nil {
										logger.Warn().Str("component", "service").Err(retryErr).Str("session", sessionID).Str("mid", msg.ID).Msg("History sync media: falha ao solicitar retry")
									} else {
										logger.Debug().Str("component", "service").Str("session", sessionID).Str("mid", msg.ID).Msg("History sync media: retry solicitado ao celular")
									}
								} else {
									logger.Debug().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", msg.ID).Str("mediaType", msg.MediaType).Msg("History sync media expired or unavailable, skipping")
								}
							} else if len(data) > 0 {
								mediaURL, err := s.storage.Upload(ctx, sessionID, msg.ID, msg.ChatJID, msg.SenderJID, msg.FromMe, data, msg.MediaType, msg.Timestamp)
								if err != nil {
									logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("Failed to upload history sync media to storage")
								} else {
									msg.MediaURL = mediaURL
								}
							}
						}
					}
				}

				if err := s.messageRepo.Save(ctx, msg); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("Failed to persist history sync message")
				}
			}
		}
	})
}

func buildLiveChatUpsert(sessionID, chatJID, lastMessageID string, timestamp int64) *model.ChatUpsert {
	chatType := inferChatType(chatJID)
	lastMessageAt := time.Unix(timestamp, 0)

	return &model.ChatUpsert{
		SessionID:     sessionID,
		ChatJID:       chatJID,
		ChatType:      &chatType,
		LastMessageID: stringPtr(lastMessageID),
		LastMessageAt: &lastMessageAt,
		Source:        "live",
	}
}

func buildHistoryChatUpsert(sessionID string, conversation *waHistorySync.Conversation, aliases jidAliasMaps, syncType string, chunkOrder int) *model.ChatUpsert {
	if conversation == nil {
		return nil
	}

	chatJID := resolveConversationChatJID(conversation, aliases)
	if chatJID == "" {
		return nil
	}

	name := firstNonEmpty(conversation.GetName(), aliases.pushnameByJID[chatJID], aliases.pushnameByJID[conversation.GetPnJID()], aliases.pushnameByJID[conversation.GetLidJID()])
	displayName := firstNonEmpty(conversation.GetDisplayName(), name)
	chatType := inferChatType(chatJID)
	archived := conversation.GetArchived()
	pinned := int(conversation.GetPinned())
	readOnly := conversation.GetReadOnly()
	markedAsUnread := conversation.GetMarkedAsUnread()
	unreadCount := int(conversation.GetUnreadCount())
	unreadMentionCount := int(conversation.GetUnreadMentionCount())
	pnJID, lidJID := resolveConversationAliases(conversation, aliases)
	username := conversation.GetUsername()
	accountLID := conversation.GetAccountLid()
	lastMessageID := conversationLastMessageID(conversation)
	lastMessageAt := unixTimePtr(uint64ToInt64(conversation.GetLastMsgTimestamp()))
	conversationTimestamp := unixTimePtr(uint64ToInt64(conversation.GetConversationTimestamp()))
	raw := map[string]any{
		"id":                   conversation.GetID(),
		"newJID":               conversation.GetNewJID(),
		"oldJID":               conversation.GetOldJID(),
		"muteEndTime":          conversation.GetMuteEndTime(),
		"createdAt":            conversation.GetCreatedAt(),
		"createdBy":            conversation.GetCreatedBy(),
		"endOfHistoryTransfer": conversation.GetEndOfHistoryTransfer(),
		"ephemeralExpiration":  conversation.GetEphemeralExpiration(),
	}

	return &model.ChatUpsert{
		SessionID:             sessionID,
		ChatJID:               chatJID,
		Name:                  stringPtr(name),
		DisplayName:           stringPtr(displayName),
		ChatType:              &chatType,
		Archived:              &archived,
		Pinned:                &pinned,
		ReadOnly:              &readOnly,
		MarkedAsUnread:        &markedAsUnread,
		UnreadCount:           &unreadCount,
		UnreadMentionCount:    &unreadMentionCount,
		LastMessageID:         stringPtr(lastMessageID),
		LastMessageAt:         lastMessageAt,
		ConversationTimestamp: conversationTimestamp,
		PnJID:                 stringPtr(pnJID),
		LidJID:                stringPtr(lidJID),
		Username:              stringPtr(username),
		AccountLID:            stringPtr(accountLID),
		Source:                "history_sync",
		SourceSyncType:        stringPtr(syncType),
		HistoryChunkOrder:     intPtr(chunkOrder),
		Raw:                   raw,
	}
}

func buildHistoryMessage(sessionID string, conversation *waHistorySync.Conversation, historyMessage *waHistorySync.HistorySyncMsg, aliases jidAliasMaps, syncType string, chunkOrder int) *model.Message {
	if historyMessage == nil {
		return nil
	}

	info := historyMessage.GetMessage()
	if info == nil || info.GetKey() == nil {
		return nil
	}

	if info.GetMessageStubType() != 0 {
		return nil
	}

	protoMsg := info.GetMessage()
	if protoMsg != nil && protoMsg.GetProtocolMessage() != nil {
		return nil
	}

	if protoMsg != nil && protoMsg.GetSenderKeyDistributionMessage() != nil {
		msgType, _, _ := extractMessageContent(protoMsg)
		if msgType == "unknown" {
			return nil
		}
	}

	key := info.GetKey()
	messageID := key.GetID()
	if messageID == "" {
		return nil
	}

	chatJID := firstNonEmpty(resolveMessageChatJID(info, conversation, aliases), resolveConversationChatJID(conversation, aliases))
	if chatJID == "" {
		return nil
	}

	senderJID := resolveMessageSenderJID(info, chatJID)
	msgType, body, mediaType := extractMessageContent(info.GetMessage())
	timestamp := int64(info.GetMessageTimestamp())
	if timestamp == 0 {
		if conversation != nil {
			timestamp = uint64ToInt64(conversation.GetConversationTimestamp())
		}
		if timestamp == 0 {
			timestamp = time.Now().Unix()
		}
	}

	var messageOrder *int64
	if order := int64(historyMessage.GetMsgOrderID()); order > 0 {
		messageOrder = &order
	}

	return &model.Message{
		ID:                  messageID,
		SessionID:           sessionID,
		ChatJID:             chatJID,
		SenderJID:           senderJID,
		FromMe:              key.GetFromMe(),
		MsgType:             msgType,
		Body:                body,
		MediaType:           mediaType,
		Source:              "history_sync",
		SourceSyncType:      syncType,
		HistoryChunkOrder:   intPtr(chunkOrder),
		HistoryMessageOrder: messageOrder,
		Raw:                 info,
		Timestamp:           time.Unix(timestamp, 0),
	}
}

func buildJIDAliasMaps(history *waHistorySync.HistorySync) jidAliasMaps {
	aliases := jidAliasMaps{
		pnToLID:       make(map[string]string),
		lidToPN:       make(map[string]string),
		pushnameByJID: make(map[string]string),
	}

	if history == nil {
		return aliases
	}

	for _, mapping := range history.GetPhoneNumberToLidMappings() {
		pn := mapping.GetPnJID()
		lid := mapping.GetLidJID()
		if pn != "" && lid != "" {
			aliases.pnToLID[pn] = lid
			aliases.lidToPN[lid] = pn
		}
	}

	for _, pushname := range history.GetPushnames() {
		id := pushname.GetID()
		name := pushname.GetPushname()
		if id != "" && name != "" {
			aliases.pushnameByJID[id] = name
		}
	}

	return aliases
}

func resolveConversationChatJID(conversation *waHistorySync.Conversation, aliases jidAliasMaps) string {
	if conversation == nil {
		return ""
	}

	chatJID := firstNonEmpty(conversation.GetNewJID(), conversation.GetID(), conversation.GetPnJID(), conversation.GetLidJID())
	if chatJID == "" {
		return ""
	}
	if resolved, ok := aliases.lidToPN[chatJID]; ok {
		return firstNonEmpty(conversation.GetPnJID(), resolved, chatJID)
	}
	return chatJID
}

func resolveConversationAliases(conversation *waHistorySync.Conversation, aliases jidAliasMaps) (string, string) {
	if conversation == nil {
		return "", ""
	}

	pnJID := conversation.GetPnJID()
	lidJID := conversation.GetLidJID()

	if pnJID == "" && lidJID != "" {
		pnJID = aliases.lidToPN[lidJID]
	}
	if lidJID == "" && pnJID != "" {
		lidJID = aliases.pnToLID[pnJID]
	}

	return pnJID, lidJID
}

func resolveMessageChatJID(info *waWeb.WebMessageInfo, conversation *waHistorySync.Conversation, aliases jidAliasMaps) string {
	if info == nil || info.GetKey() == nil {
		return resolveConversationChatJID(conversation, aliases)
	}

	chatJID := info.GetKey().GetRemoteJID()
	if chatJID == "" {
		return resolveConversationChatJID(conversation, aliases)
	}
	if resolved, ok := aliases.lidToPN[chatJID]; ok {
		return resolved
	}
	return chatJID
}

func resolveMessageSenderJID(info *waWeb.WebMessageInfo, chatJID string) string {
	if info == nil || info.GetKey() == nil {
		return ""
	}

	if senderJID := info.GetKey().GetParticipant(); senderJID != "" {
		return senderJID
	}
	if senderJID := info.GetParticipant(); senderJID != "" {
		return senderJID
	}
	if info.GetKey().GetFromMe() {
		return ""
	}
	return chatJID
}

func conversationLastMessageID(conversation *waHistorySync.Conversation) string {
	if conversation == nil {
		return ""
	}
	for i := len(conversation.GetMessages()) - 1; i >= 0; i-- {
		message := conversation.GetMessages()[i]
		if message == nil || message.GetMessage() == nil || message.GetMessage().GetKey() == nil {
			continue
		}
		if messageID := message.GetMessage().GetKey().GetID(); messageID != "" {
			return messageID
		}
	}
	return ""
}

func inferChatType(chatJID string) string {
	switch {
	case strings.HasPrefix(chatJID, "status@"):
		return "status"
	case strings.HasSuffix(chatJID, "@g.us"):
		return "group"
	case strings.HasSuffix(chatJID, "@broadcast"):
		return "broadcast"
	case strings.Contains(chatJID, "@newsletter"):
		return "newsletter"
	case chatJID == "":
		return "unknown"
	default:
		return "direct"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func intPtr(value int) *int {
	return &value
}

func unixTimePtr(timestamp int64) *time.Time {
	if timestamp <= 0 {
		return nil
	}
	t := time.Unix(timestamp, 0)
	return &t
}

func uint64ToInt64(value uint64) int64 {
	if value > uint64(math.MaxInt64) {
		return 0
	}
	return int64(value)
}

func extractMediaDownloadInfo(msg *waE2E.Message) (directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, ok bool) {
	if msg == nil {
		return "", nil, nil, nil, 0, false
	}
	switch {
	case msg.GetImageMessage() != nil:
		im := msg.GetImageMessage()
		return im.GetDirectPath(), im.GetFileEncSHA256(), im.GetFileSHA256(), im.GetMediaKey(), int(im.GetFileLength()), true
	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		return vm.GetDirectPath(), vm.GetFileEncSHA256(), vm.GetFileSHA256(), vm.GetMediaKey(), int(vm.GetFileLength()), true
	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		return am.GetDirectPath(), am.GetFileEncSHA256(), am.GetFileSHA256(), am.GetMediaKey(), int(am.GetFileLength()), true
	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		return dm.GetDirectPath(), dm.GetFileEncSHA256(), dm.GetFileSHA256(), dm.GetMediaKey(), int(dm.GetFileLength()), true
	case msg.GetStickerMessage() != nil:
		sm := msg.GetStickerMessage()
		return sm.GetDirectPath(), sm.GetFileEncSHA256(), sm.GetFileSHA256(), sm.GetMediaKey(), int(sm.GetFileLength()), true
	default:
		return "", nil, nil, nil, 0, false
	}
}

func extractMessageContent(msg *waE2E.Message) (msgType, body, mediaType string) {
	if msg == nil {
		return "unknown", "", ""
	}
	switch {
	case msg.GetConversation() != "":
		return "text", msg.GetConversation(), ""
	case msg.GetExtendedTextMessage() != nil:
		return "text", msg.GetExtendedTextMessage().GetText(), ""
	case msg.GetImageMessage() != nil:
		return "image", msg.GetImageMessage().GetCaption(), msg.GetImageMessage().GetMimetype()
	case msg.GetVideoMessage() != nil:
		return "video", msg.GetVideoMessage().GetCaption(), msg.GetVideoMessage().GetMimetype()
	case msg.GetAudioMessage() != nil:
		return "audio", "", msg.GetAudioMessage().GetMimetype()
	case msg.GetDocumentMessage() != nil:
		return "document", msg.GetDocumentMessage().GetFileName(), msg.GetDocumentMessage().GetMimetype()
	case msg.GetStickerMessage() != nil:
		return "sticker", "", msg.GetStickerMessage().GetMimetype()
	case msg.GetContactMessage() != nil:
		return "contact", msg.GetContactMessage().GetDisplayName(), ""
	case msg.GetLocationMessage() != nil:
		return "location", msg.GetLocationMessage().GetName(), ""
	case msg.GetListMessage() != nil:
		return "list", msg.GetListMessage().GetTitle(), ""
	case msg.GetButtonsMessage() != nil:
		return "buttons", msg.GetButtonsMessage().GetContentText(), ""
	case msg.GetPollCreationMessage() != nil:
		return "poll", msg.GetPollCreationMessage().GetName(), ""
	case msg.GetReactionMessage() != nil:
		return "reaction", msg.GetReactionMessage().GetText(), ""
	case msg.GetTemplateMessage() != nil:
		return "template", msg.GetTemplateMessage().GetHydratedTemplate().GetHydratedContentText(), ""
	case msg.GetInteractiveMessage() != nil:
		return "interactive", msg.GetInteractiveMessage().GetHeader().GetSubtitle(), ""
	case msg.GetPollUpdateMessage() != nil:
		return "poll_update", "", ""
	default:
		return "unknown", "", ""
	}
}

func isExpiredMediaError(err error) bool {
	return errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith403) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith404) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith410)
}
