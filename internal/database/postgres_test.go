package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func skipIfNoDatabase(t *testing.T) *DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/wzap_test?sslmode=disable&connect_timeout=2"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skip("database not available, skipping integration test")
	}

	if err := adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		t.Skip("database not available, skipping integration test")
	}

	schemaName := fmt.Sprintf("wzap_test_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		adminPool.Close()
		t.Skipf("cannot create isolated test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		t.Fatalf("failed to parse test database URL: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		t.Fatalf("failed to create isolated test pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		t.Fatalf("failed to ping isolated test schema: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
	})

	return &DB{Pool: pool}
}

func TestGetExistingTables(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	tables, err := db.getExistingTables(ctx)
	if err != nil {
		t.Fatalf("getExistingTables() error = %v", err)
	}

	if _, found := tables["wz_migrations"]; !found {
		t.Log("wz_migrations table not found - this is expected for fresh database")
	}
}

func TestBootstrapBaseline(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `
		CREATE TABLE wz_sessions (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			token VARCHAR(255) NOT NULL UNIQUE
		)
	`); err != nil {
		t.Fatalf("failed to create legacy baseline anchor: %v", err)
	}

	if err := db.BootstrapBaseline(ctx); err != nil {
		t.Fatalf("BootstrapBaseline() error = %v", err)
	}

	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("getAppliedMigrations() error = %v", err)
	}

	if !applied["001_schema.up.sql"] {
		t.Error("expected sessions baseline migration to be recorded")
	}
	if applied["002_messages.up.sql"] {
		t.Error("messages baseline must not be recorded without its anchor table")
	}

	if err := db.BootstrapBaseline(ctx); err != nil {
		t.Fatalf("second BootstrapBaseline() error = %v", err)
	}
	var count int
	if err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wz_migrations WHERE file_name = '001_schema.up.sql'").Scan(&count); err != nil {
		t.Fatalf("failed to count baseline records: %v", err)
	}
	if count != 1 {
		t.Errorf("expected one baseline record, got %d", count)
	}
}

func TestMigrationLock(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	err := db.ensureMigrationTable(ctx)
	if err != nil {
		t.Fatalf("ensureMigrationTable() error = %v", err)
	}

	rows, err := db.Pool.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'wz_migrations'")
	if err != nil {
		t.Fatalf("query error = %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("wz_migrations table should exist after ensureMigrationTable")
	}
}

func TestColumnExists(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	exists, err := db.columnExists(ctx, "wz_migrations", "file_name")
	if err != nil {
		t.Fatalf("columnExists() error = %v", err)
	}
	if !exists {
		t.Error("file_name column should exist in wz_migrations table")
	}

	exists, err = db.columnExists(ctx, "wz_migrations", "nonexistent_column")
	if err != nil {
		t.Fatalf("columnExists() error = %v", err)
	}
	if exists {
		t.Error("nonexistent_column should not exist")
	}
}

func TestIndexExists(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	_, err := db.indexExists(ctx, "nonexistent_index")
	if err != nil {
		t.Fatalf("indexExists() error = %v", err)
	}
}

func TestConcurrentMigrationSafety(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.recordMigration(ctx, "test_concurrent.up.sql"); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent migration recording error: %v", err)
	}

	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wz_migrations WHERE file_name = 'test_concurrent.up.sql'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record for concurrent migration, got %d", count)
	}

	_, _ = db.Pool.Exec(ctx, "DELETE FROM wz_migrations WHERE file_name = 'test_concurrent.up.sql'")
}

func TestMigrationIdempotency(t *testing.T) {
	db := skipIfNoDatabase(t)

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	first, err := db.recordMigration(ctx, "test_idempotent.up.sql")
	if err != nil {
		t.Fatalf("first recordMigration() error = %v", err)
	}
	second, err := db.recordMigration(ctx, "test_idempotent.up.sql")
	if err != nil {
		t.Fatalf("second recordMigration() error = %v", err)
	}
	if !first || second {
		t.Errorf("expected only the first insert to be recorded, got first=%v second=%v", first, second)
	}

	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wz_migrations WHERE file_name = 'test_idempotent.up.sql'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record after idempotent calls, got %d", count)
	}

	_, _ = db.Pool.Exec(ctx, "DELETE FROM wz_migrations WHERE file_name = 'test_idempotent.up.sql'")
}

func TestMigrateFreshSchema(t *testing.T) {
	db := skipIfNoDatabase(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() fresh schema error = %v", err)
	}
	if err := db.validateSchema(ctx); err != nil {
		t.Fatalf("validateSchema() fresh schema error = %v", err)
	}

	var applied bool
	if err := db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM wz_migrations WHERE file_name = '010_reconcile_legacy_schema.up.sql')",
	).Scan(&applied); err != nil {
		t.Fatalf("failed to check reconciliation migration: %v", err)
	}
	if !applied {
		t.Error("reconciliation migration was not applied on fresh schema")
	}
}

