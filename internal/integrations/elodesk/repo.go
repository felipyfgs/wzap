package elodesk

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo interface {
	FindBySessionID(ctx context.Context, sessionID string) (*Config, error)
	Upsert(ctx context.Context, cfg *Config) error
	Delete(ctx context.Context, sessionID string) error
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const elodeskSelectColumns = `session_id, url,
	COALESCE(inbox_identifier, ''), COALESCE(api_token, ''),
	COALESCE(hmac_token, ''), COALESCE(webhook_secret, ''),
	COALESCE(user_access_token, ''),
	COALESCE(account_id, 1),
	COALESCE(channel_id, 0),
	sign_msg, sign_delimiter, reopen_conv, conversation_pending,
	merge_br_contacts, ignore_groups, ignore_jids,
	import_on_connect, import_period,
	timeout_text_seconds, timeout_media_seconds, timeout_large_seconds,
	message_read, enabled, created_at, updated_at`

type elodeskConfigScanner interface {
	Scan(dest ...any) error
}

func scanConfig(s elodeskConfigScanner, cfg *Config) error {
	return s.Scan(
		&cfg.SessionID, &cfg.URL, &cfg.InboxIdentifier, &cfg.APIToken,
		&cfg.HMACToken, &cfg.WebhookSecret, &cfg.UserAccessToken, &cfg.AccountID, &cfg.ChannelID,
		&cfg.SignMsg, &cfg.SignDelimiter, &cfg.ReopenConv, &cfg.PendingConv,
		&cfg.MergeBRContacts, &cfg.IgnoreGroups, &cfg.IgnoreJIDs,
		&cfg.ImportOnConnect, &cfg.ImportPeriod,
		&cfg.TextTimeout, &cfg.MediaTimeout, &cfg.LargeTimeout,
		&cfg.MessageRead, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt)
}

func (r *Repository) Upsert(ctx context.Context, cfg *Config) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO wz_elodesk (
			session_id, url, inbox_identifier, api_token, hmac_token, webhook_secret, enabled,
			user_access_token, account_id, channel_id,
			sign_msg, sign_delimiter, reopen_conv, conversation_pending,
			merge_br_contacts, ignore_groups, ignore_jids,
			import_on_connect, import_period,
			timeout_text_seconds, timeout_media_seconds, timeout_large_seconds,
			message_read
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		 ON CONFLICT (session_id) DO UPDATE SET
			url = EXCLUDED.url, inbox_identifier = EXCLUDED.inbox_identifier,
			api_token = EXCLUDED.api_token, hmac_token = EXCLUDED.hmac_token,
			webhook_secret = EXCLUDED.webhook_secret, enabled = EXCLUDED.enabled,
			user_access_token = EXCLUDED.user_access_token, account_id = EXCLUDED.account_id,
			channel_id = EXCLUDED.channel_id,
			sign_msg = EXCLUDED.sign_msg, sign_delimiter = EXCLUDED.sign_delimiter,
			reopen_conv = EXCLUDED.reopen_conv,
			conversation_pending = EXCLUDED.conversation_pending,
			merge_br_contacts = EXCLUDED.merge_br_contacts,
			ignore_groups = EXCLUDED.ignore_groups, ignore_jids = EXCLUDED.ignore_jids,
			import_on_connect = EXCLUDED.import_on_connect, import_period = EXCLUDED.import_period,
			timeout_text_seconds = EXCLUDED.timeout_text_seconds,
			timeout_media_seconds = EXCLUDED.timeout_media_seconds,
			timeout_large_seconds = EXCLUDED.timeout_large_seconds,
			message_read = EXCLUDED.message_read, updated_at = NOW()`,
		cfg.SessionID, cfg.URL, cfg.InboxIdentifier, cfg.APIToken, cfg.HMACToken, cfg.WebhookSecret, cfg.Enabled,
		cfg.UserAccessToken, cfg.AccountID, cfg.ChannelID,
		cfg.SignMsg, cfg.SignDelimiter, cfg.ReopenConv, cfg.PendingConv,
		cfg.MergeBRContacts, cfg.IgnoreGroups, cfg.IgnoreJIDs,
		cfg.ImportOnConnect, cfg.ImportPeriod,
		cfg.TextTimeout, cfg.MediaTimeout, cfg.LargeTimeout,
		cfg.MessageRead)
	if err != nil {
		return fmt.Errorf("failed to upsert elodesk config: %w", err)
	}
	return nil
}

func (r *Repository) FindBySessionID(ctx context.Context, sessionID string) (*Config, error) {
	var cfg Config
	row := r.db.QueryRow(ctx,
		`SELECT `+elodeskSelectColumns+` FROM wz_elodesk WHERE session_id = $1`,
		sessionID)
	if err := scanConfig(row, &cfg); err != nil {
		return nil, fmt.Errorf("failed to find elodesk config: %w", err)
	}
	return &cfg, nil
}

func (r *Repository) Delete(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wz_elodesk WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete elodesk config: %w", err)
	}
	return nil
}
