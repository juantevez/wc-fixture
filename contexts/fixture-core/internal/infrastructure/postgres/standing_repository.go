package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
)

// standingReadModel implementa queries.StandingsReadModel, queries.BestThirdsReadModel,
// queries.FixtureReadModel, queries.GroupReadModel y queries.KnockoutReadModel.
type standingReadModel struct {
	pool *pgxpool.Pool
}

var (
	_ queries.StandingsReadModel  = (*standingReadModel)(nil)
	_ queries.BestThirdsReadModel = (*standingReadModel)(nil)
	_ queries.FixtureReadModel    = (*standingReadModel)(nil)
	_ queries.GroupReadModel      = (*standingReadModel)(nil)
	_ queries.KnockoutReadModel   = (*standingReadModel)(nil)
)

func NewStandingReadModel(pool *pgxpool.Pool) *standingReadModel {
	return &standingReadModel{pool: pool}
}

// GetStandings retorna la tabla de posiciones de un grupo ordenada por posición.
func (r *standingReadModel) GetStandings(ctx context.Context, tournamentID uuid.UUID, groupName string) ([]queries.StandingDTO, error) {
	const q = `
		SELECT gs.position, gs.team_id,
			   gs.played, gs.won, gs.drawn, gs.lost,
			   gs.goals_for, gs.goals_against,
			   gs.goals_for - gs.goals_against AS goal_difference,
			   gs.won * 3 + gs.drawn           AS points
		FROM group_standings gs
		JOIN groups g ON g.id = gs.group_id
		WHERE g.tournament_id = $1
		  AND g.name = $2
		ORDER BY gs.position`

	rows, err := r.pool.Query(ctx, q, tournamentID, groupName)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error consultando standings: %w", err)
	}
	defer rows.Close()

	var standings []queries.StandingDTO
	for rows.Next() {
		var s queries.StandingDTO
		if err := rows.Scan(
			&s.Position, &s.TeamID,
			&s.Played, &s.Won, &s.Drawn, &s.Lost,
			&s.GoalsFor, &s.GoalsAgainst,
			&s.GoalDiff, &s.Points,
		); err != nil {
			return nil, fmt.Errorf("standing_read_model: error escaneando standing: %w", err)
		}
		standings = append(standings, s)
	}

	if len(standings) == 0 {
		return nil, apperrors.NotFound("grupo", groupName)
	}

	return standings, rows.Err()
}

// GetBestThirds retorna el ranking de todos los terceros clasificados,
// ordenado de mejor a peor según criterios FIFA. Solo disponible cuando
// la fase de grupos está completa.
//
// La query materializa el cálculo de puntos y diferencia de goles directamente
// en SQL para evitar cargar el aggregate completo.
func (r *standingReadModel) GetBestThirds(ctx context.Context, tournamentID uuid.UUID) ([]queries.BestThirdDTO, error) {
	const q = `
		WITH thirds AS (
			SELECT
				g.name                                    AS group_name,
				gs.team_id,
				gs.played, gs.won, gs.drawn, gs.lost,
				gs.goals_for, gs.goals_against,
				gs.goals_for - gs.goals_against           AS goal_difference,
				gs.won * 3 + gs.drawn                     AS points,
				gs.yellow_cards * 1 + gs.red_cards * 3   AS fair_play_points,
				ROW_NUMBER() OVER (
					ORDER BY
						gs.won * 3 + gs.drawn                   DESC,
						gs.goals_for - gs.goals_against          DESC,
						gs.goals_for                             DESC,
						gs.yellow_cards * 1 + gs.red_cards * 3  ASC
				) AS rank
			FROM group_standings gs
			JOIN groups g ON g.id = gs.group_id
			WHERE g.tournament_id = $1
			  AND gs.position = 3
		)
		SELECT group_name, team_id, rank,
			   played, won, drawn, lost,
			   goals_for, goals_against, goal_difference, points,
			   rank <= 8 AS classified
		FROM thirds
		ORDER BY rank`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error consultando mejores terceros: %w", err)
	}
	defer rows.Close()

	var thirds []queries.BestThirdDTO
	for rows.Next() {
		var t queries.BestThirdDTO
		if err := rows.Scan(
			&t.GroupName, &t.TeamID, &t.Rank,
			&t.Played, &t.Won, &t.Drawn, &t.Lost,
			&t.GoalsFor, &t.GoalsAgainst, &t.GoalDiff, &t.Points,
			&t.Classified,
		); err != nil {
			return nil, fmt.Errorf("standing_read_model: error escaneando mejor tercero: %w", err)
		}
		thirds = append(thirds, t)
	}

	return thirds, rows.Err()
}

