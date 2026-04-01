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
	_, err := r.db.Exec(ctx, query, w.ID, w.SessionID, w.URL, w.Secret, w.Events, w.Enabled, w.NatsEnabled, w.CreatedAt)
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
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.NatsEnabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
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
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.NatsEnabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *WebhookRepository) Delete(ctx context.Context, sessionID, webhookID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM wz_webhooks WHERE id = $1 AND session_id = $2`,
		webhookID, sessionID)
	return err
}
