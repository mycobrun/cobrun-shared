-- Migration: 009_create_notifications
-- Description: Create notification tables

-- Device Tokens
CREATE TABLE device_tokens (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    token                   NVARCHAR(500) NOT NULL,
    platform                NVARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    app_version             NVARCHAR(20),
    device_model            NVARCHAR(100),
    is_active               BIT DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    last_used_at            DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_device_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX idx_device_tokens_is_active ON device_tokens(is_active);

-- Notification Preferences
CREATE TABLE notification_preferences (
    user_id                 NVARCHAR(36) PRIMARY KEY,
    push_enabled            BIT DEFAULT 1,
    email_enabled           BIT DEFAULT 1,
    sms_enabled             BIT DEFAULT 1,
    trip_updates            BIT DEFAULT 1,
    payment_alerts          BIT DEFAULT 1,
    promo_alerts            BIT DEFAULT 1,
    news_updates            BIT DEFAULT 0,
    quiet_hours_start       NVARCHAR(5),
    quiet_hours_end         NVARCHAR(5),
    timezone                NVARCHAR(50) DEFAULT 'UTC',
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_notification_preferences_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Notifications Log
CREATE TABLE notifications (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    type                    NVARCHAR(50) NOT NULL,
    channel                 NVARCHAR(10) NOT NULL CHECK (channel IN ('push', 'email', 'sms', 'in_app')),
    status                  NVARCHAR(15) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'read')),
    title                   NVARCHAR(200) NOT NULL,
    body                    NVARCHAR(MAX) NOT NULL,
    image_url               NVARCHAR(500),
    action_url              NVARCHAR(500),
    data                    NVARCHAR(MAX),
    priority                NVARCHAR(10) DEFAULT 'normal' CHECK (priority IN ('high', 'normal', 'low')),
    template_id             NVARCHAR(36),
    locale                  NVARCHAR(10) DEFAULT 'en',
    reference_id            NVARCHAR(36),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    sent_at                 DATETIME2,
    delivered_at            DATETIME2,
    read_at                 DATETIME2,
    failed_at               DATETIME2,
    fail_reason             NVARCHAR(500),
    
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);





