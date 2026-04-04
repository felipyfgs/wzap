-- =====================================================
-- wzap Database Schema
-- =====================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Sessions Table
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_sessions (
    id          VARCHAR(100) PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    token       VARCHAR(255) NOT NULL UNIQUE,
    jid         VARCHAR(255) NOT NULL DEFAULT '',
    qr_code     TEXT NOT NULL DEFAULT '',
    connected   INTEGER NOT NULL DEFAULT 0,
    status      VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    proxy       JSONB NOT NULL DEFAULT '{}',
    settings    JSONB NOT NULL DEFAULT '{}',
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_sessions_name
    ON wz_sessions (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_token
    ON wz_sessions (token);
CREATE INDEX IF NOT EXISTS idx_wz_sessions_status
    ON wz_sessions (status);
CREATE INDEX IF NOT EXISTS idx_wz_sessions_connected
    ON wz_sessions (connected);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_jid
    ON wz_sessions (jid)
    WHERE jid IS NOT NULL AND jid != '';

DROP TRIGGER IF EXISTS trg_wz_sessions_updated_at ON wz_sessions;
CREATE TRIGGER trg_wz_sessions_updated_at
    BEFORE UPDATE ON wz_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  wz_sessions          IS 'WhatsApp sessions managed by wzap';
COMMENT ON COLUMN wz_sessions.name     IS 'Unique URL-safe session identifier (^[a-zA-Z0-9_-]+$)';
COMMENT ON COLUMN wz_sessions.token    IS 'Session token for authentication';
COMMENT ON COLUMN wz_sessions.jid      IS 'WhatsApp device JID from whatsmeow (set after pairing)';

-- =====================================================
-- Webhooks Table
-- =====================================================
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

CREATE INDEX IF NOT EXISTS idx_wz_webhooks_session_id
    ON wz_webhooks (session_id);
CREATE INDEX IF NOT EXISTS idx_wz_webhooks_enabled
    ON wz_webhooks (enabled);

DROP TRIGGER IF EXISTS trg_wz_webhooks_updated_at ON wz_webhooks;
CREATE TRIGGER trg_wz_webhooks_updated_at
    BEFORE UPDATE ON wz_webhooks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  wz_webhooks            IS 'Webhook configurations per session';
COMMENT ON COLUMN wz_webhooks.session_id IS 'FK to wz_sessions.id';
