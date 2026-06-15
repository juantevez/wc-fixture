-- ============================================================================
-- 003_create_matches.sql
-- Partidos del torneo: fase de grupos y eliminatorias.
-- Los slots se almacenan como JSONB para soportar referencias dinámicas
-- (SlotKindWinnerOf, SlotKindBestThird) antes de que se resuelvan.
-- ============================================================================

CREATE TABLE IF NOT EXISTS matches (
    id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    tournament_id  UUID        NOT NULL,
    group_id       UUID,                 -- NULL para partidos eliminatorios
    phase          TEXT        NOT NULL
                               CHECK (phase IN (
                                   'GROUP','ROUND_OF_32','QUARTERFINAL',
                                   'SEMIFINAL','THIRD_PLACE','FINAL'
                               )),
    match_number   INT         NOT NULL CHECK (match_number BETWEEN 1 AND 104),
    home_slot      JSONB       NOT NULL, -- { kind, team_id?, source_match_id?, group_ref? }
    away_slot      JSONB       NOT NULL,
    venue_id       UUID        NOT NULL, -- referencia a venue-geo (sin FK cross-servicio)
    scheduled_at   TIMESTAMPTZ NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'SCHEDULED'
                               CHECK (status IN ('SCHEDULED','IN_PROGRESS','COMPLETED','POSTPONED')),
    parent_home_id UUID,                 -- partido del que viene el local (eliminatorias)
    parent_away_id UUID,                 -- partido del que viene el visitante (eliminatorias)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT matches_pkey              PRIMARY KEY (id),
    CONSTRAINT matches_tournament_number UNIQUE      (tournament_id, match_number),
    CONSTRAINT matches_tournament_fk     FOREIGN KEY (tournament_id)
                                         REFERENCES tournaments(tournament_id)
                                         ON DELETE CASCADE,
    CONSTRAINT matches_group_fk          FOREIGN KEY (group_id)
                                         REFERENCES groups(id)
                                         ON DELETE SET NULL,
    CONSTRAINT matches_parent_home_fk    FOREIGN KEY (parent_home_id)
                                         REFERENCES matches(id),
    CONSTRAINT matches_parent_away_fk    FOREIGN KEY (parent_away_id)
                                         REFERENCES matches(id),
    -- Los partidos de grupo deben tener group_id; los eliminatorios no
    CONSTRAINT matches_group_phase_check CHECK (
        (phase = 'GROUP' AND group_id IS NOT NULL)
        OR
        (phase != 'GROUP' AND group_id IS NULL)
    )
);

CREATE TRIGGER trg_matches_updated_at
    BEFORE UPDATE ON matches
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Validación de home_slot / away_slot JSONB ─────────────────────────────────
-- Asegura que el campo 'kind' esté presente y sea un valor válido

ALTER TABLE matches
    ADD CONSTRAINT matches_home_slot_kind CHECK (
        home_slot->>'kind' IN ('TEAM','WINNER_OF','LOSER_OF','BEST_THIRD')
    ),
    ADD CONSTRAINT matches_away_slot_kind CHECK (
        away_slot->>'kind' IN ('TEAM','WINNER_OF','LOSER_OF','BEST_THIRD')
    );

COMMENT ON TABLE  matches              IS 'Partidos del torneo (grupos y eliminatorias)';
COMMENT ON COLUMN matches.home_slot    IS 'Slot local: equipo concreto o referencia dinámica (JSONB)';
COMMENT ON COLUMN matches.away_slot    IS 'Slot visitante: equipo concreto o referencia dinámica (JSONB)';
COMMENT ON COLUMN matches.venue_id     IS 'ID del estadio (referencia a venue-geo, sin FK cross-servicio)';
COMMENT ON COLUMN matches.parent_home_id IS 'Partido del que proviene el equipo local (solo eliminatorias)';
COMMENT ON COLUMN matches.parent_away_id IS 'Partido del que proviene el equipo visitante (solo eliminatorias)';
