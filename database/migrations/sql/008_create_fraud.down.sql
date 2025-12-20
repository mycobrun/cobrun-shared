-- Rollback: 008_create_fraud

DROP INDEX IF EXISTS idx_device_fingerprints_trust_score;
DROP INDEX IF EXISTS idx_device_fingerprints_device_id;
DROP INDEX IF EXISTS idx_device_fingerprints_user_id;
DROP TABLE IF EXISTS device_fingerprints;

DROP INDEX IF EXISTS idx_fraud_rules_is_enabled;
DROP TABLE IF EXISTS fraud_rules;

DROP INDEX IF EXISTS idx_fraud_alerts_alert_type;
DROP INDEX IF EXISTS idx_fraud_alerts_severity;
DROP INDEX IF EXISTS idx_fraud_alerts_status;
DROP INDEX IF EXISTS idx_fraud_alerts_entity;
DROP TABLE IF EXISTS fraud_alerts;

DROP INDEX IF EXISTS idx_risk_assessments_requires_review;
DROP INDEX IF EXISTS idx_risk_assessments_risk_level;
DROP INDEX IF EXISTS idx_risk_assessments_entity;
DROP TABLE IF EXISTS risk_assessments;

