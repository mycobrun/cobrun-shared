-- Migration: 001_create_users
-- Description: Create users and related tables for user management

-- Users table (both riders and drivers)
CREATE TABLE users (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    email                   NVARCHAR(255) NOT NULL UNIQUE,
    phone                   NVARCHAR(20) NOT NULL,
    phone_verified          BIT DEFAULT 0,
    email_verified          BIT DEFAULT 0,
    password_hash           NVARCHAR(255) NOT NULL,
    first_name              NVARCHAR(100) NOT NULL,
    last_name               NVARCHAR(100) NOT NULL,
    profile_photo_url       NVARCHAR(500),
    user_type               NVARCHAR(20) NOT NULL CHECK (user_type IN ('rider', 'driver', 'admin')),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended', 'banned')),
    language                NVARCHAR(10) DEFAULT 'en',
    timezone                NVARCHAR(50) DEFAULT 'UTC',
    city                    NVARCHAR(100),
    country                 NVARCHAR(2),
    stripe_customer_id      NVARCHAR(100),
    referral_code           NVARCHAR(20) UNIQUE,
    referred_by             NVARCHAR(36),
    last_login_at           DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    deleted_at              DATETIME2,
    
    CONSTRAINT fk_users_referred_by FOREIGN KEY (referred_by) REFERENCES users(id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_user_type ON users(user_type);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_referral_code ON users(referral_code);

-- Driver profiles (extends users for drivers)
CREATE TABLE driver_profiles (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL UNIQUE,
    driver_license_number   NVARCHAR(50),
    license_state           NVARCHAR(50),
    license_expiry          DATE,
    average_rating          DECIMAL(3,2) DEFAULT 5.00,
    total_ratings           INT DEFAULT 0,
    total_trips             INT DEFAULT 0,
    completed_trips         INT DEFAULT 0,
    cancelled_trips         INT DEFAULT 0,
    acceptance_rate         DECIMAL(5,4) DEFAULT 1.0000,
    cancellation_rate       DECIMAL(5,4) DEFAULT 0.0000,
    online_hours_total      DECIMAL(10,2) DEFAULT 0,
    earnings_total          DECIMAL(12,2) DEFAULT 0,
    tips_total              DECIMAL(10,2) DEFAULT 0,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'suspended', 'inactive', 'rejected')),
    can_drive               BIT DEFAULT 0,
    is_online               BIT DEFAULT 0,
    last_online_at          DATETIME2,
    approved_at             DATETIME2,
    approved_by             NVARCHAR(36),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_driver_profiles_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_driver_profiles_user_id ON driver_profiles(user_id);
CREATE INDEX idx_driver_profiles_status ON driver_profiles(status);
CREATE INDEX idx_driver_profiles_is_online ON driver_profiles(is_online);
CREATE INDEX idx_driver_profiles_rating ON driver_profiles(average_rating);

