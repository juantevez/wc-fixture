package result

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// IngestionSource identifica el origen del resultado.
type IngestionSource string

const (
	SourceFIFAAPI IngestionSource = "fifa_api" // API oficial FIFA
	SourceManual  IngestionSource = "manual"   // carga manual por operador
	SourceWebhook IngestionSource = "webhook"  // webhook de proveedor externo
)

// IngestedResult representa un resultado de partido recibido y validado,
// listo para ser publicado hacia fixture-core vía NATS JetStream.
//
// Es un value object inmutable — se construye una sola vez via New()
// y no puede modificarse después.
type IngestedResult struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	MatchID      uuid.UUID
	HomeTeamID   uuid.UUID
	AwayTeamID   uuid.UUID

	// Tiempo regular
	HomeGoals int
	AwayGoals int

	// Tiempo extra — nil si no hubo
	HomeGoalsET *int
	AwayGoalsET *int

	// Penales — nil si no hubo tanda
	HomeGoalsPen *int
	AwayGoalsPen *int

	CompletedAt    time.Time
	IngestedAt     time.Time
	Source         IngestionSource
	IdempotencyKey string // hash del resultado para detectar duplicados
}

// New construye y valida un IngestedResult.
// Retorna error si los datos violan las reglas de dominio.
func New(
	tournamentID, matchID, homeTeamID, awayTeamID uuid.UUID,
	homeGoals, awayGoals int,
	homeGoalsET, awayGoalsET *int,
	homeGoalsPen, awayGoalsPen *int,
	completedAt time.Time,
	source IngestionSource,
) (*IngestedResult, error) {
	r := &IngestedResult{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		MatchID:      matchID,
		HomeTeamID:   homeTeamID,
		AwayTeamID:   awayTeamID,
		HomeGoals:    homeGoals,
		AwayGoals:    awayGoals,
		HomeGoalsET:  homeGoalsET,
		AwayGoalsET:  awayGoalsET,
		HomeGoalsPen: homeGoalsPen,
		AwayGoalsPen: awayGoalsPen,
		CompletedAt:  completedAt,
		IngestedAt:   time.Now().UTC(),
		Source:       source,
	}

	if err := r.validate(); err != nil {
		return nil, err
	}

	r.IdempotencyKey = r.buildIdempotencyKey()
	return r, nil
}

// validate aplica todas las reglas de negocio del resultado.
func (r *IngestedResult) validate() error {
	// IDs requeridos
	if r.TournamentID == uuid.Nil {
		return &DomainError{Code: ErrCodeInvalidTournament, Message: "tournament_id es requerido"}
	}
	if r.MatchID == uuid.Nil {
		return &DomainError{Code: ErrCodeInvalidMatchID, Message: "match_id es requerido"}
	}
	if r.HomeTeamID == uuid.Nil || r.AwayTeamID == uuid.Nil {
		return ErrInvalidTeams("home_team_id y away_team_id son requeridos")
	}
	if r.HomeTeamID == r.AwayTeamID {
		return ErrInvalidTeams("home_team_id y away_team_id no pueden ser iguales")
	}

	// Goles no negativos
	if r.HomeGoals < 0 || r.AwayGoals < 0 {
		return ErrInvalidGoals("los goles no pueden ser negativos")
	}

	// Tiempo extra: ambos presentes o ninguno
	if (r.HomeGoalsET == nil) != (r.AwayGoalsET == nil) {
		return ErrInvalidExtraTime("home_goals_et y away_goals_et deben informarse juntos")
	}
	if r.HomeGoalsET != nil {
		if *r.HomeGoalsET < 0 || *r.AwayGoalsET < 0 {
			return ErrInvalidExtraTime("los goles de tiempo extra no pueden ser negativos")
		}
	}

	// Penales: requieren ET y ambos valores
	if r.HomeGoalsPen != nil || r.AwayGoalsPen != nil {
		if r.HomeGoalsET == nil {
			return ErrInvalidPenalties("los penales requieren tiempo extra informado")
		}
		if *r.HomeGoalsET != *r.AwayGoalsET {
			return ErrInvalidPenalties("los penales solo aplican cuando el tiempo extra termina empatado")
		}
		if r.HomeGoalsPen == nil || r.AwayGoalsPen == nil {
			return ErrInvalidPenalties("deben informarse los penales de ambos equipos")
		}
		if *r.HomeGoalsPen < 0 || *r.AwayGoalsPen < 0 {
			return ErrInvalidPenalties("los goles de penales no pueden ser negativos")
		}
		if *r.HomeGoalsPen == *r.AwayGoalsPen {
			return ErrInvalidPenalties("los penales no pueden terminar empatados")
		}
	}

	// Fecha de completado no puede ser futura
	if r.CompletedAt.After(time.Now().UTC().Add(5 * time.Minute)) {
		return &DomainError{
			Code:    ErrCodeFutureCompletedAt,
			Message: "completed_at no puede ser una fecha futura",
		}
	}

	return nil
}

// buildIdempotencyKey construye una clave única para detectar ingestas duplicadas.
// Se basa en match_id + resultado final para permitir reingesta con resultado diferente.
func (r *IngestedResult) buildIdempotencyKey() string {
	home := r.HomeGoals
	away := r.AwayGoals
	if r.HomeGoalsET != nil {
		home += *r.HomeGoalsET
		away += *r.AwayGoalsET
	}
	if r.HomeGoalsPen != nil {
		home += *r.HomeGoalsPen
		away += *r.AwayGoalsPen
	}
	return fmt.Sprintf("%s:%d:%d", r.MatchID, home, away)
}

// HasExtraTime reporta si el partido tuvo tiempo extra.
func (r *IngestedResult) HasExtraTime() bool {
	return r.HomeGoalsET != nil
}

// HasPenalties reporta si el partido tuvo tanda de penales.
func (r *IngestedResult) HasPenalties() bool {
	return r.HomeGoalsPen != nil
}
