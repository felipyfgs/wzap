package repo

import (
	"context"
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
	query := `INSERT INTO "wzWebhooks" ("id", "sessionId", "url", "secret", "events", "enabled", "createdAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query, w.ID, w.SessionID, w.URL, w.Secret, w.Events, w.Enabled, w.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert webhook: %w", err)
	}
	return nil
}

func (r *WebhookRepository) FindBySessionID(ctx context.Context, sessionID string) ([]model.Webhook, error) {
	query := `SELECT "id", "sessionId", "url", COALESCE("secret", ''), "events", "enabled", "createdAt", "updatedAt"
		FROM "wzWebhooks" WHERE "sessionId" = $1 ORDER BY "createdAt" DESC`

	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events, &w.Enabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

func (r *WebhookRepository) Delete(ctx context.Context, sessionID, webhookID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM "wzWebhooks" WHERE "id" = $1 AND "sessionId" = $2`,
		webhookID, sessionID)
	return err
}
