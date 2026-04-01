# Query Patterns — wzap (pgx v5)

All database access uses `github.com/jackc/pgx/v5/pgxpool`. The pool is injected into each repository as `r.db *pgxpool.Pool`.

---

## Pattern 1 — QueryRow (single result)

Use for lookups by primary key or unique constraint.

```go
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
    query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
        "connected", "status", "proxy", "settings", "createdAt", "updatedAt"
        FROM "wzSessions" WHERE "id" = $1`

    var s model.Session
    err := r.db.QueryRow(ctx, query, id).Scan(
        &s.ID, &s.Name, &s.JID, &s.QRCode,
        &s.Connected, &s.Status, &s.Proxy, &s.Settings,
        &s.CreatedAt, &s.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("session not found: %w", err)
    }
    return &s, nil
}
```

Key points:
- `COALESCE("col", '')` for nullable text columns that should never be Go `""` vs `nil` ambiguity.
- Scan order must exactly match SELECT column order.
- `pgx` auto-scans `jsonb` columns into matching Go structs (`model.SessionProxy`, `model.SessionSettings`).
- `pgx.ErrNoRows` is returned when no row matches — wrap it with context.

---

## Pattern 2 — Query (multiple results)

Use for list endpoints and aggregations.

```go
func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
    query := `SELECT "id", "name", COALESCE("jid", ''), COALESCE("qrCode", ''),
        "connected", "status", "proxy", "settings", "createdAt", "updatedAt"
        FROM "wzSessions" ORDER BY "createdAt" DESC`

    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query sessions: %w", err)
    }
    defer rows.Close()   // always defer before the loop

    var sessions []model.Session
    for rows.Next() {
        var s model.Session
        if err := rows.Scan(
            &s.ID, &s.Name, &s.JID, &s.QRCode,
            &s.Connected, &s.Status, &s.Proxy, &s.Settings,
            &s.CreatedAt, &s.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan session: %w", err)
        }
        sessions = append(sessions, s)
    }
    return sessions, rows.Err()  // always check rows.Err()
}
```

Key points:
- `defer rows.Close()` immediately after the error check, before the loop.
- Return `rows.Err()` as the last error — it captures any error that occurred during iteration.
- Result slice starts as `nil`; callers should treat `nil` and `[]T{}` equivalently.

---

## Pattern 3 — Exec (insert / update / delete)

Use for writes. Check the error; ignore `pgconn.CommandTag` unless you need rows-affected.

```go
// Insert
func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
    query := `INSERT INTO "wzSessions"
        ("id", "name", "apiKey", "status", "proxy", "settings", "createdAt", "updatedAt")
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := r.db.Exec(ctx, query,
        session.ID, session.Name, session.APIKey, session.Status,
        session.Proxy, session.Settings, session.CreatedAt, session.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("failed to insert session: %w", err)
    }
    return nil
}

// Update
func (r *SessionRepository) UpdateStatus(ctx context.Context, id, status string) error {
    _, err := r.db.Exec(ctx,
        `UPDATE "wzSessions" SET "status" = $1 WHERE "id" = $2`,
        status, id,
    )
    if err != nil {
        return fmt.Errorf("failed to update status for session %s: %w", id, err)
    }
    return nil
}

// Delete
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx,
        `DELETE FROM "wzSessions" WHERE "id" = $1`, id,
    )
    if err != nil {
        return fmt.Errorf("failed to delete session %s: %w", id, err)
    }
    return nil
}
```

---

## Pattern 4 — JSONB array containment (`"events"`)

The `"wzWebhooks"."events"` column is a `jsonb` array of strings. Use the `@>` operator to check membership.

```go
func (r *WebhookRepository) FindActiveBySessionAndEvent(
    ctx context.Context, sessionID, eventType string,
) ([]model.Webhook, error) {
    query := `SELECT "id", "sessionId", "url", COALESCE("secret", ''),
        "events", "enabled", "natsEnabled", "createdAt", "updatedAt"
        FROM "wzWebhooks"
        WHERE "sessionId" = $1
          AND "enabled" = true
          AND ("events" @> $2::jsonb OR "events" @> '["All"]'::jsonb)`

    eventJSON, _ := json.Marshal([]string{eventType})  // produces '["Message"]'
    rows, err := r.db.Query(ctx, query, sessionID, eventJSON)
    // ... rows.Close(), loop, rows.Err() as in Pattern 2
}
```

To check if an event type exists at the SQL level without marshalling:
```sql
"events" @> '["Message"]'::jsonb
```

---

## Pattern 5 — JSONB struct scanning

pgx v5 scans `jsonb` columns directly into Go structs when the struct is used as the scan target:

```go
var s model.Session
rows.Scan(..., &s.Proxy, &s.Settings, ...)
// s.Proxy is model.SessionProxy (scanned from jsonb)
// s.Settings is model.SessionSettings (scanned from jsonb)
```

This works because pgx v5 uses `encoding/json` to unmarshal JSONB columns into struct pointers.

---

## Pattern 6 — Adding a new repo method

1. Add the method to the existing struct in `internal/repo/<entity>.go`.
2. Follow the naming convention: `Find...`, `Create`, `Update...`, `Delete`.
3. Use the same `r.db` pool — do not create a new connection.
4. Always accept `context.Context` as the first argument.
5. Wrap all errors with `fmt.Errorf("failed to <verb> <entity>: %w", err)`.

```go
// New method on SessionRepository
func (r *SessionRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
    query := `SELECT "status", COUNT(*)::int FROM "wzSessions" GROUP BY "status"`
    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to count sessions by status: %w", err)
    }
    defer rows.Close()

    result := make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, fmt.Errorf("failed to scan status count: %w", err)
        }
        result[status] = count
    }
    return result, rows.Err()
}
```

---

## Quick reference

| Operation | pgx method | Error check |
|---|---|---|
| Single row | `r.db.QueryRow(ctx, sql, args...).Scan(...)` | `err != nil` (includes `pgx.ErrNoRows`) |
| Multiple rows | `r.db.Query(ctx, sql, args...)` | check `err` + `rows.Err()` |
| Write (no result) | `r.db.Exec(ctx, sql, args...)` | `err != nil` |
| JSONB contains | `"col" @> $1::jsonb` | pass `json.Marshal(slice)` as arg |
| Nullable text | `COALESCE("col", '')` in SELECT | scan into `string` field directly |
