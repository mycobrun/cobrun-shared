-- Rollback: 009_create_notifications

DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP TABLE IF EXISTS notifications;

DROP TABLE IF EXISTS notification_preferences;

DROP INDEX IF EXISTS idx_device_tokens_is_active;
DROP INDEX IF EXISTS idx_device_tokens_user_id;
DROP TABLE IF EXISTS device_tokens;





