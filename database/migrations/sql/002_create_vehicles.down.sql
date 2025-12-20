-- Rollback: 002_create_vehicles

DROP INDEX IF EXISTS idx_vehicles_vehicle_type;
DROP INDEX IF EXISTS idx_vehicles_status;
DROP INDEX IF EXISTS idx_vehicles_license_plate;
DROP INDEX IF EXISTS idx_vehicles_user_id;
DROP TABLE IF EXISTS vehicles;

