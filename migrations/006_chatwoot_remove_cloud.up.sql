-- =====================================================
-- Chatwoot: remover suporte ao modo Cloud. Após esta migração, todas as
-- sessões ficam no modo API e as colunas que só existiam para o fluxo Cloud
-- (database_uri/redis_url) deixam de existir. O CHECK garante que nenhuma
-- linha volte a referenciar o modo Cloud no futuro.
-- =====================================================

UPDATE wz_chatwoot SET inbox_type = 'api' WHERE inbox_type <> 'api';

ALTER TABLE wz_chatwoot DROP COLUMN IF EXISTS database_uri;
ALTER TABLE wz_chatwoot DROP COLUMN IF EXISTS redis_url;

ALTER TABLE wz_chatwoot DROP CONSTRAINT IF EXISTS chatwoot_inbox_type_api;
ALTER TABLE wz_chatwoot
    ADD CONSTRAINT chatwoot_inbox_type_api CHECK (inbox_type = 'api');
