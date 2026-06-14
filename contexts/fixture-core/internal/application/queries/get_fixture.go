// Package queries contiene los query handlers de fixture-core (lado de lectura CQRS).
// Los queries NO cargan el aggregate Fixture — trabajan directamente contra
// read models optimizados en PostgreSQL, evitando el costo de reconstruir
// el aggregate completo para operaciones de solo lectura.
//
// Separación de responsabilidades:
//   - Commands → cargan aggregate, ejecutan lógica, persisten, publican eventos
//   - Queries  → consultan read models (vistas/tablas desnormalizadas), retornan DTOs
package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// ── DTOs de respuesta ─────────────────────────────────────────────────────────
// Los DTOs son estructuras planas optimizadas para serialización JSON.
// No contienen lógica de dominio.

// FixtureDTO es la vista completa del torneo: grupos + bracket eliminatorio.
// Es la respuesta del endpoint GET /tournaments/:id/fixture.
type FixtureDTO struct {
	TournamentID uuid.UUID    `json:"tournament_id"`
	Edition      int          `json:"edition"`
	Name         string       `json:"name"`
	Status       string       `json:"status"`
	Groups       []GroupDTO   `json:"groups"`
	Knockout     *KnockoutDTO `json:"knockout,omitempty"`
}

// GroupDTO representa un grupo con equipos, partidos y tabla de posiciones.
type GroupDTO struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Teams     []uuid.UUID    `json:"teams"`
	Standings []StandingDTO  `json:"standings"`
	Matches   []MatchDTO     `json:"matches"`
}

// StandingDTO es la fila de la tabla de posiciones de un grupo.
type StandingDTO struct {
	Position     int       `json:"position"`
	TeamID       uuid.UUID `json:"team_id"`
	Points       int       `json:"points"`
	Played       int       `json:"played"`
	Won          int       `json:"won"`
	Drawn        int       `json:"drawn"`
	Lost         int       `json:"lost"`
	GoalsFor     int       `json:"goals_for"`
	GoalsAgainst int       `json:"goals_against"`
	GoalDiff     int       `json:"goal_difference"`
}

// MatchDTO representa un partido con su resultado si ya fue jugado.
type MatchDTO struct {
	ID          uuid.UUID   `json:"id"`
	MatchNumber int         `json:"match_number"`
	Phase       string      `json:"phase"`
	HomeSlot    SlotDTO     `json:"home_slot"`
	AwaySlot    SlotDTO     `json:"away_slot"`
	VenueID     uuid.UUID   `json:"venue_id"`
	ScheduledAt time.Time   `json:"scheduled_at"`
	Status      string      `json:"status"`
	Result      *ResultDTO  `json:"result,omitempty"`
}

// SlotDTO representa un slot de partido (equipo concreto o referencia dinámica).
type SlotDTO struct {
	Kind          string     `json:"kind"`
	TeamID        *uuid.UUID `json:"team_id,omitempty"`
	SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
}

// ResultDTO contiene el resultado de un partido jugado.
type ResultDTO struct {
	HomeGoals    int        `json:"home_goals"`
	AwayGoals    int        `json:"away_goals"`
	HomeGoalsET  *int       `json:"home_goals_et,omitempty"`
	AwayGoalsET  *int       `json:"away_goals_et,omitempty"`
	HomeGoalsPen *int       `json:"home_goals_pen,omitempty"`
	AwayGoalsPen *int       `json:"away_goals_pen,omitempty"`
	WinnerID     *uuid.UUID `json:"winner_id,omitempty"`
	CompletedAt  time.Time  `json:"completed_at"`
}

// KnockoutDTO contiene el bracket eliminatorio completo.
type KnockoutDTO struct {
	RoundOf32    []MatchDTO `json:"round_of_32"`
	Quarterfinals []MatchDTO `json:"quarterfinals"`
	Semifinals   []MatchDTO `json:"semifinals"`
	ThirdPlace   *MatchDTO  `json:"third_place,omitempty"`
	Final        *MatchDTO  `json:"final,omitempty"`
}

// ── Read Model Port ───────────────────────────────────────────────────────────

// FixtureReadModel es el puerto de lectura del aggregate Fixture.
// La implementación concreta consulta tablas desnormalizadas en PostgreSQL,
// no el event store.
type FixtureReadModel interface {
	GetFixture(ctx context.Context, tournamentID uuid.UUID) (*FixtureDTO, error)
}

// ── Query y Handler ───────────────────────────────────────────────────────────

// GetFixtureQuery solicita el fixture completo de un torneo.
type GetFixtureQuery struct {
	TournamentID uuid.UUID
}

// GetFixtureHandler retorna el fixture completo (grupos + bracket).
type GetFixtureHandler struct {
	readModel FixtureReadModel
}

