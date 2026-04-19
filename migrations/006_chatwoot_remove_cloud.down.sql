-- =====================================================
-- Rollback estrutural para 006_chatwoot_remove_cloud: restaura as colunas
-- que ancoravam o modo Cloud como nullable. Os valores originais NÃO são
-- restaurados — rollback real exige `git revert` + nova configuração de
-- sessão.
-- =====================================================

ALTER TABLE wz_chatwoot DROP CONSTRAINT IF EXISTS chatwoot_inbox_type_api;

ALTER TABLE wz_chatwoot ADD COLUMN IF NOT EXISTS database_uri TEXT NOT NULL DEFAULT '';
ALTER TABLE wz_chatwoot ADD COLUMN IF NOT EXISTS redis_url VARCHAR(255) NOT NULL DEFAULT '';
