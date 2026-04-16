-- =====================================================
-- Status (WhatsApp Stories)
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
