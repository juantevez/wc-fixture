-- ============================================================================
-- 003_postgis_indexes.sql
-- Índices adicionales y datos semilla de los 16 estadios sede del Mundial 2026.
-- ============================================================================

-- ── Índice de distancias por venue origen ─────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_venue_distances_from
    ON venue_distances (from_venue_id, distance_km ASC);

CREATE INDEX IF NOT EXISTS idx_venue_distances_to
    ON venue_distances (to_venue_id, distance_km ASC);

-- ── Seed: los 16 estadios sede del Mundial 2026 ───────────────────────────────
-- ST_MakePoint(longitud, latitud) — orden lon/lat en PostGIS
-- Coordenadas aproximadas de los estadios confirmados por FIFA

INSERT INTO venues (name, city, country, country_code, capacity, surface, location, timezone, altitude_m)
VALUES
-- ── Estados Unidos (11 estadios) ─────────────────────────────────────────────
('MetLife Stadium',         'East Rutherford', 'USA', 'USA', 82500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-74.0745, 40.8135), 4326), 'America/New_York', 3),

('AT&T Stadium',            'Arlington',        'USA', 'USA', 80000, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-97.0931, 32.7473), 4326), 'America/Chicago', 187),

('SoFi Stadium',            'Inglewood',        'USA', 'USA', 70240, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-118.3394, 33.9534), 4326), 'America/Los_Angeles', 30),

('Levi''s Stadium',         'Santa Clara',      'USA', 'USA', 68500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-121.9697, 37.4033), 4326), 'America/Los_Angeles', 7),

('Arrowhead Stadium',       'Kansas City',      'USA', 'USA', 76416, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-94.4839, 39.0489), 4326), 'America/Chicago', 315),

('NRG Stadium',             'Houston',          'USA', 'USA', 72220, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-95.4101, 29.6847), 4326), 'America/Chicago', 12),

('Lincoln Financial Field',  'Philadelphia',    'USA', 'USA', 69796, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-75.1673, 39.9008), 4326), 'America/New_York', 5),

('Hard Rock Stadium',       'Miami Gardens',    'USA', 'USA', 65326, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-80.2389, 25.9580), 4326), 'America/New_York', 2),

('Gillette Stadium',        'Foxborough',       'USA', 'USA', 65878, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-71.2643, 42.0909), 4326), 'America/New_York', 36),

('Empower Field at Mile High', 'Denver',        'USA', 'USA', 76125, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-105.0200, 39.7439), 4326), 'America/Denver', 1609),

('Seattle''s CenturyLink Field', 'Seattle',     'USA', 'USA', 68740, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-122.3316, 47.5952), 4326), 'America/Los_Angeles', 6),

-- ── Canadá (2 estadios) ───────────────────────────────────────────────────────
('BC Place',                'Vancouver',        'CAN', 'CAN', 54500, 'synthetic',
    ST_SetSRID(ST_MakePoint(-123.1116, 49.2768), 4326), 'America/Vancouver', 5),

('BMO Field',               'Toronto',          'CAN', 'CAN', 45736, 'synthetic',
    ST_SetSRID(ST_MakePoint(-79.4183, 43.6333), 4326), 'America/Toronto', 76),

-- ── México (3 estadios) ───────────────────────────────────────────────────────
('Estadio Azteca',          'Ciudad de México', 'MEX', 'MEX', 87523, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-99.1500, 19.3033), 4326), 'America/Mexico_City', 2240),

('Estadio Akron',           'Guadalajara',      'MEX', 'MEX', 49850, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-103.4092, 20.6899), 4326), 'America/Mexico_City', 1600),

('Estadio BBVA',            'Monterrey',        'MEX', 'MEX', 53500, 'natural_grass',
    ST_SetSRID(ST_MakePoint(-100.4464, 25.6694), 4326), 'America/Monterrey', 540)

ON CONFLICT (name, city) DO NOTHING;
