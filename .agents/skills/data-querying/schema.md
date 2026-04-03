# Database Schema — wzap

Postgres database accessed via `pgxpool.Pool`. Table and column names are `snake_case`.

## Table: `wz_sessions`

| Column | Type | Nullable | Default | Notes |
|---|---|---|---|---|
| `id` | `VARCHAR(100)` | NOT NULL | — | PK, `uuid.NewString()` |
| `name` | `VARCHAR(100)` | NOT NULL | — | Unique, `^[a-zA-Z0-9_-]+$` |
| `token` | `VARCHAR(255)` | NOT NULL | — | Session auth token — **secret** |
| `jid` | `VARCHAR(255)` | NOT NULL | `''` | WhatsApp JID after pairing |
| `qr_code` | `TEXT` | NOT NULL | `''` | Active QR string, cleared after pair |
| `connected` | `INTEGER` | NOT NULL | `0` | `1` = connected, `0` = not |
| `status` | `VARCHAR(50)` | NOT NULL | `'disconnected'` | `disconnected`, `connecting`, `connected` |
| `proxy` | `JSONB` | NOT NULL | `'{}'` | `{host, port, protocol, username, password}` |
| `settings` | `JSONB` | NOT NULL | `'{}'` | `{alwaysOnline, rejectCall, msgRejectCall, readMessages, ignoreGroups, ignoreStatus}` |
| `metadata` | `JSONB` | NULL | — | Arbitrary metadata |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Set on insert |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Auto-updated via trigger |

### Indexes

| Name | Column(s) | Type |
|---|---|---|
| `idx_wz_sessions_name` | `name` | B-tree |
| `idx_wz_sessions_token` | `token` | Unique B-tree |
| `idx_wz_sessions_status` | `status` | B-tree |
| `idx_wz_sessions_connected` | `connected` | B-tree |
| `idx_wz_sessions_jid` | `jid` | Unique partial (WHERE `jid IS NOT NULL AND jid != ''`) |

### Common queries

```sql
SELECT id, name, status, connected, created_at FROM wz_sessions ORDER BY created_at DESC;
SELECT id, name, jid, status FROM wz_sessions WHERE connected = 1;
SELECT status, COUNT(*) AS total FROM wz_sessions GROUP BY status;
SELECT id, name, status FROM wz_sessions WHERE jid = $1;
```

## Table: `wz_webhooks`

| Column | Type | Nullable | Default | Notes |
|---|---|---|---|---|
| `id` | `VARCHAR(100)` | NOT NULL | — | PK |
| `session_id` | `VARCHAR(100)` | NOT NULL | — | FK → `wz_sessions.id` ON DELETE CASCADE |
| `url` | `VARCHAR(2048)` | NOT NULL | — | Target HTTPS endpoint |
| `secret` | `VARCHAR(255)` | NULL | — | HMAC-SHA256 secret — **secret** |
| `events` | `JSONB` | NOT NULL | `'[]'` | `["Message","Connected"]` or `["All"]` |
| `enabled` | `BOOLEAN` | NOT NULL | `true` | Active/paused |
| `nats_enabled` | `BOOLEAN` | NOT NULL | `false` | Also publish to NATS |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Set on insert |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Auto-updated via trigger |

### Indexes

| Name | Column(s) | Type |
|---|---|---|
| `idx_wz_webhooks_session_id` | `session_id` | B-tree |
| `idx_wz_webhooks_enabled` | `enabled` | B-tree |

## Table: `wz_messages`

| Column | Type | Nullable | Default | Notes |
|---|---|---|---|---|
| `id` | `VARCHAR(100)` | NOT NULL | — | PK part 1 |
| `session_id` | `VARCHAR(100)` | NOT NULL | — | PK part 2, FK → `wz_sessions.id` ON DELETE CASCADE |
| `chat_jid` | `VARCHAR(255)` | NOT NULL | — | Chat JID |
| `sender_jid` | `VARCHAR(255)` | NOT NULL | — | Sender JID |
| `from_me` | `BOOLEAN` | NOT NULL | `false` | Outgoing message |
| `msg_type` | `VARCHAR(50)` | NOT NULL | `'text'` | text, image, video, audio, document, sticker, etc. |
| `body` | `TEXT` | NOT NULL | `''` | Message text/caption |
| `media_type` | `VARCHAR(50)` | NULL | — | MIME type |
| `media_url` | `TEXT` | NULL | — | S3 URL |
| `raw` | `JSONB` | NULL | — | Full protobuf JSON |
| `timestamp` | `TIMESTAMPTZ` | NOT NULL | — | WhatsApp timestamp |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | `NOW()` | Insert time |

PK: `(id, session_id)`. Upsert: `ON CONFLICT (id, session_id) DO NOTHING`.

### Indexes

| Name | Column(s) | Type |
|---|---|---|
| `idx_wz_messages_session_chat` | `(session_id, chat_jid, timestamp DESC)` | B-tree |
| `idx_wz_messages_session_timestamp` | `(session_id, timestamp DESC)` | B-tree |
