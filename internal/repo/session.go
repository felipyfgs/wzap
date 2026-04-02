package repo

import (
	"context"
	"fmt"

	"wzap/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO wz_sessions (id, name, api_key, status, proxy, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query, session.ID, session.Name, session.APIKey, session.Status, session.Proxy, session.Settings, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
	query := `SELECT id, name, COALESCE(jid, ''), COALESCE(qr_code, ''),
		connected, status, proxy, settings, created_at, updated_at
		FROM wz_sessions ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT id, name, COALESCE(jid, ''), COALESCE(qr_code, ''),
		connected, status, proxy, settings, created_at, updated_at
		FROM wz_sessions WHERE id = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByName(ctx context.Context, name string) (*model.Session, error) {
	query := `SELECT id, name, COALESCE(jid, ''), COALESCE(qr_code, ''),
		connected, status, proxy, settings, created_at, updated_at
		FROM wz_sessions WHERE name = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, name).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByAPIKey(ctx context.Context, apiKey string) (*model.Session, error) {
	query := `SELECT id, name, COALESCE(jid, ''), COALESCE(qr_code, ''),
		connected, status, proxy, settings, created_at, updated_at
		FROM wz_sessions WHERE api_key = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, apiKey).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Proxy, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found for api_key: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wz_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) UpdateSession(ctx context.Context, session *model.Session) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET name = $1, proxy = $2, settings = $3, updated_at = NOW() WHERE id = $4`,
		session.Name, session.Proxy, session.Settings, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session %s: %w", session.ID, err)
	}
	return nil
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	connected := 0
	if status == "connected" {
		connected = 1
	}
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET status = $1, connected = $2, updated_at = NOW() WHERE id = $3`,
		status, connected, id)
	if err != nil {
		return fmt.Errorf("failed to update status for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) UpdateJID(ctx context.Context, id string, jid string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET jid = $1, connected = 1, status = 'connected', updated_at = NOW() WHERE id = $2`,
		jid, id)
	if err != nil {
		return fmt.Errorf("failed to update jid for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) SetConnected(ctx context.Context, id string, connected int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET connected = $1, updated_at = NOW() WHERE id = $2`,
		connected, id)
	if err != nil {
		return fmt.Errorf("failed to set connected status for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) ClearDevice(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET connected = 0, jid = '', status = 'disconnected', updated_at = NOW() WHERE id = $1`,
		id)
	if err != nil {
		return fmt.Errorf("failed to clear device for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) GetJID(ctx context.Context, id string) (string, error) {
	var jid string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(jid, '') FROM wz_sessions WHERE id = $1`,
		id).Scan(&jid)
	if err != nil {
		return "", fmt.Errorf("failed to get jid for session %s: %w", id, err)
	}
	return jid, nil
}

func (r *SessionRepository) FindSessionIDByJID(ctx context.Context, jid string) (string, error) {
	var sessionID string
	err := r.db.QueryRow(ctx,
		`SELECT id FROM wz_sessions WHERE jid = $1`,
		jid).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to find session ID by JID %s: %w", jid, err)
	}
	return sessionID, nil
}

func (r *SessionRepository) UpdateQRCode(ctx context.Context, id string, qrCode string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET qr_code = $1 WHERE id = $2`,
		qrCode, id)
	if err != nil {
		return fmt.Errorf("failed to update QR code for session %s: %w", id, err)
	}
	return nil
}
