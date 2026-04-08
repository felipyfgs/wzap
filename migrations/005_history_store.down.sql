DROP INDEX IF EXISTS idx_wz_messages_history_order;
DROP INDEX IF EXISTS idx_wz_messages_session_source;

ALTER TABLE wz_messages
    DROP COLUMN IF EXISTS imported_to_chatwoot_at,
    DROP COLUMN IF EXISTS history_message_order,
    DROP COLUMN IF EXISTS history_chunk_order,
    DROP COLUMN IF EXISTS source_sync_type,
    DROP COLUMN IF EXISTS source;

DROP TRIGGER IF EXISTS trg_wz_chats_updated_at ON wz_chats;
DROP INDEX IF EXISTS idx_wz_chats_session_lid_jid;
DROP INDEX IF EXISTS idx_wz_chats_session_pn_jid;
DROP INDEX IF EXISTS idx_wz_chats_session_source;
DROP INDEX IF EXISTS idx_wz_chats_session_last_message;
DROP TABLE IF EXISTS wz_chats;
