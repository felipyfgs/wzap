package service

import (
	"context"
	"time"

	"wzap/internal/async"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
)

type HistoryService struct {
	repo *repo.MessageRepository
	pool *async.Pool
}

func NewHistoryService(repo *repo.MessageRepository, pool *async.Pool) *HistoryService {
	return &HistoryService{repo: repo, pool: pool}
}

func (s *HistoryService) PersistMessage(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw interface{}) {
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
			Raw:       raw,
			Timestamp: time.Unix(timestamp, 0),
		}

		if err := s.repo.Save(ctx, msg); err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Failed to persist message")
		}
	})
}
