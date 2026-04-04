-- Remove ignore_jids and conversation_pending columns from wz_chatwoot
ALTER TABLE wz_chatwoot DROP COLUMN ignore_jids;
ALTER TABLE wz_chatwoot DROP COLUMN conversation_pending;
