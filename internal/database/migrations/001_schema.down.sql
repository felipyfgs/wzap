DROP TRIGGER IF EXISTS "updateWzWebhooksUpdatedAt" ON "wzWebhooks";
DROP TABLE IF EXISTS "wzWebhooks";
DROP TRIGGER IF EXISTS "updateWzSessionsUpdatedAt" ON "wzSessions";
DROP TABLE IF EXISTS "wzSessions";
DROP TRIGGER IF EXISTS "updateWzUsersUpdatedAt" ON "wzUsers";
DROP TABLE IF EXISTS "wzUsers";
DROP FUNCTION IF EXISTS "updateUpdatedAtColumn"();
