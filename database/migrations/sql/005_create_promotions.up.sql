-- Migration: 005_create_promotions
-- Description: Create promotions and referrals tables

-- Promotions
CREATE TABLE promotions (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    code                    NVARCHAR(20) NOT NULL UNIQUE,
    name                    NVARCHAR(100) NOT NULL,
    description             NVARCHAR(500),
    type                    NVARCHAR(20) NOT NULL CHECK (type IN ('percent_off', 'fixed_amount', 'free_ride', 'referral', 'first_ride')),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'expired', 'depleted')),
    discount_percent        DECIMAL(5,2),
    discount_amount         DECIMAL(10,2),
    max_discount            DECIMAL(10,2),
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    min_order_amount        DECIMAL(10,2) DEFAULT 0,
    max_usage_total         INT DEFAULT 0,
    max_usage_per_user      INT DEFAULT 1,
    valid_ride_types        NVARCHAR(MAX),
    valid_cities            NVARCHAR(MAX),
    first_ride_only         BIT DEFAULT 0,
    new_users_only          BIT DEFAULT 0,
    total_used              INT DEFAULT 0,
    starts_at               DATETIME2 NOT NULL,
    expires_at              DATETIME2 NOT NULL,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_promotions_code ON promotions(code);
CREATE INDEX idx_promotions_status ON promotions(status);
CREATE INDEX idx_promotions_expires_at ON promotions(expires_at);

-- Promo Usage
CREATE TABLE promo_usage (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    promo_id                NVARCHAR(36) NOT NULL,
    promo_code              NVARCHAR(20) NOT NULL,
    user_id                 NVARCHAR(36) NOT NULL,
    trip_id                 NVARCHAR(36),
    amount                  DECIMAL(10,2) NOT NULL,
    discount                DECIMAL(10,2) NOT NULL,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    used_at                 DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_promo_usage_promo FOREIGN KEY (promo_id) REFERENCES promotions(id),
    CONSTRAINT fk_promo_usage_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_promo_usage_promo_id ON promo_usage(promo_id);
CREATE INDEX idx_promo_usage_user_id ON promo_usage(user_id);
CREATE INDEX idx_promo_usage_used_at ON promo_usage(used_at);

-- User Promos
CREATE TABLE user_promos (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    promo_id                NVARCHAR(36) NOT NULL,
    promo_code              NVARCHAR(20) NOT NULL,
    promo_type              NVARCHAR(20) NOT NULL,
    discount                DECIMAL(10,2) NOT NULL,
    max_discount            DECIMAL(10,2),
    uses_left               INT DEFAULT 1,
    expires_at              DATETIME2 NOT NULL,
    used_at                 DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_user_promos_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_user_promos_promo FOREIGN KEY (promo_id) REFERENCES promotions(id)
);

CREATE INDEX idx_user_promos_user_id ON user_promos(user_id);
CREATE INDEX idx_user_promos_expires_at ON user_promos(expires_at);

-- Referrals
CREATE TABLE referrals (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    referrer_id             NVARCHAR(36) NOT NULL,
    referred_id             NVARCHAR(36) NOT NULL,
    referral_code           NVARCHAR(20) NOT NULL,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'expired')),
    referrer_bonus          DECIMAL(10,2) NOT NULL,
    referred_bonus          DECIMAL(10,2) NOT NULL,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    completed_at            DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_referrals_referrer FOREIGN KEY (referrer_id) REFERENCES users(id),
    CONSTRAINT fk_referrals_referred FOREIGN KEY (referred_id) REFERENCES users(id)
);

CREATE INDEX idx_referrals_referrer_id ON referrals(referrer_id);
CREATE INDEX idx_referrals_referred_id ON referrals(referred_id);
CREATE INDEX idx_referrals_status ON referrals(status);

-- Wallet Credits
CREATE TABLE wallet_credits (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    amount                  DECIMAL(10,2) NOT NULL,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    type                    NVARCHAR(30) NOT NULL,
    description             NVARCHAR(500),
    reference_id            NVARCHAR(36),
    expires_at              DATETIME2 NOT NULL,
    used_amount             DECIMAL(10,2) DEFAULT 0,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'fully_used')),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_wallet_credits_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_wallet_credits_user_id ON wallet_credits(user_id);
CREATE INDEX idx_wallet_credits_status ON wallet_credits(status);
CREATE INDEX idx_wallet_credits_expires_at ON wallet_credits(expires_at);


