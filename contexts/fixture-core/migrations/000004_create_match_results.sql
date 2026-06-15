-- ============================================================================
-- 004_create_match_results.sql
-- Resultados de partidos — separados de matches para normalización.
-- Relación 1:1 con matches: un partido tiene como máximo un resultado.
-- ============================================================================

CREATE TABLE IF NOT EXISTS match_results (
    id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    match_id       UUID        NOT NULL,
    home_team_id   UUID        NOT NULL,
    away_team_id   UUID        NOT NULL,

    -- Tiempo regular (obligatorio)
    home_goals     INT         NOT NULL CHECK (home_goals >= 0),
    away_goals     INT         NOT NULL CHECK (away_goals >= 0),

    -- Tiempo extra (solo eliminatorias con empate en 90')
    home_goals_et  INT                  CHECK (home_goals_et >= 0),
    away_goals_et  INT                  CHECK (away_goals_et >= 0),

    -- Penales (solo cuando hay empate en tiempo extra)
    home_goals_pen INT                  CHECK (home_goals_pen >= 0),
    away_goals_pen INT                  CHECK (away_goals_pen >= 0),

    -- Ganador precalculado para facilitar queries de standings
    -- NULL solo en empates de fase de grupos
    winner_id      UUID,

    registered_by  TEXT        NOT NULL DEFAULT 'system',
    completed_at   TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT match_results_pkey      PRIMARY KEY (id),
    CONSTRAINT match_results_match_uk  UNIQUE      (match_id),   -- 1:1 con matches
    CONSTRAINT match_results_match_fk  FOREIGN KEY (match_id)
                                       REFERENCES matches(id)
                                       ON DELETE CASCADE,
    CONSTRAINT match_results_teams_diff CHECK (home_team_id != away_team_id),

    -- Si hay penales, debe haber tiempo extra
    CONSTRAINT match_results_pen_requires_et CHECK (
        (home_goals_pen IS NULL AND away_goals_pen IS NULL)
        OR
        (home_goals_et IS NOT NULL AND away_goals_et IS NOT NULL)
    ),
    -- Los penales no pueden terminar empatados
    CONSTRAINT match_results_pen_no_draw CHECK (
        home_goals_pen IS NULL
        OR away_goals_pen IS NULL
        OR home_goals_pen != away_goals_pen
    ),
    -- Si hay ET, ambos valores deben estar presentes
    CONSTRAINT match_results_et_both_or_none CHECK (
        (home_goals_et IS NULL) = (away_goals_et IS NULL)
    ),
    -- Si hay penales, ambos valores deben estar presentes
    CONSTRAINT match_results_pen_both_or_none CHECK (
        (home_goals_pen IS NULL) = (away_goals_pen IS NULL)
    )
);

COMMENT ON TABLE  match_results               IS 'Resultados de partidos — relación 1:1 con matches';
COMMENT ON COLUMN match_results.winner_id     IS 'Equipo ganador precalculado; NULL en empates de grupos';
COMMENT ON COLUMN match_results.registered_by IS 'Fuente del resultado: fifa_api, manual, result-ingestion';
COMMENT ON COLUMN match_results.home_goals_et IS 'Goles en tiempo extra (ambos NULL si no hubo ET)';
COMMENT ON COLUMN match_results.home_goals_pen IS 'Goles en penales (ambos NULL si no hubo tanda)';
