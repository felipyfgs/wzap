package chatwoot

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo interface {
	FindBySessionID(ctx context.Context, sessionID string) (*ChatwootConfig, error)
	Upsert(ctx context.Context, cfg *ChatwootConfig) error
	Delete(ctx context.Context, sessionID string) error
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, cfg *ChatwootConfig) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_chatwoot (session_id, url, account_id, token, inbox_id, inbox_name, sign_msg, sign_delimiter, reopen_conversation, merge_br_contacts, ignore_groups, ignore_jids, conversation_pending, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 ON CONFLICT (session_id) DO UPDATE SET
			url = EXCLUDED.url,
			account_id = EXCLUDED.account_id,
			token = EXCLUDED.token,
			inbox_id = EXCLUDED.inbox_id,
			inbox_name = EXCLUDED.inbox_name,
			sign_msg = EXCLUDED.sign_msg,
			sign_delimiter = EXCLUDED.sign_delimiter,
			reopen_conversation = EXCLUDED.reopen_conversation,
			merge_br_contacts = EXCLUDED.merge_br_contacts,
			ignore_groups = EXCLUDED.ignore_groups,
			ignore_jids = EXCLUDED.ignore_jids,
			conversation_pending = EXCLUDED.conversation_pending,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()`,
		cfg.SessionID, cfg.URL, cfg.AccountID, cfg.Token, cfg.InboxID, cfg.InboxName,
		cfg.SignMsg, cfg.SignDelimiter, cfg.ReopenConversation, cfg.MergeBRContacts,
		cfg.IgnoreGroups, cfg.IgnoreJIDs, cfg.ConversationPending, cfg.Enabled)
	if err != nil {
		return fmt.Errorf("failed to upsert chatwoot config: %w", err)
	}
	return nil
}

func (r *Repository) FindBySessionID(ctx context.Context, sessionID string) (*ChatwootConfig, error) {
	var cfg ChatwootConfig
	err := r.db.QueryRow(ctx,
		`SELECT session_id, url, account_id, token, inbox_id, inbox_name, sign_msg, sign_delimiter,
			reopen_conversation, merge_br_contacts, ignore_groups, ignore_jids, conversation_pending, enabled, created_at, updated_at
		 FROM wz_chatwoot WHERE session_id = $1`,
		sessionID).Scan(
		&cfg.SessionID, &cfg.URL, &cfg.AccountID, &cfg.Token, &cfg.InboxID, &cfg.InboxName,
		&cfg.SignMsg, &cfg.SignDelimiter, &cfg.ReopenConversation, &cfg.MergeBRContacts,
		&cfg.IgnoreGroups, &cfg.IgnoreJIDs, &cfg.ConversationPending, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to find chatwoot config: %w", err)
	}
	return &cfg, nil
}

func (r *Repository) Delete(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wz_chatwoot WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete chatwoot config: %w", err)
	}
	return nil
}
