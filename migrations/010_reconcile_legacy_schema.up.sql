-- =====================================================
-- Reconcile databases upgraded across migration consolidation
-- =====================================================
--
-- Releases before the 9-to-5 migration consolidation stored part of the
-- current schema in separate migrations. A legacy database could therefore
-- have the consolidated file names recorded as a baseline while still
-- missing objects folded into those files. Every operation below is
-- idempotent and preserves existing application data.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Core tables may have been partially baselined because 001_schema contains
-- both sessions and webhooks but uses sessions as its legacy anchor.
CREATE TABLE IF NOT EXISTS wz_sessions (
    id          VARCHAR(100) PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    token       VARCHAR(255) NOT NULL UNIQUE,
    jid         VARCHAR(255) NOT NULL DEFAULT '',
    qr_code     TEXT NOT NULL DEFAULT '',
    status      VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    connected   INTEGER NOT NULL DEFAULT 0,
    engine      VARCHAR(20) NOT NULL DEFAULT 'whatsmeow',
    proxy       JSONB NOT NULL DEFAULT '{}',
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE wz_sessions
    ADD COLUMN IF NOT EXISTS jid VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS qr_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    ADD COLUMN IF NOT EXISTS connected INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS engine VARCHAR(20) NOT NULL DEFAULT 'whatsmeow',
    ADD COLUMN IF NOT EXISTS proxy JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE wz_sessions SET engine = 'whatsmeow' WHERE engine IS NULL;
ALTER TABLE wz_sessions
    ALTER COLUMN engine SET DEFAULT 'whatsmeow',
    ALTER COLUMN engine SET NOT NULL;

CREATE TABLE IF NOT EXISTS wz_webhooks (
    id           VARCHAR(100) PRIMARY KEY,
    session_id   VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url          VARCHAR(2048) NOT NULL,
    secret       VARCHAR(255),
    events       JSONB NOT NULL DEFAULT '[]',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    nats_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE wz_webhooks
    ADD COLUMN IF NOT EXISTS secret VARCHAR(255),
    ADD COLUMN IF NOT EXISTS events JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS nats_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Message history columns and TEXT widening were separate migrations before
-- consolidation and can be absent independently of the base messages table.
CREATE TABLE IF NOT EXISTS wz_messages (
    id                      VARCHAR(100) NOT NULL,
    session_id              VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid                VARCHAR(255) NOT NULL,
    sender_jid              VARCHAR(255) NOT NULL,
    from_me                 BOOLEAN NOT NULL DEFAULT false,
    msg_type                TEXT NOT NULL DEFAULT 'text',
    body                    TEXT NOT NULL DEFAULT '',
    media_type              TEXT,
    media_url               TEXT,
    raw                     JSONB,
    timestamp               TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cw_message_id           INTEGER,
    cw_conversation_id      INTEGER,
    cw_source_id            TEXT,
    source                  VARCHAR(32) NOT NULL DEFAULT 'live',
    source_sync_type        VARCHAR(64),
    history_chunk_order     INTEGER,
    history_message_order   BIGINT,
    imported_to_chatwoot_at TIMESTAMPTZ,
    PRIMARY KEY (id, session_id)
);

ALTER TABLE wz_messages
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'live',
    ADD COLUMN IF NOT EXISTS source_sync_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS history_chunk_order INTEGER,
    ADD COLUMN IF NOT EXISTS history_message_order BIGINT,
    ADD COLUMN IF NOT EXISTS imported_to_chatwoot_at TIMESTAMPTZ,
    ALTER COLUMN msg_type TYPE TEXT,
    ALTER COLUMN media_type TYPE TEXT;

CREATE TABLE IF NOT EXISTS wz_chats (
    session_id             VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid               VARCHAR(255) NOT NULL,
    name                   TEXT,
    display_name           TEXT,
    chat_type              VARCHAR(50),
    archived               BOOLEAN,
    pinned                 INTEGER,
    read_only              BOOLEAN,
    marked_as_unread       BOOLEAN,
    unread_count           INTEGER,
    unread_mention_count   INTEGER,
    last_message_id        VARCHAR(100),
    last_message_at        TIMESTAMPTZ,
    conversation_timestamp TIMESTAMPTZ,
    pn_jid                 VARCHAR(255),
    lid_jid                VARCHAR(255),
    username               VARCHAR(255),
    account_lid            VARCHAR(255),
    source                 VARCHAR(32) NOT NULL DEFAULT 'live',
    source_sync_type       VARCHAR(64),
    history_chunk_order    INTEGER,
    raw                    JSONB,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, chat_jid)
);

ALTER TABLE wz_chats
    ADD COLUMN IF NOT EXISTS name TEXT,
    ADD COLUMN IF NOT EXISTS display_name TEXT,
    ADD COLUMN IF NOT EXISTS chat_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS archived BOOLEAN,
    ADD COLUMN IF NOT EXISTS pinned INTEGER,
    ADD COLUMN IF NOT EXISTS read_only BOOLEAN,
    ADD COLUMN IF NOT EXISTS marked_as_unread BOOLEAN,
    ADD COLUMN IF NOT EXISTS unread_count INTEGER,
    ADD COLUMN IF NOT EXISTS unread_mention_count INTEGER,
    ADD COLUMN IF NOT EXISTS last_message_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS conversation_timestamp TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS pn_jid VARCHAR(255),
    ADD COLUMN IF NOT EXISTS lid_jid VARCHAR(255),
    ADD COLUMN IF NOT EXISTS username VARCHAR(255),
    ADD COLUMN IF NOT EXISTS account_lid VARCHAR(255),
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'live',
    ADD COLUMN IF NOT EXISTS source_sync_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS history_chunk_order INTEGER,
    ADD COLUMN IF NOT EXISTS raw JSONB,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS wz_statuses (
    id          VARCHAR(100) NOT NULL,
    session_id  VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    sender_jid  VARCHAR(255) NOT NULL,
    from_me     BOOLEAN NOT NULL DEFAULT false,
    status_type VARCHAR(50) NOT NULL DEFAULT 'status_text',
    body        TEXT NOT NULL DEFAULT '',
    media_type  VARCHAR(50),
    media_url   TEXT,
    raw         JSONB,
    timestamp   TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, session_id)
);