// ── FixtureReadModel ──────────────────────────────────────────────────────────

// GetFixture retorna el fixture completo: torneo + grupos + bracket eliminatorio.
func (r *standingReadModel) GetFixture(ctx context.Context, tournamentID uuid.UUID) (*queries.FixtureDTO, error) {
	const tq = `
		SELECT edition, name, status
		FROM tournaments
		WHERE tournament_id = $1`

	var dto queries.FixtureDTO
	dto.TournamentID = tournamentID
	var statusStr string
	if err := r.pool.QueryRow(ctx, tq, tournamentID).Scan(&dto.Edition, &dto.Name, &statusStr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("torneo", tournamentID.String())
		}
		return nil, fmt.Errorf("standing_read_model: error cargando torneo: %w", err)
	}
	dto.Status = statusStr

	groups, err := r.loadGroupDTOs(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	dto.Groups = groups

	if statusStr == "KNOCKOUT" || statusStr == "FINISHED" {
		knockout, err := r.loadKnockoutDTO(ctx, tournamentID)
		if err != nil {
			return nil, err
		}
		dto.Knockout = knockout
	}

	return &dto, nil
}

// ── GroupReadModel ────────────────────────────────────────────────────────────

// ListGroups retorna todos los grupos con sus standings y partidos.
func (r *standingReadModel) ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]queries.GroupDTO, error) {
	return r.loadGroupDTOs(ctx, tournamentID)
}

// GetGroup retorna el detalle de un grupo específico.
func (r *standingReadModel) GetGroup(ctx context.Context, tournamentID uuid.UUID, groupName string) (*queries.GroupDTO, error) {
	groups, err := r.loadGroupDTOs(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].Name == groupName {
			return &groups[i], nil
		}
	}
	return nil, apperrors.NotFound("grupo", groupName)
}

// ── KnockoutReadModel ─────────────────────────────────────────────────────────

// GetKnockout retorna el bracket eliminatorio completo.
func (r *standingReadModel) GetKnockout(ctx context.Context, tournamentID uuid.UUID) (*queries.KnockoutDTO, error) {
	dto, err := r.loadKnockoutDTO(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if dto == nil {
		return nil, apperrors.NotFound("bracket eliminatorio", tournamentID.String())
	}
	return dto, nil
}

// GetKnockoutRound retorna los partidos de una ronda eliminatoria específica.
func (r *standingReadModel) GetKnockoutRound(ctx context.Context, tournamentID uuid.UUID, phase string) ([]queries.MatchDTO, error) {
	const q = `
		SELECT m.id, m.match_number, m.phase, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE m.tournament_id = $1 AND m.phase = $2
		ORDER BY m.match_number`

	rows, err := r.pool.Query(ctx, q, tournamentID, phase)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error consultando ronda %q: %w", phase, err)
	}
	defer rows.Close()

	var matches []queries.MatchDTO
	for rows.Next() {
		dto, err := scanKnockoutMatchDTO(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, dto)
	}
	return matches, rows.Err()
}

// ── Helpers de carga ──────────────────────────────────────────────────────────

