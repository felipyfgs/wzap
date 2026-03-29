-- =====================================================
-- wzap Database Schema
-- =====================================================

CREATE OR REPLACE FUNCTION "updateUpdatedAtColumn"()
RETURNS TRIGGER AS $$
BEGIN
    NEW."updatedAt" = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- =====================================================
-- Sessions Table (primary auth entity)
-- =====================================================
CREATE TABLE IF NOT EXISTS "wzSessions" (
    "id" VARCHAR(100) PRIMARY KEY,
    "name" VARCHAR(100) NOT NULL UNIQUE,
    "token" VARCHAR(255) NOT NULL UNIQUE,
    "jid" VARCHAR(255) DEFAULT '',
    "qrCode" TEXT DEFAULT '',
    "connected" INTEGER DEFAULT 0,
    "status" VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    "proxy" JSONB NOT NULL DEFAULT '{}',
    "settings" JSONB NOT NULL DEFAULT '{}',
    "metadata" JSONB,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idxWzSessionsName" ON "wzSessions" ("name");
CREATE INDEX IF NOT EXISTS "idxWzSessionsToken" ON "wzSessions" ("token");
CREATE INDEX IF NOT EXISTS "idxWzSessionsStatus" ON "wzSessions" ("status");
CREATE INDEX IF NOT EXISTS "idxWzSessionsConnected" ON "wzSessions" ("connected");

CREATE UNIQUE INDEX IF NOT EXISTS "idxWzSessionsJidUnique"
    ON "wzSessions" ("jid")
    WHERE "jid" IS NOT NULL AND "jid" != '';

DROP TRIGGER IF EXISTS "updateWzSessionsUpdatedAt" ON "wzSessions";
CREATE TRIGGER "updateWzSessionsUpdatedAt"
    BEFORE UPDATE ON "wzSessions"
    FOR EACH ROW
    EXECUTE FUNCTION "updateUpdatedAtColumn"();

COMMENT ON TABLE "wzSessions" IS 'WhatsApp sessions managed by wzap';
COMMENT ON COLUMN "wzSessions"."name" IS 'Unique URL-safe session identifier (^[a-zA-Z0-9_-]+$)';
COMMENT ON COLUMN "wzSessions"."token" IS 'Token for session-scoped authentication';
COMMENT ON COLUMN "wzSessions"."jid" IS 'WhatsApp device JID from whatsmeow (set after pairing)';

-- =====================================================
-- Webhooks Table
-- =====================================================
CREATE TABLE IF NOT EXISTS "wzWebhooks" (
    "id" VARCHAR(100) PRIMARY KEY,
    "sessionId" VARCHAR(100) NOT NULL REFERENCES "wzSessions"("id") ON DELETE CASCADE,
    "url" VARCHAR(2048) NOT NULL,
    "secret" VARCHAR(255),
    "events" JSONB NOT NULL DEFAULT '[]',
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "natsEnabled" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idxWzWebhooksSessionId" ON "wzWebhooks" ("sessionId");
CREATE INDEX IF NOT EXISTS "idxWzWebhooksEnabled" ON "wzWebhooks" ("enabled");

DROP TRIGGER IF EXISTS "updateWzWebhooksUpdatedAt" ON "wzWebhooks";
CREATE TRIGGER "updateWzWebhooksUpdatedAt"
    BEFORE UPDATE ON "wzWebhooks"
    FOR EACH ROW
    EXECUTE FUNCTION "updateUpdatedAtColumn"();

COMMENT ON TABLE "wzWebhooks" IS 'Webhook configurations per session';
