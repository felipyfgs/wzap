DROP TRIGGER IF EXISTS update_wz_webhooks_updated_at ON wz_webhooks;
DROP TABLE IF EXISTS wz_webhooks;
DROP TRIGGER IF EXISTS update_wz_sessions_updated_at ON wz_sessions;
DROP TABLE IF EXISTS wz_sessions;
DROP FUNCTION IF EXISTS update_updated_at_column();
