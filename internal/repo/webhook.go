package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"wzap/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookRepository struct {
	db *pgxpool.Pool
}

func NewWebhookRepository(db *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Create(ctx context.Context, w *model.Webhook) error {
	query := `INSERT INTO wz_webhooks (id, session_id, url, secret, events, enabled, nats_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`
	_, err := r.db.Exec(ctx, query, w.ID, w.SessionID, w.URL, w.Secret, w.Events, w.Enabled, w.NATSEnabled, w.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert webhook: %w", err)
	}
	return nil
}

func (r *WebhookRepository) FindBySessionID(ctx context.Context, sessionID string) ([]model.Webhook, error) {
	query := `SELECT id, session_id, url, COALESCE(secret, ''), events, enabled, nats_enabled, created_at, updated_at
		FROM wz_webhooks WHERE session_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.NATSEnabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *WebhookRepository) FindActiveBySessionAndEvent(ctx context.Context, sessionID string, eventType string) ([]model.Webhook, error) {
	query := `SELECT id, session_id, url, COALESCE(secret, ''), events, enabled, nats_enabled, created_at, updated_at
		FROM wz_webhooks
		WHERE session_id = $1
		  AND enabled = true
		  AND (events @> $2::jsonb OR events @> '["All"]'::jsonb)`

	eventJSON, _ := json.Marshal([]string{eventType})
	rows, err := r.db.Query(ctx, query, sessionID, eventJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to query active webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.NATSEnabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *WebhookRepository) FindByID(ctx context.Context, sessionID, webhookID string) (*model.Webhook, error) {
	query := `SELECT id, session_id, url, COALESCE(secret, ''), events, enabled, nats_enabled, created_at, updated_at
		FROM wz_webhooks WHERE id = $1 AND session_id = $2`
	var w model.Webhook
	err := r.db.QueryRow(ctx, query, webhookID, sessionID).Scan(
		&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.NATSEnabled, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %w", err)
	}
	return &w, nil
}

func (r *WebhookRepository) Update(ctx context.Context, w *model.Webhook) error {
	_, err := r.db.Exec(ctx,
		`UPDATE wz_webhooks SET url = $1, secret = $2, events = $3, enabled = $4, nats_enabled = $5, updated_at = NOW()
		 WHERE id = $6 AND session_id = $7`,
		w.URL, w.Secret, w.Events, w.Enabled, w.NATSEnabled, w.ID, w.SessionID)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, sessionID, webhookID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM wz_webhooks WHERE id = $1 AND session_id = $2`,
		webhookID, sessionID)
	return err
}
