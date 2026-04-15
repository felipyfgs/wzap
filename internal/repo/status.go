package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wzap/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

const statusSelectColumns = `id, session_id, sender_jid, from_me, status_type, body,
	COALESCE(media_type, ''), COALESCE(media_url, ''), COALESCE(raw, 'null'::jsonb), timestamp, expires_at, created_at`

type statusScanner interface {
	Scan(dest ...any) error
}

type StatusRepo interface {
	Save(ctx context.Context, status *model.Status) error
	FindBySession(ctx context.Context, sessionID string, limit, offset int) ([]model.Status, error)
	FindBySender(ctx context.Context, sessionID, senderJID string) ([]model.Status, error)
	UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	DeleteBySender(ctx context.Context, sessionID, senderJID string) error
}

type StatusRepository struct {
	db *pgxpool.Pool
}

func NewStatusRepository(db *pgxpool.Pool) *StatusRepository {
	return &StatusRepository{db: db}
}

func scanStatus(scanner statusScanner, s *model.Status) error {
	var raw []byte
	if err := scanner.Scan(
		&s.ID,
		&s.SessionID,
		&s.SenderJID,
		&s.FromMe,
		&s.StatusType,
		&s.Body,
		&s.MediaType,
		&s.MediaURL,
		&raw,
		&s.Timestamp,
		&s.ExpiresAt,
		&s.CreatedAt,
	); err != nil {
		return err
	}
	s.Raw = raw
	return nil
}

func (r *StatusRepository) Save(ctx context.Context, status *model.Status) error {
	rawJSON, _ := json.Marshal(status.Raw)
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_statuses (
			id, session_id, sender_jid, from_me, status_type, body, media_type, media_url, raw, timestamp, expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id, session_id) DO UPDATE SET
			body = CASE WHEN EXCLUDED.body != '' THEN EXCLUDED.body ELSE wz_statuses.body END,
			media_type = COALESCE(NULLIF(EXCLUDED.media_type, ''), wz_statuses.media_type),
			media_url = COALESCE(NULLIF(EXCLUDED.media_url, ''), wz_statuses.media_url),
			raw = CASE WHEN EXCLUDED.raw IS NOT NULL AND EXCLUDED.raw != 'null'::jsonb THEN EXCLUDED.raw ELSE wz_statuses.raw END`,
		status.ID, status.SessionID, status.SenderJID, status.FromMe, status.StatusType, status.Body,
		status.MediaType, status.MediaURL, rawJSON, status.Timestamp, status.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to save status: %w", err)
	}
	return nil
}

func (r *StatusRepository) FindBySession(ctx context.Context, sessionID string, limit, offset int) ([]model.Status, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT ` + statusSelectColumns + `
		FROM wz_statuses
		WHERE session_id = $1 AND expires_at > NOW()
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query statuses: %w", err)
	}
	defer rows.Close()

	var statuses []model.Status
	for rows.Next() {
		var s model.Status
		if err := scanStatus(rows, &s); err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (r *StatusRepository) FindBySender(ctx context.Context, sessionID, senderJID string) ([]model.Status, error) {
	query := `SELECT ` + statusSelectColumns + `
		FROM wz_statuses
		WHERE session_id = $1 AND sender_jid = $2 AND expires_at > NOW()
		ORDER BY timestamp ASC`

	rows, err := r.db.Query(ctx, query, sessionID, senderJID)
	if err != nil {
		return nil, fmt.Errorf("failed to query statuses by sender: %w", err)
	}
	defer rows.Close()

	var statuses []model.Status
	for rows.Next() {
		var s model.Status
		if err := scanStatus(rows, &s); err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (r *StatusRepository) UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_statuses SET media_url = $1 WHERE id = $2 AND session_id = $3`,
		mediaURL, msgID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update status media url: %w", err)
	}
	return nil
}

func (r *StatusRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM wz_statuses WHERE expires_at < $1`,
		before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired statuses: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *StatusRepository) DeleteBySender(ctx context.Context, sessionID, senderJID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM wz_statuses WHERE session_id = $1 AND sender_jid = $2`,
		sessionID, senderJID)
	if err != nil {
		return fmt.Errorf("failed to delete statuses by sender: %w", err)
	}
	return nil
}
