-- Rollback: 010_create_geo_pricing

DROP INDEX IF EXISTS idx_surge_zones_service_area_id;
DROP INDEX IF EXISTS idx_surge_zones_center;
DROP TABLE IF EXISTS surge_zones;

DROP INDEX IF EXISTS idx_rate_cards_is_active;
DROP INDEX IF EXISTS idx_rate_cards_service_area_id;
DROP INDEX IF EXISTS idx_rate_cards_ride_type;
DROP TABLE IF EXISTS rate_cards;

DROP INDEX IF EXISTS idx_geofences_center;
DROP INDEX IF EXISTS idx_geofences_is_active;
DROP INDEX IF EXISTS idx_geofences_city;
DROP INDEX IF EXISTS idx_geofences_type;
DROP TABLE IF EXISTS geofences;

DROP INDEX IF EXISTS idx_service_areas_center;
DROP INDEX IF EXISTS idx_service_areas_is_active;
DROP INDEX IF EXISTS idx_service_areas_city;
DROP TABLE IF EXISTS service_areas;

