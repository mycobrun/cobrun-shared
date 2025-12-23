-- Rollback: 003_create_compliance

DROP INDEX IF EXISTS idx_driver_onboarding_status;
DROP INDEX IF EXISTS idx_driver_onboarding_user_id;
DROP TABLE IF EXISTS driver_onboarding;

DROP INDEX IF EXISTS idx_background_checks_status;
DROP INDEX IF EXISTS idx_background_checks_user_id;
DROP TABLE IF EXISTS background_checks;

DROP INDEX IF EXISTS idx_documents_expiry_date;
DROP INDEX IF EXISTS idx_documents_status;
DROP INDEX IF EXISTS idx_documents_type;
DROP INDEX IF EXISTS idx_documents_user_id;
DROP TABLE IF EXISTS documents;





