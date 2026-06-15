-- ============================================================================
-- 002_create_venue_distances.sql
-- Cache de distancias geodésicas entre pares de venues.
-- Se precalcula con PostGIS al inicializar el torneo y se actualiza
-- si se agregan venues. Solo almacena pares únicos (from_id < to_id).
-- ============================================================================

CREATE TABLE IF NOT EXISTS venue_distances (
    from_venue_id UUID           NOT NULL,
    to_venue_id   UUID           NOT NULL,
    distance_km   NUMERIC(8, 2)  NOT NULL CHECK (distance_km > 0),

    CONSTRAINT venue_distances_pkey     PRIMARY KEY (from_venue_id, to_venue_id),
    CONSTRAINT venue_distances_from_fk  FOREIGN KEY (from_venue_id)
                                        REFERENCES venues(id) ON DELETE CASCADE,
    CONSTRAINT venue_distances_to_fk    FOREIGN KEY (to_venue_id)
                                        REFERENCES venues(id) ON DELETE CASCADE,
    -- Garantizar que solo guardamos pares canónicos (evita duplicar A→B y B→A)
    CONSTRAINT venue_distances_canonical CHECK (from_venue_id < to_venue_id)
);

-- Vista simétrica que expone ambas direcciones para queries sin ordenar los IDs
CREATE OR REPLACE VIEW venue_distances_symmetric AS
    SELECT from_venue_id, to_venue_id, distance_km FROM venue_distances
    UNION ALL
    SELECT to_venue_id AS from_venue_id, from_venue_id AS to_venue_id, distance_km
    FROM venue_distances;

COMMENT ON TABLE venue_distances IS
    'Cache precalculado de distancias geodésicas entre pares de venues. '
    'Solo contiene pares canónicos (from_id < to_id). '
    'Reconstruir con DistanceCacheBuilder.Rebuild() al agregar venues.';

COMMENT ON VIEW venue_distances_symmetric IS
    'Vista que expone ambas direcciones de cada par de distancias. '
    'Usar para queries donde el orden from/to no está garantizado.';
