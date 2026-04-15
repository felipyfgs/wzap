-- Reverse the migration: move status data back into wz_messages

INSERT INTO wz_messages (id, session_id, chat_jid, sender_jid, from_me, msg_type, body, media_type, media_url, raw, timestamp, created_at)
SELECT
    s.id,
    s.session_id,
    'status@broadcast',
    s.sender_jid,
    s.from_me,
    s.status_type,
    s.body,
    s.media_type,
    s.media_url,
    s.raw,
    s.timestamp,
    s.created_at
FROM wz_statuses s
ON CONFLICT (id, session_id) DO NOTHING;

DROP TABLE IF EXISTS wz_statuses;