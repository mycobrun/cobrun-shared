-- Migration: 012_create_wa_approval_workflow
-- Description: Create Washington State compliant driver approval workflow tables
-- Implements per-step approval, annual rechecks, audit logs, and jurisdictional configuration

-- ============================================================================
-- STEP DEFINITIONS (configurable by jurisdiction)
-- ============================================================================
CREATE TABLE onboarding_step_definitions (
    id                      NVARCHAR(50) PRIMARY KEY,           -- e.g., 'IDENTITY', 'LICENSE', 'MVR'
    name                    NVARCHAR(100) NOT NULL,
    description             NVARCHAR(500),
    step_order              INT NOT NULL,
    is_required             BIT NOT NULL DEFAULT 1,
    is_annual_recheck       BIT NOT NULL DEFAULT 0,            -- True for background/MVR/insurance
    requires_document       BIT NOT NULL DEFAULT 0,
    requires_vendor_check   BIT NOT NULL DEFAULT 0,            -- True for background check, MVR
    vendor_type             NVARCHAR(50),                      -- 'background_check', 'mvr', 'document_verification'
    who_can_approve         NVARCHAR(30) NOT NULL DEFAULT 'ADMIN', -- SYSTEM, ADMIN, SYSTEM_AND_ADMIN
    auto_approve_rules      NVARCHAR(MAX),                     -- JSON: rules for auto-approval
    jurisdiction_code       NVARCHAR(20) NOT NULL DEFAULT 'WA', -- State/region code
    is_active               BIT NOT NULL DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_step_definitions_order ON onboarding_step_definitions(jurisdiction_code, step_order);

-- Insert WA-required steps
INSERT INTO onboarding_step_definitions (id, name, description, step_order, is_required, is_annual_recheck, requires_document, requires_vendor_check, vendor_type, who_can_approve) VALUES
('ACCOUNT_IDENTITY', 'Account & Identity', 'Email/phone verified, profile photo, govt ID + selfie verification', 1, 1, 0, 1, 0, NULL, 'SYSTEM_AND_ADMIN'),
('DRIVER_APPLICATION', 'Driver Application', 'Name, address, phone, age, license #, vehicle, insurance info', 2, 1, 0, 0, 0, NULL, 'SYSTEM'),
('LICENSE_VALIDATION', 'License Validation', 'Upload license front/back, verify not expired, age >= 20', 3, 1, 0, 1, 0, NULL, 'SYSTEM_AND_ADMIN'),
('DRIVING_HISTORY', 'Driving History (MVR)', 'Pull/verify driving record, check violations', 4, 1, 1, 0, 1, 'mvr', 'SYSTEM_AND_ADMIN'),
('CRIMINAL_BACKGROUND', 'Criminal Background Check', 'Nationwide DB + sex offender search, 7-year lookback', 5, 1, 1, 0, 1, 'background_check', 'SYSTEM_AND_ADMIN'),
('VEHICLE_ELIGIBILITY', 'Vehicle Eligibility', 'VIN, plate, year <= 15 years old, registration', 6, 1, 0, 1, 0, NULL, 'ADMIN'),
('INSURANCE_VERIFICATION', 'Insurance Verification', 'TNC coverage proof, issue written proof of coverage', 7, 1, 1, 1, 0, NULL, 'ADMIN'),
('POLICY_ACKNOWLEDGEMENTS', 'Policy Acknowledgements', 'Zero tolerance, prohibited activities, nondiscrimination', 8, 1, 0, 0, 0, NULL, 'SYSTEM'),
('JURISDICTIONAL_COMPLIANCE', 'Jurisdictional Compliance', 'Local permits, decals, inspections if required', 9, 0, 0, 1, 0, NULL, 'ADMIN'),
('FINAL_APPROVAL', 'Final Admin Approval', 'Admin final review before activation', 10, 1, 0, 0, 0, NULL, 'ADMIN');

-- ============================================================================
-- DRIVER APPROVAL CASES
-- ============================================================================
CREATE TABLE driver_approval_cases (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    case_type               NVARCHAR(30) NOT NULL DEFAULT 'INITIAL' CHECK (case_type IN ('INITIAL', 'ANNUAL_RECHECK', 'REINSTATEMENT', 'VEHICLE_CHANGE')),
    status                  NVARCHAR(30) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'SUBMITTED', 'IN_REVIEW', 'ACTION_REQUIRED', 'APPROVED', 'REJECTED', 'SUSPENDED', 'DEACTIVATED', 'EXPIRED')),
    jurisdiction_code       NVARCHAR(20) NOT NULL DEFAULT 'WA',
    completion_percent      DECIMAL(5,2) DEFAULT 0,
    current_step_id         NVARCHAR(50),
    blocked_reason          NVARCHAR(500),
    
    -- Approval tracking
    submitted_at            DATETIME2,
    reviewed_by             NVARCHAR(36),
    reviewed_at             DATETIME2,
    approved_at             DATETIME2,
    approved_by             NVARCHAR(36),
    rejected_at             DATETIME2,
    rejected_by             NVARCHAR(36),
    rejection_reason        NVARCHAR(500),
    rejection_codes         NVARCHAR(MAX),                     -- JSON array of reason codes
    
    -- Written proof of coverage (WA requirement)
    insurance_proof_issued_at    DATETIME2,
    insurance_proof_document_id  NVARCHAR(36),
    insurance_acknowledged_at    DATETIME2,
    
    -- Expiration tracking
    expires_at              DATETIME2,                         -- For annual rechecks
    next_recheck_date       DATE,
    
    -- Metadata
    version                 INT NOT NULL DEFAULT 1,
    is_active               BIT NOT NULL DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_approval_cases_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_approval_cases_driver_id ON driver_approval_cases(driver_id);
