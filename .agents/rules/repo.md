# Repository Rules

## Struct and Constructor

```go
type XxxRepository struct {
    db *pgxpool.Pool
}

func NewXxxRepository(db *pgxpool.Pool) *XxxRepository {
    return &XxxRepository{db: db}
}
```

- Single unexported field `db` of type `*pgxpool.Pool`.
- Constructor always `New<Type>Name(db *pgxpool.Pool) *<TypeName>`.
- No error return from constructors.

## Method Signature

```go
func (r *XxxRepository) Method(ctx context.Context, ...) (T, error)
```

- Receiver is always `r`.
- `context.Context` is always the first parameter.
- Pointer receivers for all methods.

## SQL Query Conventions

- Table names prefixed with `wz_`: `wz_sessions`, `wz_messages`, `wz_webhooks`.
- Queries use backtick raw strings with positional params: `$1`, `$2`, ...
- `updated_at` is set explicitly with `NOW()` in UPDATE queries (even though a DB trigger exists).

## INSERT Pattern

```go
func (r *XxxRepository) Create(ctx context.Context, entity *model.T) error {
    query := `INSERT INTO wz_table (col1, col2, ...) VALUES ($1, $2, ...)`
    _, err := r.db.Exec(ctx, query, entity.Field1, entity.Field2, ...)
    if err != nil {
        return fmt.Errorf("failed to insert entity: %w", err)
    }
    return nil
}
```

For idempotent inserts: `ON CONFLICT (id, session_id) DO NOTHING`.

## SELECT Single Row (QueryRow)

```go
var s model.Session
err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, ...)
if err != nil {
    return nil, fmt.Errorf("session not found: %w", err)
}
return &s, nil
```

- Declare value, scan into it, return `&s` (pointer to value).
- Nullable columns: `COALESCE(col, '')` in SQL to avoid NULL → empty string at DB level.

## SELECT Multiple Rows (Query)

```go
rows, err := r.db.Query(ctx, query)
if err != nil {
    return nil, fmt.Errorf("failed to query entities: %w", err)
}
defer rows.Close()

var entities []model.T
for rows.Next() {
    var e model.T
    if err := rows.Scan(&e.Field1, ...); err != nil {
        return nil, fmt.Errorf("failed to scan entity: %w", err)
    }
    entities = append(entities, e)
}
return entities, rows.Err()
```

- Slice declared as nil (`var entities []model.T`), appended to in loop.
- Always `defer rows.Close()`.
- Return `rows.Err()` as final error check.

## UPDATE Pattern

```go
_, err := r.db.Exec(ctx,
    `UPDATE wz_sessions SET name = $1, updated_at = NOW() WHERE id = $2`,
    session.Name, session.ID)
if err != nil {
    return fmt.Errorf("failed to update session: %w", err)
}
```

## DELETE Pattern

```go
_, err := r.db.Exec(ctx, `DELETE FROM wz_table WHERE id = $1`, id)
if err != nil {
    return fmt.Errorf("failed to delete entity: %w", err)
}
```

Compound delete: `DELETE FROM wz_webhooks WHERE id = $1 AND session_id = $2`.

## Error Wrapping

- Write ops: `fmt.Errorf("failed to <verb> <noun>: %w", err)`
- Read ops (find by ID): `fmt.Errorf("<noun> not found: %w", err)`
- Read ops (query): `fmt.Errorf("failed to query <nouns>: %w", err)`
- Read ops (scalar): `fmt.Errorf("failed to get <field> for <noun> %s: %w", id, err)`

## Method Naming

| Operation | Name | Return |
|---|---|---|
| Insert | `Create` | `error` |
| Upsert | `Save` | `error` |
| Get by ID | `FindByID` | `(*model.T, error)` |
| Get by name | `FindByName` | `(*model.T, error)` |
| List all | `FindAll` | `([]model.T, error)` |
| List by parent | `FindBySessionID` | `([]model.T, error)` |
| Scalar get | `GetJID` | `(string, error)` |
| Partial update | `UpdateStatus`, `UpdateJID` | `error` |
| Delete | `Delete` | `error` |

## Pagination (in MessageRepository)

```go
if limit <= 0 || limit > 100 {
    limit = 50
}
```

- Default limit: 50, max: 100.
- Uses `ORDER BY timestamp DESC LIMIT $n OFFSET $n`.

## JSONB Handling

- `any` / `interface{}` fields: `json.Marshal(msg.Raw)` before passing to SQL (error silently ignored with `_`).
- JSONB containment: `"events" @> $2::jsonb OR events @> '["All"]'::jsonb`.

## Wiring

Repos constructed in `internal/server/router.go`:

```go
sessionRepo := repo.NewSessionRepository(s.db.Pool)
```

Passed to services, middleware (`middleware.Auth`, `middleware.RequiredSession`), and the `wa.Manager`.
