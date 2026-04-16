-- =====================================================
-- wzap Database Schema — Core Tables
-- =====================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Sessions
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_sessions (
    id                   VARCHAR(100)  PRIMARY KEY,
    name                 VARCHAR(100)  NOT NULL UNIQUE,
    token                VARCHAR(255)  NOT NULL UNIQUE,
    jid                  VARCHAR(255)  NOT NULL DEFAULT '',
    qr_code              TEXT          NOT NULL DEFAULT '',
    status               VARCHAR(50)   NOT NULL DEFAULT 'disconnected',
    connected            INTEGER       NOT NULL DEFAULT 0,
    engine               VARCHAR(20)   NOT NULL DEFAULT 'whatsmeow',

    -- Extensible config
    proxy                JSONB         NOT NULL DEFAULT '{}',
    settings             JSONB         NOT NULL DEFAULT '{}',

    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
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

-- =====================================================
-- Webhooks
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_webhooks (
    id           VARCHAR(100)  PRIMARY KEY,
    session_id   VARCHAR(100)  NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url          VARCHAR(2048) NOT NULL,
    secret       VARCHAR(255),
    events       JSONB         NOT NULL DEFAULT '[]',
    enabled      BOOLEAN       NOT NULL DEFAULT true,
    nats_enabled BOOLEAN       NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_webhooks_session_id
    ON wz_webhooks (session_id);
CREATE INDEX IF NOT EXISTS idx_wz_webhooks_enabled
    ON wz_webhooks (enabled);

DROP TRIGGER IF EXISTS trg_wz_webhooks_updated_at ON wz_webhooks;
CREATE TRIGGER trg_wz_webhooks_updated_at
    BEFORE UPDATE ON wz_webhooks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
