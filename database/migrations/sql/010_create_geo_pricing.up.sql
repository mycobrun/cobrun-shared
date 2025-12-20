-- Migration: 010_create_geo_pricing
-- Description: Create geofencing and pricing tables

-- Service Areas
CREATE TABLE service_areas (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    name                    NVARCHAR(100) NOT NULL,
    city                    NVARCHAR(100) NOT NULL,
    state                   NVARCHAR(50),
    country                 NVARCHAR(2) NOT NULL,
    timezone                NVARCHAR(50) NOT NULL,
    polygon                 NVARCHAR(MAX) NOT NULL, -- GeoJSON
    center_lat              DECIMAL(10,7) NOT NULL,
    center_lng              DECIMAL(10,7) NOT NULL,
    is_active               BIT DEFAULT 1,
    launch_date             DATE,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    default_language        NVARCHAR(10) DEFAULT 'en',
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_service_areas_city ON service_areas(city);
CREATE INDEX idx_service_areas_is_active ON service_areas(is_active);
CREATE INDEX idx_service_areas_center ON service_areas(center_lat, center_lng);

-- Geofences
CREATE TABLE geofences (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    name                    NVARCHAR(100) NOT NULL,
    type                    NVARCHAR(20) NOT NULL CHECK (type IN ('service_area', 'surge_zone', 'no_pickup', 'no_dropoff', 'airport', 'event')),
    city                    NVARCHAR(100) NOT NULL,
    polygon                 NVARCHAR(MAX) NOT NULL, -- GeoJSON
    center_lat              DECIMAL(10,7),
    center_lng              DECIMAL(10,7),
    is_active               BIT DEFAULT 1,
    priority                INT DEFAULT 0,
    metadata                NVARCHAR(MAX),
    effective_at            DATETIME2,
    expires_at              DATETIME2,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

CREATE INDEX idx_geofences_type ON geofences(type);
CREATE INDEX idx_geofences_city ON geofences(city);
CREATE INDEX idx_geofences_is_active ON geofences(is_active);
CREATE INDEX idx_geofences_center ON geofences(center_lat, center_lng);

-- Rate Cards
CREATE TABLE rate_cards (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    service_area_id         NVARCHAR(36),
    ride_type               NVARCHAR(20) NOT NULL CHECK (ride_type IN ('standard', 'premium', 'xl', 'pool')),
    base_fare               DECIMAL(10,2) NOT NULL,
    per_km                  DECIMAL(10,4) NOT NULL,
    per_minute              DECIMAL(10,4) NOT NULL,
    min_fare                DECIMAL(10,2) NOT NULL,
    booking_fee             DECIMAL(10,2) DEFAULT 0,
    cancellation_fee        DECIMAL(10,2) DEFAULT 5.00,
    wait_time_per_min       DECIMAL(10,4) DEFAULT 0,
    currency                NVARCHAR(3) NOT NULL DEFAULT 'USD',
    description             NVARCHAR(500),
    is_active               BIT DEFAULT 1,
    effective_from          DATE NOT NULL,
    effective_to            DATE,
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_rate_cards_service_area FOREIGN KEY (service_area_id) REFERENCES service_areas(id)
);

CREATE INDEX idx_rate_cards_ride_type ON rate_cards(ride_type);
CREATE INDEX idx_rate_cards_service_area_id ON rate_cards(service_area_id);
CREATE INDEX idx_rate_cards_is_active ON rate_cards(is_active);

-- Surge Zones (for surge pricing tracking)
CREATE TABLE surge_zones (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    name                    NVARCHAR(100) NOT NULL,
    service_area_id         NVARCHAR(36),
    geofence_id             NVARCHAR(36),
    center_lat              DECIMAL(10,7) NOT NULL,
    center_lng              DECIMAL(10,7) NOT NULL,
    radius_km               DECIMAL(5,2) NOT NULL,
    current_multiplier      DECIMAL(3,2) DEFAULT 1.00,
    base_multiplier         DECIMAL(3,2) DEFAULT 1.00,
    max_multiplier          DECIMAL(3,2) DEFAULT 3.00,
    active_requests         INT DEFAULT 0,
    available_drivers       INT DEFAULT 0,
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_surge_zones_service_area FOREIGN KEY (service_area_id) REFERENCES service_areas(id),
    CONSTRAINT fk_surge_zones_geofence FOREIGN KEY (geofence_id) REFERENCES geofences(id)
);

CREATE INDEX idx_surge_zones_center ON surge_zones(center_lat, center_lng);
CREATE INDEX idx_surge_zones_service_area_id ON surge_zones(service_area_id);

