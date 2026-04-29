DROP INDEX IF EXISTS idx_wz_messages_forwarded;

ALTER TABLE wz_messages
    DROP COLUMN IF EXISTS forwarding_score,
    DROP COLUMN IF EXISTS is_forwarded;
