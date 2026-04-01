# Database Schema — wzap

Postgres database accessed via `pgxpool.Pool`. All table and column names use camelCase wrapped in double quotes.

---

## Table: `"wzSessions"`

Stores one row per WhatsApp session managed by wzap.

| Column | Type | Nullable | Default | Notes |
|---|---|---|---|---|
| `"id"` | `VARCHAR(100)` | NOT NULL | — | Primary key, format `uuid.NewString()` |
| `"name"` | `VARCHAR(100)` | NOT NULL | — | Unique human-readable identifier; `^[a-zA-Z0-9_-]+$` |
| `"apiKey"` | `VARCHAR(255)` | NOT NULL | — | Bearer token for session-scoped auth; `sk_<uuid>` if auto-generated, otherwise the caller-supplied value — **treat as secret** |
| `"jid"` | `VARCHAR(255)` | NULL | `''` | WhatsApp JID once paired, e.g. `5511999@s.whatsapp.net`; empty string when unpaired |
| `"qrCode"` | `TEXT` | NULL | `''` | Raw QR string for active pairing; cleared after successful pair |
| `"connected"` | `INTEGER` | NULL | `0` | `1` = connected, `0` = not connected |
| `"status"` | `VARCHAR(50)` | NOT NULL | `'disconnected'` | One of: `"disconnected"`, `"connecting"`, `"connected"` |
| `"proxy"` | `JSONB` | NOT NULL | `'{}'` | `SessionProxy{Host, Port, Protocol, Username, Password}` |
| `"settings"` | `JSONB` | NOT NULL | `'{}'` | `SessionSettings{alwaysOnline, rejectCall, msgRejectCall, readMessages, ignoreGroups, ignoreStatus}` |
| `"metadata"` | `JSONB` | NULL | — | Arbitrary metadata, no enforced schema |
| `"createdAt"` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Set on insert |
| `"updatedAt"` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | **Auto-updated** via `BEFORE UPDATE` trigger (`updateWzSessionsUpdatedAt`) |

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

### Indexes

| Name | Columns | Type |
|---|---|---|
| `idxWzSessionsName` | `"name"` | B-tree |
| `idxWzSessionsApiKey` | `"apiKey"` | Unique B-tree |
| `idxWzSessionsStatus` | `"status"` | B-tree |
| `idxWzSessionsConnected` | `"connected"` | B-tree |
| `idxWzSessionsJidUnique` | `"jid"` | Unique partial (WHERE `"jid" IS NOT NULL AND "jid" != ''`) |

> Never SELECT `"apiKey"` in list queries. Only fetch it when the caller is authenticated as admin and the purpose is session creation confirmation.

---

## Table: `"wzWebhooks"`

Stores webhook subscriptions. Each session can have multiple webhooks.

| Column | Type | Nullable | Default | Notes |
|---|---|---|---|---|
| `"id"` | `VARCHAR(100)` | NOT NULL | — | Primary key |
| `"sessionId"` | `VARCHAR(100)` | NOT NULL | — | FK → `"wzSessions"."id"` **ON DELETE CASCADE** |
| `"url"` | `VARCHAR(2048)` | NOT NULL | — | Target HTTPS endpoint for event delivery |
| `"secret"` | `VARCHAR(255)` | NULL | — | HMAC-SHA256 signing secret — **treat as secret** |
| `"events"` | `JSONB` | NOT NULL | `'[]'` | Array of `EventType` strings, e.g. `["Message","Connected"]`; `["All"]` = wildcard |
| `"enabled"` | `BOOLEAN` | NOT NULL | `true` | `true` = active; `false` = paused |
| `"natsEnabled"` | `BOOLEAN` | NOT NULL | `false` | `true` = also publish to NATS subject |
| `"createdAt"` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Set on insert |
| `"updatedAt"` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | **Auto-updated** via `BEFORE UPDATE` trigger (`updateWzWebhooksUpdatedAt`) |

### `"events"` JSONB array examples

```json
["All"]                               // receives every event
["Message", "Connected"]              // receives only these two
["GroupInfo", "JoinedGroup"]
```

### Common queries

```sql
-- Webhook count per session
SELECT "sessionId", COUNT(*) AS total, SUM(CASE WHEN "enabled" THEN 1 ELSE 0 END) AS active
FROM "wzWebhooks"
GROUP BY "sessionId";
```

### Indexes

| Name | Columns | Type |
|---|---|---|
| `idxWzWebhooksSessionId` | `"sessionId"` | B-tree |
| `idxWzWebhooksEnabled` | `"enabled"` | B-tree |

> Never SELECT `"secret"` in list queries.

---

## Naming convention summary

- Tables: `"wzPascalCase"` (e.g. `"wzSessions"`, `"wzWebhooks"`)
- Columns: `"camelCase"` always wrapped in double quotes
- Parameters: positional `$1`, `$2`, … (pgx / PostgreSQL style)
- Timestamps: `timestamptz` — Go type `time.Time`, scanned directly by pgx
- JSONB structs: scanned directly into Go structs by pgx (auto-marshalling)
- Both tables have a `BEFORE UPDATE` trigger that auto-updates `"updatedAt"` to `NOW()` — no manual update needed
- `"wzWebhooks"."sessionId"` has `ON DELETE CASCADE` — deleting a session removes its webhooks automatically
