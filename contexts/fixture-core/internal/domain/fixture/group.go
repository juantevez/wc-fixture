package fixture

import (
	"github.com/google/uuid"
)

// GroupStatus refleja el estado del grupo dentro de la fase de grupos.
type GroupStatus string

const (
	GroupStatusPending    GroupStatus = "PENDING"
	GroupStatusInProgress GroupStatus = "IN_PROGRESS"
	GroupStatusCompleted  GroupStatus = "COMPLETED"
)

const (
	teamsPerGroup   = 4
	matchesPerGroup = 6 // C(4,2) = 6 partidos por grupo
)

// Group representa uno de los 12 grupos del torneo (A–L).
// Contiene los 4 equipos, los 6 partidos inter-grupo y la tabla de posiciones.
//
// Un grupo está COMPLETED cuando sus 6 partidos tienen resultado registrado.
// En ese momento el aggregate Fixture puede leer sus standings para determinar
// los clasificados a la fase eliminatoria.
type Group struct {
	ID       uuid.UUID
	Name     string // "A" … "L"
	Status   GroupStatus
	Teams    [teamsPerGroup]uuid.UUID // IDs de los 4 equipos
	Matches  []Match                  // los 6 partidos del grupo
	Standings []GroupStanding          // tabla de posiciones, ordenada
}

// CompletedMatchCount retorna cuántos partidos del grupo tienen resultado.
func (g Group) CompletedMatchCount() int {
	count := 0
	for _, m := range g.Matches {
		if m.IsCompleted() {
			count++
		}
	}
	return count
}

// IsComplete reporta si todos los partidos del grupo tienen resultado.
func (g Group) IsComplete() bool {
	return g.CompletedMatchCount() == matchesPerGroup
}

// ClassifiedTeams retorna los IDs del 1° y 2° clasificado del grupo.
// Solo debe llamarse cuando el grupo está COMPLETED.
func (g Group) ClassifiedTeams() (first, second uuid.UUID) {
	if len(g.Standings) < 2 {
		return uuid.Nil, uuid.Nil
	}
	return g.Standings[0].TeamID, g.Standings[1].TeamID
}

// ThirdPlace retorna el standing del equipo en 3° posición del grupo.
// Solo debe llamarse cuando el grupo está COMPLETED.
func (g Group) ThirdPlace() (GroupStanding, bool) {
	if len(g.Standings) < 3 {
		return GroupStanding{}, false
	}
	return g.Standings[2], true
}

// StandingFor retorna el standing de un equipo específico en el grupo.
func (g Group) StandingFor(teamID uuid.UUID) (GroupStanding, bool) {
	for _, s := range g.Standings {
		if s.TeamID == teamID {
			return s, true
		}
	}
	return GroupStanding{}, false
}

// findMatch busca un partido dentro del grupo por su ID.
func (g *Group) findMatch(matchID uuid.UUID) (*Match, bool) {
	for i := range g.Matches {
		if g.Matches[i].ID == matchID {
			return &g.Matches[i], true
		}
	}
	return nil, false
}

// applyResult registra el resultado de un partido en el grupo,
// recalcula la tabla de posiciones y actualiza el status del grupo.
// Retorna error si el resultado es inválido.
func (g *Group) applyResult(matchID uuid.UUID, result MatchResult) error {
	match, ok := g.findMatch(matchID)
	if !ok {
		return errMatchNotFound(matchID.String())
	}
	if err := match.validateResult(result); err != nil {
		return err
	}

	// Aplicar resultado al partido
	match.Result = &result
	match.Status = MatchStatusCompleted

	// Recalcular tabla de posiciones con todos los resultados del grupo
	g.recalculateStandings()

	// Actualizar estado del grupo
	if g.Status == GroupStatusPending {
		g.Status = GroupStatusInProgress
	}
	if g.IsComplete() {
		g.Status = GroupStatusCompleted
	}

	return nil
}

// recalculateStandings recalcula la tabla de posiciones desde cero
// procesando todos los resultados completados del grupo.
func (g *Group) recalculateStandings() {
	// Inicializar standings con todos los equipos en 0
	standings := make([]GroupStanding, teamsPerGroup)
	for i, teamID := range g.Teams {
		standings[i] = GroupStanding{TeamID: teamID}
	}

	// Aplicar cada resultado completado
	for _, m := range g.Matches {
		if m.IsCompleted() && m.Result != nil {
			standings = applyMatchToStandings(standings, *m.Result)
		}
	}

	g.Standings = standings
}

// HasTeam reporta si el equipo dado pertenece a este grupo.
func (g Group) HasTeam(teamID uuid.UUID) bool {
	for _, t := range g.Teams {
		if t == teamID {
			return true
		}
	}
	return false
}
