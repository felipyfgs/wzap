---
name: data-querying
description: Add or extend Postgres queries in wzap's repo layer to answer questions about sessions, webhooks, and events. Use when a stakeholder needs session metrics, webhook delivery history, event statistics, or any data slice from the wzap database — and the answer should be a typed Go method or a raw SQL artifact.
---

Add or extend repository methods in `internal/repo/` to query the wzap Postgres database.

## Workflow

1. Identify which table(s) are involved — read [schema.md](schema.md).
2. Check if an existing repo method already covers the need (`SessionRepository`, `WebhookRepository`, `MessageRepository`).
3. Draft the SQL, validate against a limited filter first.
4. Implement the Go method following the pgx patterns in [query-patterns.md](query-patterns.md).
5. Add the method to the right repository struct; do not add raw SQL outside the repo layer.
6. Verify: `go build ./... && go test -v -race ./internal/repo/...`

## Tables

| Table | Repo | Purpose |
|---|---|---|
| `wz_sessions` | `SessionRepository` | WhatsApp sessions |
| `wz_webhooks` | `WebhookRepository` | Webhook subscriptions |
| `wz_messages` | `MessageRepository` | Persisted messages |

## Rules

- All queries go through `r.db *pgxpool.Pool` via `QueryRow`, `Query`, or `Exec`.
- Table/column names are `snake_case`: `wz_sessions`, `token`, `created_at`. No double quotes needed.
- Nullable text columns: `COALESCE(col, '')` in SELECT.
- Parameters: positional `$1`, `$2`, …
- JSONB containment: `events @> $1::jsonb`.
- Wrap errors: `fmt.Errorf("failed to <verb> <noun>: %w", err)`.
- `context.Context` always first param. Receiver is `r`.

## Sensitive fields

- `token` in `wz_sessions` is a bearer token — never SELECT it in list queries or log it.
- `secret` in `wz_webhooks` is for HMAC signing — same rule.

## Gotchas

- `wz_sessions.jid` has a unique partial index (`WHERE jid IS NOT NULL AND jid != ''`) — empty strings are allowed for multiple unpaired sessions.
- `wz_webhooks.session_id` has `ON DELETE CASCADE` — deleting a session removes its webhooks.
- `wz_messages` has a composite primary key `(id, session_id)` — use upsert with `ON CONFLICT (id, session_id) DO NOTHING`.
- `updated_at` is auto-updated by a trigger — no need to set manually in UPDATE queries (though the repo code does set it defensively).
