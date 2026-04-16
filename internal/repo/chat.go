package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"wzap/internal/model"
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) Upsert(ctx context.Context, chat *model.ChatUpdate) error {
	rawJSON, _ := json.Marshal(chat.Raw)
	_, err := r.db.Exec(ctx, `INSERT INTO wz_chats (
		session_id, chat_jid, name, display_name, chat_type, archived, pinned, read_only,
		marked_as_unread, unread_count, unread_mention_count, last_message_id, last_message_at,
		conversation_timestamp, pn_jid, lid_jid, username, account_lid, source, source_sync_type,
		history_chunk_order, raw
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13,
		$14, $15, $16, $17, $18, $19, $20,
		$21, $22
	)
	ON CONFLICT (session_id, chat_jid) DO UPDATE SET
		name = COALESCE(EXCLUDED.name, wz_chats.name),
		display_name = COALESCE(EXCLUDED.display_name, wz_chats.display_name),
		chat_type = COALESCE(EXCLUDED.chat_type, wz_chats.chat_type),
		archived = COALESCE(EXCLUDED.archived, wz_chats.archived),
		pinned = COALESCE(EXCLUDED.pinned, wz_chats.pinned),
		read_only = COALESCE(EXCLUDED.read_only, wz_chats.read_only),
		marked_as_unread = COALESCE(EXCLUDED.marked_as_unread, wz_chats.marked_as_unread),
		unread_count = COALESCE(EXCLUDED.unread_count, wz_chats.unread_count),
		unread_mention_count = COALESCE(EXCLUDED.unread_mention_count, wz_chats.unread_mention_count),
		last_message_id = COALESCE(EXCLUDED.last_message_id, wz_chats.last_message_id),
		last_message_at = COALESCE(EXCLUDED.last_message_at, wz_chats.last_message_at),
		conversation_timestamp = COALESCE(EXCLUDED.conversation_timestamp, wz_chats.conversation_timestamp),
		pn_jid = COALESCE(EXCLUDED.pn_jid, wz_chats.pn_jid),
		lid_jid = COALESCE(EXCLUDED.lid_jid, wz_chats.lid_jid),
		username = COALESCE(EXCLUDED.username, wz_chats.username),
		account_lid = COALESCE(EXCLUDED.account_lid, wz_chats.account_lid),
		source = CASE
			WHEN EXCLUDED.source = 'live' THEN 'live'
			ELSE COALESCE(EXCLUDED.source, wz_chats.source)
		END,
		source_sync_type = COALESCE(EXCLUDED.source_sync_type, wz_chats.source_sync_type),
		history_chunk_order = COALESCE(EXCLUDED.history_chunk_order, wz_chats.history_chunk_order),
		raw = CASE
			WHEN EXCLUDED.raw IS NOT NULL AND EXCLUDED.raw != 'null'::jsonb THEN EXCLUDED.raw
			ELSE wz_chats.raw
		END`,
		chat.SessionID,
		chat.ChatJID,
		chat.Name,
		chat.DisplayName,
		chat.ChatType,
		chat.Archived,
		chat.Pinned,
		chat.ReadOnly,
		chat.MarkedAsUnread,
		chat.UnreadCount,
		chat.UnreadMentionCount,
		chat.LastMessageID,
		chat.LastMessageAt,
		chat.ConvTimestamp,
		chat.PnJID,
		chat.LidJID,
		chat.Username,
		chat.AccountLID,
		chat.Source,
		chat.SyncType,
		chat.ChunkOrder,
		rawJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert chat: %w", err)
	}
	return nil
}

func (r *ChatRepository) FindBySessionAndChat(ctx context.Context, sessionID, chatJID string) (*model.Chat, error) {
	var chat model.Chat
	var raw []byte
	err := r.db.QueryRow(ctx, `SELECT session_id, chat_jid,
		COALESCE(name, ''), COALESCE(display_name, ''), COALESCE(chat_type, ''),
		COALESCE(archived, false), COALESCE(pinned, 0), COALESCE(read_only, false), COALESCE(marked_as_unread, false),
		COALESCE(unread_count, 0), COALESCE(unread_mention_count, 0), COALESCE(last_message_id, ''),
		last_message_at, conversation_timestamp, COALESCE(pn_jid, ''), COALESCE(lid_jid, ''), COALESCE(username, ''),
		COALESCE(account_lid, ''), source, COALESCE(source_sync_type, ''), history_chunk_order,
		COALESCE(raw, '{}'::jsonb), created_at, updated_at
		FROM wz_chats
		WHERE session_id = $1 AND chat_jid = $2`, sessionID, chatJID).Scan(
		&chat.SessionID,
		&chat.ChatJID,
		&chat.Name,
		&chat.DisplayName,
		&chat.ChatType,
		&chat.Archived,
		&chat.Pinned,
		&chat.ReadOnly,
		&chat.MarkedAsUnread,
		&chat.UnreadCount,
		&chat.UnreadMentionCount,
		&chat.LastMessageID,
		&chat.LastMessageAt,
		&chat.ConvTimestamp,
		&chat.PnJID,
		&chat.LidJID,
		&chat.Username,
		&chat.AccountLID,
		&chat.Source,
		&chat.SyncType,
		&chat.ChunkOrder,
		&raw,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find chat by session and chat: %w", err)
	}
	chat.Raw = raw
	return &chat, nil
}

func (r *ChatRepository) ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]model.Chat, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `SELECT session_id, chat_jid,
		COALESCE(name, ''), COALESCE(display_name, ''), COALESCE(chat_type, ''),
		COALESCE(archived, false), COALESCE(pinned, 0), COALESCE(read_only, false), COALESCE(marked_as_unread, false),
		COALESCE(unread_count, 0), COALESCE(unread_mention_count, 0), COALESCE(last_message_id, ''),
		last_message_at, conversation_timestamp, COALESCE(pn_jid, ''), COALESCE(lid_jid, ''), COALESCE(username, ''),
		COALESCE(account_lid, ''), source, COALESCE(source_sync_type, ''), history_chunk_order,
		COALESCE(raw, '{}'::jsonb), created_at, updated_at
		FROM wz_chats
		WHERE session_id = $1
		ORDER BY last_message_at DESC NULLS LAST, updated_at DESC
		LIMIT $2 OFFSET $3`, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats by session: %w", err)
	}
	defer rows.Close()

	chats := make([]model.Chat, 0)
	for rows.Next() {
		var chat model.Chat
		var raw []byte
		if err := rows.Scan(
			&chat.SessionID,
			&chat.ChatJID,
			&chat.Name,
			&chat.DisplayName,
			&chat.ChatType,
			&chat.Archived,
			&chat.Pinned,
			&chat.ReadOnly,
			&chat.MarkedAsUnread,
			&chat.UnreadCount,
			&chat.UnreadMentionCount,
			&chat.LastMessageID,
			&chat.LastMessageAt,
			&chat.ConvTimestamp,
			&chat.PnJID,
			&chat.LidJID,
			&chat.Username,
			&chat.AccountLID,
			&chat.Source,
			&chat.SyncType,
			&chat.ChunkOrder,
			&raw,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan chat: %w", err)
		}
		chat.Raw = raw
		chats = append(chats, chat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate chats: %w", err)
	}
	return chats, nil
}
