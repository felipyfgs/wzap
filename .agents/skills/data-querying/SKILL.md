---
name: data-querying
description: Add or extend Postgres queries in wzap's repo layer to answer operational questions about sessions, webhooks, and events. Use when a stakeholder needs session metrics, webhook delivery history, event statistics, or any data slice from the wzap database — and the answer should be reproducible via a typed Go method or a raw SQL artifact.
---
# Skill: Data querying — wzap

## Purpose

Add or extend repository methods in `internal/repo/` to answer operational questions about sessions, webhooks, and WhatsApp events from the wzap Postgres database — producing results that are typed, safe, and re-runnable.

## When to use this skill

- Someone needs **session metrics**: how many sessions are connected, disconnected, or in pairing state.
- Someone needs **webhook analytics**: which webhooks fired, for which events, and to which URLs.
- A new admin/monitoring feature needs a query not yet covered by existing repo methods.
- A bug investigation requires understanding the state stored in `"wzSessions"` or `"wzWebhooks"`.

## Data sources

| Table | Package | Access via |
|---|---|---|
| `"wzSessions"` | `internal/repo/` | `SessionRepository` |
| `"wzWebhooks"` | `internal/repo/` | `WebhookRepository` |

Full column details are in `schema.md`. Query patterns and pgx snippets are in `query-patterns.md`.

## Inputs

- **Business question**: one or two sentences describing what needs to be known.
- **Filters**: session ID, status, event type, date range, enabled flag, etc.
- **Sensitivity**: `"apiKey"` and `"secret"` are sensitive — never include them in logs, response payloads, or analysis artifacts.

## Out of scope

- Direct queries against the whatsmeow SQLite device store (managed by the library).
- Creating new Postgres tables or running migrations without an explicit migration PR.
- Exporting raw `apiKey` or webhook `secret` values outside the secured API layer.

## Conventions

- All queries go through `pgxpool.Pool` via `r.db.QueryRow`, `r.db.Query`, or `r.db.Exec`.
- Table and column names are camelCase wrapped in double quotes: `"wzSessions"`, `"apiKey"`, `"createdAt"`.
- Nullable columns use `COALESCE`: `COALESCE("jid", '')`.
- JSONB array containment for event filtering: `"events" @> $1::jsonb`.
- Parameters are positional `$1`, `$2`, …
- Errors are wrapped: `fmt.Errorf("failed to <action>: %w", err)`.

## Required artifacts

- A typed Go method on the appropriate repository struct, committed to `internal/repo/`.
- OR a raw SQL file committed to a clearly named location with a comment header explaining intent.
- A short summary of the query's methodology, assumptions, and any known data-quality caveats.

## Implementation checklist

1. Identify which table(s) are involved — consult `schema.md`.
2. Determine whether an existing repo method already covers the need (check `SessionRepository` and `WebhookRepository`).
3. Draft the SQL, validate it against a limited filter (e.g. single session ID) first.
4. Implement the Go method following the pgx patterns in `query-patterns.md`.
5. Add the method to the right repository struct; do not add raw SQL outside the repo layer.
6. Verify with `go build ./...` and `go test -v -race ./internal/repo/...`.

## Verification

```bash
go build ./...
go test -v -race ./internal/repo/...
```

The skill is complete when:

- The query returns correct results for known sessions and webhooks.
- No `apiKey` or `secret` value appears in logs, test output, or analysis artifacts.
- Another engineer can re-run the query or call the new repo method without modification.

## Safety and escalation

- `"apiKey"` in `"wzSessions"` is equivalent to a bearer token — treat it like a password; never select it in list queries or log it.
- `"secret"` in `"wzWebhooks"` is used for HMAC signing — same rule applies.
- If the query requires joining tables that don't exist yet, stop and create a migration PR first.

## Companion files

- `schema.md` — full column list for `"wzSessions"` and `"wzWebhooks"`.
- `query-patterns.md` — pgx QueryRow, Query, Exec, and JSONB patterns with real examples.