// loadGroupDTOs carga todos los grupos de un torneo con teams, standings y partidos.
func (r *standingReadModel) loadGroupDTOs(ctx context.Context, tournamentID uuid.UUID) ([]queries.GroupDTO, error) {
	const gq = `
		SELECT g.id, g.name, g.status,
			   array_agg(gt.team_id ORDER BY gt.seeding_pot) AS team_ids
		FROM groups g
		JOIN group_teams gt ON gt.group_id = g.id
		WHERE g.tournament_id = $1
		GROUP BY g.id, g.name, g.status
		ORDER BY g.name`

	type entry struct {
		id  uuid.UUID
		dto queries.GroupDTO
	}
	var entries []entry
	groupIdx := make(map[uuid.UUID]int)

	grows, err := r.pool.Query(ctx, gq, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error cargando grupos: %w", err)
	}
	defer grows.Close()

	for grows.Next() {
		var id uuid.UUID
		var statusStr string
		var teamIDs []uuid.UUID
		var dto queries.GroupDTO
		if err := grows.Scan(&id, &dto.Name, &statusStr, &teamIDs); err != nil {
			return nil, fmt.Errorf("standing_read_model: error escaneando grupo: %w", err)
		}
		dto.Status = statusStr
		dto.Teams = teamIDs
		groupIdx[id] = len(entries)
		entries = append(entries, entry{id: id, dto: dto})
	}
	if err := grows.Err(); err != nil {
		return nil, err
	}

	const sq = `
		SELECT gs.group_id, gs.position, gs.team_id,
			   gs.played, gs.won, gs.drawn, gs.lost,
			   gs.goals_for, gs.goals_against,
			   gs.won * 3 + gs.drawn          AS points,
			   gs.goals_for - gs.goals_against AS goal_difference
		FROM group_standings gs
		JOIN groups g ON g.id = gs.group_id
		WHERE g.tournament_id = $1
		ORDER BY gs.group_id, gs.position`

	srows, err := r.pool.Query(ctx, sq, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error cargando standings: %w", err)
	}
	defer srows.Close()

	for srows.Next() {
		var groupID uuid.UUID
		var s queries.StandingDTO
		if err := srows.Scan(
			&groupID, &s.Position, &s.TeamID,
			&s.Played, &s.Won, &s.Drawn, &s.Lost,
			&s.GoalsFor, &s.GoalsAgainst,
			&s.Points, &s.GoalDiff,
		); err != nil {
			return nil, fmt.Errorf("standing_read_model: error escaneando standing: %w", err)
		}
		if idx, ok := groupIdx[groupID]; ok {
			entries[idx].dto.Standings = append(entries[idx].dto.Standings, s)
		}
	}
	if err := srows.Err(); err != nil {
		return nil, err
	}

	const mq = `
		SELECT m.id, m.match_number, m.phase, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id,
			   m.group_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE m.tournament_id = $1 AND m.phase = 'GROUP'
		ORDER BY m.match_number`

	mrows, err := r.pool.Query(ctx, mq, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error cargando partidos de grupos: %w", err)
	}
	defer mrows.Close()

	for mrows.Next() {
		var dto queries.MatchDTO
		var homeSlotJSON, awaySlotJSON []byte
		var phaseStr, statusStr string
		var homeGoals, awayGoals *int
		var homeGoalsET, awayGoalsET, homeGoalsPen, awayGoalsPen *int
		var winnerID, homeTeamID, awayTeamID *uuid.UUID
		var completedAt *time.Time
		var groupID uuid.UUID

		if err := mrows.Scan(
			&dto.ID, &dto.MatchNumber, &phaseStr, &homeSlotJSON, &awaySlotJSON,
			&dto.VenueID, &dto.ScheduledAt, &statusStr,
			&homeGoals, &awayGoals,
			&homeGoalsET, &awayGoalsET,
			&homeGoalsPen, &awayGoalsPen,
			&winnerID, &completedAt,
			&homeTeamID, &awayTeamID,
			&groupID,
		); err != nil {
			return nil, fmt.Errorf("standing_read_model: error escaneando partido de grupo: %w", err)
		}
		dto.Phase = phaseStr
		dto.Status = statusStr
		if err := unmarshalSlotDTO(homeSlotJSON, &dto.HomeSlot); err != nil {
			return nil, err
		}
		if err := unmarshalSlotDTO(awaySlotJSON, &dto.AwaySlot); err != nil {
			return nil, err
		}
		if homeGoals != nil {
			res := &queries.ResultDTO{
				HomeGoals:    *homeGoals,
				AwayGoals:    *awayGoals,
				HomeGoalsET:  homeGoalsET,
				AwayGoalsET:  awayGoalsET,
				HomeGoalsPen: homeGoalsPen,
				AwayGoalsPen: awayGoalsPen,
				WinnerID:     winnerID,
			}
			if completedAt != nil {
				res.CompletedAt = *completedAt
			}
			dto.Result = res
		}
		if idx, ok := groupIdx[groupID]; ok {
			entries[idx].dto.Matches = append(entries[idx].dto.Matches, dto)
		}
	}
	if err := mrows.Err(); err != nil {
		return nil, err
	}

	result := make([]queries.GroupDTO, len(entries))
	for i, e := range entries {
		result[i] = e.dto
	}
	return result, nil
}

