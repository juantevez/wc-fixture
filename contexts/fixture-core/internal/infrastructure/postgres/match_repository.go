package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// matchReadModel implementa queries.MatchReadModel para consultas de partidos.
// Lee directamente de la tabla matches + match_results (desnormalizada),
// sin cargar el aggregate completo.
type matchReadModel struct {
	pool *pgxpool.Pool
}

// Verificación en tiempo de compilación.
var _ queries.MatchReadModel = (*matchReadModel)(nil)

func NewMatchReadModel(pool *pgxpool.Pool) queries.MatchReadModel {
	return &matchReadModel{pool: pool}
}

// ListMatches retorna partidos filtrados y paginados.
func (r *matchReadModel) ListMatches(
	ctx context.Context,
	tournamentID uuid.UUID,
	filters queries.MatchFilters,
	page httputil.PageParams,
) ([]queries.MatchDTO, int, error) {

	where, args := buildMatchWhereClause(tournamentID, filters)

	// Query de count para paginación
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE %s`, where)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("match_read_model: error contando partidos: %w", err)
	}

	// Query de datos con paginación
	dataQuery := fmt.Sprintf(`
		SELECT m.id, m.match_number, m.phase, m.home_slot, m.away_slot,
			   m.venue_id, m.scheduled_at, m.status,
			   mr.home_goals, mr.away_goals,
			   mr.home_goals_et, mr.away_goals_et,
			   mr.home_goals_pen, mr.away_goals_pen,
			   mr.winner_id, mr.completed_at,
			   mr.home_team_id, mr.away_team_id
		FROM matches m
		LEFT JOIN match_results mr ON mr.match_id = m.id
		WHERE %s
		ORDER BY m.match_number
		LIMIT $%d OFFSET $%d`,
		where,
		len(args)+1,
		len(args)+2,
	)

	args = append(args, page.Limit(), page.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("match_read_model: error listando partidos: %w", err)
	}
	defer rows.Close()

	var matches []queries.MatchDTO
	for rows.Next() {
		dto, err := scanMatchDTO(rows)
		if err != nil {
			return nil, 0, err
		}
		matches = append(matches, dto)
	}

	return matches, total, rows.Err()
}

// GetMatch retorna el detalle de un partido por su ID.
func (r *matchReadModel) GetMatch(ctx context.Context, matchID uuid.UUID) (*queries.MatchDTO, error) {
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
		WHERE m.id = $1`

	rows, err := r.pool.Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("match_read_model: error consultando partido: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, apperrors.NotFound("partido", matchID.String())
	}

	dto, err := scanMatchDTO(rows)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

// buildMatchWhereClause construye la cláusula WHERE y los args según los filtros activos.
func buildMatchWhereClause(tournamentID uuid.UUID, f queries.MatchFilters) (string, []any) {
	conditions := []string{"m.tournament_id = $1"}
	args := []any{tournamentID}
	n := 2

	if f.Phase != "" {
		conditions = append(conditions, fmt.Sprintf("m.phase = $%d", n))
		args = append(args, f.Phase)
		n++
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("m.status = $%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.GroupName != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (SELECT 1 FROM groups g WHERE g.id = m.group_id AND g.name = $%d)`, n))
		args = append(args, f.GroupName)
		n++
	}
	if f.VenueID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("m.venue_id = $%d", n))
		args = append(args, f.VenueID)
		n++
	}
	if !f.DateFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("m.scheduled_at >= $%d", n))
		args = append(args, f.DateFrom)
		n++
	}
	if !f.DateTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("m.scheduled_at <= $%d", n))
		args = append(args, f.DateTo)
		n++
	}
	if f.TeamID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf(`
			(m.home_slot->>'team_id' = $%d OR m.away_slot->>'team_id' = $%d)`, n, n))
		args = append(args, f.TeamID.String())
		n++
	}

	_ = n
	return strings.Join(conditions, " AND "), args
}

// scanMatchDTO escanea una fila del resultado de query en un MatchDTO.
func scanMatchDTO(rows interface {
	Scan(dest ...any) error
}) (queries.MatchDTO, error) {
	var dto queries.MatchDTO
	var homeSlotJSON, awaySlotJSON []byte
	var phaseStr, statusStr string

	var homeGoals, awayGoals *int
	var homeGoalsET, awayGoalsET *int
	var homeGoalsPen, awayGoalsPen *int
	var winnerID, homeTeamID, awayTeamID *uuid.UUID

	if err := rows.Scan(
		&dto.ID, &dto.MatchNumber, &phaseStr, &homeSlotJSON, &awaySlotJSON,
		&dto.VenueID, &dto.ScheduledAt, &statusStr,
		&homeGoals, &awayGoals,
		&homeGoalsET, &awayGoalsET,
		&homeGoalsPen, &awayGoalsPen,
		&winnerID, &dto.Result,
		&homeTeamID, &awayTeamID,
	); err != nil {
		return queries.MatchDTO{}, fmt.Errorf("match_read_model: error escaneando partido: %w", err)
	}

	dto.Phase = phaseStr
	dto.Status = statusStr

	// Deserializar slots desde JSONB
	if err := unmarshalSlotDTO(homeSlotJSON, &dto.HomeSlot); err != nil {
		return queries.MatchDTO{}, err
	}
	if err := unmarshalSlotDTO(awaySlotJSON, &dto.AwaySlot); err != nil {
		return queries.MatchDTO{}, err
	}

	// Construir ResultDTO si el partido tiene resultado
	if homeGoals != nil {
		dto.Result = &queries.ResultDTO{
			HomeGoals:    *homeGoals,
			AwayGoals:    *awayGoals,
			HomeGoalsET:  homeGoalsET,
			AwayGoalsET:  awayGoalsET,
			HomeGoalsPen: homeGoalsPen,
			AwayGoalsPen: awayGoalsPen,
			WinnerID:     winnerID,
		}
		if homeTeamID != nil {
			// completedAt lo ignoramos en el DTO de lista para evitar scan adicional
		}
	}

	return dto, nil
}

// unmarshalSlotDTO deserializa el JSONB del slot en un SlotDTO.
func unmarshalSlotDTO(data []byte, dst *queries.SlotDTO) error {
	var raw struct {
		Kind          string     `json:"kind"`
		TeamID        *uuid.UUID `json:"team_id"`
		SourceMatchID *uuid.UUID `json:"source_match_id"`
	}
	if err := unmarshalJSON(data, &raw); err != nil {
		return fmt.Errorf("match_read_model: slot inválido: %w", err)
	}
	dst.Kind = raw.Kind
	dst.TeamID = raw.TeamID
	dst.SourceMatchID = raw.SourceMatchID
	return nil
}
