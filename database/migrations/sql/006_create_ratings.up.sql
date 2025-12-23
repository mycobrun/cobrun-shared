-- Migration: 006_create_ratings
-- Description: Create ratings and reviews tables

-- Ratings
CREATE TABLE ratings (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    trip_id                 NVARCHAR(36) NOT NULL,
    rater_id                NVARCHAR(36) NOT NULL,
    rater_type              NVARCHAR(10) NOT NULL CHECK (rater_type IN ('rider', 'driver')),
    ratee_id                NVARCHAR(36) NOT NULL,
    ratee_type              NVARCHAR(10) NOT NULL CHECK (ratee_type IN ('rider', 'driver')),
    overall_score           INT NOT NULL CHECK (overall_score BETWEEN 1 AND 5),
    categories              NVARCHAR(MAX),
    tags                    NVARCHAR(MAX),
    comment                 NVARCHAR(1000),
    is_anonymous            BIT DEFAULT 0,
    tip_amount              DECIMAL(10,2),
    tip_currency            NVARCHAR(3),
    response_id             NVARCHAR(36),
    is_flagged              BIT DEFAULT 0,
    flag_reason             NVARCHAR(500),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_ratings_rater FOREIGN KEY (rater_id) REFERENCES users(id),
    CONSTRAINT fk_ratings_ratee FOREIGN KEY (ratee_id) REFERENCES users(id)
);

CREATE INDEX idx_ratings_trip_id ON ratings(trip_id);
CREATE INDEX idx_ratings_rater_id ON ratings(rater_id);
CREATE INDEX idx_ratings_ratee_id ON ratings(ratee_id);
CREATE INDEX idx_ratings_overall_score ON ratings(overall_score);
CREATE INDEX idx_ratings_created_at ON ratings(created_at);

-- Rating Responses
CREATE TABLE rating_responses (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    rating_id               NVARCHAR(36) NOT NULL UNIQUE,
    responder_id            NVARCHAR(36) NOT NULL,
    response                NVARCHAR(1000) NOT NULL,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_rating_responses_rating FOREIGN KEY (rating_id) REFERENCES ratings(id),
    CONSTRAINT fk_rating_responses_responder FOREIGN KEY (responder_id) REFERENCES users(id)
);

-- User Rating Summaries
CREATE TABLE user_rating_summaries (
    user_id                 NVARCHAR(36) PRIMARY KEY,
    user_type               NVARCHAR(10) NOT NULL,
    average_rating          DECIMAL(3,2) NOT NULL DEFAULT 5.00,
    total_ratings           INT DEFAULT 0,
    rating_1_count          INT DEFAULT 0,
    rating_2_count          INT DEFAULT 0,
    rating_3_count          INT DEFAULT 0,
    rating_4_count          INT DEFAULT 0,
    rating_5_count          INT DEFAULT 0,
    category_averages       NVARCHAR(MAX),
    top_tags                NVARCHAR(MAX),
    last_30_days_avg        DECIMAL(3,2),
    last_30_days_count      INT DEFAULT 0,
    total_trips             INT DEFAULT 0,
    tips_received           DECIMAL(10,2) DEFAULT 0,
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_user_rating_summaries_user FOREIGN KEY (user_id) REFERENCES users(id)
);





