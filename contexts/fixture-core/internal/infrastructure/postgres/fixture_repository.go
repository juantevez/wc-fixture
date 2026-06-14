package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
)

// fixtureRepo implementa ports.FixtureRepository usando PostgreSQL con
// event sourcing: persiste los domain events y reconstruye el aggregate
// reproduciendo el estado desde el event store + snapshot del estado actual.
//
// Estrategia de persistencia:
//   - fixture_events    → event store (append-only, fuente de verdad)
//   - tournaments       → estado actual del aggregate (snapshot optimizado)
//   - matches           → partidos desnormalizados para read models
//   - group_standings   → tabla de posiciones materializada
//
// La reconstrucción del aggregate lee el snapshot de tournaments y los
// matches/standings en lugar de reproducir todos los eventos — mucho más
// eficiente que event sourcing puro para un aggregate tan grande.
type fixtureRepo struct {
	pool       *pgxpool.Pool
	eventStore *EventStore
}

// Verificación en tiempo de compilación que fixtureRepo implementa la interfaz.
var _ ports.FixtureRepository = (*fixtureRepo)(nil)

func NewFixtureRepository(pool *pgxpool.Pool) ports.FixtureRepository {
	return &fixtureRepo{
		pool:       pool,
		eventStore: NewEventStore(pool),
	}
}

// ── GetByTournamentID ─────────────────────────────────────────────────────────

