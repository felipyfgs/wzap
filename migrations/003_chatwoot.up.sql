CREATE TABLE IF NOT EXISTS wz_chatwoot (
    session_id          VARCHAR(100) PRIMARY KEY REFERENCES wz_sessions(id) ON DELETE CASCADE,
    url                 VARCHAR(2048) NOT NULL,
    account_id          INTEGER NOT NULL,
    token               VARCHAR(255) NOT NULL,
    inbox_id            INTEGER NOT NULL,
    inbox_name          VARCHAR(255) NOT NULL DEFAULT 'wzap',
    sign_msg            BOOLEAN NOT NULL DEFAULT false,
    sign_delimiter      VARCHAR(50) NOT NULL DEFAULT '\n',
    reopen_conversation BOOLEAN NOT NULL DEFAULT true,
    merge_br_contacts   BOOLEAN NOT NULL DEFAULT true,
    ignore_groups       BOOLEAN NOT NULL DEFAULT false,
    enabled             BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_chatwoot_enabled
    ON wz_chatwoot (enabled);

DROP TRIGGER IF EXISTS trg_wz_chatwoot_updated_at ON wz_chatwoot;
CREATE TRIGGER trg_wz_chatwoot_updated_at
    BEFORE UPDATE ON wz_chatwoot
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE wz_chatwoot IS 'Chatwoot integration configuration per session';
COMMENT ON COLUMN wz_chatwoot.session_id IS 'FK to wz_sessions.id';
COMMENT ON COLUMN wz_chatwoot.url IS 'Chatwoot instance base URL';
COMMENT ON COLUMN wz_chatwoot.account_id IS 'Chatwoot account ID';
COMMENT ON COLUMN wz_chatwoot.token IS 'Chatwoot agent/inbox access token';
COMMENT ON COLUMN wz_chatwoot.inbox_id IS 'Chatwoot inbox ID for this integration';
COMMENT ON COLUMN wz_chatwoot.inbox_name IS 'Name for auto-created inbox';
COMMENT ON COLUMN wz_chatwoot.sign_msg IS 'Whether to prefix messages with sender info';
COMMENT ON COLUMN wz_chatwoot.sign_delimiter IS 'Delimiter for sign_msg prefix';
COMMENT ON COLUMN wz_chatwoot.reopen_conversation IS 'Whether to reopen resolved conversations';
COMMENT ON COLUMN wz_chatwoot.merge_br_contacts IS 'Whether to deduplicate +55 9th-digit variants';
COMMENT ON COLUMN wz_chatwoot.ignore_groups IS 'Whether to ignore group messages';