func TestMigrateReconcilesLegacySchema(t *testing.T) {
	db := skipIfNoDatabase(t)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, legacySchemaSQL); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}
	if err := db.BootstrapBaseline(ctx); err != nil {
		t.Fatalf("BootstrapBaseline() legacy schema error = %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() legacy schema error = %v", err)
	}
	if err := db.validateSchema(ctx); err != nil {
		t.Fatalf("validateSchema() reconciled schema error = %v", err)
	}

	for _, column := range []string{
		"inbox_type", "webhook_token", "conversation_pending", "message_read",
		"ignore_jids", "import_on_connect", "import_period", "timeout_text_seconds",
		"timeout_media_seconds", "timeout_large_seconds", "redis_url", "database_uri",
	} {
		exists, err := db.columnExists(ctx, "wz_chatwoot", column)
		if err != nil {
			t.Fatalf("columnExists(wz_chatwoot.%s) error = %v", column, err)
		}
		if !exists {
			t.Errorf("expected reconciled column wz_chatwoot.%s", column)
		}
	}

	var engine, nullable string
	if err := db.Pool.QueryRow(ctx, "SELECT engine FROM wz_sessions WHERE id = 'legacy-session'").Scan(&engine); err != nil {
		t.Fatalf("failed to read reconciled engine: %v", err)
	}
	if engine != "whatsmeow" {
		t.Errorf("engine = %q, want whatsmeow", engine)
	}
	if err := db.Pool.QueryRow(ctx, `
		SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'wz_sessions' AND column_name = 'engine'
	`).Scan(&nullable); err != nil {
		t.Fatalf("failed to inspect engine nullability: %v", err)
	}
	if nullable != "NO" {
		t.Errorf("engine is_nullable = %q, want NO", nullable)
	}

	for _, column := range []string{"msg_type", "media_type"} {
		var dataType string
		if err := db.Pool.QueryRow(ctx, `
			SELECT data_type FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'wz_messages' AND column_name = $1
		`, column).Scan(&dataType); err != nil {
			t.Fatalf("failed to inspect wz_messages.%s type: %v", column, err)
		}
		if dataType != "text" {
			t.Errorf("wz_messages.%s type = %q, want text", column, dataType)
		}
	}

	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO wz_chatwoot (session_id, url, account_id, token, inbox_id)
		VALUES ('legacy-session', 'https://chatwoot.example', 1, 'token', 2)
	`); err != nil {
		t.Fatalf("failed to insert Chatwoot config after reconciliation: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		"DELETE FROM wz_migrations WHERE file_name = '010_reconcile_legacy_schema.up.sql'",
	); err != nil {
		t.Fatalf("failed to reset reconciliation migration for idempotency test: %v", err)
	}

	if err := db.BootstrapBaseline(ctx); err != nil {
		t.Fatalf("second BootstrapBaseline() legacy schema error = %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate() legacy schema error = %v", err)
	}

	var migrationCount int
	if err := db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM wz_migrations WHERE file_name = '010_reconcile_legacy_schema.up.sql'",
	).Scan(&migrationCount); err != nil {
		t.Fatalf("failed to count reconciliation migration: %v", err)
	}
	if migrationCount != 1 {
		t.Errorf("reconciliation migration count = %d, want 1", migrationCount)
	}
}

func TestValidateSchemaReportsDrift(t *testing.T) {
	db := skipIfNoDatabase(t)

	err := db.validateSchema(context.Background())
	if err == nil {
		t.Fatal("validateSchema() expected an error for empty schema")
	}
	for _, object := range []string{"wz_chatwoot.inbox_type", "wz_messages.source", "wz_webhooks.id"} {
		if !strings.Contains(err.Error(), object) {
			t.Errorf("validateSchema() error %q does not mention %s", err, object)
		}
	}
}

const legacySchemaSQL = `
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE wz_sessions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    token VARCHAR(255) NOT NULL UNIQUE,
    jid VARCHAR(255) NOT NULL DEFAULT '',
    qr_code TEXT NOT NULL DEFAULT '',
    connected INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    engine VARCHAR(20) DEFAULT 'whatsmeow',
    proxy JSONB NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO wz_sessions (id, name, token, engine)
VALUES ('legacy-session', 'legacy', 'legacy-token', NULL);

CREATE TABLE wz_messages (
    id VARCHAR(100) NOT NULL,
    session_id VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid VARCHAR(255) NOT NULL,
    sender_jid VARCHAR(255) NOT NULL,
    from_me BOOLEAN NOT NULL DEFAULT false,
    msg_type VARCHAR(50) NOT NULL DEFAULT 'text',
    body TEXT NOT NULL DEFAULT '',
    media_type VARCHAR(50),
    media_url TEXT,
    raw JSONB,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cw_message_id INTEGER,
    cw_conversation_id INTEGER,
    cw_source_id TEXT,
    PRIMARY KEY (id, session_id)
);

CREATE TABLE wz_chats (
    session_id VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid VARCHAR(255) NOT NULL,
    PRIMARY KEY (session_id, chat_jid)
);

CREATE TABLE wz_statuses (
    id VARCHAR(100) NOT NULL,
    session_id VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    sender_jid VARCHAR(255) NOT NULL,
    from_me BOOLEAN NOT NULL DEFAULT false,
    status_type VARCHAR(50) NOT NULL DEFAULT 'status_text',
    body TEXT NOT NULL DEFAULT '',
    media_type VARCHAR(50),
    media_url TEXT,
    raw JSONB,
    timestamp TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, session_id)
);

CREATE TABLE wz_chatwoot (
    session_id VARCHAR(100) PRIMARY KEY REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url VARCHAR(2048) NOT NULL,
    account_id INTEGER NOT NULL,
    token VARCHAR(255) NOT NULL,
    inbox_id INTEGER NOT NULL,
    inbox_name VARCHAR(255) NOT NULL DEFAULT 'wzap',
    sign_msg BOOLEAN NOT NULL DEFAULT false,
    sign_delimiter VARCHAR(50) NOT NULL DEFAULT '\n',
    reopen_conversation BOOLEAN NOT NULL DEFAULT true,
    merge_br_contacts BOOLEAN NOT NULL DEFAULT true,
    ignore_groups BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
