-- =====================================================
-- Migrate camelCase identifiers → snake_case
-- Safe to run even on already-migrated databases.
-- =====================================================

-- Rename function
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- wz_sessions (rename from "wzSessions")
-- =====================================================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '"wzSessions"' OR table_name = 'wzSessions') THEN
        ALTER TABLE IF EXISTS "wzSessions" RENAME TO wz_sessions;
    END IF;
END $$;

ALTER TABLE IF EXISTS wz_sessions
    RENAME COLUMN "apiKey"    TO api_key;
ALTER TABLE IF EXISTS wz_sessions
    RENAME COLUMN "qrCode"    TO qr_code;
ALTER TABLE IF EXISTS wz_sessions
    RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE IF EXISTS wz_sessions
    RENAME COLUMN "updatedAt" TO updated_at;

-- Re-create indexes with new names (old ones dropped implicitly on rename)
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_api_key ON wz_sessions (api_key);
CREATE INDEX        IF NOT EXISTS idx_wz_sessions_name    ON wz_sessions (name);
CREATE INDEX        IF NOT EXISTS idx_wz_sessions_status  ON wz_sessions (status);
CREATE INDEX        IF NOT EXISTS idx_wz_sessions_connected ON wz_sessions (connected);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wz_sessions_jid
    ON wz_sessions (jid) WHERE jid IS NOT NULL AND jid != '';

-- Re-create trigger
DROP TRIGGER IF EXISTS "updateWzSessionsUpdatedAt" ON wz_sessions;
DROP TRIGGER IF EXISTS trg_wz_sessions_updated_at  ON wz_sessions;
CREATE TRIGGER trg_wz_sessions_updated_at
    BEFORE UPDATE ON wz_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Migrate JSONB proxy/settings keys from camelCase → PascalCase
UPDATE wz_sessions
SET proxy = (
    SELECT jsonb_strip_nulls(jsonb_build_object(
        'Host',     proxy->>'host',
        'Port',     (proxy->>'port')::int,
        'Protocol', proxy->>'protocol',
        'Username', proxy->>'username',
        'Password', proxy->>'password'
    ))
)
WHERE proxy != '{}' AND proxy ? 'host';

UPDATE wz_sessions
SET settings = jsonb_build_object(
    'AlwaysOnline',  COALESCE((settings->>'alwaysOnline')::bool, false),
    'RejectCall',    COALESCE((settings->>'rejectCall')::bool, false),
    'MsgRejectCall', COALESCE(settings->>'msgRejectCall', ''),
    'ReadMessages',  COALESCE((settings->>'readMessages')::bool, false),
    'IgnoreGroups',  COALESCE((settings->>'ignoreGroups')::bool, false),
    'IgnoreStatus',  COALESCE((settings->>'ignoreStatus')::bool, false)
)
WHERE settings != '{}' AND settings ? 'alwaysOnline';

-- =====================================================
-- wz_webhooks (rename from "wzWebhooks")
-- =====================================================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'wzWebhooks' OR table_name = '"wzWebhooks"') THEN
        ALTER TABLE IF EXISTS "wzWebhooks" RENAME TO wz_webhooks;
    END IF;
END $$;

ALTER TABLE IF EXISTS wz_webhooks
    RENAME COLUMN "sessionId"   TO session_id;
ALTER TABLE IF EXISTS wz_webhooks
    RENAME COLUMN "natsEnabled" TO nats_enabled;
ALTER TABLE IF EXISTS wz_webhooks
    RENAME COLUMN "createdAt"   TO created_at;
ALTER TABLE IF EXISTS wz_webhooks
    RENAME COLUMN "updatedAt"   TO updated_at;

CREATE INDEX IF NOT EXISTS idx_wz_webhooks_session_id ON wz_webhooks (session_id);
CREATE INDEX IF NOT EXISTS idx_wz_webhooks_enabled    ON wz_webhooks (enabled);

DROP TRIGGER IF EXISTS "updateWzWebhooksUpdatedAt" ON wz_webhooks;
DROP TRIGGER IF EXISTS trg_wz_webhooks_updated_at  ON wz_webhooks;
CREATE TRIGGER trg_wz_webhooks_updated_at
    BEFORE UPDATE ON wz_webhooks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Drop old function (replaced by set_updated_at)
DROP FUNCTION IF EXISTS "updateUpdatedAtColumn"();
