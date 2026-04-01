package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"wzap/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Save(ctx context.Context, msg *model.Message) error {
	rawJSON, _ := json.Marshal(msg.Raw)
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_messages (id, session_id, chat_jid, sender_jid, from_me, msg_type, body, media_type, media_url, raw, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (id, session_id) DO NOTHING`,
		msg.ID, msg.SessionID, msg.ChatJID, msg.SenderJID, msg.FromMe, msg.MsgType, msg.Body, msg.MediaType, msg.MediaURL, rawJSON, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func (r *MessageRepository) FindByChat(ctx context.Context, sessionID, chatJID string, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, session_id, chat_jid, sender_jid, from_me, msg_type, body,
		COALESCE(media_type, ''), COALESCE(media_url, ''), timestamp, created_at
		FROM wz_messages
		WHERE session_id = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, sessionID, chatJID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.ChatJID, &m.SenderJID, &m.FromMe, &m.MsgType, &m.Body,
			&m.MediaType, &m.MediaURL, &m.Timestamp, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepository) FindBySession(ctx context.Context, sessionID string, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, session_id, chat_jid, sender_jid, from_me, msg_type, body,
		COALESCE(media_type, ''), COALESCE(media_url, ''), timestamp, created_at
		FROM wz_messages
		WHERE session_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.ChatJID, &m.SenderJID, &m.FromMe, &m.MsgType, &m.Body,
			&m.MediaType, &m.MediaURL, &m.Timestamp, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
