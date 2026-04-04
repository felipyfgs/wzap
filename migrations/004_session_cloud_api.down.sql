ALTER TABLE wz_sessions
    DROP COLUMN IF EXISTS engine,
    DROP COLUMN IF EXISTS phone_number_id,
    DROP COLUMN IF EXISTS access_token,
    DROP COLUMN IF EXISTS business_account_id,
    DROP COLUMN IF EXISTS app_secret,
    DROP COLUMN IF EXISTS webhook_verify_token;
