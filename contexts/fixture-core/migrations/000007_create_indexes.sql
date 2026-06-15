-- ============================================================================
-- 007_create_indexes.sql
-- Índices de rendimiento para los patrones de acceso más frecuentes.
-- Separados en su propia migración para poder añadir/eliminar sin
-- reconstruir las tablas.
-- ============================================================================

-- ── tournaments ───────────────────────────────────────────────────────────────

-- Búsqueda por estado (ej: todos los torneos en curso)
CREATE INDEX IF NOT EXISTS idx_tournaments_status
    ON tournaments (status)
    WHERE status != 'FINISHED';

-- ── groups ────────────────────────────────────────────────────────────────────

-- Grupos por torneo — el acceso más frecuente
CREATE INDEX IF NOT EXISTS idx_groups_tournament_id
    ON groups (tournament_id);

-- Grupos incompletos de un torneo — usado para verificar si la fase de grupos terminó
CREATE INDEX IF NOT EXISTS idx_groups_tournament_status
    ON groups (tournament_id, status)
    WHERE status != 'COMPLETED';

-- ── group_teams ───────────────────────────────────────────────────────────────

-- Buscar en qué grupo está un equipo
CREATE INDEX IF NOT EXISTS idx_group_teams_team_id
    ON group_teams (team_id);

-- ── matches ───────────────────────────────────────────────────────────────────

-- Partidos por torneo y fase — el filtro más común en ListMatches
CREATE INDEX IF NOT EXISTS idx_matches_tournament_phase
    ON matches (tournament_id, phase);

-- Partidos de un grupo específico
CREATE INDEX IF NOT EXISTS idx_matches_group_id
    ON matches (group_id)
    WHERE group_id IS NOT NULL;

-- Partidos pendientes — para saber cuántos faltan en un torneo
CREATE INDEX IF NOT EXISTS idx_matches_tournament_status
    ON matches (tournament_id, status)
    WHERE status != 'COMPLETED';

-- Partidos por fecha — para agenda del día
CREATE INDEX IF NOT EXISTS idx_matches_scheduled_at
    ON matches (scheduled_at DESC);

-- Partidos por venue — para queries geoespaciales cruzadas
CREATE INDEX IF NOT EXISTS idx_matches_venue_id
    ON matches (venue_id);

-- Búsqueda de equipo en slots JSONB (local o visitante)
-- Usado en el filtro TeamID de ListMatches
CREATE INDEX IF NOT EXISTS idx_matches_home_slot_team
    ON matches ((home_slot->>'team_id'))
    WHERE home_slot->>'kind' = 'TEAM';

CREATE INDEX IF NOT EXISTS idx_matches_away_slot_team
    ON matches ((away_slot->>'team_id'))
    WHERE away_slot->>'kind' = 'TEAM';

-- Partidos eliminatorios con parent — para navegar el bracket
CREATE INDEX IF NOT EXISTS idx_matches_parent_home_id
    ON matches (parent_home_id)
    WHERE parent_home_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_matches_parent_away_id
    ON matches (parent_away_id)
    WHERE parent_away_id IS NOT NULL;

-- ── match_results ─────────────────────────────────────────────────────────────

-- Resultados por equipo — para estadísticas individuales
CREATE INDEX IF NOT EXISTS idx_match_results_home_team
    ON match_results (home_team_id);

CREATE INDEX IF NOT EXISTS idx_match_results_away_team
    ON match_results (away_team_id);

-- Resultados por fecha de completado — para cronología de resultados
CREATE INDEX IF NOT EXISTS idx_match_results_completed_at
    ON match_results (completed_at DESC);

-- ── group_standings ───────────────────────────────────────────────────────────

-- Standings ordenados por posición — el acceso más común
CREATE INDEX IF NOT EXISTS idx_group_standings_group_position
    ON group_standings (group_id, position);

-- Standings de terceros — para el cálculo de mejores terceros
CREATE INDEX IF NOT EXISTS idx_group_standings_third_place
    ON group_standings (group_id, points DESC, goal_difference DESC, goals_for DESC)
    WHERE position = 3;

-- ── fixture_events ────────────────────────────────────────────────────────────

-- Carga de eventos por torneo ordenada — usado en LoadEvents()
CREATE INDEX IF NOT EXISTS idx_fixture_events_tournament_version
    ON fixture_events (tournament_id, aggregate_version ASC);

-- Búsqueda de eventos por tipo — para auditoría y debugging
CREATE INDEX IF NOT EXISTS idx_fixture_events_event_type
    ON fixture_events (event_type, occurred_at DESC);

-- Carga desde una versión específica — usado en LoadEventsFrom()
CREATE INDEX IF NOT EXISTS idx_fixture_events_tournament_from_version
    ON fixture_events (tournament_id, aggregate_version)
    WHERE aggregate_version > 0;

-- ============================================================================
-- Estadísticas de índices — ejecutar luego de cargar datos iniciales
-- ANALYZE tournaments, groups, group_teams, matches, match_results,
--         group_standings, fixture_events;
-- ============================================================================
