CREATE TABLE IF NOT EXISTS wz_messages (
    id               VARCHAR(100) NOT NULL,
    session_id       VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    chat_jid         VARCHAR(255) NOT NULL,
    sender_jid       VARCHAR(255) NOT NULL,
    from_me          BOOLEAN NOT NULL DEFAULT false,
    msg_type         VARCHAR(50) NOT NULL DEFAULT 'text',
    body             TEXT NOT NULL DEFAULT '',
    media_type       VARCHAR(50),
    media_url        TEXT,
    raw              JSONB,
    timestamp        TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cw_message_id      INTEGER,
    cw_conversation_id INTEGER,
    cw_source_id       TEXT,

    PRIMARY KEY (id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_wz_messages_session_chat
    ON wz_messages (session_id, chat_jid, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wz_messages_session_timestamp
    ON wz_messages (session_id, timestamp DESC);