CREATE INDEX idx_approval_cases_status ON driver_approval_cases(status);
CREATE INDEX idx_approval_cases_type ON driver_approval_cases(case_type);
CREATE INDEX idx_approval_cases_expires_at ON driver_approval_cases(expires_at);

-- ============================================================================
-- DRIVER STEP INSTANCES (per-step tracking)
-- ============================================================================
CREATE TABLE driver_step_instances (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    case_id                 NVARCHAR(36) NOT NULL,
    step_definition_id      NVARCHAR(50) NOT NULL,
    driver_id               NVARCHAR(36) NOT NULL,
    status                  NVARCHAR(30) NOT NULL DEFAULT 'NOT_STARTED' CHECK (status IN ('NOT_STARTED', 'PENDING_DRIVER', 'SUBMITTED', 'UNDER_REVIEW', 'APPROVED', 'REJECTED', 'EXPIRED', 'WAIVED')),
    
    -- Vendor check reference (if applicable)
    vendor_check_id         NVARCHAR(100),                     -- External provider reference
    vendor_check_status     NVARCHAR(50),
    vendor_check_completed_at DATETIME2,
    
    -- Auto-evaluation results
    auto_evaluation_result  NVARCHAR(30),                      -- PASS, FAIL, MANUAL_REVIEW
    auto_evaluation_details NVARCHAR(MAX),                     -- JSON: rule results
    auto_evaluated_at       DATETIME2,
    
    -- Admin decision
    decision                NVARCHAR(30) CHECK (decision IN ('APPROVE', 'REJECT', 'REQUEST_INFO', 'WAIVE')),
    decision_by             NVARCHAR(36),
    decision_at             DATETIME2,
    decision_reason_codes   NVARCHAR(MAX),                     -- JSON array of reason codes
    decision_notes          NVARCHAR(1000),
    
    -- If more info requested
    info_requested_at       DATETIME2,
    info_request_message    NVARCHAR(500),
    info_received_at        DATETIME2,
    
    -- Expiration (for annual rechecks)
    valid_from              DATETIME2,
    valid_until             DATETIME2,
    
    -- Evidence links
    evidence_document_ids   NVARCHAR(MAX),                     -- JSON array of document IDs
    
    -- Timestamps
    started_at              DATETIME2,
    submitted_at            DATETIME2,
    completed_at            DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_step_instances_case FOREIGN KEY (case_id) REFERENCES driver_approval_cases(id),
    CONSTRAINT fk_step_instances_step FOREIGN KEY (step_definition_id) REFERENCES onboarding_step_definitions(id),
    CONSTRAINT fk_step_instances_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_step_instances_case_id ON driver_step_instances(case_id);
CREATE INDEX idx_step_instances_driver_id ON driver_step_instances(driver_id);
CREATE INDEX idx_step_instances_status ON driver_step_instances(status);
CREATE INDEX idx_step_instances_valid_until ON driver_step_instances(valid_until);

-- ============================================================================
-- ONBOARDING EVIDENCE (documents + vendor reports)
-- ============================================================================
CREATE TABLE onboarding_evidence (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    step_instance_id        NVARCHAR(36) NOT NULL,
    driver_id               NVARCHAR(36) NOT NULL,
    evidence_type           NVARCHAR(50) NOT NULL CHECK (evidence_type IN ('DOCUMENT', 'VENDOR_REPORT', 'ACKNOWLEDGEMENT', 'SIGNATURE', 'PHOTO', 'SYSTEM_CHECK')),
    
    -- Document reference (if document)
    document_id             NVARCHAR(36),
    
    -- Vendor report (if vendor check)
    vendor_name             NVARCHAR(50),
    vendor_report_id        NVARCHAR(100),
    vendor_report_status    NVARCHAR(50),
    vendor_report_raw       NVARCHAR(MAX),                     -- Encrypted/redacted raw JSON
    
    -- Secure storage
    storage_url             NVARCHAR(500),                     -- Private blob URL
    storage_container       NVARCHAR(100),
    storage_path            NVARCHAR(255),
    content_type            NVARCHAR(100),
    file_size               BIGINT,
    checksum                NVARCHAR(64),                      -- SHA-256
    
    -- Acknowledgement (if ack)
    policy_version          NVARCHAR(50),
    acknowledged_at         DATETIME2,
    signature_data          NVARCHAR(MAX),                     -- Base64 signature image
    ip_address              NVARCHAR(45),
    user_agent              NVARCHAR(500),
    
    -- Metadata
    metadata                NVARCHAR(MAX),                     -- JSON
    expires_at              DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_evidence_step_instance FOREIGN KEY (step_instance_id) REFERENCES driver_step_instances(id),
    CONSTRAINT fk_evidence_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_evidence_document FOREIGN KEY (document_id) REFERENCES documents(id)
);

CREATE INDEX idx_evidence_step_instance ON onboarding_evidence(step_instance_id);
CREATE INDEX idx_evidence_driver_id ON onboarding_evidence(driver_id);
CREATE INDEX idx_evidence_type ON onboarding_evidence(evidence_type);

-- ============================================================================
-- BACKGROUND CHECK RESULTS (normalized from vendor)
-- ============================================================================
CREATE TABLE background_check_results (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    step_instance_id        NVARCHAR(36) NOT NULL,
    driver_id               NVARCHAR(36) NOT NULL,
    
    -- Vendor info
    vendor_name             NVARCHAR(50) NOT NULL,             -- 'checkr', 'sterling', etc.
    vendor_report_id        NVARCHAR(100) NOT NULL,
    vendor_status           NVARCHAR(50) NOT NULL,
    
    -- Check components
    criminal_check_status   NVARCHAR(30),                      -- CLEAR, REVIEW, FAIL
    criminal_check_details  NVARCHAR(MAX),                     -- JSON
    
    sex_offender_check_status NVARCHAR(30),                    -- CLEAR, FOUND
    sex_offender_check_details NVARCHAR(MAX),
    
    national_db_check_status NVARCHAR(30),                     -- CLEAR, REVIEW, FAIL
    national_db_check_details NVARCHAR(MAX),
    
    ssn_trace_status        NVARCHAR(30),
    identity_verified       BIT,
    
    -- 7-year lookback results
    felony_class_a_found    BIT DEFAULT 0,
    felony_class_b_found    BIT DEFAULT 0,
    violent_offense_found   BIT DEFAULT 0,
    dui_offense_found       BIT DEFAULT 0,
    hit_and_run_found       BIT DEFAULT 0,
    
    -- Overall evaluation
    overall_status          NVARCHAR(30) NOT NULL,             -- PASS, FAIL, MANUAL_REVIEW
    disqualification_reason NVARCHAR(500),
    disqualification_codes  NVARCHAR(MAX),                     -- JSON array
    
    -- Raw report reference (encrypted storage)
    raw_report_storage_url  NVARCHAR(500),
    
    -- Timestamps
    initiated_at            DATETIME2 NOT NULL,
    completed_at            DATETIME2,
    expires_at              DATETIME2,                         -- Annual expiry
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_bg_results_step FOREIGN KEY (step_instance_id) REFERENCES driver_step_instances(id),
    CONSTRAINT fk_bg_results_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_bg_results_driver_id ON background_check_results(driver_id);
CREATE INDEX idx_bg_results_expires_at ON background_check_results(expires_at);

-- ============================================================================
-- DRIVING HISTORY RESULTS (MVR)
-- ============================================================================
CREATE TABLE driving_history_results (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    step_instance_id        NVARCHAR(36) NOT NULL,
    driver_id               NVARCHAR(36) NOT NULL,
    
    -- Vendor info
    vendor_name             NVARCHAR(50) NOT NULL,
    vendor_report_id        NVARCHAR(100) NOT NULL,
    vendor_status           NVARCHAR(50) NOT NULL,
    
    -- License validation
    license_number          NVARCHAR(50),
    license_state           NVARCHAR(2),
    license_status          NVARCHAR(30),                      -- VALID, SUSPENDED, REVOKED, EXPIRED
    license_class           NVARCHAR(10),
    license_expiry          DATE,
    
    -- Violation counts (3-year lookback)
    total_violations_3yr    INT DEFAULT 0,
    moving_violations_3yr   INT DEFAULT 0,
    major_violations_3yr    INT DEFAULT 0,
    
    -- Specific disqualifying violations
    eluding_police_found    BIT DEFAULT 0,
    reckless_driving_found  BIT DEFAULT 0,
    suspended_license_driving BIT DEFAULT 0,
    
    -- Violation details
    violations              NVARCHAR(MAX),                     -- JSON array of violations
    
    -- Overall evaluation
    overall_status          NVARCHAR(30) NOT NULL,             -- PASS, FAIL, MANUAL_REVIEW
    disqualification_reason NVARCHAR(500),
    
    -- Raw report reference
    raw_report_storage_url  NVARCHAR(500),
    
    -- Timestamps
    initiated_at            DATETIME2 NOT NULL,
    completed_at            DATETIME2,
    expires_at              DATETIME2,                         -- Annual expiry
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_mvr_results_step FOREIGN KEY (step_instance_id) REFERENCES driver_step_instances(id),
    CONSTRAINT fk_mvr_results_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_mvr_results_driver_id ON driving_history_results(driver_id);
CREATE INDEX idx_mvr_results_expires_at ON driving_history_results(expires_at);

-- ============================================================================
-- INSURANCE POLICIES
-- ============================================================================
CREATE TABLE insurance_policies (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    vehicle_id              NVARCHAR(36),
    
    -- Policy details
    provider_name           NVARCHAR(100) NOT NULL,
    policy_number           NVARCHAR(100) NOT NULL,
    insured_name            NVARCHAR(200),
    
    -- Coverage info
    coverage_type           NVARCHAR(50),                      -- personal, commercial, tnc_rideshare
    liability_limit         DECIMAL(12,2),
    
    -- Dates
    effective_date          DATE NOT NULL,
    expiration_date         DATE NOT NULL,
    
    -- Verification
    verification_status     NVARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (verification_status IN ('PENDING', 'VERIFIED', 'REJECTED', 'EXPIRED')),
    verified_by             NVARCHAR(36),
    verified_at             DATETIME2,
    rejection_reason        NVARCHAR(500),
    
    -- Document reference
    document_id             NVARCHAR(36),
    
    -- Written proof of coverage (WA requirement)
    proof_of_coverage_issued BIT DEFAULT 0,
    proof_of_coverage_issued_at DATETIME2,
    proof_of_coverage_doc_id NVARCHAR(36),
    driver_acknowledged_at  DATETIME2,
    
    -- Metadata
    is_active               BIT NOT NULL DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_insurance_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_insurance_vehicle FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
    CONSTRAINT fk_insurance_document FOREIGN KEY (document_id) REFERENCES documents(id)
);

CREATE INDEX idx_insurance_driver_id ON insurance_policies(driver_id);
CREATE INDEX idx_insurance_expiration ON insurance_policies(expiration_date);
CREATE INDEX idx_insurance_status ON insurance_policies(verification_status);

-- ============================================================================
-- POLICY ACKNOWLEDGEMENTS
-- ============================================================================
CREATE TABLE policy_acknowledgements (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    case_id                 NVARCHAR(36),
    
    policy_type             NVARCHAR(50) NOT NULL CHECK (policy_type IN ('DRIVER_TERMS', 'PRIVACY', 'ZERO_TOLERANCE', 'PROHIBITED_ACTIVITIES', 'NONDISCRIMINATION', 'INSURANCE_DISCLOSURE', 'BACKGROUND_CHECK_CONSENT', 'CONTINUOUS_MONITORING')),
    policy_version          NVARCHAR(50) NOT NULL,
    policy_document_url     NVARCHAR(500),
    
    -- Acknowledgement
    acknowledged_at         DATETIME2 NOT NULL,
    signature_full_name     NVARCHAR(200),
    signature_data          NVARCHAR(MAX),                     -- Base64 signature
    
    -- Capture info
    ip_address              NVARCHAR(45),
    user_agent              NVARCHAR(500),
    device_fingerprint      NVARCHAR(100),
    
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_ack_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_ack_case FOREIGN KEY (case_id) REFERENCES driver_approval_cases(id)
);

CREATE INDEX idx_ack_driver_id ON policy_acknowledgements(driver_id);
CREATE INDEX idx_ack_policy_type ON policy_acknowledgements(policy_type);

-- ============================================================================
-- VEHICLE PROFILES (enhanced)
-- ============================================================================
ALTER TABLE vehicles ADD 
    tnc_eligible            BIT DEFAULT 0,
    eligibility_check_date  DATE,
    eligibility_notes       NVARCHAR(500),
    inspection_required     BIT DEFAULT 0,
    inspection_expires_at   DATE,
    inspection_document_id  NVARCHAR(36);

-- ============================================================================
-- RECHECK SCHEDULES
-- ============================================================================
CREATE TABLE recheck_schedules (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    check_type              NVARCHAR(50) NOT NULL CHECK (check_type IN ('BACKGROUND_CHECK', 'MVR', 'INSURANCE', 'LICENSE', 'VEHICLE_INSPECTION')),
    
    -- Schedule
    due_date                DATE NOT NULL,
    grace_period_days       INT DEFAULT 30,
    
    -- Status
    status                  NVARCHAR(30) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'OVERDUE', 'FAILED')),
    
    -- References
    case_id                 NVARCHAR(36),
    step_instance_id        NVARCHAR(36),
    
    -- Notification tracking
    reminder_30_days_sent   BIT DEFAULT 0,
    reminder_7_days_sent    BIT DEFAULT 0,
    reminder_1_day_sent     BIT DEFAULT 0,
    overdue_notification_sent BIT DEFAULT 0,
    
    -- Completion
    completed_at            DATETIME2,
    new_expiry_date         DATE,
    
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_recheck_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_recheck_case FOREIGN KEY (case_id) REFERENCES driver_approval_cases(id),
    CONSTRAINT fk_recheck_step FOREIGN KEY (step_instance_id) REFERENCES driver_step_instances(id)
);

CREATE INDEX idx_recheck_driver_id ON recheck_schedules(driver_id);
CREATE INDEX idx_recheck_due_date ON recheck_schedules(due_date);
CREATE INDEX idx_recheck_status ON recheck_schedules(status);

-- ============================================================================
-- APPROVAL AUDIT LOG (append-only, compliance-ready)
-- ============================================================================
CREATE TABLE approval_audit_events (
    id                      BIGINT IDENTITY(1,1) PRIMARY KEY,
    event_type              NVARCHAR(50) NOT NULL,             -- STEP_DECISION, STATUS_CHANGE, DOCUMENT_UPLOAD, etc.
    driver_id               NVARCHAR(36) NOT NULL,
    case_id                 NVARCHAR(36),
    step_instance_id        NVARCHAR(36),
    
    -- Actor
    actor_id                NVARCHAR(36) NOT NULL,             -- User ID or 'SYSTEM'
    actor_type              NVARCHAR(20) NOT NULL CHECK (actor_type IN ('DRIVER', 'ADMIN', 'SYSTEM', 'WEBHOOK')),
    actor_email             NVARCHAR(255),
    
    -- Event details
    old_status              NVARCHAR(50),
    new_status              NVARCHAR(50),
    decision                NVARCHAR(30),
    reason_codes            NVARCHAR(MAX),                     -- JSON array
    notes                   NVARCHAR(1000),
    
    -- Evidence reference
    evidence_ids            NVARCHAR(MAX),                     -- JSON array
    
    -- Request info
    ip_address              NVARCHAR(45),
    user_agent              NVARCHAR(500),
    request_id              NVARCHAR(100),
    
    -- Immutable timestamp
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_audit_driver_id ON approval_audit_events(driver_id);
CREATE INDEX idx_audit_case_id ON approval_audit_events(case_id);
CREATE INDEX idx_audit_event_type ON approval_audit_events(event_type);
CREATE INDEX idx_audit_created_at ON approval_audit_events(created_at);

-- ============================================================================
-- REASON CODES (lookup table)
-- ============================================================================
CREATE TABLE approval_reason_codes (
    code                    NVARCHAR(50) PRIMARY KEY,
    category                NVARCHAR(50) NOT NULL,             -- REJECTION, INFO_REQUEST, WAIVER
    description             NVARCHAR(500) NOT NULL,
    is_disqualifying        BIT DEFAULT 0,                     -- Cannot be overridden
    step_ids                NVARCHAR(MAX),                     -- JSON array of applicable steps
    is_active               BIT DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

-- Insert WA-specific reason codes
INSERT INTO approval_reason_codes (code, category, description, is_disqualifying, step_ids) VALUES
-- Disqualifying (auto-reject)
('DQ_AGE_UNDER_20', 'REJECTION', 'Driver is under 20 years of age', 1, '["LICENSE_VALIDATION"]'),
('DQ_VEHICLE_TOO_OLD', 'REJECTION', 'Vehicle is more than 15 years old', 1, '["VEHICLE_ELIGIBILITY"]'),
('DQ_3_PLUS_MOVING_VIOLATIONS', 'REJECTION', 'More than 3 moving violations in prior 3 years', 1, '["DRIVING_HISTORY"]'),
('DQ_MAJOR_VIOLATION_3YR', 'REJECTION', 'Major violation in prior 3 years (eluding police, reckless driving, suspended license)', 1, '["DRIVING_HISTORY"]'),
('DQ_FELONY_7YR', 'REJECTION', 'Class A or B felony conviction within past 7 years', 1, '["CRIMINAL_BACKGROUND"]'),
('DQ_VIOLENT_OFFENSE', 'REJECTION', 'Violent or serious violent offense conviction', 1, '["CRIMINAL_BACKGROUND"]'),
('DQ_DUI_7YR', 'REJECTION', 'DUI conviction within past 7 years', 1, '["CRIMINAL_BACKGROUND"]'),
('DQ_HIT_AND_RUN', 'REJECTION', 'Hit and run conviction', 1, '["CRIMINAL_BACKGROUND"]'),
('DQ_SEX_OFFENDER', 'REJECTION', 'Match on National Sex Offender Registry', 1, '["CRIMINAL_BACKGROUND"]'),
('DQ_LICENSE_SUSPENDED', 'REJECTION', 'Driver license is suspended or revoked', 1, '["LICENSE_VALIDATION", "DRIVING_HISTORY"]'),
('DQ_LICENSE_EXPIRED', 'REJECTION', 'Driver license has expired', 1, '["LICENSE_VALIDATION"]'),
('DQ_INSURANCE_EXPIRED', 'REJECTION', 'Insurance policy has expired', 1, '["INSURANCE_VERIFICATION"]'),

-- Manual rejection (admin discretion)
('REJ_DOCUMENT_UNREADABLE', 'REJECTION', 'Document is unreadable or low quality', 0, NULL),
('REJ_DOCUMENT_MISMATCH', 'REJECTION', 'Document information does not match application', 0, NULL),
('REJ_DOCUMENT_EXPIRED', 'REJECTION', 'Document has expired', 0, NULL),
('REJ_DOCUMENT_FRAUDULENT', 'REJECTION', 'Document appears to be fraudulent', 0, NULL),
('REJ_PHOTO_UNSUITABLE', 'REJECTION', 'Photo does not meet requirements', 0, NULL),
('REJ_VEHICLE_DAMAGE', 'REJECTION', 'Vehicle shows significant damage', 0, '["VEHICLE_ELIGIBILITY"]'),
('REJ_INSURANCE_INSUFFICIENT', 'REJECTION', 'Insurance coverage does not meet TNC requirements', 0, '["INSURANCE_VERIFICATION"]'),
('REJ_BACKGROUND_CONCERN', 'REJECTION', 'Background check requires manual review - denied', 0, '["CRIMINAL_BACKGROUND"]'),

-- Request more info
('INFO_DOCUMENT_BLURRY', 'INFO_REQUEST', 'Please upload a clearer photo of the document', 0, NULL),
('INFO_DOCUMENT_MISSING', 'INFO_REQUEST', 'Required document is missing', 0, NULL),
('INFO_DOCUMENT_PARTIAL', 'INFO_REQUEST', 'Document is partially visible - please upload complete document', 0, NULL),
('INFO_INFO_MISMATCH', 'INFO_REQUEST', 'Information does not match - please verify and resubmit', 0, NULL),
('INFO_ADDITIONAL_DOCS', 'INFO_REQUEST', 'Additional documentation required', 0, NULL);

-- ============================================================================
-- JURISDICTIONAL REQUIREMENTS (configurable)
-- ============================================================================
CREATE TABLE jurisdictional_requirements (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    jurisdiction_code       NVARCHAR(20) NOT NULL,             -- 'WA', 'WA-SEATTLE', 'WA-KING', 'WA-SEATAC'
    jurisdiction_name       NVARCHAR(100) NOT NULL,
    parent_jurisdiction     NVARCHAR(20),                      -- For inheritance
    
    -- Requirements
    requires_for_hire_permit BIT DEFAULT 0,
    requires_tnc_decal      BIT DEFAULT 0,
    requires_vehicle_inspection BIT DEFAULT 0,
    requires_local_training BIT DEFAULT 0,
    requires_local_exam     BIT DEFAULT 0,
    
    -- Thresholds
    min_driver_age          INT DEFAULT 20,
    max_vehicle_age_years   INT DEFAULT 15,
    max_moving_violations_3yr INT DEFAULT 3,
    background_check_lookback_years INT DEFAULT 7,
    
    -- Retention
    record_retention_years  INT DEFAULT 7,
    complaint_retention_years INT DEFAULT 2,
    
    -- Config
    config_json             NVARCHAR(MAX),                     -- Additional JSON config
    is_active               BIT DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

-- Insert WA jurisdictions
INSERT INTO jurisdictional_requirements (jurisdiction_code, jurisdiction_name, min_driver_age, max_vehicle_age_years) VALUES
('WA', 'Washington State', 20, 15),
('WA-SEATTLE', 'Seattle', 20, 15),
('WA-KING', 'King County', 20, 15),
('WA-SEATAC', 'SeaTac Airport', 20, 15);

-- ============================================================================
-- DRIVER COMPLAINTS (for zero-tolerance + 2-year retention)
-- ============================================================================
CREATE TABLE driver_complaints (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    complainant_id          NVARCHAR(36),                      -- Rider or anonymous
    trip_id                 NVARCHAR(36),
    
    complaint_type          NVARCHAR(50) NOT NULL,             -- SAFETY, DISCRIMINATION, IMPAIRMENT, HARASSMENT, OTHER
    severity                NVARCHAR(20) DEFAULT 'MEDIUM' CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    description             NVARCHAR(MAX) NOT NULL,
    
    -- Investigation
    status                  NVARCHAR(30) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'INVESTIGATING', 'RESOLVED', 'DISMISSED', 'ESCALATED')),
    assigned_to             NVARCHAR(36),
    resolution              NVARCHAR(MAX),
    resolution_date         DATETIME2,
    
    -- Action taken
    action_taken            NVARCHAR(50),                      -- WARNING, SUSPENSION, DEACTIVATION, NONE
    action_notes            NVARCHAR(MAX),
    
    -- Evidence
    evidence_document_ids   NVARCHAR(MAX),                     -- JSON array
    
    -- Retention (WA requires 2 years minimum)
    retention_until         DATE NOT NULL,
    
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_complaint_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_complaint_driver_id ON driver_complaints(driver_id);
CREATE INDEX idx_complaint_status ON driver_complaints(status);
CREATE INDEX idx_complaint_retention ON driver_complaints(retention_until);

-- ============================================================================
-- APPEALS / REVIEW REQUESTS
-- ============================================================================
CREATE TABLE driver_appeals (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    case_id                 NVARCHAR(36),
    step_instance_id        NVARCHAR(36),
    
    appeal_type             NVARCHAR(30) NOT NULL CHECK (appeal_type IN ('REJECTION_APPEAL', 'SUSPENSION_APPEAL', 'DOCUMENT_REVIEW', 'GENERAL')),
    status                  NVARCHAR(30) NOT NULL DEFAULT 'SUBMITTED' CHECK (status IN ('SUBMITTED', 'UNDER_REVIEW', 'APPROVED', 'DENIED', 'WITHDRAWN')),
    
    -- Driver submission
    explanation             NVARCHAR(MAX) NOT NULL,
    evidence_document_ids   NVARCHAR(MAX),                     -- JSON array
    submitted_at            DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    -- Admin review
    reviewed_by             NVARCHAR(36),
    reviewed_at             DATETIME2,
    decision_notes          NVARCHAR(MAX),
    
    -- Link to support ticket
    support_ticket_id       NVARCHAR(36),
    
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_appeal_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_appeal_case FOREIGN KEY (case_id) REFERENCES driver_approval_cases(id),
    CONSTRAINT fk_appeal_step FOREIGN KEY (step_instance_id) REFERENCES driver_step_instances(id)
);

CREATE INDEX idx_appeal_driver_id ON driver_appeals(driver_id);
CREATE INDEX idx_appeal_status ON driver_appeals(status);
