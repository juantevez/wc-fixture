-- ============================================================================
-- 001_create_teams.sql
-- Equipos nacionales participantes del Mundial 2026.
-- ============================================================================

CREATE TABLE IF NOT EXISTS teams (
    id                 UUID        NOT NULL DEFAULT gen_random_uuid(),
    name               TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    short_name         CHAR(3)     NOT NULL,  -- código FIFA de 3 letras (ARG, FRA, etc.)
    country_code       CHAR(3)     NOT NULL,  -- ISO 3166-1 alpha-3
    confederation      TEXT        NOT NULL
                       CHECK (confederation IN ('UEFA','CONMEBOL','CONCACAF','CAF','AFC','OFC')),
    fifa_ranking_date  INT         NOT NULL CHECK (fifa_ranking_date > 0),
    flag_url           TEXT        NOT NULL DEFAULT '',
    qualified          BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT teams_pkey       PRIMARY KEY (id),
    CONSTRAINT teams_short_name UNIQUE (UPPER(short_name)),
    CONSTRAINT teams_country    UNIQUE (country_code)
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Índices de acceso frecuente
CREATE INDEX IF NOT EXISTS idx_teams_confederation
    ON teams (confederation);

CREATE INDEX IF NOT EXISTS idx_teams_qualified
    ON teams (qualified)
    WHERE qualified = TRUE;

CREATE INDEX IF NOT EXISTS idx_teams_ranking
    ON teams (confederation, fifa_ranking_date ASC);

COMMENT ON TABLE  teams                 IS 'Equipos nacionales participantes del Mundial 2026';
COMMENT ON COLUMN teams.short_name      IS 'Código FIFA de 3 letras: ARG, FRA, BRA, etc.';
COMMENT ON COLUMN teams.fifa_ranking_date IS 'Ranking FIFA al momento del sorteo (para desempate)';
COMMENT ON COLUMN teams.qualified       IS 'TRUE = clasificado al Mundial 2026';
