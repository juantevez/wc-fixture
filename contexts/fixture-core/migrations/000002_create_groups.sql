-- ============================================================================
-- 002_create_groups.sql
-- Grupos del torneo (A–L) y asignación de equipos a grupos.
-- ============================================================================

CREATE TABLE IF NOT EXISTS groups (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    tournament_id UUID        NOT NULL,
    name          CHAR(1)     NOT NULL CHECK (name IN ('A','B','C','D','E','F','G','H','I','J','K','L')),
    status        TEXT        NOT NULL DEFAULT 'PENDING'
                              CHECK (status IN ('PENDING','IN_PROGRESS','COMPLETED')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT groups_pkey               PRIMARY KEY (id),
    CONSTRAINT groups_tournament_id_name UNIQUE      (tournament_id, name),
    CONSTRAINT groups_tournament_fk      FOREIGN KEY (tournament_id)
                                         REFERENCES tournaments(tournament_id)
                                         ON DELETE CASCADE
);

CREATE TRIGGER trg_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Asignación de equipos a grupos ───────────────────────────────────────────
-- Un equipo pertenece exactamente a un grupo por torneo.
-- seeding_pot: bombo del sorteo (1–4), determina el rol del equipo en el grupo.

CREATE TABLE IF NOT EXISTS group_teams (
    group_id    UUID NOT NULL,
    team_id     UUID NOT NULL,
    seeding_pot INT  NOT NULL CHECK (seeding_pot BETWEEN 1 AND 4),

    CONSTRAINT group_teams_pkey     PRIMARY KEY (group_id, team_id),
    CONSTRAINT group_teams_group_fk FOREIGN KEY (group_id)
                                    REFERENCES groups(id)
                                    ON DELETE CASCADE
);

-- Un equipo no puede estar en dos grupos del mismo torneo
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_teams_unique_team_per_tournament
    ON group_teams (team_id, group_id);

COMMENT ON TABLE  groups            IS '12 grupos (A-L) del torneo';
COMMENT ON COLUMN groups.name       IS 'Letra del grupo: A a L';
COMMENT ON COLUMN groups.status     IS 'Estado del grupo dentro de la fase de grupos';
COMMENT ON TABLE  group_teams       IS 'Asignación de equipos a grupos (resultado del sorteo)';
COMMENT ON COLUMN group_teams.seeding_pot IS 'Bombo del sorteo: 1=cabeza de serie, 4=último bombo';
