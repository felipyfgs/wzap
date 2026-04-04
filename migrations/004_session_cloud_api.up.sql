ALTER TABLE wz_sessions
    ADD COLUMN IF NOT EXISTS engine VARCHAR(20) DEFAULT 'whatsmeow',
    ADD COLUMN IF NOT EXISTS phone_number_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS access_token TEXT,
    ADD COLUMN IF NOT EXISTS business_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS app_secret TEXT,
    ADD COLUMN IF NOT EXISTS webhook_verify_token VARCHAR(255);
