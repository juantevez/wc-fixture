package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
)

// standingReadModel implementa queries.StandingsReadModel y queries.BestThirdsReadModel.
type standingReadModel struct {
	pool *pgxpool.Pool
}

var (
	_ queries.StandingsReadModel  = (*standingReadModel)(nil)
	_ queries.BestThirdsReadModel = (*standingReadModel)(nil)
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

// ── Helpers internos ──────────────────────────────────────────────────────────

// unmarshalJSON es un helper centralizado para deserializar JSON.
// Los errores incluyen contexto del campo para facilitar debugging.
func unmarshalJSON(data []byte, dst any) error {
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("error deserializando JSON: %w", err)
	}
	return nil
}
