-- Rollback: 001_create_users

DROP INDEX IF EXISTS idx_driver_profiles_rating;
DROP INDEX IF EXISTS idx_driver_profiles_is_online;
DROP INDEX IF EXISTS idx_driver_profiles_status;
DROP INDEX IF EXISTS idx_driver_profiles_user_id;
DROP TABLE IF EXISTS driver_profiles;

DROP INDEX IF EXISTS idx_users_referral_code;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_user_type;
DROP INDEX IF EXISTS idx_users_phone;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;





