-- =============================================================================
-- seed_venues.sql — 16 estadios sede del Mundial 2026
-- Ejecutar contra venue_db después de correr las migraciones.
-- Uso: psql -U wc2026 -d venue_db -f seed_venues.sql
--
-- La migración 003_postgis_indexes.sql ya insertó venues SIN UUIDs fijos.
-- Este script los reemplaza con UUIDs predecibles para testing y Postman.
-- =============================================================================

-- Limpiar datos previos (cascade elimina venue_distances también)
TRUNCATE TABLE venue_distances;
TRUNCATE TABLE venues RESTART IDENTITY CASCADE;

INSERT INTO venues (id, name, city, country, country_code, capacity, surface, location, timezone, altitude_m)
VALUES

-- ═══════════════════════════════════════════════════════════════════════════
-- ESTADOS UNIDOS — 11 estadios
-- ═══════════════════════════════════════════════════════════════════════════

(
    '00000000-0000-0000-0002-000000000001',
    'MetLife Stadium', 'East Rutherford', 'USA', 'USA',
    82500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-74.0745, 40.8135), 4326),
    'America/New_York', 3
),
(
    '00000000-0000-0000-0002-000000000002',
    'AT&T Stadium', 'Arlington', 'USA', 'USA',
    80000, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-97.0931, 32.7473), 4326),
    'America/Chicago', 187
),
(
    '00000000-0000-0000-0002-000000000003',
    'SoFi Stadium', 'Inglewood', 'USA', 'USA',
    70240, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-118.3394, 33.9534), 4326),
    'America/Los_Angeles', 30
),
(
    '00000000-0000-0000-0002-000000000004',
    'Levi''s Stadium', 'Santa Clara', 'USA', 'USA',
    68500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-121.9697, 37.4033), 4326),
    'America/Los_Angeles', 7
),
(
    '00000000-0000-0000-0002-000000000005',
    'Arrowhead Stadium', 'Kansas City', 'USA', 'USA',
    76416, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-94.4839, 39.0489), 4326),
    'America/Chicago', 315
),
(
    '00000000-0000-0000-0002-000000000006',
    'NRG Stadium', 'Houston', 'USA', 'USA',
    72220, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-95.4101, 29.6847), 4326),
    'America/Chicago', 12
),
(
    '00000000-0000-0000-0002-000000000007',
    'Lincoln Financial Field', 'Philadelphia', 'USA', 'USA',
    69796, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-75.1673, 39.9008), 4326),
    'America/New_York', 5
),
(
    '00000000-0000-0000-0002-000000000008',
    'Hard Rock Stadium', 'Miami Gardens', 'USA', 'USA',
    65326, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-80.2389, 25.9580), 4326),
    'America/New_York', 2
),
(
    '00000000-0000-0000-0002-000000000009',
    'Gillette Stadium', 'Foxborough', 'USA', 'USA',
    65878, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-71.2643, 42.0909), 4326),
    'America/New_York', 36
),
(
    '00000000-0000-0000-0002-000000000010',
    'Empower Field at Mile High', 'Denver', 'USA', 'USA',
    76125, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-105.0200, 39.7439), 4326),
    'America/Denver', 1609
),
(
    '00000000-0000-0000-0002-000000000011',
    'Lumen Field', 'Seattle', 'USA', 'USA',
    68740, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-122.3316, 47.5952), 4326),
    'America/Los_Angeles', 6
),

-- ═══════════════════════════════════════════════════════════════════════════
-- CANADÁ — 2 estadios
-- ═══════════════════════════════════════════════════════════════════════════

(
    '00000000-0000-0000-0002-000000000012',
    'BC Place', 'Vancouver', 'CAN', 'CAN',
    54500, 'synthetic',
    ST_SetSRID(ST_MakePoint(-123.1116, 49.2768), 4326),
    'America/Vancouver', 5
),
(
    '00000000-0000-0000-0002-000000000013',
    'BMO Field', 'Toronto', 'CAN', 'CAN',
    45736, 'synthetic',
    ST_SetSRID(ST_MakePoint(-79.4183, 43.6333), 4326),
    'America/Toronto', 76
),

