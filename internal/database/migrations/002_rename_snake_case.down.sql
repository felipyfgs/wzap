-- Rollback: restore camelCase identifiers (for reference only)
ALTER TABLE IF EXISTS wz_webhooks RENAME COLUMN session_id   TO "sessionId";
ALTER TABLE IF EXISTS wz_webhooks RENAME COLUMN nats_enabled TO "natsEnabled";
ALTER TABLE IF EXISTS wz_webhooks RENAME COLUMN created_at   TO "createdAt";
ALTER TABLE IF EXISTS wz_webhooks RENAME COLUMN updated_at   TO "updatedAt";
ALTER TABLE IF EXISTS wz_webhooks RENAME TO "wzWebhooks";

ALTER TABLE IF EXISTS wz_sessions RENAME COLUMN api_key    TO "apiKey";
ALTER TABLE IF EXISTS wz_sessions RENAME COLUMN qr_code    TO "qrCode";
ALTER TABLE IF EXISTS wz_sessions RENAME COLUMN created_at TO "createdAt";
ALTER TABLE IF EXISTS wz_sessions RENAME COLUMN updated_at TO "updatedAt";
ALTER TABLE IF EXISTS wz_sessions RENAME TO "wzSessions";
