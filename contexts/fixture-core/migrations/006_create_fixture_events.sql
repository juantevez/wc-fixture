-- ============================================================================
-- 006_create_fixture_events.sql
-- Event store del aggregate Fixture — append-only, fuente de verdad.
-- Cada fila representa un domain event emitido durante un comando.
-- ============================================================================

CREATE TABLE IF NOT EXISTS fixture_events (
    id                BIGSERIAL   NOT NULL,
    tournament_id     UUID        NOT NULL,
    event_type        TEXT        NOT NULL CHECK (char_length(event_type) > 0),
    event_version     INT         NOT NULL DEFAULT 1 CHECK (event_version >= 1),
    payload           JSONB       NOT NULL,
    occurred_at       TIMESTAMPTZ NOT NULL,
    aggregate_version BIGINT      NOT NULL CHECK (aggregate_version > 0),

    CONSTRAINT fixture_events_pkey    PRIMARY KEY (id),
    CONSTRAINT fixture_events_tourn_fk FOREIGN KEY (tournament_id)
                                        REFERENCES tournaments(tournament_id)
                                        ON DELETE CASCADE,

    -- Garantía de orden sin gaps dentro del mismo aggregate
    CONSTRAINT fixture_events_version_unique UNIQUE (tournament_id, aggregate_version)
);

-- Tipos válidos de eventos — útil para auditoría y debugging
COMMENT ON COLUMN fixture_events.event_type IS
    'Valores válidos: TournamentInitialized, MatchResultRegistered, '
    'GroupStageCompleted, KnockoutBracketGenerated, KnockoutMatchAdvanced, '
    'MatchScheduleUpdated, TournamentFinished';

COMMENT ON COLUMN fixture_events.aggregate_version IS
    'Versión del aggregate al momento del evento — monotónica sin gaps por torneo';

COMMENT ON COLUMN fixture_events.payload IS
    'Payload JSON del evento — schema específico por event_type';

COMMENT ON TABLE fixture_events IS
    'Event store append-only del aggregate Fixture. '
    'Nunca se hace UPDATE ni DELETE en esta tabla.';

-- Función para prevenir UPDATE y DELETE en el event store
CREATE OR REPLACE FUNCTION prevent_event_store_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION
        'fixture_events es append-only: UPDATE y DELETE están prohibidos. '
        'event_id=%, tournament_id=%', OLD.id, OLD.tournament_id;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_fixture_events_immutable_update
    BEFORE UPDATE ON fixture_events
    FOR EACH ROW EXECUTE FUNCTION prevent_event_store_mutation();

CREATE TRIGGER trg_fixture_events_immutable_delete
    BEFORE DELETE ON fixture_events
    FOR EACH ROW EXECUTE FUNCTION prevent_event_store_mutation();
