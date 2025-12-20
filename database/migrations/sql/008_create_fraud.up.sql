-- Migration: 008_create_fraud
-- Description: Create fraud detection tables

-- Risk Assessments
CREATE TABLE risk_assessments (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    entity_id               NVARCHAR(36) NOT NULL,
    entity_type             NVARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'driver', 'trip', 'payment')),
    risk_score              DECIMAL(5,2) NOT NULL,
    risk_level              NVARCHAR(10) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    signals                 NVARCHAR(MAX),
    factor_scores           NVARCHAR(MAX),
    is_blocked              BIT DEFAULT 0,
    block_reason            NVARCHAR(500),
    requires_review         BIT DEFAULT 0,
    reviewed_by             NVARCHAR(36),
    reviewed_at             DATETIME2,
    review_notes            NVARCHAR(MAX),
    expires_at              DATETIME2 NOT NULL,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_risk_assessments_entity ON risk_assessments(entity_id, entity_type);
CREATE INDEX idx_risk_assessments_risk_level ON risk_assessments(risk_level);
CREATE INDEX idx_risk_assessments_requires_review ON risk_assessments(requires_review);

-- Fraud Alerts
CREATE TABLE fraud_alerts (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    entity_id               NVARCHAR(36) NOT NULL,
    entity_type             NVARCHAR(20) NOT NULL,
    alert_type              NVARCHAR(30) NOT NULL CHECK (alert_type IN ('multiple_accounts', 'payment_fraud', 'promo_abuse', 'fake_rides', 'driver_collusion', 'gps_spoofing', 'velocity_anomaly', 'device_anomaly', 'account_takeover', 'suspicious_activity')),
    severity                NVARCHAR(10) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    title                   NVARCHAR(200) NOT NULL,
    description             NVARCHAR(MAX) NOT NULL,
    evidence                NVARCHAR(MAX),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'false_positive', 'escalated')),
    assigned_to             NVARCHAR(36),
    resolved_by             NVARCHAR(36),
    resolution              NVARCHAR(MAX),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    resolved_at             DATETIME2
);

CREATE INDEX idx_fraud_alerts_entity ON fraud_alerts(entity_id, entity_type);
CREATE INDEX idx_fraud_alerts_status ON fraud_alerts(status);
CREATE INDEX idx_fraud_alerts_severity ON fraud_alerts(severity);
CREATE INDEX idx_fraud_alerts_alert_type ON fraud_alerts(alert_type);

-- Fraud Rules
CREATE TABLE fraud_rules (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    name                    NVARCHAR(100) NOT NULL,
    description             NVARCHAR(500),
    rule_type               NVARCHAR(20) NOT NULL CHECK (rule_type IN ('threshold', 'pattern', 'velocity')),
    condition               NVARCHAR(MAX) NOT NULL,
    action                  NVARCHAR(20) NOT NULL CHECK (action IN ('block', 'flag', 'alert')),
    severity                NVARCHAR(10) NOT NULL,
    is_enabled              BIT DEFAULT 1,
    priority                INT DEFAULT 0,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_fraud_rules_is_enabled ON fraud_rules(is_enabled);

-- Device Fingerprints
CREATE TABLE device_fingerprints (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    device_id               NVARCHAR(100) NOT NULL,
    platform                NVARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    os_version              NVARCHAR(50),
    app_version             NVARCHAR(20),
    device_model            NVARCHAR(100),
    is_emulator             BIT DEFAULT 0,
    is_rooted               BIT DEFAULT 0,
    is_vpn                  BIT DEFAULT 0,
    ip_address              NVARCHAR(45),
    country                 NVARCHAR(2),
    city                    NVARCHAR(100),
    timezone                NVARCHAR(50),
    language                NVARCHAR(10),
    screen_resolution       NVARCHAR(20),
    trust_score             DECIMAL(5,2) DEFAULT 100,
    first_seen_at           DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    last_seen_at            DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    login_count             INT DEFAULT 1,
    
    CONSTRAINT fk_device_fingerprints_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_device_fingerprints_user_id ON device_fingerprints(user_id);
CREATE INDEX idx_device_fingerprints_device_id ON device_fingerprints(device_id);
CREATE INDEX idx_device_fingerprints_trust_score ON device_fingerprints(trust_score);