-- ═══════════════════════════════════════════════════════════════════════════
-- MÉXICO — 3 estadios
-- ═══════════════════════════════════════════════════════════════════════════

(
    '00000000-0000-0000-0002-000000000014',
    'Estadio Azteca', 'Ciudad de México', 'MEX', 'MEX',
    87523, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-99.1500, 19.3033), 4326),
    'America/Mexico_City', 2240
),
(
    '00000000-0000-0000-0002-000000000015',
    'Estadio Akron', 'Guadalajara', 'MEX', 'MEX',
    49850, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-103.4092, 20.6899), 4326),
    'America/Mexico_City', 1600
),
(
    '00000000-0000-0000-0002-000000000016',
    'Estadio BBVA', 'Monterrey', 'MEX', 'MEX',
    53500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-100.4464, 25.6694), 4326),
    'America/Monterrey', 540
);

-- ── Calcular matriz de distancias completa con PostGIS ────────────────────────
-- C(16,2) = 120 pares únicos
INSERT INTO venue_distances (from_venue_id, to_venue_id, distance_km)
SELECT
    v1.id,
    v2.id,
    ROUND(
        (ST_DistanceSphere(v1.location::geometry, v2.location::geometry) / 1000.0)::numeric,
        2
    )
FROM venues v1
CROSS JOIN venues v2
WHERE v1.id < v2.id;

-- ── Verificación ──────────────────────────────────────────────────────────────
DO $$
DECLARE
    total_venues   INT;
    usa_venues     INT;
    can_venues     INT;
    mex_venues     INT;
    total_pairs    INT;
    max_dist       NUMERIC;
    min_dist       NUMERIC;
    far_from       TEXT;
    far_to         TEXT;
BEGIN
    SELECT COUNT(*)                                    INTO total_venues FROM venues;
    SELECT COUNT(*) FILTER (WHERE country = 'USA')     INTO usa_venues   FROM venues;
    SELECT COUNT(*) FILTER (WHERE country = 'CAN')     INTO can_venues   FROM venues;
    SELECT COUNT(*) FILTER (WHERE country = 'MEX')     INTO mex_venues   FROM venues;
    SELECT COUNT(*)                                    INTO total_pairs  FROM venue_distances;

    SELECT d.distance_km, v1.name, v2.name
    INTO max_dist, far_from, far_to
    FROM venue_distances d
    JOIN venues v1 ON v1.id = d.from_venue_id
    JOIN venues v2 ON v2.id = d.to_venue_id
    ORDER BY d.distance_km DESC LIMIT 1;

    SELECT MIN(distance_km) INTO min_dist FROM venue_distances;

    RAISE NOTICE '=== Verificación seed_venues ===';
    RAISE NOTICE 'Total venues     : %  (esperado: 16)',  total_venues;
    RAISE NOTICE 'USA              : %  (esperado: 11)',  usa_venues;
    RAISE NOTICE 'Canadá           : %  (esperado: 2)',   can_venues;
    RAISE NOTICE 'México           : %  (esperado: 3)',   mex_venues;
    RAISE NOTICE 'Pares distancias : %  (esperado: 120)', total_pairs;
    RAISE NOTICE 'Par más lejano   : % ↔ % (% km)', far_from, far_to, max_dist;
    RAISE NOTICE 'Distancia mínima : % km', min_dist;

    IF total_venues != 16 THEN
        RAISE EXCEPTION 'Error: se esperaban 16 venues, se insertaron %', total_venues;
    END IF;
    IF total_pairs != 120 THEN
        RAISE EXCEPTION 'Error: se esperaban 120 pares de distancias, se calcularon %', total_pairs;
    END IF;

    RAISE NOTICE '✅ seed_venues completado correctamente.';
END $$;
