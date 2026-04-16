package chatwoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo interface {
	FindBySessionID(ctx context.Context, sessionID string) (*Config, error)
	FindByPhoneAndInboxType(ctx context.Context, phone, inboxType string) (*Config, error)
	Upsert(ctx context.Context, cfg *Config) error
	Delete(ctx context.Context, sessionID string) error
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const chatwootSelectColumns = `session_id, url, account_id, token, COALESCE(webhook_token, ''),
	inbox_id, inbox_name, COALESCE(inbox_type, 'api'),
	sign_msg, sign_delimiter, reopen_conversation, conversation_pending,
	merge_br_contacts, ignore_groups, ignore_jids,
	import_on_connect, import_period,
	timeout_text_seconds, timeout_media_seconds, timeout_large_seconds,
	COALESCE(message_read, false), COALESCE(database_uri, ''),
	redis_url, enabled, created_at, updated_at`

type chatwootConfigScanner interface {
	Scan(dest ...any) error
}

func qualifyChatwootColumns(alias string) string {
	lines := strings.Split(chatwootSelectColumns, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		var parts []string
		for _, col := range strings.Split(line, ",") {
			col = strings.TrimSpace(col)
			if col == "" {
				continue
			}
			if strings.HasPrefix(col, "COALESCE(") {
				// COALESCE(col, val) → COALESCE(alias.col, val)
				col = strings.Replace(col, "COALESCE(", "COALESCE("+alias+".", 1)
			} else {
				col = alias + "." + col
			}
			parts = append(parts, col)
		}
		lines[i] = strings.Join(parts, ", ")
	}
	return strings.Join(lines, "\n\t")
}

func scanConfig(s chatwootConfigScanner, cfg *Config) error {
	return s.Scan(
		&cfg.SessionID, &cfg.URL, &cfg.AccountID, &cfg.Token, &cfg.WebhookToken,
		&cfg.InboxID, &cfg.InboxName, &cfg.InboxType,
		&cfg.SignMsg, &cfg.SignDelimiter, &cfg.ReopenConv, &cfg.PendingConv,
		&cfg.MergeBRContacts, &cfg.IgnoreGroups, &cfg.IgnoreJIDs,
		&cfg.ImportOnConnect, &cfg.ImportPeriod,
		&cfg.TextTimeout, &cfg.MediaTimeout, &cfg.LargeTimeout,
		&cfg.MessageRead, &cfg.DatabaseURI,
		&cfg.RedisURL, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt)
}

func (r *Repository) Upsert(ctx context.Context, cfg *Config) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_chatwoot (
			session_id, url, account_id, token, webhook_token, inbox_id, inbox_name, inbox_type, enabled,
			sign_msg, sign_delimiter, reopen_conversation, conversation_pending,
			merge_br_contacts, ignore_groups, ignore_jids,
			import_on_connect, import_period,
			timeout_text_seconds, timeout_media_seconds, timeout_large_seconds,
			message_read, database_uri, redis_url
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
		 ON CONFLICT (session_id) DO UPDATE SET
			url = EXCLUDED.url, account_id = EXCLUDED.account_id,
			token = EXCLUDED.token, webhook_token = EXCLUDED.webhook_token,
			inbox_id = EXCLUDED.inbox_id,
			inbox_name = EXCLUDED.inbox_name, inbox_type = EXCLUDED.inbox_type, enabled = EXCLUDED.enabled,
			sign_msg = EXCLUDED.sign_msg, sign_delimiter = EXCLUDED.sign_delimiter,
			reopen_conversation = EXCLUDED.reopen_conversation,
			conversation_pending = EXCLUDED.conversation_pending,
			merge_br_contacts = EXCLUDED.merge_br_contacts,
			ignore_groups = EXCLUDED.ignore_groups, ignore_jids = EXCLUDED.ignore_jids,
			import_on_connect = EXCLUDED.import_on_connect, import_period = EXCLUDED.import_period,
			timeout_text_seconds = EXCLUDED.timeout_text_seconds,
			timeout_media_seconds = EXCLUDED.timeout_media_seconds,
			timeout_large_seconds = EXCLUDED.timeout_large_seconds,
			message_read = EXCLUDED.message_read, database_uri = EXCLUDED.database_uri,
			redis_url = EXCLUDED.redis_url, updated_at = NOW()`,
		cfg.SessionID, cfg.URL, cfg.AccountID, cfg.Token, cfg.WebhookToken,
		cfg.InboxID, cfg.InboxName, cfg.InboxType, cfg.Enabled,
		cfg.SignMsg, cfg.SignDelimiter, cfg.ReopenConv, cfg.PendingConv,
		cfg.MergeBRContacts, cfg.IgnoreGroups, cfg.IgnoreJIDs,
		cfg.ImportOnConnect, cfg.ImportPeriod,
		cfg.TextTimeout, cfg.MediaTimeout, cfg.LargeTimeout,
		cfg.MessageRead, cfg.DatabaseURI, cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to upsert chatwoot config: %w", err)
	}
	return nil
}

func (r *Repository) FindBySessionID(ctx context.Context, sessionID string) (*Config, error) {
	var cfg Config
	row := r.db.QueryRow(ctx,
		`SELECT `+chatwootSelectColumns+` FROM wz_chatwoot WHERE session_id = $1`,
		sessionID)
	if err := scanConfig(row, &cfg); err != nil {
		return nil, fmt.Errorf("failed to find chatwoot config: %w", err)
	}
	return &cfg, nil
}

func (r *Repository) FindByPhoneAndInboxType(ctx context.Context, phone, inboxType string) (*Config, error) {
	var cfg Config
	row := r.db.QueryRow(ctx,
		`SELECT `+qualifyChatwootColumns("c")+`
		 FROM wz_chatwoot c
		 JOIN wz_sessions s ON c.session_id = s.id
		 WHERE c.inbox_type = $1
		 AND split_part(split_part(s.jid, '@', 1), ':', 1) = $2`,
		inboxType, phone)
	if err := scanConfig(row, &cfg); err != nil {
		return nil, fmt.Errorf("failed to find chatwoot config by phone: %w", err)
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
