-- ============================================================================
-- 001_create_venues.sql
-- Estadios sede del torneo con soporte geoespacial PostGIS.
-- ============================================================================

-- Habilitar la extensión PostGIS (si no está habilitada a nivel de cluster)
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS venues (
    id           UUID         NOT NULL DEFAULT gen_random_uuid(),
    name         TEXT         NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    city         TEXT         NOT NULL,
    country      TEXT         NOT NULL CHECK (country IN ('USA', 'CAN', 'MEX')),
    country_code CHAR(3)      NOT NULL,
    capacity     INT          NOT NULL CHECK (capacity > 0),
    surface      TEXT         NOT NULL DEFAULT 'natural_grass'
                              CHECK (surface IN ('natural_grass', 'synthetic')),
    -- GEOGRAPHY(Point, 4326): usa WGS84 y permite cálculos geodésicos precisos
    -- con ST_DWithin y ST_DistanceSphere sin conversión manual de unidades.
    location     GEOGRAPHY(Point, 4326) NOT NULL,
    timezone     TEXT         NOT NULL,     -- IANA timezone (America/New_York, etc.)
    altitude_m   INT          NOT NULL DEFAULT 0 CHECK (altitude_m >= 0),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT venues_pkey      PRIMARY KEY (id),
    CONSTRAINT venues_name_city UNIQUE (name, city)
);

-- Índice espacial GIST — obligatorio para performance de ST_DWithin y ST_Distance
-- Sin este índice las queries geoespaciales hacen sequential scan sobre todos los venues
CREATE INDEX IF NOT EXISTS idx_venues_location
    ON venues USING GIST (location);

-- Índice para filtro por país — el filtro más común en ListVenues
CREATE INDEX IF NOT EXISTS idx_venues_country
    ON venues (country);

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_venues_updated_at
    BEFORE UPDATE ON venues
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  venues            IS 'Estadios sede del Mundial 2026 (16 venues en USA, CAN y MEX)';
COMMENT ON COLUMN venues.location   IS 'Coordenada geográfica WGS84 (GEOGRAPHY para cálculos geodésicos)';
COMMENT ON COLUMN venues.timezone   IS 'IANA timezone string para conversión de horarios';
COMMENT ON COLUMN venues.altitude_m IS 'Altitud sobre el mar — relevante para Ciudad de México (2240m)';
