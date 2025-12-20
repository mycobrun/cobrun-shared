-- Migration: 002_create_vehicles
-- Description: Create vehicles table

CREATE TABLE vehicles (
    id                      NVARCHAR(36) PRIMARY KEY DEFAULT NEWID(),
    user_id                 NVARCHAR(36) NOT NULL,
    make                    NVARCHAR(50) NOT NULL,
    model                   NVARCHAR(50) NOT NULL,
    year                    INT NOT NULL,
    color                   NVARCHAR(30) NOT NULL,
    license_plate           NVARCHAR(20) NOT NULL,
    vin                     NVARCHAR(17),
    vehicle_type            NVARCHAR(20) NOT NULL CHECK (vehicle_type IN ('sedan', 'suv', 'van', 'luxury')),
    capacity                INT NOT NULL DEFAULT 4,
    is_active               BIT DEFAULT 0,
    status                  NVARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewing', 'approved', 'rejected', 'expired')),
    registration_doc_id     NVARCHAR(36),
    insurance_doc_id        NVARCHAR(36),
    verified_at             DATETIME2,
    verified_by             NVARCHAR(36),
    created_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at              DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    
    CONSTRAINT fk_vehicles_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_vehicles_user_id ON vehicles(user_id);
CREATE INDEX idx_vehicles_license_plate ON vehicles(license_plate);
CREATE INDEX idx_vehicles_status ON vehicles(status);
CREATE INDEX idx_vehicles_vehicle_type ON vehicles(vehicle_type);

