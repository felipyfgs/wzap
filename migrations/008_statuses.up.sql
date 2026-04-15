-- =====================================================
-- Status (WhatsApp Stories) — separated from wz_messages
-- =====================================================

CREATE TABLE IF NOT EXISTS wz_statuses (
    id              VARCHAR(100)  NOT NULL,
    session_id      VARCHAR(100)  NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    sender_jid      VARCHAR(255)  NOT NULL,
    from_me         BOOLEAN       NOT NULL DEFAULT false,
    status_type     VARCHAR(50)   NOT NULL DEFAULT 'status_text',
    body            TEXT          NOT NULL DEFAULT '',
    media_type      VARCHAR(50),
    media_url       TEXT,
    raw             JSONB,
    timestamp       TIMESTAMPTZ   NOT NULL,
    expires_at      TIMESTAMPTZ   NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_wz_statuses_session_timestamp
    ON wz_statuses (session_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_wz_statuses_session_sender
    ON wz_statuses (session_id, sender_jid, timestamp DESC);

-- Migrate existing status messages from wz_messages
INSERT INTO wz_statuses (id, session_id, sender_jid, from_me, status_type, body, media_type, media_url, raw, timestamp, expires_at, created_at)
SELECT
    m.id,
    m.session_id,
    COALESCE(NULLIF(m.sender_jid, ''), m.chat_jid),
    m.from_me,
    m.msg_type,
    m.body,
    m.media_type,
    m.media_url,
    m.raw,
    m.timestamp,
    m.timestamp + INTERVAL '24 hours',
    m.created_at
FROM wz_messages m
WHERE m.chat_jid LIKE 'status@%'
ON CONFLICT (id, session_id) DO NOTHING;

-- Remove migrated status messages from wz_messages
DELETE FROM wz_messages WHERE chat_jid LIKE 'status@%';
