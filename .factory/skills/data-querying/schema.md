# Database Schema — wzap

Postgres database accessed via `pgxpool.Pool`. All table and column names use camelCase wrapped in double quotes.

---

## Table: `"wzSessions"`

Stores one row per WhatsApp session managed by wzap.

| Column | Type | Nullable | Notes |
|---|---|---|---|
| `"id"` | `text` (UUID) | NOT NULL | Primary key, format `uuid.NewString()` |
| `"name"` | `text` | NOT NULL | Unique human-readable identifier; `^[a-zA-Z0-9_-]+$` |
| `"apiKey"` | `text` | NOT NULL | Bearer token for session-scoped auth; format `sk_<uuid>` — **treat as secret** |
| `"jid"` | `text` | NULL | WhatsApp JID once paired, e.g. `5511999@s.whatsapp.net`; empty string when unpaired |
| `"qrCode"` | `text` | NULL | Raw QR string for active pairing; cleared after successful pair |
| `"connected"` | `integer` | NOT NULL | `1` = connected, `0` = not connected |
| `"status"` | `text` | NOT NULL | One of: `"disconnected"`, `"connecting"`, `"pairing"`, `"connected"` |
| `"proxy"` | `jsonb` | NOT NULL | `SessionProxy{Host, Port, Protocol, Username, Password}` — empty object by default |
| `"settings"` | `jsonb` | NOT NULL | `SessionSettings{alwaysOnline, rejectCall, msgRejectCall, readMessages, ignoreGroups, ignoreStatus}` |
| `"createdAt"` | `timestamptz` | NOT NULL | Set on insert |
| `"updatedAt"` | `timestamptz` | NOT NULL | Set on insert; update manually on writes |

### `"proxy"` JSONB shape

```json
{
  "host": "",
  "port": 0,
  "protocol": "",
  "username": "",
  "password": ""
}
```

### `"settings"` JSONB shape

```json
{
  "alwaysOnline": false,
  "rejectCall": false,
  "msgRejectCall": "",
  "readMessages": false,
  "ignoreGroups": false,
  "ignoreStatus": false
}
```

### Common queries

```sql
-- All sessions ordered by creation
SELECT "id", "name", "status", "connected", "createdAt"
FROM "wzSessions"
ORDER BY "createdAt" DESC;

-- Connected sessions only
SELECT "id", "name", "jid", "status"
FROM "wzSessions"
WHERE "connected" = 1;

-- Session count by status
SELECT "status", COUNT(*) AS total
FROM "wzSessions"
GROUP BY "status";

-- Find session by JID
SELECT "id", "name", "status"
FROM "wzSessions"
WHERE "jid" = $1;
```

> Never SELECT `"apiKey"` in list queries. Only fetch it when the caller is authenticated as admin and the purpose is session creation confirmation.

---

## Table: `"wzWebhooks"`

Stores webhook subscriptions. Each session can have multiple webhooks.

| Column | Type | Nullable | Notes |
|---|---|---|---|
| `"id"` | `text` (UUID) | NOT NULL | Primary key |
| `"sessionId"` | `text` (UUID) | NOT NULL | FK → `"wzSessions"."id"` |
| `"url"` | `text` | NOT NULL | Target HTTPS endpoint for event delivery |
| `"secret"` | `text` | NULL | HMAC-SHA256 signing secret — **treat as secret** |
| `"events"` | `jsonb` | NOT NULL | Array of `EventType` strings, e.g. `["Message","Connected"]`; `["All"]` = wildcard |
| `"enabled"` | `boolean` | NOT NULL | `true` = active; `false` = paused |
| `"natsEnabled"` | `boolean` | NOT NULL | `true` = also publish to NATS subject |
| `"createdAt"` | `timestamptz` | NOT NULL | Set on insert |
| `"updatedAt"` | `timestamptz` | NULL | Updated on webhook edits |

### `"events"` JSONB array examples

```json
["All"]                               // receives every event
["Message", "Connected"]              // receives only these two
["GroupInfo", "JoinedGroup"]
```

### Common queries

```sql
-- All webhooks for a session
SELECT "id", "url", "events", "enabled"
FROM "wzWebhooks"
WHERE "sessionId" = $1
ORDER BY "createdAt" DESC;

-- Active webhooks that match a specific event
SELECT "id", "url", "natsEnabled"
FROM "wzWebhooks"
WHERE "sessionId" = $1
  AND "enabled" = true
  AND ("events" @> $2::jsonb OR "events" @> '["All"]'::jsonb);
-- $2 = JSON array of one event, e.g. '["Message"]'

-- Webhook count per session
SELECT "sessionId", COUNT(*) AS total, SUM(CASE WHEN "enabled" THEN 1 ELSE 0 END) AS active
FROM "wzWebhooks"
GROUP BY "sessionId";
```

> Never SELECT `"secret"` in list queries.

---

## Naming convention summary

- Tables: `"wzPascalCase"` (e.g. `"wzSessions"`, `"wzWebhooks"`)
- Columns: `"camelCase"` always wrapped in double quotes
- Parameters: positional `$1`, `$2`, … (pgx / PostgreSQL style)
- Timestamps: `timestamptz` — Go type `time.Time`, scanned directly by pgx
- JSONB structs: scanned directly into Go structs by pgx (auto-marshalling)
