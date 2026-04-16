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
		if err := db.applyMigration(ctx, name); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", name, err)
		}
		logger.Info().Str("component", "db").Str("file", name).Msg("Migration applied")
	}

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

func (db *DB) applyMigration(ctx context.Context, fileName string) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1)"); err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM wz_migrations WHERE file_name = $1)", fileName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}
	if exists {
		logger.Info().Str("component", "db").Str("file", fileName).Msg("Migration already applied, skipping")
		return nil
	}

	sqlBytes, err := migrations.FS.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to read migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, "INSERT INTO wz_migrations (file_name) VALUES ($1)", fileName); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", fileName, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", fileName, err)
	}

	return nil
}

func (db *DB) BootstrapBaseline(ctx context.Context) error {
	if err := db.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration tracking table: %w", err)
	}

	existingTables, err := db.getExistingTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}

	tableToMigration := []struct {
		table     string
		migration string
	}{
		{"wz_sessions", "001_schema.up.sql"},
		{"wz_messages", "002_messages.up.sql"},
		{"wz_chats", "003_chats.up.sql"},
		{"wz_statuses", "004_statuses.up.sql"},
		{"wz_chatwoot", "005_chatwoot.up.sql"},
	}

	for _, tm := range tableToMigration {
		if !contains(existingTables, tm.table) {
			continue
		}
		if err := db.recordMigration(ctx, tm.migration); err != nil {
			return fmt.Errorf("failed to record baseline migration %s: %w", tm.migration, err)
		}
		logger.Info().Str("component", "db").Str("file", tm.migration).Msg("Baseline migration recorded")
	}

	return nil
}

func (db *DB) columnExists(ctx context.Context, table, column string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2)`,
		table, column).Scan(&exists)
	return exists, err
}

func (db *DB) indexExists(ctx context.Context, indexName string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1)`,
		indexName).Scan(&exists)
	return exists, err
}

func (db *DB) getExistingTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, rows.Err()
}

func (db *DB) recordMigration(ctx context.Context, fileName string) error {
	var exists bool
	if err := db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM wz_migrations WHERE file_name = $1)", fileName).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		_, err := db.Pool.Exec(ctx, "INSERT INTO wz_migrations (file_name) VALUES ($1)", fileName)
		return err
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
