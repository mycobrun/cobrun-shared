-- Rollback: 005_create_promotions

DROP INDEX IF EXISTS idx_wallet_credits_expires_at;
DROP INDEX IF EXISTS idx_wallet_credits_status;
DROP INDEX IF EXISTS idx_wallet_credits_user_id;
DROP TABLE IF EXISTS wallet_credits;

DROP INDEX IF EXISTS idx_referrals_status;
DROP INDEX IF EXISTS idx_referrals_referred_id;
DROP INDEX IF EXISTS idx_referrals_referrer_id;
DROP TABLE IF EXISTS referrals;

DROP INDEX IF EXISTS idx_user_promos_expires_at;
DROP INDEX IF EXISTS idx_user_promos_user_id;
DROP TABLE IF EXISTS user_promos;

DROP INDEX IF EXISTS idx_promo_usage_used_at;
DROP INDEX IF EXISTS idx_promo_usage_user_id;
DROP INDEX IF EXISTS idx_promo_usage_promo_id;
DROP TABLE IF EXISTS promo_usage;

DROP INDEX IF EXISTS idx_promotions_expires_at;
DROP INDEX IF EXISTS idx_promotions_status;
DROP INDEX IF EXISTS idx_promotions_code;
DROP TABLE IF EXISTS promotions;


