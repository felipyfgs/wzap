ALTER TABLE wz_messages
    ADD COLUMN IF NOT EXISTS is_forwarded     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS forwarding_score INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_wz_messages_forwarded
    ON wz_messages(session_id, is_forwarded)
    WHERE is_forwarded = TRUE;
