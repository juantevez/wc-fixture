-- ============================================================================
-- 002_create_confederations.sql
-- Confederaciones FIFA con seed de datos oficiales y cupos WC2026.
-- ============================================================================

CREATE TABLE IF NOT EXISTS confederations (
    code       CHAR(8)     NOT NULL,
    name       TEXT        NOT NULL,
    short_name TEXT        NOT NULL,
    region     TEXT        NOT NULL,
    wc2026_slots INT       NOT NULL DEFAULT 0 CHECK (wc2026_slots >= 0),

    CONSTRAINT confederations_pkey PRIMARY KEY (code)
);

COMMENT ON TABLE confederations IS 'Confederaciones FIFA con cupos asignados al Mundial 2026';
COMMENT ON COLUMN confederations.wc2026_slots IS 'Cupos directos asignados (sin contar plazas de playoff)';

-- Seed con los datos oficiales de las 6 confederaciones FIFA
-- Cupos WC2026: total 48 equipos
--   UEFA:     16  (Europa)
--   CAF:       9  (África)
--   AFC:       8  (Asia)
--   CONMEBOL:  6  (Sudamérica)
--   CONCACAF:  6  (N/C América y Caribe)
--   OFC:       1  (Oceanía)
--   Playoff:   2  (inter-confederaciones)

INSERT INTO confederations (code, name, short_name, region, wc2026_slots)
VALUES
    ('UEFA',     'Union of European Football Associations',                                        'UEFA',     'Europe',                            16),
    ('CAF',      'Confederation of African Football',                                              'CAF',      'Africa',                             9),
    ('AFC',      'Asian Football Confederation',                                                   'AFC',      'Asia',                               8),
    ('CONMEBOL', 'Confederación Sudamericana de Fútbol',                                           'CONMEBOL', 'South America',                      6),
    ('CONCACAF', 'Confederation of North, Central America and Caribbean Association Football',     'CONCACAF', 'North/Central America & Caribbean',  6),
    ('OFC',      'Oceania Football Confederation',                                                 'OFC',      'Oceania',                            1)
ON CONFLICT (code) DO UPDATE SET
    name         = EXCLUDED.name,
    short_name   = EXCLUDED.short_name,
    region       = EXCLUDED.region,
    wc2026_slots = EXCLUDED.wc2026_slots;
