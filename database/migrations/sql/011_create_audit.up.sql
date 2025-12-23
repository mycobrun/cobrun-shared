-- Migration: 011_create_audit
-- Description: Create audit log and schema migrations tables

-- Audit Logs
CREATE TABLE audit_logs (
    id                      BIGINT IDENTITY(1,1) PRIMARY KEY,
    entity_type             NVARCHAR(50) NOT NULL,
    entity_id               NVARCHAR(36) NOT NULL,
    action                  NVARCHAR(20) NOT NULL CHECK (action IN ('create', 'update', 'delete', 'read')),
    actor_id                NVARCHAR(36),
    actor_type              NVARCHAR(20) CHECK (actor_type IN ('user', 'driver', 'admin', 'system')),
    old_values              NVARCHAR(MAX),
    new_values              NVARCHAR(MAX),
    ip_address              NVARCHAR(45),
    user_agent              NVARCHAR(500),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Schema Migrations (for tracking applied migrations)
CREATE TABLE schema_migrations (
    version                 INT PRIMARY KEY,
    name                    NVARCHAR(255) NOT NULL,
    applied_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);





