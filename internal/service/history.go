package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/async"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/storage"
	"wzap/internal/wa"
	"wzap/internal/wautil"
)

type messageRepo interface {
	Save(ctx context.Context, msg *model.Message) error
}

type chatRepo interface {
	Upsert(ctx context.Context, chat *model.ChatUpdate) error
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
	messageRepo    messageRepo
	chatRepo       chatRepo
	pool           *async.Pool
	downloader     MediaDownloader
	storage        MediaStorage
	retryRequester MediaRetryRequester
}

func NewHistoryService(messageRepo messageRepo, chatRepo chatRepo, pool *async.Pool) *HistoryService {
	return &HistoryService{messageRepo: messageRepo, chatRepo: chatRepo, pool: pool}
}

func (s *HistoryService) SetMediaDownloader(d MediaDownloader)    { s.downloader = d }
func (s *HistoryService) SetMediaStorage(ms MediaStorage)         { s.storage = ms }
func (s *HistoryService) SetRetryRequester(r MediaRetryRequester) { s.retryRequester = r }

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

func (s *HistoryService) PersistMessage(input wa.PersistInput) {
	enqueuedAt := time.Now()
	_ = s.pool.Submit(func(ctx context.Context) {
		waited := time.Since(enqueuedAt)
		msg := &model.Message{
			ID:        input.MessageID,
			SessionID: input.SessionID,
			ChatJID:   input.ChatJID,
			SenderJID: input.SenderJID,
			FromMe:    input.FromMe,
			MsgType:   input.MsgType,
			Body:      input.Body,
			MediaType: input.MediaType,
			Source:    "live",
			Raw:       input.Raw,
			Timestamp: time.Unix(input.Timestamp, 0),
		}

		saveStart := time.Now()
		if err := s.messageRepo.Save(ctx, msg); err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", input.SessionID).Str("mid", input.MessageID).Msg("Failed to persist message")
		} else {
			logger.Debug().Str("component", "service").Str("session", input.SessionID).Str("mid", input.MessageID).Bool("fromMe", input.FromMe).Dur("queueWait", waited).Dur("saveDur", time.Since(saveStart)).Msg("PersistMessage: wz_messages row saved")
		}

		if s.chatRepo == nil || input.ChatJID == "" {
			return
		}

		chat := buildLiveChatUpdate(input.SessionID, input.ChatJID, input.MessageID, input.Timestamp)
		if err := s.chatRepo.Upsert(ctx, chat); err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", input.SessionID).Str("chat", input.ChatJID).Msg("Failed to persist canonical chat from live message")
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
			chatJID := resolveConversationChatJID(conversation, aliases)
			if strings.HasPrefix(chatJID, "status@") {
				continue
			}

			chat := buildHistoryChatUpdate(sessionID, conversation, aliases, syncType, chunkOrder)
			if chat != nil && s.chatRepo != nil {
				if err := s.chatRepo.Upsert(ctx, chat); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("chat", chat.ChatJID).Msg("Failed to persist canonical chat from history sync")
				}
			}

			if s.messageRepo == nil {
				continue
			}

			var retryIdx int
			for _, historyMessage := range conversation.GetMessages() {
				msg := buildHistoryMessage(sessionID, conversation, historyMessage, aliases, syncType, chunkOrder)
				if msg == nil {
					continue
				}

				if s.downloader != nil && s.storage != nil && msg.MediaURL == "" {
					info := historyMessage.GetMessage()
					if info != nil {
						protoMsg := info.GetMessage()
						directPath, encFileHash, fileHash, mediaKey, fileLength, hasMedia := wautil.ExtractMediaDownloadInfo(protoMsg)
						if hasMedia && directPath != "" {
							dlCtx, dlCancel := context.WithTimeout(ctx, 30*time.Second)
							data, err := s.downloader.DownloadMediaByPath(dlCtx, sessionID, directPath, encFileHash, fileHash, mediaKey, fileLength, msg.MediaType)
							dlCancel()
							if err != nil {
								if s.retryRequester != nil && isExpiredMediaError(err) {
									logger.Debug().Str("component", "service").Str("session", sessionID).Str("mid", msg.ID).Msg("History sync media retry: agendando retry")
									requester := s.retryRequester
									sessID, msgID, chatJID2, senderJID2 := sessionID, msg.ID, msg.ChatJID, msg.SenderJID
									fromMe2, mediaType2, ts2 := msg.FromMe, msg.MediaType, msg.Timestamp
									encHash, fHash, mKey, fLen := encFileHash, fileHash, mediaKey, fileLength
									retryIdx++
									delay := time.Duration(retryIdx) * mediaRetryInterval
									time.AfterFunc(delay, func() {
										if retryErr := requester.RequestMediaRetry(context.Background(), sessID, msgID, chatJID2, senderJID2, fromMe2, mediaType2, ts2, encHash, fHash, mKey, fLen); retryErr != nil {
											logger.Warn().Str("component", "service").Err(retryErr).Str("session", sessID).Str("mid", msgID).Msg("History sync media: falha ao solicitar retry")
										} else {
											logger.Debug().Str("component", "service").Str("session", sessID).Str("mid", msgID).Msg("History sync media: retry solicitado ao celular")
										}
									})
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

func buildLiveChatUpdate(sessionID, chatJID, lastMessageID string, timestamp int64) *model.ChatUpdate {
	chatType := wautil.InferChatType(chatJID)
	lastMessageAt := time.Unix(timestamp, 0)

	return &model.ChatUpdate{
		SessionID:     sessionID,
		ChatJID:       chatJID,
		ChatType:      &chatType,
		LastMessageID: wautil.StringPtr(lastMessageID),
		LastMessageAt: &lastMessageAt,
		Source:        "live",
	}
}

func buildHistoryChatUpdate(sessionID string, conversation *waHistorySync.Conversation, aliases jidAliasMaps, syncType string, chunkOrder int) *model.ChatUpdate {
	if conversation == nil {
		return nil
	}

	chatJID := resolveConversationChatJID(conversation, aliases)
	if chatJID == "" {
		return nil
	}

	name := wautil.FirstNonEmpty(conversation.GetName(), aliases.pushnameByJID[chatJID], aliases.pushnameByJID[conversation.GetPnJID()], aliases.pushnameByJID[conversation.GetLidJID()])
	displayName := wautil.FirstNonEmpty(conversation.GetDisplayName(), name)
	chatType := wautil.InferChatType(chatJID)
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
	lastMessageAt := wautil.UnixTimePtr(wautil.Uint64ToInt64(conversation.GetLastMsgTimestamp()))
	conversationTimestamp := wautil.UnixTimePtr(wautil.Uint64ToInt64(conversation.GetConversationTimestamp()))
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

	return &model.ChatUpdate{
		SessionID:          sessionID,
		ChatJID:            chatJID,
		Name:               wautil.StringPtr(name),
		DisplayName:        wautil.StringPtr(displayName),
		ChatType:           &chatType,
		Archived:           &archived,
		Pinned:             &pinned,
		ReadOnly:           &readOnly,
		MarkedAsUnread:     &markedAsUnread,
		UnreadCount:        &unreadCount,
		UnreadMentionCount: &unreadMentionCount,
		LastMessageID:      wautil.StringPtr(lastMessageID),
		LastMessageAt:      lastMessageAt,
		ConvTimestamp:      conversationTimestamp,
		PnJID:              wautil.StringPtr(pnJID),
		LidJID:             wautil.StringPtr(lidJID),
		Username:           wautil.StringPtr(username),
		AccountLID:         wautil.StringPtr(accountLID),
		Source:             "history_sync",
		SyncType:           wautil.StringPtr(syncType),
		ChunkOrder:         wautil.IntPtr(chunkOrder),
		Raw:                raw,
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
		msgType, _, _ := wautil.ExtractMessageContent(protoMsg)
		if msgType == "unknown" {
			return nil
		}
	}

	key := info.GetKey()
	messageID := key.GetID()
	if messageID == "" {
		return nil
	}

	chatJID := wautil.FirstNonEmpty(resolveMessageChatJID(info, conversation, aliases), resolveConversationChatJID(conversation, aliases))
	if chatJID == "" {
		return nil
	}

	senderJID := resolveMessageSenderJID(info, chatJID)
	msgType, body, mediaType := wautil.ExtractMessageContent(info.GetMessage())
	timestamp := wautil.Uint64ToInt64(info.GetMessageTimestamp())
	if timestamp == 0 {
		if conversation != nil {
			timestamp = wautil.Uint64ToInt64(conversation.GetConversationTimestamp())
		}
		if timestamp == 0 {
			timestamp = time.Now().Unix()
		}
	}

	var messageOrder *int64
	if order := wautil.Uint64ToInt64(historyMessage.GetMsgOrderID()); order > 0 {
		messageOrder = &order
	}

	return &model.Message{
		ID:         messageID,
		SessionID:  sessionID,
		ChatJID:    chatJID,
		SenderJID:  senderJID,
		FromMe:     key.GetFromMe(),
		MsgType:    msgType,
		Body:       body,
		MediaType:  mediaType,
		Source:     "history_sync",
		SyncType:   syncType,
		ChunkOrder: wautil.IntPtr(chunkOrder),
		MsgOrder:   messageOrder,
		Raw:        info,
		Timestamp:  time.Unix(timestamp, 0),
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

	chatJID := wautil.FirstNonEmpty(conversation.GetNewJID(), conversation.GetID(), conversation.GetPnJID(), conversation.GetLidJID())
	if chatJID == "" {
		return ""
	}
	if resolved, ok := aliases.lidToPN[chatJID]; ok {
		return wautil.FirstNonEmpty(conversation.GetPnJID(), resolved, chatJID)
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

func isExpiredMediaError(err error) bool {
	return errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith403) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith404) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith410)
}