-- Chatwoot gained fields both by editing its original CREATE TABLE and via
-- incremental migrations. Reconcile every field absent from the first
-- released schema so the next error is not merely exposed after inbox_type.
CREATE TABLE IF NOT EXISTS wz_chatwoot (
    session_id            VARCHAR(100) PRIMARY KEY REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url                   VARCHAR(2048) NOT NULL,
    account_id            INTEGER NOT NULL,
    token                 VARCHAR(255) NOT NULL,
    inbox_id              INTEGER NOT NULL,
    inbox_name            VARCHAR(255) NOT NULL DEFAULT 'wzap',
    inbox_type            VARCHAR(20) NOT NULL DEFAULT 'api',
    enabled               BOOLEAN NOT NULL DEFAULT true,
    webhook_token         VARCHAR(255) NOT NULL DEFAULT '',
    sign_msg              BOOLEAN NOT NULL DEFAULT false,
    sign_delimiter        VARCHAR(50) NOT NULL DEFAULT '\n',
    reopen_conversation   BOOLEAN NOT NULL DEFAULT true,
    conversation_pending  BOOLEAN NOT NULL DEFAULT false,
    message_read          BOOLEAN NOT NULL DEFAULT false,
    merge_br_contacts     BOOLEAN NOT NULL DEFAULT true,
    ignore_groups         BOOLEAN NOT NULL DEFAULT false,
    ignore_jids           TEXT[] NOT NULL DEFAULT '{}',
    import_on_connect     BOOLEAN NOT NULL DEFAULT false,
    import_period         VARCHAR(10) NOT NULL DEFAULT '7d',
    timeout_text_seconds  INTEGER NOT NULL DEFAULT 10,
    timeout_media_seconds INTEGER NOT NULL DEFAULT 60,
    timeout_large_seconds INTEGER NOT NULL DEFAULT 300,
    redis_url             VARCHAR(255) NOT NULL DEFAULT '',
    database_uri          TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE wz_chatwoot
    ADD COLUMN IF NOT EXISTS inbox_type VARCHAR(20) NOT NULL DEFAULT 'api',
    ADD COLUMN IF NOT EXISTS webhook_token VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS conversation_pending BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS message_read BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS ignore_jids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS import_on_connect BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS import_period VARCHAR(10) NOT NULL DEFAULT '7d',
    ADD COLUMN IF NOT EXISTS timeout_text_seconds INTEGER NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS timeout_media_seconds INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS timeout_large_seconds INTEGER NOT NULL DEFAULT 300,
    ADD COLUMN IF NOT EXISTS redis_url VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS database_uri TEXT NOT NULL DEFAULT '';

-- Recreate the complete index and trigger set. IF NOT EXISTS makes this safe
-- for both fresh and upgraded databases.
CREATE INDEX IF NOT EXISTS idx_wz_sessions_name ON wz_sessions (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_token ON wz_sessions (token);
CREATE INDEX IF NOT EXISTS idx_wz_sessions_status ON wz_sessions (status);
CREATE INDEX IF NOT EXISTS idx_wz_sessions_connected ON wz_sessions (connected);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_jid
    ON wz_sessions (jid) WHERE jid IS NOT NULL AND jid != '';

CREATE INDEX IF NOT EXISTS idx_wz_webhooks_session_id ON wz_webhooks (session_id);
CREATE INDEX IF NOT EXISTS idx_wz_webhooks_enabled ON wz_webhooks (enabled);

CREATE INDEX IF NOT EXISTS idx_wz_messages_session_chat
    ON wz_messages (session_id, chat_jid, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_timestamp
    ON wz_messages (session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_conversation
    ON wz_messages (session_id, cw_conversation_id) WHERE cw_conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_message
    ON wz_messages (session_id, cw_message_id) WHERE cw_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_source
    ON wz_messages (session_id, cw_source_id) WHERE cw_source_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_source
    ON wz_messages (session_id, source, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_messages_history_order
    ON wz_messages (session_id, history_chunk_order, timestamp, history_message_order)
    WHERE history_chunk_order IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wz_chats_session_last_message
    ON wz_chats (session_id, last_message_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_wz_chats_session_source ON wz_chats (session_id, source);
CREATE INDEX IF NOT EXISTS idx_wz_chats_session_pn_jid
    ON wz_chats (session_id, pn_jid) WHERE pn_jid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wz_chats_session_lid_jid
    ON wz_chats (session_id, lid_jid) WHERE lid_jid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wz_statuses_session_timestamp
    ON wz_statuses (session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_statuses_session_sender
    ON wz_statuses (session_id, sender_jid, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_statuses_expires_at ON wz_statuses (expires_at);

CREATE INDEX IF NOT EXISTS idx_wz_chatwoot_enabled ON wz_chatwoot (enabled);
CREATE INDEX IF NOT EXISTS idx_wz_chatwoot_inbox_type ON wz_chatwoot (inbox_type);

DROP TRIGGER IF EXISTS trg_wz_sessions_updated_at ON wz_sessions;
CREATE TRIGGER trg_wz_sessions_updated_at
    BEFORE UPDATE ON wz_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_wz_webhooks_updated_at ON wz_webhooks;
CREATE TRIGGER trg_wz_webhooks_updated_at
    BEFORE UPDATE ON wz_webhooks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_wz_chats_updated_at ON wz_chats;
CREATE TRIGGER trg_wz_chats_updated_at
    BEFORE UPDATE ON wz_chats
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_wz_chatwoot_updated_at ON wz_chatwoot;
CREATE TRIGGER trg_wz_chatwoot_updated_at
    BEFORE UPDATE ON wz_chatwoot
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
