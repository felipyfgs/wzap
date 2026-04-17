package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"wzap/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

const messageSelectColumns = `id, session_id, chat_jid, sender_jid, from_me, msg_type, body,
	COALESCE(media_type, ''), COALESCE(media_url, ''), COALESCE(source, 'live'), COALESCE(source_sync_type, ''),
	history_chunk_order, history_message_order, COALESCE(raw, 'null'::jsonb), timestamp, created_at,
	cw_message_id, cw_conversation_id, cw_source_id, imported_to_chatwoot_at`

type messageScanner interface {
	Scan(dest ...any) error
}

type MessageRepo interface {
	Save(ctx context.Context, msg *model.Message) error
	FindByChat(ctx context.Context, sessionID, chatJID string, limit, offset int) ([]model.Message, error)
	FindBySession(ctx context.Context, sessionID string, limit, offset int) ([]model.Message, error)
	FindByID(ctx context.Context, sessionID, msgID string) (*model.Message, error)
	FindSessionByMessageID(ctx context.Context, msgID string) (string, error)
	FindByCWMessageID(ctx context.Context, sessionID string, cwMsgID int) (*model.Message, error)
	FindAllByCWMessageID(ctx context.Context, sessionID string, cwMsgID int) ([]model.Message, error)
	FindBySourceID(ctx context.Context, sessionID, sourceID string) (*model.Message, error)
	FindBySourceIDPrefix(ctx context.Context, sessionID, sourceIDPrefix string) (*model.Message, error)
	FindByBody(ctx context.Context, sessionID, body string, fromMe bool) (*model.Message, error)
	FindByBodyAndChat(ctx context.Context, sessionID, chatJID, body string, fromMe bool) (*model.Message, error)
	FindByBodyAndChatAny(ctx context.Context, sessionID, chatJID, body string, fromMe bool) (*model.Message, error)
	FindByTimestamp(ctx context.Context, sessionID, chatJID string, timestamp int64, windowSeconds int64) (*model.Message, error)
	FindLastReceived(ctx context.Context, sessionID, chatJID string) (*model.Message, error)
	UpdateChatwootRef(ctx context.Context, sessionID, msgID string, cwMsgID, cwConvID int, cwSourceID string) error
	ExistsBySourceID(ctx context.Context, sessionID, sourceID string) (bool, error)
	FindUnimportedHistory(ctx context.Context, sessionID string, since time.Time, limit, offset int) ([]model.Message, error)
	MarkImported(ctx context.Context, sessionID, msgID string) error
	UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error
	FindMedia(ctx context.Context, sessionID string, f MediaFilter) ([]model.Message, int, error)
}

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Save(ctx context.Context, msg *model.Message) error {
	rawJSON, _ := json.Marshal(msg.Raw)
	if msg.Source == "" {
		msg.Source = "live"
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_messages (
			id, session_id, chat_jid, sender_jid, from_me, msg_type, body, media_type, media_url,
			source, source_sync_type, history_chunk_order, history_message_order, raw, timestamp, imported_to_chatwoot_at
		)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 ON CONFLICT (id, session_id) DO UPDATE SET
		   body = CASE WHEN EXCLUDED.body != '' THEN EXCLUDED.body ELSE wz_messages.body END,
		   msg_type = CASE WHEN EXCLUDED.msg_type != '' AND EXCLUDED.msg_type != 'unknown' THEN EXCLUDED.msg_type ELSE wz_messages.msg_type END,
		   media_type = COALESCE(NULLIF(EXCLUDED.media_type, ''), wz_messages.media_type),
		   media_url = COALESCE(NULLIF(EXCLUDED.media_url, ''), wz_messages.media_url),
		   raw = CASE WHEN EXCLUDED.raw IS NOT NULL AND EXCLUDED.raw != 'null'::jsonb THEN EXCLUDED.raw ELSE wz_messages.raw END,
		   sender_jid = CASE WHEN EXCLUDED.sender_jid != '' THEN EXCLUDED.sender_jid ELSE wz_messages.sender_jid END,
		   source = CASE WHEN EXCLUDED.source = 'live' OR wz_messages.source = 'live' THEN 'live' ELSE COALESCE(EXCLUDED.source, wz_messages.source) END,
		   source_sync_type = COALESCE(NULLIF(EXCLUDED.source_sync_type, ''), wz_messages.source_sync_type),
		   history_chunk_order = COALESCE(EXCLUDED.history_chunk_order, wz_messages.history_chunk_order),
		   history_message_order = COALESCE(EXCLUDED.history_message_order, wz_messages.history_message_order),
		   imported_to_chatwoot_at = COALESCE(EXCLUDED.imported_to_chatwoot_at, wz_messages.imported_to_chatwoot_at),
		   timestamp = LEAST(wz_messages.timestamp, EXCLUDED.timestamp)`,
		msg.ID, msg.SessionID, msg.ChatJID, msg.SenderJID, msg.FromMe, msg.MsgType, msg.Body, msg.MediaType, msg.MediaURL,
		msg.Source, msg.SyncType, msg.ChunkOrder, msg.MsgOrder, rawJSON, msg.Timestamp, msg.CWImportedAt)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func scanMessage(scanner messageScanner, m *model.Message) error {
	var raw []byte
	if err := scanner.Scan(
		&m.ID,
		&m.SessionID,
		&m.ChatJID,
		&m.SenderJID,
		&m.FromMe,
		&m.MsgType,
		&m.Body,
		&m.MediaType,
		&m.MediaURL,
		&m.Source,
		&m.SyncType,
		&m.ChunkOrder,
		&m.MsgOrder,
		&raw,
		&m.Timestamp,
		&m.CreatedAt,
		&m.CWMessageID,
		&m.CWConvID,
		&m.CWSrcID,
		&m.CWImportedAt,
	); err != nil {
		return err
	}
	m.Raw = raw
	return nil
}

func (r *MessageRepository) FindByChat(ctx context.Context, sessionID, chatJID string, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT ` + messageSelectColumns + `
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
		if err := scanMessage(rows, &m); err != nil {
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
	query := `SELECT ` + messageSelectColumns + `
		FROM wz_messages
		WHERE session_id = $1 AND chat_jid NOT LIKE 'status@%'
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
		if err := scanMessage(rows, &m); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// ErrInvalidCursor is returned when the cursor query parameter cannot be parsed.
var ErrInvalidCursor = errors.New("invalid cursor")

// MediaFilter holds server-side filters for media queries.
type MediaFilter struct {
	MsgType string // optional: image, video, document, audio, sticker
	Chat    string // optional: chat JID
	Cursor  string // cursor-based pagination: "timestamp|id" (RFC3339 timestamp + message ID)
	Limit   int
	Search  string // optional: search in body or chat JID
	Since   string // optional: ISO date string (start of range)
	Until   string // optional: ISO date string (end of range)
	Sort    string // optional: desc (default) or asc
}

// escapeILIKE escapes % and _ characters for use in ILIKE patterns.
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// buildMediaWhere constructs a WHERE clause and args slice from the filter.
// If addCursor is true, the cursor condition is appended.
// Returns an error if the cursor is malformed.
func buildMediaWhere(sessionID string, f MediaFilter, sortOrder string, addCursor bool) (string, []any, error) {
	where := `session_id = $1 AND msg_type IN ('image', 'video', 'document', 'audio', 'sticker')`
	args := []any{sessionID}
	argN := 2

	if f.MsgType != "" {
		where += fmt.Sprintf(` AND msg_type = $%d`, argN)
		args = append(args, f.MsgType)
		argN++
	}

	if f.Chat != "" {
		where += fmt.Sprintf(` AND chat_jid = $%d`, argN)
		args = append(args, f.Chat)
		argN++
	}

	if f.Search != "" {
		escaped := "%" + escapeILIKE(f.Search) + "%"
		where += fmt.Sprintf(` AND (body ILIKE $%d OR chat_jid ILIKE $%d)`, argN, argN+1)
		args = append(args, escaped, escaped)
		argN += 2
	}

	if f.Since != "" {
		where += fmt.Sprintf(` AND timestamp >= $%d::timestamptz`, argN)
		args = append(args, f.Since)
		argN++
	}

	if f.Until != "" {
		where += fmt.Sprintf(` AND timestamp <= $%d::timestamptz`, argN)
		args = append(args, f.Until)
		argN++
	}

	if addCursor && f.Cursor != "" {
		parts := strings.SplitN(f.Cursor, "|", 2)
		cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return "", nil, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
		}
		cursorID := ""
		if len(parts) == 2 {
			cursorID = parts[1]
		}
		if sortOrder == "DESC" {
			where += fmt.Sprintf(` AND (timestamp, id) < ($%d, $%d)`, argN, argN+1)
		} else {
			where += fmt.Sprintf(` AND (timestamp, id) > ($%d, $%d)`, argN, argN+1)
		}
		args = append(args, cursorTime, cursorID)
		argN += 2
	}

	return where, args, nil
}

// FindMedia returns media messages with server-side filtering and cursor pagination.
func (r *MessageRepository) FindMedia(ctx context.Context, sessionID string, f MediaFilter) ([]model.Message, int, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 100
	}

	sortOrder := "DESC"
	if f.Sort == "asc" {
		sortOrder = "ASC"
	}

	// Build WHERE clauses (with cursor for data, without cursor for count)
	where, args, err := buildMediaWhere(sessionID, f, sortOrder, true)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid cursor: %w", err)
	}
	countWhere, countArgs, err := buildMediaWhere(sessionID, f, sortOrder, false)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid cursor: %w", err)
	}

	// Count total matching items (without cursor)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wz_messages WHERE `+countWhere, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count media: %w", err)
	}

	// Fetch page
	query := `SELECT ` + messageSelectColumns + `
		FROM wz_messages
		WHERE ` + where + `
		ORDER BY timestamp ` + sortOrder + `, id ` + sortOrder + `
		LIMIT ` + fmt.Sprintf("%d", f.Limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query media: %w", err)
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := scanMessage(rows, &m); err != nil {
			return nil, 0, err
		}
		msgs = append(msgs, m)
	}
	return msgs, total, rows.Err()
}

