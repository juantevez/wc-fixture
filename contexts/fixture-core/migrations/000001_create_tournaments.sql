-- ============================================================================
-- 001_create_tournaments.sql
-- Tabla principal del torneo — snapshot del estado del aggregate Fixture.
-- ============================================================================

CREATE TABLE IF NOT EXISTS tournaments (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    tournament_id UUID        NOT NULL,
    edition       INT         NOT NULL CHECK (edition >= 1930),
    name          TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    status        TEXT        NOT NULL DEFAULT 'DRAFT'
                              CHECK (status IN ('DRAFT','GROUP_STAGE','KNOCKOUT','FINISHED')),
    config        JSONB       NOT NULL DEFAULT '{}',
    version       BIGINT      NOT NULL DEFAULT 0 CHECK (version >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tournaments_pkey          PRIMARY KEY (id),
    CONSTRAINT tournaments_tournament_id UNIQUE      (tournament_id)
);

-- Trigger para mantener updated_at sincronizado automáticamente
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tournaments_updated_at
    BEFORE UPDATE ON tournaments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  tournaments              IS 'Snapshot del aggregate Fixture por torneo';
COMMENT ON COLUMN tournaments.tournament_id IS 'ID de negocio del torneo (el que usan los demás contextos)';
COMMENT ON COLUMN tournaments.version       IS 'Versión para optimistic locking del aggregate';
COMMENT ON COLUMN tournaments.config        IS 'Configuración extendida del torneo (BestThirdsPolicy, etc.)';
