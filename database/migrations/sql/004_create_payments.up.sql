-- Migration: 004_create_payments
-- Description: Create payment-related tables

-- Payment Methods
CREATE TABLE payment_methods (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    stripe_payment_method_id NVARCHAR(100) NOT NULL,
    stripe_customer_id      NVARCHAR(100),
    type                    NVARCHAR(20) NOT NULL CHECK (type IN ('card', 'bank_account')),
    card_last4              NVARCHAR(4),
    card_brand              NVARCHAR(20),
    card_exp_month          INT,
    card_exp_year           INT,
    is_default              BIT DEFAULT 0,
    is_active               BIT DEFAULT 1,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_payment_methods_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX idx_payment_methods_is_default ON payment_methods(user_id, is_default);

-- Payments (transactions)
CREATE TABLE payments (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    trip_id                 NVARCHAR(36),
    rider_id                NVARCHAR(36) NOT NULL,
    driver_id               NVARCHAR(36),
    type                    NVARCHAR(20) NOT NULL CHECK (type IN ('trip_fare', 'cancellation', 'tip', 'promo_credit', 'refund')),
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'refunded', 'cancelled')),
    amount                  DECIMAL(10,2) NOT NULL,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    payment_method_id       NVARCHAR(36),
    stripe_payment_id       NVARCHAR(100),
    stripe_charge_id        NVARCHAR(100),
    platform_fee            DECIMAL(10,2) DEFAULT 0,
    processing_fee          DECIMAL(10,2) DEFAULT 0,
    driver_payout           DECIMAL(10,2) DEFAULT 0,
    description             NVARCHAR(500),
    receipt_url             NVARCHAR(500),
    failure_reason          NVARCHAR(500),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    processed_at            DATETIME2,
    
    CONSTRAINT fk_payments_rider FOREIGN KEY (rider_id) REFERENCES users(id),
    CONSTRAINT fk_payments_driver FOREIGN KEY (driver_id) REFERENCES users(id),
    CONSTRAINT fk_payments_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id)
);

CREATE INDEX idx_payments_trip_id ON payments(trip_id);
CREATE INDEX idx_payments_rider_id ON payments(rider_id);
CREATE INDEX idx_payments_driver_id ON payments(driver_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at);

-- Wallets
CREATE TABLE wallets (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL UNIQUE,
    balance                 DECIMAL(10,2) NOT NULL DEFAULT 0,
    pending_balance         DECIMAL(10,2) NOT NULL DEFAULT 0,
    available_balance       DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_wallets_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);

-- Wallet Transactions
CREATE TABLE wallet_transactions (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    wallet_id               NVARCHAR(36) NOT NULL,
    user_id                 NVARCHAR(36) NOT NULL,
    type                    NVARCHAR(20) NOT NULL CHECK (type IN ('credit', 'debit')),
    amount                  DECIMAL(10,2) NOT NULL,
    balance_after           DECIMAL(10,2) NOT NULL,
    description             NVARCHAR(500),
    reference_id            NVARCHAR(36),
    reference_type          NVARCHAR(30),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_wallet_transactions_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id),
    CONSTRAINT fk_wallet_transactions_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX idx_wallet_transactions_user_id ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_transactions_created_at ON wallet_transactions(created_at);

-- Driver Payouts
CREATE TABLE driver_payouts (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    driver_id               NVARCHAR(36) NOT NULL,
    amount                  DECIMAL(10,2) NOT NULL,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    stripe_transfer_id      NVARCHAR(100),
    payout_method           NVARCHAR(30),
    period_start            DATE,
    period_end              DATE,
    trips_count             INT DEFAULT 0,
    gross_earnings          DECIMAL(10,2) DEFAULT 0,
    platform_fees           DECIMAL(10,2) DEFAULT 0,
    tips_amount             DECIMAL(10,2) DEFAULT 0,
    failure_reason          NVARCHAR(500),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    completed_at            DATETIME2,
    
    CONSTRAINT fk_driver_payouts_driver FOREIGN KEY (driver_id) REFERENCES users(id)
);

CREATE INDEX idx_driver_payouts_driver_id ON driver_payouts(driver_id);
CREATE INDEX idx_driver_payouts_status ON driver_payouts(status);
CREATE INDEX idx_driver_payouts_created_at ON driver_payouts(created_at);

