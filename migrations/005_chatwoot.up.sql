-- =====================================================
-- Chatwoot Integration Config
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_chatwoot (
    session_id           VARCHAR(100)  PRIMARY KEY REFERENCES wz_sessions(id) ON DELETE CASCADE,

    -- Connection
    url                  VARCHAR(2048) NOT NULL,
    account_id           INTEGER       NOT NULL,
    token                VARCHAR(255)  NOT NULL,
    inbox_id             INTEGER       NOT NULL,
    inbox_name           VARCHAR(255)  NOT NULL DEFAULT 'wzap',
    inbox_type           VARCHAR(20)   NOT NULL DEFAULT 'api',
    enabled              BOOLEAN       NOT NULL DEFAULT true,

    -- Webhook
    webhook_token        VARCHAR(255)  NOT NULL DEFAULT '',

    -- Message behavior
    sign_msg             BOOLEAN       NOT NULL DEFAULT false,
    sign_delimiter       VARCHAR(50)   NOT NULL DEFAULT '\n',
    reopen_conversation  BOOLEAN       NOT NULL DEFAULT true,
    conversation_pending BOOLEAN       NOT NULL DEFAULT false,
    message_read         BOOLEAN       NOT NULL DEFAULT false,

    -- Contact handling
    merge_br_contacts    BOOLEAN       NOT NULL DEFAULT true,

    -- Filtering
    ignore_groups        BOOLEAN       NOT NULL DEFAULT false,
    ignore_jids          TEXT[]        NOT NULL DEFAULT '{}',

    -- History import
    import_on_connect    BOOLEAN       NOT NULL DEFAULT false,
    import_period        VARCHAR(10)   NOT NULL DEFAULT '7d',

    -- Timeouts (seconds)
    timeout_text_seconds  INTEGER      NOT NULL DEFAULT 10,
    timeout_media_seconds INTEGER      NOT NULL DEFAULT 60,
    timeout_large_seconds INTEGER      NOT NULL DEFAULT 300,

    -- External
    redis_url            VARCHAR(255)  NOT NULL DEFAULT '',
    database_uri         TEXT          NOT NULL DEFAULT '',

    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_chatwoot_enabled
    ON wz_chatwoot (enabled);

DROP TRIGGER IF EXISTS trg_wz_chatwoot_updated_at ON wz_chatwoot;
CREATE TRIGGER trg_wz_chatwoot_updated_at
    BEFORE UPDATE ON wz_chatwoot
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
