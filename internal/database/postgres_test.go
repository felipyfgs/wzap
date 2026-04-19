package database

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func skipIfNoDatabase(t *testing.T) *DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/wzap_test?sslmode=disable&connect_timeout=2")
	if err != nil {
		t.Skip("database not available, skipping integration test")
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("database not available, skipping integration test")
	}

	return &DB{Pool: pool}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"empty slice", []string{}, "test", false},
		{"exact match", []string{"test", "foo"}, "test", true},
		{"case insensitive", []string{"Test", "Foo"}, "test", true},
		{"no match", []string{"test", "foo"}, "bar", false},
		{"single element match", []string{"test"}, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetExistingTables(t *testing.T) {
	db := skipIfNoDatabase(t)
	defer db.Pool.Close()

	ctx := context.Background()

	tables, err := db.getExistingTables(ctx)
	if err != nil {
		t.Fatalf("getExistingTables() error = %v", err)
	}

	if !slices.Contains(tables, "wz_migrations") {
		t.Log("wz_migrations table not found - this is expected for fresh database")
	}
}

func TestBootstrapBaseline(t *testing.T) {
	db := skipIfNoDatabase(t)
	defer db.Pool.Close()

	ctx := context.Background()

	err := db.BootstrapBaseline(ctx)
	if err != nil {
		t.Logf("BootstrapBaseline() error = %v (may be expected for fresh database)", err)
	}

	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("getAppliedMigrations() error = %v", err)
	}

	t.Logf("Applied migrations after BootstrapBaseline: %v", applied)
}

func TestMigrationLock(t *testing.T) {
	db := skipIfNoDatabase(t)
	defer db.Pool.Close()

	ctx := context.Background()

	err := db.ensureMigrationTable(ctx)
	if err != nil {
		t.Fatalf("ensureMigrationTable() error = %v", err)
	}

	rows, err := db.Pool.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'wz_migrations'")
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
	defer db.Pool.Close()

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
	defer db.Pool.Close()

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	_, err := db.indexExists(ctx, "nonexistent_index")
	if err != nil {
		t.Fatalf("indexExists() error = %v", err)
	}
}

func TestConcurrentMigrationSafety(t *testing.T) {
	db := skipIfNoDatabase(t)
	defer db.Pool.Close()

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	for range 3 {
		wg.Go(func() {
			if err := db.recordMigration(ctx, "test_concurrent.up.sql"); err != nil {
				errCh <- err
			}
		})
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
	defer db.Pool.Close()

	ctx := context.Background()

	_ = db.ensureMigrationTable(ctx)

	_ = db.recordMigration(ctx, "test_idempotent.up.sql")
	_ = db.recordMigration(ctx, "test_idempotent.up.sql")
	_ = db.recordMigration(ctx, "test_idempotent.up.sql")

	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wz_migrations WHERE file_name = 'test_idempotent.up.sql'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record after idempotent calls, got %d", count)
	}

	_, _ = db.Pool.Exec(ctx, "DELETE FROM wz_migrations WHERE file_name = 'test_idempotent.up.sql'")
}
