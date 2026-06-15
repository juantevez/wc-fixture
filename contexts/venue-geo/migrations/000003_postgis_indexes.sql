-- ============================================================================
-- 003_postgis_indexes.sql
-- Índices adicionales de PostGIS para venue-geo.
-- Los datos de venues se insertan via deploy/postgres/seed_venues.sql
-- para garantizar UUIDs fijos y consistencia entre entornos.
-- ============================================================================

-- Índice de distancias por venue origen
CREATE INDEX IF NOT EXISTS idx_venue_distances_from
    ON venue_distances (from_venue_id, distance_km ASC);

CREATE INDEX IF NOT EXISTS idx_venue_distances_to
    ON venue_distances (to_venue_id, distance_km ASC);
