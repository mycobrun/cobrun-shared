-- Rollback: 007_create_support

DROP INDEX IF EXISTS idx_support_agents_is_available;
DROP INDEX IF EXISTS idx_support_agents_team;
DROP TABLE IF EXISTS support_agents;

DROP INDEX IF EXISTS idx_ticket_attachments_ticket_id;
DROP TABLE IF EXISTS ticket_attachments;

DROP INDEX IF EXISTS idx_ticket_messages_created_at;
DROP INDEX IF EXISTS idx_ticket_messages_ticket_id;
DROP TABLE IF EXISTS ticket_messages;

DROP INDEX IF EXISTS idx_support_tickets_created_at;
DROP INDEX IF EXISTS idx_support_tickets_assigned_to;
DROP INDEX IF EXISTS idx_support_tickets_category;
DROP INDEX IF EXISTS idx_support_tickets_priority;
DROP INDEX IF EXISTS idx_support_tickets_status;
DROP INDEX IF EXISTS idx_support_tickets_trip_id;
DROP INDEX IF EXISTS idx_support_tickets_user_id;
DROP TABLE IF EXISTS support_tickets;





