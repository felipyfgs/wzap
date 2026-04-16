-- =====================================================
-- Messages
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_messages (
    id                      VARCHAR(100)  NOT NULL,
    session_id              VARCHAR(100)  NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,

    -- WhatsApp context
    chat_jid                VARCHAR(255)  NOT NULL,
    sender_jid              VARCHAR(255)  NOT NULL,
    from_me                 BOOLEAN       NOT NULL DEFAULT false,
    msg_type                TEXT          NOT NULL DEFAULT 'text',
    body                    TEXT          NOT NULL DEFAULT '',
    media_type              TEXT,
    media_url               TEXT,
    raw                     JSONB,
    timestamp               TIMESTAMPTZ   NOT NULL,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Chatwoot cross-reference (populated after CW sync)
    cw_message_id           INTEGER,
    cw_conversation_id      INTEGER,
    cw_source_id            TEXT,

    -- History sync
    source                  VARCHAR(32)   NOT NULL DEFAULT 'live',
    source_sync_type        VARCHAR(64),
    history_chunk_order     INTEGER,
    history_message_order   BIGINT,
    imported_to_chatwoot_at TIMESTAMPTZ,

    PRIMARY KEY (id, session_id)
);

-- Query: list messages for a chat
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_chat
    ON wz_messages (session_id, chat_jid, timestamp DESC);

-- Query: list messages for a session
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_timestamp
    ON wz_messages (session_id, timestamp DESC);

-- Query: find message by CW conversation ID
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_conversation
    ON wz_messages (session_id, cw_conversation_id)
    WHERE cw_conversation_id IS NOT NULL;

-- Query: find message by CW message ID
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_message
    ON wz_messages (session_id, cw_message_id)
    WHERE cw_message_id IS NOT NULL;

-- Query: find message by CW source ID (idempotency + reply resolution)
CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_source
    ON wz_messages (session_id, cw_source_id)
    WHERE cw_source_id IS NOT NULL;

-- Query: messages by source (live vs history)
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_source
    ON wz_messages (session_id, source, timestamp DESC);

-- Query: history chunk ordering
CREATE INDEX IF NOT EXISTS idx_wz_messages_history_order
    ON wz_messages (session_id, history_chunk_order, timestamp, history_message_order)
    WHERE history_chunk_order IS NOT NULL;
