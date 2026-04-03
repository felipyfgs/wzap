# Query Patterns — wzap (pgx v5)

All database access uses `github.com/jackc/pgx/v5/pgxpool`. Pool is `r.db *pgxpool.Pool`.

## QueryRow — single result

```go
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
    query := `SELECT id, name, COALESCE(token, ''), COALESCE(jid, ''), COALESCE(qr_code, ''),
        connected, status, proxy, settings, created_at, updated_at
        FROM wz_sessions WHERE id = $1`

    var s model.Session
    err := r.db.QueryRow(ctx, query, id).Scan(
        &s.ID, &s.Name, &s.Token, &s.JID, &s.QRCode,
        &s.Connected, &s.Status, &s.Proxy, &s.Settings,
        &s.CreatedAt, &s.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("session not found: %w", err)
    }
    return &s, nil
}
```

- `COALESCE(col, '')` for nullable text columns.
- Scan order must match SELECT order exactly.
- pgx auto-scans JSONB into Go structs (`model.SessionProxy`, `model.SessionSettings`).

## Query — multiple results

```go
func (r *SessionRepository) FindAll(ctx context.Context) ([]model.Session, error) {
    query := `SELECT id, name, COALESCE(token, ''), COALESCE(jid, ''), COALESCE(qr_code, ''),
        connected, status, proxy, settings, created_at, updated_at
        FROM wz_sessions ORDER BY created_at DESC`

    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query sessions: %w", err)
    }
    defer rows.Close()

    var sessions []model.Session
    for rows.Next() {
        var s model.Session
        if err := rows.Scan(
            &s.ID, &s.Name, &s.Token, &s.JID, &s.QRCode,
            &s.Connected, &s.Status, &s.Proxy, &s.Settings,
            &s.CreatedAt, &s.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan session: %w", err)
        }
        sessions = append(sessions, s)
    }
    return sessions, rows.Err()
}
```

- `defer rows.Close()` immediately after error check.
- Return `rows.Err()` as final error check.
- Result slice starts as `nil`.

## Exec — insert / update / delete

```go
func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
    query := `INSERT INTO wz_sessions (id, name, token, status, proxy, settings, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := r.db.Exec(ctx, query,
        session.ID, session.Name, session.Token, session.Status,
        session.Proxy, session.Settings, session.CreatedAt, session.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("failed to insert session: %w", err)
    }
    return nil
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id, status string) error {
    _, err := r.db.Exec(ctx,
        `UPDATE wz_sessions SET status = $1 WHERE id = $2`,
        status, id,
    )
    if err != nil {
        return fmt.Errorf("failed to update status for session %s: %w", id, err)
    }
    return nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM wz_sessions WHERE id = $1`, id)
    if err != nil {
        return fmt.Errorf("failed to delete session %s: %w", id, err)
    }
    return nil
}
```

## JSONB array containment

```go
func (r *WebhookRepository) FindActiveBySessionAndEvent(
    ctx context.Context, sessionID, eventType string,
) ([]model.Webhook, error) {
    query := `SELECT id, session_id, url, COALESCE(secret, ''),
        events, enabled, nats_enabled, created_at, updated_at
        FROM wz_webhooks
        WHERE session_id = $1 AND enabled = true
          AND (events @> $2::jsonb OR events @> '["All"]'::jsonb)`

    eventJSON, _ := json.Marshal([]string{eventType})
    rows, err := r.db.Query(ctx, query, sessionID, eventJSON)
    // ... rows.Close(), loop, rows.Err()
}
```

## Upsert (messages)

```go
query := `INSERT INTO wz_messages (id, session_id, chat_jid, sender_jid, from_me, msg_type, body, media_type, media_url, raw, timestamp)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    ON CONFLICT (id, session_id) DO NOTHING`
```

## Adding a new repo method

1. Add method to existing struct in `internal/repo/<entity>.go`.
2. Naming: `FindByID`, `FindAll`, `FindBySessionID`, `Create`, `UpdateStatus`, `Delete`.
3. Receiver is `r`. `context.Context` always first param.
4. Wrap all errors: `fmt.Errorf("failed to <verb> <noun>: %w", err)`.
5. Wire in `internal/server/router.go` if it's a new repo.

```go
func (r *SessionRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
    query := `SELECT status, COUNT(*)::int FROM wz_sessions GROUP BY status`
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

## Quick reference

| Operation | pgx method | Error check |
|---|---|---|
| Single row | `r.db.QueryRow(ctx, sql, args...).Scan(...)` | `err != nil` |
| Multiple rows | `r.db.Query(ctx, sql, args...)` | `err` + `rows.Err()` |
| Write | `r.db.Exec(ctx, sql, args...)` | `err != nil` |
| JSONB contains | `col @> $1::jsonb` | pass `json.Marshal(slice)` as arg |
| Nullable text | `COALESCE(col, '')` in SELECT | scan into `string` |
