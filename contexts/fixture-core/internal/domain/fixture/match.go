package fixture

import (
	"time"

	"github.com/google/uuid"
)

// MatchPhase indica en qué etapa del torneo se juega un partido.
type MatchPhase string

const (
	PhaseGroup       MatchPhase = "GROUP"
	PhaseRoundOf32   MatchPhase = "ROUND_OF_32"
	PhaseQuarterfinal MatchPhase = "QUARTERFINAL"
	PhaseSemifinal   MatchPhase = "SEMIFINAL"
	PhaseThirdPlace  MatchPhase = "THIRD_PLACE"
	PhaseFinal       MatchPhase = "FINAL"
)

// MatchStatus refleja el estado de vida de un partido.
type MatchStatus string

const (
	MatchStatusScheduled  MatchStatus = "SCHEDULED"
	MatchStatusInProgress MatchStatus = "IN_PROGRESS"
	MatchStatusCompleted  MatchStatus = "COMPLETED"
	MatchStatusPostponed  MatchStatus = "POSTPONED"
)

// Match representa un partido del torneo — tanto de fase de grupos
// como de fase eliminatoria.
//
// En fase de grupos: HomeSlot y AwaySlot siempre son SlotKindTeam.
// En fase eliminatoria: los slots pueden ser dinámicos hasta que el
// partido padre finaliza y el aggregate los resuelve.
type Match struct {
	ID          uuid.UUID
	Phase       MatchPhase
	MatchNumber int // número secuencial dentro del torneo (M1..M104)

	HomeSlot MatchSlot
	AwaySlot MatchSlot

	VenueID     uuid.UUID
	ScheduledAt time.Time // en UTC; la zona horaria del venue está en venue-geo

	Status MatchStatus
	Result *MatchResult // nil hasta que el partido está COMPLETED

	// Solo eliminatorias: IDs de los dos partidos de los que provienen
	// el local y el visitante respectivamente.
	ParentHomeMatchID *uuid.UUID
	ParentAwayMatchID *uuid.UUID
}

// MatchResult contiene el resultado final de un partido.
// Para partidos de grupos solo se registran los goles regulares.
// Para eliminatorias se incluye tiempo extra y penales si aplica.
type MatchResult struct {
	HomeTeamID uuid.UUID
	AwayTeamID uuid.UUID

	HomeGoals int
	AwayGoals int

	// Extra time — solo para eliminatorias
	HomeGoalsET *int
	AwayGoalsET *int

	// Penales — solo cuando hay empate en ET
	HomeGoalsPen *int
	AwayGoalsPen *int

	CompletedAt time.Time
}

// Winner retorna el UUID del equipo ganador.
// Para partidos de grupos puede retornar uuid.Nil si hay empate.
// Para eliminatorias siempre retorna un ganador (penales incluidos).
func (r MatchResult) Winner() uuid.UUID {
	homeTotal := r.HomeGoals
	awayTotal := r.AwayGoals

	if r.HomeGoalsET != nil {
		homeTotal += *r.HomeGoalsET
	}
	if r.AwayGoalsET != nil {
		awayTotal += *r.AwayGoalsET
	}
	if r.HomeGoalsPen != nil {
		homeTotal += *r.HomeGoalsPen
	}
	if r.AwayGoalsPen != nil {
		awayTotal += *r.AwayGoalsPen
	}

	switch {
	case homeTotal > awayTotal:
		return r.HomeTeamID
	case awayTotal > homeTotal:
		return r.AwayTeamID
	default:
		return uuid.Nil // empate (solo posible en fase de grupos)
	}
}

// Loser retorna el UUID del equipo perdedor.
// Retorna uuid.Nil en caso de empate.
func (r MatchResult) Loser() uuid.UUID {
	w := r.Winner()
	if w == uuid.Nil {
		return uuid.Nil
	}
	if w == r.HomeTeamID {
		return r.AwayTeamID
	}
	return r.HomeTeamID
}

// IsKnockoutPhase reporta si el partido pertenece a la fase eliminatoria.
func (m Match) IsKnockoutPhase() bool {
	return m.Phase != PhaseGroup
}

// IsCompleted reporta si el partido tiene resultado registrado.
func (m Match) IsCompleted() bool {
	return m.Status == MatchStatusCompleted && m.Result != nil
}

// BothSlotsResolved reporta si ambos equipos ya están determinados.
// Un partido eliminatorio puede estar SCHEDULED pero con slots aún no resueltos.
func (m Match) BothSlotsResolved() bool {
	return m.HomeSlot.IsResolved() && m.AwaySlot.IsResolved()
}

// validate verifica las reglas de negocio de un resultado antes de aplicarlo.
func (m Match) validateResult(result MatchResult) error {
	if m.IsCompleted() {
		return errMatchAlreadyCompleted(m.ID.String())
	}
	if !m.BothSlotsResolved() {
		return errSlotNotResolved(m.ID.String())
	}
	if result.HomeGoals < 0 || result.AwayGoals < 0 {
		return errInvalidResult("los goles no pueden ser negativos")
	}
	// En eliminatorias no puede haber empate sin tiempo extra
	if m.IsKnockoutPhase() {
		if result.HomeGoals == result.AwayGoals && result.HomeGoalsET == nil {
			return errInvalidResult("un partido eliminatorio en empate requiere tiempo extra")
		}
	}
	return nil
}
