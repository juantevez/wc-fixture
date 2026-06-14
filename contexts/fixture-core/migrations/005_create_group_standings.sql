-- ============================================================================
-- 005_create_group_standings.sql
-- Tabla de posiciones materializada por grupo.
-- Se actualiza via upsert en cada Save() del aggregate Fixture.
-- ============================================================================

CREATE TABLE IF NOT EXISTS group_standings (
    group_id      UUID        NOT NULL,
    team_id       UUID        NOT NULL,
    position      INT         NOT NULL CHECK (position BETWEEN 1 AND 4),
    played        INT         NOT NULL DEFAULT 0 CHECK (played >= 0),
    won           INT         NOT NULL DEFAULT 0 CHECK (won >= 0),
    drawn         INT         NOT NULL DEFAULT 0 CHECK (drawn >= 0),
    lost          INT         NOT NULL DEFAULT 0 CHECK (lost >= 0),
    goals_for     INT         NOT NULL DEFAULT 0 CHECK (goals_for >= 0),
    goals_against INT         NOT NULL DEFAULT 0 CHECK (goals_against >= 0),
    yellow_cards  INT         NOT NULL DEFAULT 0 CHECK (yellow_cards >= 0),
    red_cards     INT         NOT NULL DEFAULT 0 CHECK (red_cards >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Columnas generadas para facilitar queries sin cálculo en app
    goal_difference INT GENERATED ALWAYS AS (goals_for - goals_against) STORED,
    points          INT GENERATED ALWAYS AS (won * 3 + drawn)           STORED,
    fair_play_pts   INT GENERATED ALWAYS AS (yellow_cards + red_cards * 3) STORED,

    CONSTRAINT group_standings_pkey     PRIMARY KEY (group_id, team_id),
    CONSTRAINT group_standings_group_fk FOREIGN KEY (group_id)
                                        REFERENCES groups(id)
                                        ON DELETE CASCADE,

    -- Integridad: partidos jugados = won + drawn + lost
    CONSTRAINT group_standings_played_check CHECK (played = won + drawn + lost),

    -- Posición única dentro del grupo
    CONSTRAINT group_standings_position_unique UNIQUE (group_id, position)
);

-- Vista para consultas de mejor tercero con ranking en una sola query
CREATE OR REPLACE VIEW best_thirds_ranked AS
SELECT
    g.tournament_id,
    g.name                                                  AS group_name,
    gs.team_id,
    gs.position,
    gs.points,
    gs.played,
    gs.won,
    gs.drawn,
    gs.lost,
    gs.goals_for,
    gs.goals_against,
    gs.goal_difference,
    gs.fair_play_pts,
    ROW_NUMBER() OVER (
        PARTITION BY g.tournament_id
        ORDER BY
            gs.points          DESC,
            gs.goal_difference DESC,
            gs.goals_for       DESC,
            gs.fair_play_pts   ASC
    )                                                       AS rank,
    ROW_NUMBER() OVER (
        PARTITION BY g.tournament_id
        ORDER BY
            gs.points          DESC,
            gs.goal_difference DESC,
            gs.goals_for       DESC,
            gs.fair_play_pts   ASC
    ) <= 8                                                  AS classified
FROM group_standings gs
JOIN groups g ON g.id = gs.group_id
WHERE gs.position = 3;

COMMENT ON TABLE  group_standings               IS 'Tabla de posiciones materializada — se recalcula con cada resultado';
COMMENT ON COLUMN group_standings.goal_difference IS 'Columna generada: goals_for - goals_against';
COMMENT ON COLUMN group_standings.points          IS 'Columna generada: won*3 + drawn';
COMMENT ON COLUMN group_standings.fair_play_pts   IS 'Columna generada: amarillas + rojas*3 (menor es mejor)';
COMMENT ON VIEW   best_thirds_ranked              IS 'Vista con ranking de mejores terceros por torneo';
