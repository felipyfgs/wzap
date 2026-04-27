ALTER TABLE wz_elodesk
    ALTER COLUMN inbox_identifier SET NOT NULL,
    ALTER COLUMN api_token SET NOT NULL,
    DROP COLUMN IF EXISTS user_access_token,
    DROP COLUMN IF EXISTS account_id,
    DROP COLUMN IF EXISTS channel_id;
