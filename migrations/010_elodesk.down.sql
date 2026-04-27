DROP INDEX IF EXISTS idx_wz_messages_elodesk_src;

ALTER TABLE wz_messages
    DROP COLUMN IF EXISTS elodesk_src_id,
    DROP COLUMN IF EXISTS elodesk_conv_id,
    DROP COLUMN IF EXISTS elodesk_message_id;

DROP TRIGGER IF EXISTS trg_wz_elodesk_updated_at ON wz_elodesk;
DROP TABLE IF EXISTS wz_elodesk;
