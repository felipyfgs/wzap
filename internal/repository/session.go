package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"wzap/internal/model"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO wz_sessions (id, api_key, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, session.ID, session.APIKey, session.Status, session.Metadata, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
	query := `SELECT id, api_key, COALESCE(device_jid, ''), status, is_connected,
		COALESCE(qr_code, ''), metadata, created_at, updated_at
		FROM wz_sessions ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(&s.ID, &s.APIKey, &s.DeviceJID, &s.Status, &s.IsConnected, &s.QRCode, &s.Metadata, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT id, api_key, COALESCE(device_jid, ''), status, is_connected,
		COALESCE(qr_code, ''), metadata, created_at, updated_at
		FROM wz_sessions WHERE id = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.APIKey, &s.DeviceJID, &s.Status, &s.IsConnected, &s.QRCode, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByAPIKey(ctx context.Context, apiKey string) (*model.Session, error) {
	query := `SELECT id, api_key, COALESCE(device_jid, ''), status, is_connected,
		COALESCE(qr_code, ''), metadata, created_at, updated_at
		FROM wz_sessions WHERE api_key = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, apiKey).Scan(&s.ID, &s.APIKey, &s.DeviceJID, &s.Status, &s.IsConnected, &s.QRCode, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found with this api key: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByDeviceJID(ctx context.Context, jid string) (*model.Session, error) {
	query := `SELECT id, device_jid, status, is_connected, COALESCE(qr_code, ''), metadata, created_at, updated_at
		FROM wz_sessions WHERE device_jid = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, jid).Scan(&s.ID, &s.DeviceJID, &s.Status, &s.IsConnected, &s.QRCode, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wz_sessions WHERE id = $1`, id)
	return err
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE wz_sessions SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *SessionRepository) UpdateDeviceJID(ctx context.Context, id string, jid string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET device_jid = $1, is_connected = true, status = 'READY' WHERE id = $2`,
		jid, id)
	return err
}

func (r *SessionRepository) SetConnected(ctx context.Context, id string, connected bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET is_connected = $1 WHERE id = $2`,
		connected, id)
	return err
}

func (r *SessionRepository) ClearDevice(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_sessions SET is_connected = false, device_jid = NULL, status = 'CLOSED' WHERE id = $1`,
		id)
	return err
}

func (r *SessionRepository) GetDeviceJID(ctx context.Context, id string) (string, error) {
	var jid string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(device_jid, '') FROM wz_sessions WHERE id = $1`,
		id).Scan(&jid)
	return jid, err
}

func (r *SessionRepository) FindSessionIDByJID(ctx context.Context, jid string) (string, error) {
	var sessionID string
	err := r.db.QueryRow(ctx,
		`SELECT id FROM wz_sessions WHERE device_jid = $1`,
		jid).Scan(&sessionID)
	return sessionID, err
}
