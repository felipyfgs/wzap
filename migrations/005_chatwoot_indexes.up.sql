CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_conversation
    ON wz_messages (session_id, cw_conversation_id)
    WHERE cw_conversation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_message
    ON wz_messages (session_id, cw_message_id)
    WHERE cw_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wz_messages_cw_source
    ON wz_messages (session_id, cw_source_id)
    WHERE cw_source_id IS NOT NULL;