func NewGetFixtureHandler(rm FixtureReadModel) *GetFixtureHandler {
	return &GetFixtureHandler{readModel: rm}
}

func (h *GetFixtureHandler) Handle(ctx context.Context, q GetFixtureQuery) (*FixtureDTO, error) {
	if q.TournamentID == uuid.Nil {
		return nil, apperrors.Validation("tournament_id es requerido")
	}

	dto, err := h.readModel.GetFixture(ctx, q.TournamentID)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("fixture consultado",
		"tournament_id", q.TournamentID,
		"status", dto.Status,
		"grupos", len(dto.Groups),
	)
	return dto, nil
}

// ── Helpers de mapeo dominio → DTO ───────────────────────────────────────────
// Se usan cuando el read model retorna el aggregate en lugar de una proyección.

func ToFixtureDTO(f *fixture.Fixture) *FixtureDTO {
	groups := make([]GroupDTO, len(f.Groups))
	for i, g := range f.Groups {
		groups[i] = ToGroupDTO(g)
	}

	dto := &FixtureDTO{
		TournamentID: f.TournamentID,
		Edition:      f.Edition,
		Name:         f.Name,
		Status:       string(f.Status),
		Groups:       groups,
	}

	if len(f.KnockoutRounds) > 0 {
		dto.Knockout = ToKnockoutDTO(f.KnockoutRounds)
	}

	return dto
}

func ToGroupDTO(g fixture.Group) GroupDTO {
	teams := make([]uuid.UUID, len(g.Teams))
	copy(teams[:], g.Teams[:])

	standings := make([]StandingDTO, len(g.Standings))
	for i, s := range g.Standings {
		standings[i] = toStandingDTO(s)
	}

	matches := make([]MatchDTO, len(g.Matches))
	for i, m := range g.Matches {
		matches[i] = ToMatchDTO(m)
	}

	return GroupDTO{
		Name:      g.Name,
		Status:    string(g.Status),
		Teams:     teams,
		Standings: standings,
		Matches:   matches,
	}
}

func toStandingDTO(s fixture.GroupStanding) StandingDTO {
	return StandingDTO{
		Position:     s.Position,
		TeamID:       s.TeamID,
		Points:       s.Points(),
		Played:       s.Played,
		Won:          s.Won,
		Drawn:        s.Drawn,
		Lost:         s.Lost,
		GoalsFor:     s.GoalsFor,
		GoalsAgainst: s.GoalsAgainst,
		GoalDiff:     s.GoalDifference(),
	}
}

func ToMatchDTO(m fixture.Match) MatchDTO {
	dto := MatchDTO{
		ID:          m.ID,
		MatchNumber: m.MatchNumber,
		Phase:       string(m.Phase),
		HomeSlot:    toSlotDTO(m.HomeSlot),
		AwaySlot:    toSlotDTO(m.AwaySlot),
		VenueID:     m.VenueID,
		ScheduledAt: m.ScheduledAt,
		Status:      string(m.Status),
	}
	if m.Result != nil {
		dto.Result = toResultDTO(m.Result)
	}
	return dto
}

func toSlotDTO(s fixture.MatchSlot) SlotDTO {
	return SlotDTO{
		Kind:          string(s.Kind),
		TeamID:        s.TeamID,
		SourceMatchID: s.SourceMatchID,
	}
}

func toResultDTO(r *fixture.MatchResult) *ResultDTO {
	winner := r.Winner()
	var winnerPtr *uuid.UUID
	if winner != uuid.Nil {
		winnerPtr = &winner
	}
	return &ResultDTO{
		HomeGoals:    r.HomeGoals,
		AwayGoals:    r.AwayGoals,
		HomeGoalsET:  r.HomeGoalsET,
		AwayGoalsET:  r.AwayGoalsET,
		HomeGoalsPen: r.HomeGoalsPen,
		AwayGoalsPen: r.AwayGoalsPen,
		WinnerID:     winnerPtr,
		CompletedAt:  r.CompletedAt,
	}
}

func ToKnockoutDTO(rounds []fixture.Round) *KnockoutDTO {
	dto := &KnockoutDTO{}
	for _, r := range rounds {
		matches := make([]MatchDTO, len(r.Matches))
		for i, m := range r.Matches {
			matches[i] = ToMatchDTO(m)
		}
		switch r.Phase {
		case fixture.PhaseRoundOf32:
			dto.RoundOf32 = matches
		case fixture.PhaseQuarterfinal:
			dto.Quarterfinals = matches
		case fixture.PhaseSemifinal:
			dto.Semifinals = matches
		case fixture.PhaseThirdPlace:
			if len(matches) > 0 {
				dto.ThirdPlace = &matches[0]
			}
		case fixture.PhaseFinal:
			if len(matches) > 0 {
				dto.Final = &matches[0]
			}
		}
	}
	return dto
}
