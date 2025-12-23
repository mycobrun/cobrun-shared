-- Migration: 007_create_support
-- Description: Create support ticket tables

-- Support Tickets
CREATE TABLE support_tickets (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    user_type               NVARCHAR(10) NOT NULL CHECK (user_type IN ('rider', 'driver')),
    trip_id                 NVARCHAR(36),
    category                NVARCHAR(30) NOT NULL CHECK (category IN ('trip_issue', 'payment_issue', 'driver_behavior', 'rider_behavior', 'safety_incident', 'lost_item', 'account_issue', 'app_bug', 'feedback', 'refund_request', 'promo_issue', 'other')),
    subject                 NVARCHAR(200) NOT NULL,
    description             NVARCHAR(MAX) NOT NULL,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'pending', 'resolved', 'closed', 'escalated')),
    priority                NVARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    assigned_to             NVARCHAR(36),
    assigned_team           NVARCHAR(50),
    tags                    NVARCHAR(MAX),
    resolution              NVARCHAR(MAX),
    resolution_type         NVARCHAR(50),
    satisfaction_score      INT CHECK (satisfaction_score BETWEEN 1 AND 5),
    first_response_at       DATETIME2,
    resolved_at             DATETIME2,
    closed_at               DATETIME2,
    escalated_at            DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_support_tickets_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_support_tickets_user_id ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_trip_id ON support_tickets(trip_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_category ON support_tickets(category);
CREATE INDEX idx_support_tickets_assigned_to ON support_tickets(assigned_to);
CREATE INDEX idx_support_tickets_created_at ON support_tickets(created_at);

-- Ticket Messages
CREATE TABLE ticket_messages (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    ticket_id               NVARCHAR(36) NOT NULL,
    sender_id               NVARCHAR(36) NOT NULL,
    sender_type             NVARCHAR(10) NOT NULL CHECK (sender_type IN ('user', 'agent', 'system')),
    sender_name             NVARCHAR(100) NOT NULL,
    message                 NVARCHAR(MAX) NOT NULL,
    is_internal             BIT DEFAULT 0,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_ticket_messages_ticket FOREIGN KEY (ticket_id) REFERENCES support_tickets(id)
);

CREATE INDEX idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
CREATE INDEX idx_ticket_messages_created_at ON ticket_messages(created_at);

-- Ticket Attachments
CREATE TABLE ticket_attachments (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    ticket_id               NVARCHAR(36),
    message_id              NVARCHAR(36),
    file_name               NVARCHAR(255) NOT NULL,
    file_type               NVARCHAR(100) NOT NULL,
    file_size               BIGINT NOT NULL,
    url                     NVARCHAR(500) NOT NULL,
    uploaded_by             NVARCHAR(36) NOT NULL,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_ticket_attachments_ticket FOREIGN KEY (ticket_id) REFERENCES support_tickets(id),
    CONSTRAINT fk_ticket_attachments_message FOREIGN KEY (message_id) REFERENCES ticket_messages(id)
);

CREATE INDEX idx_ticket_attachments_ticket_id ON ticket_attachments(ticket_id);

-- Support Agents
CREATE TABLE support_agents (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    name                    NVARCHAR(100) NOT NULL,
    email                   NVARCHAR(255) NOT NULL UNIQUE,
    team                    NVARCHAR(50),
    is_available            BIT DEFAULT 1,
    current_load            INT DEFAULT 0,
    max_load                INT DEFAULT 20,
    skills                  NVARCHAR(MAX),
    languages               NVARCHAR(MAX),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_support_agents_team ON support_agents(team);
CREATE INDEX idx_support_agents_is_available ON support_agents(is_available);





