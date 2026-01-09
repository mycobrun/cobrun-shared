-- Migration: 012_create_wa_approval_workflow (rollback)
-- Description: Drop Washington State compliant driver approval workflow tables

-- Drop tables in reverse order of creation (due to foreign keys)
DROP TABLE IF EXISTS driver_appeals;
DROP TABLE IF EXISTS driver_complaints;
DROP TABLE IF EXISTS jurisdictional_requirements;
DROP TABLE IF EXISTS approval_reason_codes;
DROP TABLE IF EXISTS approval_audit_events;
DROP TABLE IF EXISTS recheck_schedules;
DROP TABLE IF EXISTS policy_acknowledgements;
DROP TABLE IF EXISTS insurance_policies;
DROP TABLE IF EXISTS driving_history_results;
DROP TABLE IF EXISTS background_check_results;
DROP TABLE IF EXISTS onboarding_evidence;
DROP TABLE IF EXISTS driver_step_instances;
DROP TABLE IF EXISTS driver_approval_cases;
DROP TABLE IF EXISTS onboarding_step_definitions;

-- Remove columns added to vehicles table
ALTER TABLE vehicles DROP COLUMN IF EXISTS tnc_eligible;
ALTER TABLE vehicles DROP COLUMN IF EXISTS eligibility_check_date;
ALTER TABLE vehicles DROP COLUMN IF EXISTS eligibility_notes;
ALTER TABLE vehicles DROP COLUMN IF EXISTS inspection_required;
ALTER TABLE vehicles DROP COLUMN IF EXISTS inspection_expires_at;
ALTER TABLE vehicles DROP COLUMN IF EXISTS inspection_document_id;
