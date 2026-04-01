package service

import (
	"context"
	"time"

	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
)

type HistoryService struct {
	repo *repo.MessageRepository
}

func NewHistoryService(repo *repo.MessageRepository) *HistoryService {
	return &HistoryService{repo: repo}
}

func (s *HistoryService) PersistMessage(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw interface{}) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		msg := &model.Message{
			ID:        messageID,
			SessionID: sessionID,
			ChatJID:   chatJID,
			SenderJID: senderJID,
			FromMe:    fromMe,
			MsgType:   msgType,
			Body:      body,
			MediaType: mediaType,
			Raw:       raw,
			Timestamp: time.Unix(timestamp, 0),
		}

		if err := s.repo.Save(ctx, msg); err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Failed to persist message")
		}
	}()
}
