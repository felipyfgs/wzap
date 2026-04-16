package repo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"wzap/internal/database"
)

const testDatabaseDSN = "postgres://postgres:postgres@localhost:5432/wzap_test?sslmode=disable&connect_timeout=2"

func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, testDatabaseDSN)
	if err != nil {
		t.Skip("database not available, skipping integration test")
	}
	if err := db.Migrate(ctx); err != nil {
		db.Close()
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func insertTestSession(t *testing.T, db *database.DB, prefix string) string {
	t.Helper()

	suffix := time.Now().UnixNano()
	sessionID := fmt.Sprintf("%s-%d", prefix, suffix)
	name := fmt.Sprintf("%s-name-%d", prefix, suffix)
	token := fmt.Sprintf("%s-token-%d", prefix, suffix)

	if _, err := db.Pool.Exec(context.Background(), `INSERT INTO wz_sessions (id, name, token) VALUES ($1, $2, $3)`, sessionID, name, token); err != nil {
		t.Fatalf("failed to insert test session: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM wz_sessions WHERE id = $1`, sessionID)
	})

	return sessionID
}
