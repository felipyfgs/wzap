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
-- Users Table (wuzapi-inspired)
-- =====================================================
CREATE TABLE IF NOT EXISTS "wzUsers" (
    "id" VARCHAR(100) PRIMARY KEY,
    "name" VARCHAR(255) NOT NULL DEFAULT '',
    "token" VARCHAR(255) NOT NULL UNIQUE,
    "webhook" VARCHAR(2048) NOT NULL DEFAULT '',
    "events" TEXT NOT NULL DEFAULT '',
    "expiration" INTEGER DEFAULT 0,
    "proxyUrl" TEXT DEFAULT '',
    "history" INTEGER DEFAULT 0,
    "hmacKey" BYTEA,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idxWzUsersToken" ON "wzUsers" ("token");

DROP TRIGGER IF EXISTS "updateWzUsersUpdatedAt" ON "wzUsers";
CREATE TRIGGER "updateWzUsersUpdatedAt"
    BEFORE UPDATE ON "wzUsers"
    FOR EACH ROW
    EXECUTE FUNCTION "updateUpdatedAtColumn"();

COMMENT ON TABLE "wzUsers" IS 'User instances that own WhatsApp sessions';

-- =====================================================
-- Sessions Table (WhatsApp connection state)
-- =====================================================
CREATE TABLE IF NOT EXISTS "wzSessions" (
    "id" VARCHAR(100) PRIMARY KEY,
    "userId" VARCHAR(100) NOT NULL REFERENCES "wzUsers"("id") ON DELETE CASCADE,
    "jid" VARCHAR(255) DEFAULT '',
    "qrCode" TEXT DEFAULT '',
    "connected" INTEGER DEFAULT 0,
    "status" VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    "metadata" JSONB,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idxWzSessionsUserId" ON "wzSessions" ("userId");
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
COMMENT ON COLUMN "wzSessions"."jid" IS 'WhatsApp device JID from whatsmeow (set after pairing)';

-- =====================================================
-- Webhooks Table
-- =====================================================
CREATE TABLE IF NOT EXISTS "wzWebhooks" (
    "id" VARCHAR(100) PRIMARY KEY,
    "userId" VARCHAR(100) NOT NULL REFERENCES "wzUsers"("id") ON DELETE CASCADE,
    "url" VARCHAR(2048) NOT NULL,
    "secret" VARCHAR(255),
    "events" JSONB NOT NULL DEFAULT '[]',
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idxWzWebhooksUserId" ON "wzWebhooks" ("userId");
CREATE INDEX IF NOT EXISTS "idxWzWebhooksEnabled" ON "wzWebhooks" ("enabled");

DROP TRIGGER IF EXISTS "updateWzWebhooksUpdatedAt" ON "wzWebhooks";
CREATE TRIGGER "updateWzWebhooksUpdatedAt"
    BEFORE UPDATE ON "wzWebhooks"
    FOR EACH ROW
    EXECUTE FUNCTION "updateUpdatedAtColumn"();

COMMENT ON TABLE "wzWebhooks" IS 'Webhook configurations for users';
