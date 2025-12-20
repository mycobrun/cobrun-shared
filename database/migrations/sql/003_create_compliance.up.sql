-- Migration: 003_create_compliance
-- Description: Create compliance tables (documents, background checks, onboarding)

-- Documents table
CREATE TABLE documents (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    type                    NVARCHAR(30) NOT NULL CHECK (type IN ('driver_license', 'vehicle_registration', 'insurance', 'profile_photo', 'vehicle_photo', 'background_check', 'id_card', 'passport', 'proof_of_address', 'bank_statement')),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewing', 'approved', 'rejected', 'expired')),
    file_url                NVARCHAR(500) NOT NULL,
    file_name               NVARCHAR(255) NOT NULL,
    file_size               BIGINT NOT NULL,
    mime_type               NVARCHAR(100) NOT NULL,
    document_number         NVARCHAR(100),
    expiry_date             DATE,
    issue_date              DATE,
    issuing_country         NVARCHAR(2),
    issuing_state           NVARCHAR(50),
    extracted_data          NVARCHAR(MAX), -- JSON
    verified_by             NVARCHAR(36),
    verified_at             DATETIME2,
    rejection_reason        NVARCHAR(500),
    uploaded_at             DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_documents_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_documents_user_id ON documents(user_id);
CREATE INDEX idx_documents_type ON documents(type);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_expiry_date ON documents(expiry_date);

-- Background Checks
CREATE TABLE background_checks (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    provider_id             NVARCHAR(100),
    type                    NVARCHAR(30) NOT NULL CHECK (type IN ('criminal', 'driving', 'employment')),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'clear', 'failed', 'requires_review')),
    criminal_records        BIT,
    driving_violations      INT,
    mvr_status              NVARCHAR(30),
    identity_verified       BIT,
    address_verified        BIT,
    ssn_verified            BIT,
    overall_risk            NVARCHAR(20) CHECK (overall_risk IN ('low', 'medium', 'high')),
    findings                NVARCHAR(MAX), -- JSON array
    reviewed_by             NVARCHAR(36),
    reviewed_at             DATETIME2,
    review_notes            NVARCHAR(MAX),
    valid_until             DATE,
    requested_at            DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    completed_at            DATETIME2,
    
    CONSTRAINT fk_background_checks_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_background_checks_user_id ON background_checks(user_id);
CREATE INDEX idx_background_checks_status ON background_checks(status);

-- Driver Onboarding
CREATE TABLE driver_onboarding (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL UNIQUE,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'in_progress', 'pending_review', 'approved', 'rejected', 'expired')),
    current_step            NVARCHAR(50),
    completed_steps         NVARCHAR(MAX), -- JSON array
    submitted_docs          NVARCHAR(MAX), -- JSON object
    background_check_id     NVARCHAR(36),
    vehicle_id              NVARCHAR(36),
    approved_at             DATETIME2,
    approved_by             NVARCHAR(36),
    rejected_at             DATETIME2,
    rejected_by             NVARCHAR(36),
    rejection_reason        NVARCHAR(500),
    started_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_driver_onboarding_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_driver_onboarding_bg_check FOREIGN KEY (background_check_id) REFERENCES background_checks(id),
    CONSTRAINT fk_driver_onboarding_vehicle FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
);

CREATE INDEX idx_driver_onboarding_user_id ON driver_onboarding(user_id);
CREATE INDEX idx_driver_onboarding_status ON driver_onboarding(status);


