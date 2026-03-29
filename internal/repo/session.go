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
	query := `INSERT INTO "wzSessions" ("id", "name", "apiKey", "status", "metadata", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query, session.ID, session.Name, session.APIKey, session.Status, session.Metadata, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
	query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
		"connected", "status", "metadata", "createdAt", "updatedAt"
		FROM "wzSessions" ORDER BY "createdAt" DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Metadata, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
		"connected", "status", "metadata", "createdAt", "updatedAt"
		FROM "wzSessions" WHERE "id" = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByName(ctx context.Context, name string) (*model.Session, error) {
	query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
		"connected", "status", "metadata", "createdAt", "updatedAt"
		FROM "wzSessions" WHERE "name" = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, name).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) FindByAPIKey(ctx context.Context, apiKey string) (*model.Session, error) {
	query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
		"connected", "status", "metadata", "createdAt", "updatedAt"
		FROM "wzSessions" WHERE "apiKey" = $1`

	var s model.Session
	err := r.db.QueryRow(ctx, query, apiKey).Scan(&s.ID, &s.Name, &s.JID, &s.QRCode, &s.Connected, &s.Status, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found for api key: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "wzSessions" WHERE "id" = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE "wzSessions" SET "status" = $1 WHERE "id" = $2`, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) UpdateJid(ctx context.Context, id string, jid string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "wzSessions" SET "jid" = $1, "connected" = 1, "status" = 'connected' WHERE "id" = $2`,
		jid, id)
	if err != nil {
		return fmt.Errorf("failed to update jid for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) SetConnected(ctx context.Context, id string, connected int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "wzSessions" SET "connected" = $1 WHERE "id" = $2`,
		connected, id)
	if err != nil {
		return fmt.Errorf("failed to set connected status for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) ClearDevice(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "wzSessions" SET "connected" = 0, "jid" = '', "status" = 'disconnected' WHERE "id" = $1`,
		id)
	if err != nil {
		return fmt.Errorf("failed to clear device for session %s: %w", id, err)
	}
	return nil
}

func (r *SessionRepository) GetJid(ctx context.Context, id string) (string, error) {
	var jid string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE("jid", '') FROM "wzSessions" WHERE "id" = $1`,
		id).Scan(&jid)
	if err != nil {
		return "", fmt.Errorf("failed to get jid for session %s: %w", id, err)
	}
	return jid, nil
}

func (r *SessionRepository) FindSessionIDByJID(ctx context.Context, jid string) (string, error) {
	var sessionID string
	err := r.db.QueryRow(ctx,
		`SELECT "id" FROM "wzSessions" WHERE "jid" = $1`,
		jid).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to find session ID by JID %s: %w", jid, err)
	}
	return sessionID, nil
}

func (r *SessionRepository) UpdateQRCode(ctx context.Context, id string, qrCode string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "wzSessions" SET "qrCode" = $1 WHERE "id" = $2`,
		qrCode, id)
	if err != nil {
		return fmt.Errorf("failed to update QR code for session %s: %w", id, err)
	}
	return nil
}