// loadKnockoutDTO carga el bracket eliminatorio completo de un torneo.
// Retorna nil si no hay partidos eliminatorios.
func (r *standingReadModel) loadKnockoutDTO(ctx context.Context, tournamentID uuid.UUID) (*queries.KnockoutDTO, error) {
	const q = `
		SELECT m.id, m.match_number, m.phase, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE m.tournament_id = $1 AND m.phase != 'GROUP'
		ORDER BY
			CASE m.phase
				WHEN 'ROUND_OF_32'  THEN 1
				WHEN 'QUARTERFINAL' THEN 2
				WHEN 'SEMIFINAL'    THEN 3
				WHEN 'THIRD_PLACE'  THEN 4
				WHEN 'FINAL'        THEN 5
			END,
			m.match_number`

	rows, err := r.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("standing_read_model: error cargando bracket: %w", err)
	}
	defer rows.Close()

	dto := &queries.KnockoutDTO{}
	hasMatches := false

	for rows.Next() {
		m, err := scanKnockoutMatchDTO(rows)
		if err != nil {
			return nil, err
		}
		hasMatches = true
		switch m.Phase {
		case "ROUND_OF_32":
			dto.RoundOf32 = append(dto.RoundOf32, m)
		case "QUARTERFINAL":
			dto.Quarterfinals = append(dto.Quarterfinals, m)
		case "SEMIFINAL":
			dto.Semifinals = append(dto.Semifinals, m)
		case "THIRD_PLACE":
			cp := m
			dto.ThirdPlace = &cp
		case "FINAL":
			cp := m
			dto.Final = &cp
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !hasMatches {
		return nil, nil
	}
	return dto, nil
}

// scanKnockoutMatchDTO escanea una fila de partido (con LEFT JOIN a match_results)
// en un MatchDTO. Usa una variable *time.Time para completed_at en lugar de
// escanear directamente en ResultDTO.
func scanKnockoutMatchDTO(rows interface{ Scan(dest ...any) error }) (queries.MatchDTO, error) {
	var dto queries.MatchDTO
	var homeSlotJSON, awaySlotJSON []byte
	var phaseStr, statusStr string
	var homeGoals, awayGoals *int
	var homeGoalsET, awayGoalsET, homeGoalsPen, awayGoalsPen *int
	var winnerID, homeTeamID, awayTeamID *uuid.UUID
	var completedAt *time.Time

	if err := rows.Scan(
		&dto.ID, &dto.MatchNumber, &phaseStr, &homeSlotJSON, &awaySlotJSON,
		&dto.VenueID, &dto.ScheduledAt, &statusStr,
		&homeGoals, &awayGoals,
		&homeGoalsET, &awayGoalsET,
		&homeGoalsPen, &awayGoalsPen,
		&winnerID, &completedAt,
		&homeTeamID, &awayTeamID,
	); err != nil {
		return queries.MatchDTO{}, fmt.Errorf("standing_read_model: error escaneando partido: %w", err)
	}
	dto.Phase = phaseStr
	dto.Status = statusStr
	if err := unmarshalSlotDTO(homeSlotJSON, &dto.HomeSlot); err != nil {
		return queries.MatchDTO{}, err
	}
	if err := unmarshalSlotDTO(awaySlotJSON, &dto.AwaySlot); err != nil {
		return queries.MatchDTO{}, err
	}
	if homeGoals != nil {
		res := &queries.ResultDTO{
			HomeGoals:    *homeGoals,
			AwayGoals:    *awayGoals,
			HomeGoalsET:  homeGoalsET,
			AwayGoalsET:  awayGoalsET,
			HomeGoalsPen: homeGoalsPen,
			AwayGoalsPen: awayGoalsPen,
			WinnerID:     winnerID,
		}
		if completedAt != nil {
			res.CompletedAt = *completedAt
		}
		dto.Result = res
	}
	return dto, nil
}

// ── Helpers internos ──────────────────────────────────────────────────────────

// unmarshalJSON es un helper centralizado para deserializar JSON.
// Los errores incluyen contexto del campo para facilitar debugging.
func unmarshalJSON(data []byte, dst any) error {
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("error deserializando JSON: %w", err)
	}
	return nil
}
