CREATE TABLE IF NOT EXISTS wz_chats (
    session_id              VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid                VARCHAR(255) NOT NULL,
    name                    TEXT,
    display_name            TEXT,
    chat_type               VARCHAR(50),
    archived                BOOLEAN,
    pinned                  INTEGER,
    read_only               BOOLEAN,
    marked_as_unread        BOOLEAN,
    unread_count            INTEGER,
    unread_mention_count    INTEGER,
    last_message_id         VARCHAR(100),
    last_message_at         TIMESTAMPTZ,
    conversation_timestamp  TIMESTAMPTZ,
    pn_jid                  VARCHAR(255),
    lid_jid                 VARCHAR(255),
    username                VARCHAR(255),
    account_lid             VARCHAR(255),
    source                  VARCHAR(32)  NOT NULL DEFAULT 'live',
    source_sync_type        VARCHAR(64),
    history_chunk_order     INTEGER,
    raw                     JSONB,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, chat_jid)
);

CREATE INDEX IF NOT EXISTS idx_wz_chats_session_last_message
    ON wz_chats (session_id, last_message_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_wz_chats_session_source
    ON wz_chats (session_id, source);

CREATE INDEX IF NOT EXISTS idx_wz_chats_session_pn_jid
    ON wz_chats (session_id, pn_jid)
    WHERE pn_jid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wz_chats_session_lid_jid
    ON wz_chats (session_id, lid_jid)
    WHERE lid_jid IS NOT NULL;

DROP TRIGGER IF EXISTS trg_wz_chats_updated_at ON wz_chats;
CREATE TRIGGER trg_wz_chats_updated_at
    BEFORE UPDATE ON wz_chats
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE wz_messages
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'live',
    ADD COLUMN IF NOT EXISTS source_sync_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS history_chunk_order INTEGER,
    ADD COLUMN IF NOT EXISTS history_message_order BIGINT,
    ADD COLUMN IF NOT EXISTS imported_to_chatwoot_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_wz_messages_session_source
    ON wz_messages (session_id, source, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_wz_messages_history_order
    ON wz_messages (session_id, history_chunk_order, timestamp, history_message_order)
    WHERE history_chunk_order IS NOT NULL;
