-- Add ignore_jids and conversation_pending columns to wz_chatwoot
ALTER TABLE wz_chatwoot ADD COLUMN ignore_jids TEXT[] DEFAULT '{}';
ALTER TABLE wz_chatwoot ADD COLUMN conversation_pending BOOLEAN DEFAULT false;

-- Migrate existing ignore_groups to ignore_jids
UPDATE wz_chatwoot SET ignore_jids = ARRAY['@g.us'] WHERE ignore_groups = true;
