-- =====================================================
-- wzap Database Schema
-- =====================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- =====================================================
-- Sessions Table
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_sessions (
    id VARCHAR(100) PRIMARY KEY,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    device_jid VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'INIT',
    is_connected BOOLEAN NOT NULL DEFAULT false,
    qr_code TEXT,
    qr_expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Ensure api_key column exists (migration helper)
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='wz_sessions' AND column_name='api_key') THEN
        ALTER TABLE wz_sessions ADD COLUMN api_key VARCHAR(255);
        UPDATE wz_sessions SET api_key = 'sk_' || id WHERE api_key IS NULL;
        ALTER TABLE wz_sessions ALTER COLUMN api_key SET NOT NULL;
        ALTER TABLE wz_sessions ADD CONSTRAINT wz_sessions_api_key_unique UNIQUE (api_key);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_wz_sessions_status ON wz_sessions (status);
CREATE INDEX IF NOT EXISTS idx_wz_sessions_connected ON wz_sessions (is_connected);

CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_device_jid_unique
    ON wz_sessions (device_jid)
    WHERE device_jid IS NOT NULL AND device_jid != '';

DROP TRIGGER IF EXISTS update_wz_sessions_updated_at ON wz_sessions;
CREATE TRIGGER update_wz_sessions_updated_at
    BEFORE UPDATE ON wz_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE wz_sessions IS 'WhatsApp sessions managed by wzap';
COMMENT ON COLUMN wz_sessions.device_jid IS 'WhatsApp device JID from whatsmeow (set after pairing)';

-- =====================================================
-- Webhooks Table
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_webhooks (
    id VARCHAR(100) PRIMARY KEY,
    session_id VARCHAR(100) NOT NULL REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255),
    events JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_webhooks_session_id ON wz_webhooks (session_id);
CREATE INDEX IF NOT EXISTS idx_wz_webhooks_enabled ON wz_webhooks (enabled);

DROP TRIGGER IF EXISTS update_wz_webhooks_updated_at ON wz_webhooks;
CREATE TRIGGER update_wz_webhooks_updated_at
    BEFORE UPDATE ON wz_webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE wz_webhooks IS 'Webhook configurations for wzap sessions';
