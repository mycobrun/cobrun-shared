-- Rollback: 004_create_payments

DROP INDEX IF EXISTS idx_driver_payouts_created_at;
DROP INDEX IF EXISTS idx_driver_payouts_status;
DROP INDEX IF EXISTS idx_driver_payouts_driver_id;
DROP TABLE IF EXISTS driver_payouts;

DROP INDEX IF EXISTS idx_wallet_transactions_created_at;
DROP INDEX IF EXISTS idx_wallet_transactions_user_id;
DROP INDEX IF EXISTS idx_wallet_transactions_wallet_id;
DROP TABLE IF EXISTS wallet_transactions;

DROP INDEX IF EXISTS idx_wallets_user_id;
DROP TABLE IF EXISTS wallets;

DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_driver_id;
DROP INDEX IF EXISTS idx_payments_rider_id;
DROP INDEX IF EXISTS idx_payments_trip_id;
DROP TABLE IF EXISTS payments;

DROP INDEX IF EXISTS idx_payment_methods_is_default;
DROP INDEX IF EXISTS idx_payment_methods_user_id;
DROP TABLE IF EXISTS payment_methods;





