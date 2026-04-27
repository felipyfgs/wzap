-- =====================================================
-- Elodesk Integration Config (espelha wz_chatwoot; ver migration 005)
-- =====================================================
CREATE TABLE IF NOT EXISTS wz_elodesk (
    session_id           VARCHAR(100)  PRIMARY KEY REFERENCES wz_sessions(id) ON DELETE CASCADE,

    -- Connection
    url                  VARCHAR(2048) NOT NULL,
    inbox_identifier     VARCHAR(255)  NOT NULL,
    api_token            VARCHAR(255)  NOT NULL,
    hmac_token           VARCHAR(255)  NOT NULL DEFAULT '',
    webhook_secret       VARCHAR(255)  NOT NULL DEFAULT '',
    enabled              BOOLEAN       NOT NULL DEFAULT true,

    -- Message behavior
    sign_msg             BOOLEAN       NOT NULL DEFAULT false,
    sign_delimiter       VARCHAR(50)   NOT NULL DEFAULT '\n',
    reopen_conv          BOOLEAN       NOT NULL DEFAULT true,
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

    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wz_elodesk_enabled
    ON wz_elodesk (enabled);

DROP TRIGGER IF EXISTS trg_wz_elodesk_updated_at ON wz_elodesk;
CREATE TRIGGER trg_wz_elodesk_updated_at
    BEFORE UPDATE ON wz_elodesk
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- =====================================================
-- Refs de elodesk em wz_messages (paralelo a cw_*)
-- =====================================================
ALTER TABLE wz_messages
    ADD COLUMN IF NOT EXISTS elodesk_message_id BIGINT,
    ADD COLUMN IF NOT EXISTS elodesk_conv_id    BIGINT,
    ADD COLUMN IF NOT EXISTS elodesk_src_id     TEXT;

CREATE INDEX IF NOT EXISTS idx_wz_messages_elodesk_src
    ON wz_messages (session_id, elodesk_src_id)
    WHERE elodesk_src_id IS NOT NULL;