func (r *fixtureRepo) GetByTournamentID(ctx context.Context, tournamentID uuid.UUID) (*fixture.Fixture, error) {
	// 1. Cargar el snapshot del torneo
	f, err := r.loadTournamentSnapshot(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// 2. Cargar grupos con sus equipos
	groups, err := r.loadGroups(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	f.Groups = groups

	// 3. Cargar partidos de grupos y asignarlos a cada grupo
	if err := r.loadGroupMatches(ctx, tournamentID, f.Groups); err != nil {
		return nil, err
	}

	// 4. Cargar standings de cada grupo
	if err := r.loadGroupStandings(ctx, tournamentID, f.Groups); err != nil {
		return nil, err
	}

	// 5. Cargar rondas eliminatorias si el torneo está en fase knockout
	if f.Status == fixture.StatusKnockout || f.Status == fixture.StatusFinished {
		rounds, err := r.loadKnockoutRounds(ctx, tournamentID)
		if err != nil {
			return nil, err
		}
		f.KnockoutRounds = rounds
	}

	return f, nil
}

// loadTournamentSnapshot carga el estado base del torneo desde la tabla tournaments.
func (r *fixtureRepo) loadTournamentSnapshot(ctx context.Context, tournamentID uuid.UUID) (*fixture.Fixture, error) {
	const q = `
		SELECT id, tournament_id, edition, name, status, version
		FROM tournaments
		WHERE tournament_id = $1`

	var f fixture.Fixture
	var statusStr string

	err := r.pool.QueryRow(ctx, q, tournamentID).Scan(
		&f.ID, &f.TournamentID, &f.Edition, &f.Name, &statusStr, &f.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("torneo", tournamentID.String())
	}
	if err != nil {
		return nil, fmt.Errorf("fixture_repo: error cargando torneo: %w", err)
	}

	f.Status = fixture.TournamentStatus(statusStr)
	return &f, nil
}

// loadGroups carga los 12 grupos con sus equipos.
func (r *fixtureRepo) loadGroups(ctx context.Context, tournamentID uuid.UUID) ([]fixture.Group, error) {
	const q = `
		SELECT g.id, g.name, g.status,
			   array_agg(gt.team_id ORDER BY gt.seeding_pot) AS team_ids
		FROM groups g
		JOIN group_teams gt ON gt.group_id = g.id
		WHERE g.tournament_id = $1
		GROUP BY g.id, g.name, g.status
		ORDER BY g.name`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("fixture_repo: error cargando grupos: %w", err)
	}
	defer rows.Close()

	var groups []fixture.Group
	for rows.Next() {
		var g fixture.Group
		var statusStr string
		var teamIDs []uuid.UUID

		if err := rows.Scan(&g.ID, &g.Name, &statusStr, &teamIDs); err != nil {
			return nil, fmt.Errorf("fixture_repo: error escaneando grupo: %w", err)
		}

		g.Status = fixture.GroupStatus(statusStr)
		for i, t := range teamIDs {
			if i < 4 {
				g.Teams[i] = t
			}
		}
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

// loadGroupMatches carga los partidos de grupos y los asigna al grupo correspondiente.
func (r *fixtureRepo) loadGroupMatches(ctx context.Context, tournamentID uuid.UUID, groups []fixture.Group) error {
	const q = `
		SELECT m.id, m.group_id, m.match_number, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE m.tournament_id = $1
		  AND m.phase = 'GROUP'
		ORDER BY m.match_number`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return fmt.Errorf("fixture_repo: error cargando partidos de grupos: %w", err)
	}
	defer rows.Close()

	// Mapa groupID → índice en el slice groups para asignación O(1)
	groupIdx := make(map[uuid.UUID]int, len(groups))
	for i, g := range groups {
		groupIdx[g.ID] = i
	}

	for rows.Next() {
		m, groupID, err := r.scanMatch(rows)
		if err != nil {
			return err
		}
		if idx, ok := groupIdx[groupID]; ok {
			groups[idx].Matches = append(groups[idx].Matches, m)
		}
	}

	return rows.Err()
}

// loadGroupStandings carga los standings de todos los grupos.
func (r *fixtureRepo) loadGroupStandings(ctx context.Context, tournamentID uuid.UUID, groups []fixture.Group) error {
	const q = `
		SELECT gs.group_id, gs.team_id, gs.position,
			   gs.played, gs.won, gs.drawn, gs.lost,
			   gs.goals_for, gs.goals_against,
			   gs.yellow_cards, gs.red_cards
		FROM group_standings gs
		JOIN groups g ON g.id = gs.group_id
		WHERE g.tournament_id = $1
		ORDER BY gs.group_id, gs.position`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return fmt.Errorf("fixture_repo: error cargando standings: %w", err)
	}
	defer rows.Close()

	groupIdx := make(map[uuid.UUID]int, len(groups))
	for i, g := range groups {
		groupIdx[g.ID] = i
	}

	for rows.Next() {
		var s fixture.GroupStanding
		var groupID uuid.UUID

		if err := rows.Scan(
			&groupID, &s.TeamID, &s.Position,
			&s.Played, &s.Won, &s.Drawn, &s.Lost,
			&s.GoalsFor, &s.GoalsAgainst,
			&s.YellowCards, &s.RedCards,
		); err != nil {
			return fmt.Errorf("fixture_repo: error escaneando standing: %w", err)
		}

		if idx, ok := groupIdx[groupID]; ok {
			groups[idx].Standings = append(groups[idx].Standings, s)
		}
	}

	return rows.Err()
}

// loadKnockoutRounds carga las rondas eliminatorias con sus partidos.
func (r *fixtureRepo) loadKnockoutRounds(ctx context.Context, tournamentID uuid.UUID) ([]fixture.Round, error) {
	const q = `
		SELECT m.id, m.phase, m.match_number, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   m.parent_home_id, m.parent_away_id,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE m.tournament_id = $1
		  AND m.phase != 'GROUP'
		ORDER BY
			CASE m.phase
				WHEN 'ROUND_OF_32'   THEN 1
				WHEN 'QUARTERFINAL'  THEN 2
				WHEN 'SEMIFINAL'     THEN 3
				WHEN 'THIRD_PLACE'   THEN 4
				WHEN 'FINAL'         THEN 5
			END,
			m.match_number`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("fixture_repo: error cargando rondas eliminatorias: %w", err)
	}
	defer rows.Close()

	// Agrupar matches por phase manteniendo el orden
	phaseOrder := []fixture.MatchPhase{
		fixture.PhaseRoundOf32,
		fixture.PhaseQuarterfinal,
		fixture.PhaseSemifinal,
		fixture.PhaseThirdPlace,
		fixture.PhaseFinal,
	}
	byPhase := make(map[fixture.MatchPhase][]fixture.Match)

	for rows.Next() {
		var m fixture.Match
		var phaseStr string
		var homeSlotJSON, awaySlotJSON []byte

		err := rows.Scan(
			&m.ID, &phaseStr, &m.MatchNumber, &homeSlotJSON, &awaySlotJSON,
			&m.VenueID, &m.ScheduledAt, &m.Status,
			&m.ParentHomeMatchID, &m.ParentAwayMatchID,
			// result fields scanned via helper
		)
		if err != nil {
			return nil, fmt.Errorf("fixture_repo: error escaneando partido eliminatorio: %w", err)
		}

		m.Phase = fixture.MatchPhase(phaseStr)
		if err := json.Unmarshal(homeSlotJSON, &m.HomeSlot); err != nil {
			return nil, fmt.Errorf("fixture_repo: error deserializando home_slot: %w", err)
		}
		if err := json.Unmarshal(awaySlotJSON, &m.AwaySlot); err != nil {
			return nil, fmt.Errorf("fixture_repo: error deserializando away_slot: %w", err)
		}

		byPhase[m.Phase] = append(byPhase[m.Phase], m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rounds := make([]fixture.Round, 0, len(phaseOrder))
	for _, phase := range phaseOrder {
		if matches, ok := byPhase[phase]; ok {
			rounds = append(rounds, fixture.Round{Phase: phase, Matches: matches})
		}
	}

	return rounds, nil
}

// ── Save ──────────────────────────────────────────────────────────────────────

// Save persiste el estado del aggregate en una transacción atómica:
//  1. Actualiza el snapshot en tournaments (con optimistic locking)
//  2. Upsert de matches y match_results
//  3. Upsert de group_standings
//  4. Append de eventos pendientes en fixture_events
func (r *fixtureRepo) Save(ctx context.Context, f *fixture.Fixture) error {
	evts := f.PendingEvents()
	prevVersion := f.Version - int64(len(evts))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("fixture_repo: error iniciando transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Actualizar snapshot con optimistic locking
	if err := r.updateTournamentSnapshot(ctx, tx, f); err != nil {
		return err
	}

	// 2. Upsert de todos los partidos del fixture
	if err := r.upsertMatches(ctx, tx, f); err != nil {
		return err
	}

	// 3. Upsert de standings de todos los grupos
	if err := r.upsertStandings(ctx, tx, f); err != nil {
		return err
	}

	// 4. Append de eventos pendientes
	if err := r.eventStore.AppendEvents(ctx, tx, f.TournamentID, evts, prevVersion); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("fixture_repo: error en commit: %w", err)
	}

	return nil
}

func (r *fixtureRepo) updateTournamentSnapshot(ctx context.Context, tx pgx.Tx, f *fixture.Fixture) error {
	const q = `
		UPDATE tournaments
		SET status = $1, version = $2, updated_at = NOW()
		WHERE tournament_id = $3 AND version = $4`

	tag, err := tx.Exec(ctx, q, string(f.Status), f.Version, f.TournamentID, f.Version-1)
	if err != nil {
		return fmt.Errorf("fixture_repo: error actualizando snapshot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.Conflict("conflicto de versión en el aggregate Fixture — reintente la operación")
	}
	return nil
}

func (r *fixtureRepo) upsertMatches(ctx context.Context, tx pgx.Tx, f *fixture.Fixture) error {
	// Upsert partidos de grupos
	for _, g := range f.Groups {
		for _, m := range g.Matches {
			if err := r.upsertMatch(ctx, tx, f.TournamentID, &g.ID, m); err != nil {
				return err
			}
		}
	}
	// Upsert partidos eliminatorios
	for _, round := range f.KnockoutRounds {
		for _, m := range round.Matches {
			if err := r.upsertMatch(ctx, tx, f.TournamentID, nil, m); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *fixtureRepo) upsertMatch(ctx context.Context, tx pgx.Tx, tournamentID uuid.UUID, groupID *uuid.UUID, m fixture.Match) error {
	homeSlotJSON, err := json.Marshal(m.HomeSlot)
	if err != nil {
		return fmt.Errorf("fixture_repo: error serializando home_slot: %w", err)
	}
	awaySlotJSON, err := json.Marshal(m.AwaySlot)
	if err != nil {
		return fmt.Errorf("fixture_repo: error serializando away_slot: %w", err)
	}

	const q = `
		INSERT INTO matches
			(id, tournament_id, group_id, phase, match_number, home_slot, away_slot,
			 venue_id, scheduled_at, status, parent_home_id, parent_away_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			home_slot    = EXCLUDED.home_slot,
			away_slot    = EXCLUDED.away_slot,
			venue_id     = EXCLUDED.venue_id,
			scheduled_at = EXCLUDED.scheduled_at,
			status       = EXCLUDED.status,
			updated_at   = NOW()`

	if _, err := tx.Exec(ctx, q,
		m.ID, tournamentID, groupID, string(m.Phase), m.MatchNumber,
		homeSlotJSON, awaySlotJSON,
		m.VenueID, m.ScheduledAt, string(m.Status),
		m.ParentHomeMatchID, m.ParentAwayMatchID,
	); err != nil {
		return fmt.Errorf("fixture_repo: error en upsert de partido %d: %w", m.MatchNumber, err)
	}

	// Si hay resultado, persitirlo también
	if m.Result != nil {
		if err := r.upsertMatchResult(ctx, tx, m); err != nil {
			return err
		}
	}
	return nil
}

func (r *fixtureRepo) upsertMatchResult(ctx context.Context, tx pgx.Tx, m fixture.Match) error {
	res := m.Result
	winner := res.Winner()
	var winnerPtr *uuid.UUID
	if winner != uuid.Nil {
		winnerPtr = &winner
	}

	const q = `
		INSERT INTO match_results
			(match_id, home_team_id, away_team_id,
			 home_goals, away_goals,
			 home_goals_et, away_goals_et,
			 home_goals_pen, away_goals_pen,
			 winner_id, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (match_id) DO UPDATE SET
			home_goals     = EXCLUDED.home_goals,
			away_goals     = EXCLUDED.away_goals,
			home_goals_et  = EXCLUDED.home_goals_et,
			away_goals_et  = EXCLUDED.away_goals_et,
			home_goals_pen = EXCLUDED.home_goals_pen,
			away_goals_pen = EXCLUDED.away_goals_pen,
			winner_id      = EXCLUDED.winner_id,
			completed_at   = EXCLUDED.completed_at`

	if _, err := tx.Exec(ctx, q,
		m.ID, res.HomeTeamID, res.AwayTeamID,
		res.HomeGoals, res.AwayGoals,
		res.HomeGoalsET, res.AwayGoalsET,
		res.HomeGoalsPen, res.AwayGoalsPen,
		winnerPtr, res.CompletedAt,
	); err != nil {
		return fmt.Errorf("fixture_repo: error en upsert de resultado partido %s: %w", m.ID, err)
	}
	return nil
}

func (r *fixtureRepo) upsertStandings(ctx context.Context, tx pgx.Tx, f *fixture.Fixture) error {
	const q = `
		INSERT INTO group_standings
			(group_id, team_id, position, played, won, drawn, lost,
			 goals_for, goals_against, yellow_cards, red_cards, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		ON CONFLICT (group_id, team_id) DO UPDATE SET
			position      = EXCLUDED.position,
			played        = EXCLUDED.played,
			won           = EXCLUDED.won,
			drawn         = EXCLUDED.drawn,
			lost          = EXCLUDED.lost,
			goals_for     = EXCLUDED.goals_for,
			goals_against = EXCLUDED.goals_against,
			yellow_cards  = EXCLUDED.yellow_cards,
			red_cards     = EXCLUDED.red_cards,
			updated_at    = NOW()`

	for _, g := range f.Groups {
		for _, s := range g.Standings {
			if _, err := tx.Exec(ctx, q,
				g.ID, s.TeamID, s.Position,
				s.Played, s.Won, s.Drawn, s.Lost,
				s.GoalsFor, s.GoalsAgainst,
				s.YellowCards, s.RedCards,
			); err != nil {
				return fmt.Errorf("fixture_repo: error en upsert de standing grupo %s equipo %s: %w",
					g.Name, s.TeamID, err)
			}
		}
	}
	return nil
}

// ── Helpers de scan ───────────────────────────────────────────────────────────

// scanMatch escanea una fila de la query de partidos con JOIN a match_results.
// Retorna el partido y el groupID (puede ser nil para eliminatorias).
func (r *fixtureRepo) scanMatch(rows pgx.Rows) (fixture.Match, uuid.UUID, error) {
	var m fixture.Match
	var groupID uuid.UUID
	var homeSlotJSON, awaySlotJSON []byte
	var statusStr, phaseStr string

	// Campos de match_results (todos opcionales via LEFT JOIN)
	var homeGoals, awayGoals *int
	var homeGoalsET, awayGoalsET *int
	var homeGoalsPen, awayGoalsPen *int
	var winnerID *uuid.UUID
	var homeTeamID, awayTeamID *uuid.UUID

	err := rows.Scan(
		&m.ID, &groupID, &m.MatchNumber, &homeSlotJSON, &awaySlotJSON,
		&m.VenueID, &m.ScheduledAt, &statusStr,
		&homeGoals, &awayGoals,
		&homeGoalsET, &awayGoalsET,
		&homeGoalsPen, &awayGoalsPen,
		&winnerID,
		&m.Result, // placeholder — se sobrescribe abajo
		&homeTeamID, &awayTeamID,
	)
	if err != nil {
		return fixture.Match{}, uuid.Nil, fmt.Errorf("fixture_repo: error escaneando partido: %w", err)
	}

	m.Phase = fixture.MatchPhase(phaseStr)
	m.Status = fixture.MatchStatus(statusStr)

	if err := json.Unmarshal(homeSlotJSON, &m.HomeSlot); err != nil {
		return fixture.Match{}, uuid.Nil, fmt.Errorf("fixture_repo: home_slot inválido: %w", err)
	}
	if err := json.Unmarshal(awaySlotJSON, &m.AwaySlot); err != nil {
		return fixture.Match{}, uuid.Nil, fmt.Errorf("fixture_repo: away_slot inválido: %w", err)
	}

	// Reconstruir MatchResult si el partido está completado
	if homeGoals != nil && homeTeamID != nil && awayTeamID != nil {
		m.Result = &fixture.MatchResult{
			HomeTeamID:   *homeTeamID,
			AwayTeamID:   *awayTeamID,
			HomeGoals:    *homeGoals,
			AwayGoals:    *awayGoals,
			HomeGoalsET:  homeGoalsET,
			AwayGoalsET:  awayGoalsET,
			HomeGoalsPen: homeGoalsPen,
			AwayGoalsPen: awayGoalsPen,
		}
		_ = winnerID // derivado de Winner(), no necesitamos persistirlo
	}

	return m, groupID, nil
}