func (r *MessageRepository) FindByID(ctx context.Context, sessionID, msgID string) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE id = $1 AND session_id = $2`,
		msgID, sessionID), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by ID: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindSessionByMessageID(ctx context.Context, msgID string) (string, error) {
	var sessionID string
	err := r.db.QueryRow(ctx,
		`SELECT session_id FROM wz_messages WHERE id = $1 ORDER BY created_at DESC LIMIT 1`,
		msgID).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to find session by message ID: %w", err)
	}
	return sessionID, nil
}

func (r *MessageRepository) UpdateChatwootRef(ctx context.Context, sessionID, msgID string, cwMsgID, cwConvID int, sourceID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_messages SET cw_message_id = $1, cw_conversation_id = $2, cw_source_id = $3
		 WHERE id = $4 AND session_id = $5`,
		cwMsgID, cwConvID, sourceID, msgID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update chatwoot ref: %w", err)
	}
	return nil
}

func (r *MessageRepository) FindByCWMessageID(ctx context.Context, sessionID string, cwMsgID int) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE cw_message_id = $1 AND session_id = $2`,
		cwMsgID, sessionID), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by CW message ID: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindAllByCWMessageID(ctx context.Context, sessionID string, cwMsgID int) ([]model.Message, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE cw_message_id = $1 AND session_id = $2`,
		cwMsgID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find messages by CW message ID: %w", err)
	}
	defer rows.Close()
	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := scanMessage(rows, &m); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepository) ExistsBySourceID(ctx context.Context, sessionID, sourceID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM wz_messages WHERE cw_source_id = $1 AND session_id = $2 AND cw_message_id IS NOT NULL)`,
		sourceID, sessionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check source_id existence: %w", err)
	}
	return exists, nil
}

func (r *MessageRepository) FindBySourceID(ctx context.Context, sessionID, sourceID string) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE cw_source_id = $1 AND session_id = $2`,
		sourceID, sessionID), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by source ID: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindBySourceIDPrefix(ctx context.Context, sessionID, sourceIDPrefix string) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE cw_source_id LIKE $1 AND session_id = $2 AND cw_message_id IS NOT NULL
		 ORDER BY created_at DESC LIMIT 1`,
		sourceIDPrefix+"%", sessionID), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by source ID prefix: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindByBody(ctx context.Context, sessionID, body string, fromMe bool) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE body = $1 AND session_id = $2 AND from_me = $3 AND cw_message_id IS NOT NULL
		 ORDER BY created_at DESC LIMIT 1`,
		body, sessionID, fromMe), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by body: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindByBodyAndChat(ctx context.Context, sessionID, chatJID, body string, fromMe bool) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE body = $1 AND session_id = $2 AND chat_jid = $3 AND from_me = $4 AND cw_message_id IS NOT NULL
		 ORDER BY created_at DESC LIMIT 1`,
		body, sessionID, chatJID, fromMe), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by body and chat: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindByBodyAndChatAny(ctx context.Context, sessionID, chatJID, body string, fromMe bool) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages WHERE body = $1 AND session_id = $2 AND chat_jid = $3 AND from_me = $4
		 ORDER BY created_at DESC LIMIT 1`,
		body, sessionID, chatJID, fromMe), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by body and chat (any): %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindByTimestamp(ctx context.Context, sessionID, chatJID string, timestamp int64, windowSeconds int64) (*model.Message, error) {
	var m model.Message
	tLow := time.Unix(timestamp-windowSeconds, 0)
	tHigh := time.Unix(timestamp+windowSeconds, 0)
	tCenter := time.Unix(timestamp, 0)
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages 
		 WHERE session_id = $1 AND chat_jid = $2 AND from_me = true 
		 AND timestamp BETWEEN $3 AND $4
		 ORDER BY ABS(EXTRACT(EPOCH FROM timestamp - $5)) ASC LIMIT 1`,
		sessionID, chatJID, tLow, tHigh, tCenter), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find message by timestamp window: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindLastReceived(ctx context.Context, sessionID, chatJID string) (*model.Message, error) {
	var m model.Message
	err := scanMessage(r.db.QueryRow(ctx,
		`SELECT `+messageSelectColumns+` FROM wz_messages
		 WHERE session_id = $1 AND chat_jid = $2 AND from_me = false
		 ORDER BY timestamp DESC LIMIT 1`,
		sessionID, chatJID), &m)
	if err != nil {
		return nil, fmt.Errorf("failed to find last received message by chat: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) FindUnimportedHistory(ctx context.Context, sessionID string, since time.Time, limit, offset int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	query := `SELECT ` + messageSelectColumns + `
		FROM wz_messages
		WHERE session_id = $1 AND source = 'history_sync' AND imported_to_chatwoot_at IS NULL AND timestamp >= $2
		ORDER BY timestamp ASC, id ASC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, sessionID, since, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query unimported history: %w", err)
	}
	defer rows.Close()

	msgs := make([]model.Message, 0)
	for rows.Next() {
		var m model.Message
		if err := scanMessage(rows, &m); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepository) MarkImported(ctx context.Context, sessionID, msgID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_messages SET imported_to_chatwoot_at = NOW() WHERE id = $1 AND session_id = $2`,
		msgID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to mark message as imported to chatwoot: %w", err)
	}
	return nil
}

func (r *MessageRepository) UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_messages SET media_url = $1 WHERE id = $2 AND session_id = $3`,
		mediaURL, msgID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update media url: %w", err)
	}
	return nil
}
