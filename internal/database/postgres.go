package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

type baselineMigration struct {
	table string
	file  string
}

var legacyBaselineMigrations = []baselineMigration{
	{table: "wz_sessions", file: "001_schema.up.sql"},
	{table: "wz_messages", file: "002_messages.up.sql"},
	{table: "wz_chats", file: "003_chats.up.sql"},
	{table: "wz_statuses", file: "004_statuses.up.sql"},
	{table: "wz_chatwoot", file: "005_chatwoot.up.sql"},
}

// requiredSchema mirrors the database objects read and written by the
// application. It turns schema drift into a startup error instead of a later
// HTTP 500 on the first affected request.
var requiredSchema = map[string][]string{
	"wz_sessions": {
		"id", "name", "token", "jid", "qr_code", "status", "connected",
		"engine", "proxy", "settings", "created_at", "updated_at",
	},
	"wz_webhooks": {
		"id", "session_id", "url", "secret", "events", "enabled",
		"nats_enabled", "created_at", "updated_at",
	},
	"wz_messages": {
		"id", "session_id", "chat_jid", "sender_jid", "from_me", "msg_type",
		"body", "media_type", "media_url", "raw", "timestamp", "created_at",
		"cw_message_id", "cw_conversation_id", "cw_source_id", "source",
		"source_sync_type", "history_chunk_order", "history_message_order",
		"imported_to_chatwoot_at",
	},
	"wz_chats": {
		"session_id", "chat_jid", "name", "display_name", "chat_type", "archived",
		"pinned", "read_only", "marked_as_unread", "unread_count",
		"unread_mention_count", "last_message_id", "last_message_at",
		"conversation_timestamp", "pn_jid", "lid_jid", "username", "account_lid",
		"source", "source_sync_type", "history_chunk_order", "raw", "created_at",
		"updated_at",
	},
	"wz_statuses": {
		"id", "session_id", "sender_jid", "from_me", "status_type", "body",
		"media_type", "media_url", "raw", "timestamp", "expires_at", "created_at",
	},
	"wz_chatwoot": {
		"session_id", "url", "account_id", "token", "inbox_id", "inbox_name",
		"inbox_type", "enabled", "webhook_token", "sign_msg", "sign_delimiter",
		"reopen_conversation", "conversation_pending", "message_read",
		"merge_br_contacts", "ignore_groups", "ignore_jids", "import_on_connect",
		"import_period", "timeout_text_seconds", "timeout_media_seconds",
		"timeout_large_seconds", "redis_url", "database_uri", "created_at", "updated_at",
	},
}

func New(ctx context.Context, url string) (*DB, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Str("component", "db").Msg("Successfully connected to PostgreSQL")

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		logger.Info().Str("component", "db").Msg("Closing PostgreSQL connection pool")
		db.Pool.Close()
	}
}

func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func (db *DB) Migrate(ctx context.Context) error {
	if err := db.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration tracking table: %w", err)
	}

	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var pending []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || len(name) < 7 || name[len(name)-7:] != ".up.sql" {
			continue
		}
		if !applied[name] {
			pending = append(pending, name)
		}
	}

	sort.Strings(pending)

	for _, name := range pending {
		migrationApplied, err := db.applyMigration(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", name, err)
		}
		if migrationApplied {
			logger.Info().Str("component", "db").Str("file", name).Msg("Migration applied")
		}
	}

	if err := db.validateSchema(ctx); err != nil {
		return fmt.Errorf("failed to validate database schema: %w", err)
	}
	logger.Info().Str("component", "db").Msg("Database schema validated")

	return nil
}

func (db *DB) ensureMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS wz_migrations (
			id SERIAL PRIMARY KEY,
			file_name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`
	_, err := db.Pool.Exec(ctx, query)
	return err
}

func (db *DB) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := db.Pool.Query(ctx, "SELECT file_name FROM wz_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var fileName string
		if err := rows.Scan(&fileName); err != nil {
			return nil, err
		}
		applied[fileName] = true
	}
	return applied, rows.Err()
}

func (db *DB) applyMigration(ctx context.Context, fileName string) (bool, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1)"); err != nil {
		return false, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM wz_migrations WHERE file_name = $1)", fileName).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	if exists {
		logger.Info().Str("component", "db").Str("file", fileName).Msg("Migration already applied, skipping")
		return false, nil
	}

	sqlBytes, err := migrations.FS.ReadFile(fileName)
	if err != nil {
		return false, fmt.Errorf("failed to read migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return false, fmt.Errorf("failed to execute migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, "INSERT INTO wz_migrations (file_name) VALUES ($1)", fileName); err != nil {
		return false, fmt.Errorf("failed to record migration %s: %w", fileName, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit migration %s: %w", fileName, err)
	}

	return true, nil
}

func (db *DB) BootstrapBaseline(ctx context.Context) error {
	if err := db.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration tracking table: %w", err)
	}

	existingTables, err := db.getExistingTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}

	for _, baseline := range legacyBaselineMigrations {
		if _, exists := existingTables[baseline.table]; !exists {
			continue
		}
		recorded, err := db.recordMigration(ctx, baseline.file)
		if err != nil {
			return fmt.Errorf("failed to record baseline migration %s: %w", baseline.file, err)
		}
		if recorded {
			logger.Info().Str("component", "db").Str("file", baseline.file).Str("table", baseline.table).Msg("Legacy baseline migration recorded")
		}
	}

	return nil
}

func (db *DB) columnExists(ctx context.Context, table, column string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2)`,
		table, column).Scan(&exists)
	return exists, err
}

func (db *DB) indexExists(ctx context.Context, indexName string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1)`,
		indexName).Scan(&exists)
	return exists, err
}

func (db *DB) getExistingTables(ctx context.Context) (map[string]struct{}, error) {
	query := `
		SELECT table_name FROM information_schema.tables 
		WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string]struct{})
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables[tableName] = struct{}{}
	}
	return tables, rows.Err()
}

func (db *DB) recordMigration(ctx context.Context, fileName string) (bool, error) {
	tag, err := db.Pool.Exec(ctx,
		"INSERT INTO wz_migrations (file_name) VALUES ($1) ON CONFLICT (file_name) DO NOTHING",
		fileName,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (db *DB) validateSchema(ctx context.Context) error {
	rows, err := db.Pool.Query(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
	`)
	if err != nil {
		return fmt.Errorf("failed to inspect schema columns: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]map[string]struct{})
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			return fmt.Errorf("failed to scan schema column: %w", err)
		}
		if existing[table] == nil {
			existing[table] = make(map[string]struct{})
		}
		existing[table][column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to list schema columns: %w", err)
	}

	tables := make([]string, 0, len(requiredSchema))
	for table := range requiredSchema {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	var missing []string
	for _, table := range tables {
		for _, column := range requiredSchema[table] {
			if _, exists := existing[table][column]; !exists {
				missing = append(missing, table+"."+column)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return nil
}
