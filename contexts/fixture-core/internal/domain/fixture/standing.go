package fixture

import "github.com/google/uuid"

// GroupStanding representa la posición de un equipo en la tabla de su grupo.
// Es un value object: se recalcula completo cada vez que se registra un resultado.
// No tiene identidad propia — su identidad es (GroupID, TeamID).
type GroupStanding struct {
	TeamID   uuid.UUID
	Position int // 1 a 4, calculado al ordenar

	// Estadísticas acumuladas
	Played       int
	Won          int
	Drawn        int
	Lost         int
	GoalsFor     int
	GoalsAgainst int
	YellowCards  int
	RedCards     int
}

// Points retorna los puntos acumulados según el sistema FIFA (3/1/0).
func (s GroupStanding) Points() int {
	return s.Won*3 + s.Drawn
}

// GoalDifference retorna la diferencia de goles (GoalsFor - GoalsAgainst).
func (s GroupStanding) GoalDifference() int {
	return s.GoalsFor - s.GoalsAgainst
}

// FairPlayPoints retorna los puntos de fair play acumulados.
// FIFA usa: tarjeta amarilla = 1, roja directa = 3, amarilla+roja = 4.
// Se usa como criterio de desempate (menor es mejor).
func (s GroupStanding) FairPlayPoints() int {
	return s.YellowCards*1 + s.RedCards*3
}

// ── Ordenamiento de standings ─────────────────────────────────────────────────

// StandingLess reporta si a debe ir antes que b en la tabla del grupo.
// Implementa los criterios de desempate FIFA en cascada:
//  1. Mayor cantidad de puntos
//  2. Mayor diferencia de goles
//  3. Mayor cantidad de goles a favor
//  4. Menor puntos de fair play
//
// Nota: los criterios 5 y 6 (desempate directo entre empatados y ranking FIFA)
// se aplican a nivel del aggregate Fixture ya que requieren contexto adicional.
func StandingLess(a, b GroupStanding) bool {
	if a.Points() != b.Points() {
		return a.Points() > b.Points()
	}
	if a.GoalDifference() != b.GoalDifference() {
		return a.GoalDifference() > b.GoalDifference()
	}
	if a.GoalsFor != b.GoalsFor {
		return a.GoalsFor > b.GoalsFor
	}
	return a.FairPlayPoints() < b.FairPlayPoints()
}

// SortStandings ordena el slice de standings in-place según los criterios FIFA.
// Usa insertion sort — con 4 elementos siempre es O(1) en la práctica.
func SortStandings(standings []GroupStanding) {
	for i := 1; i < len(standings); i++ {
		for j := i; j > 0 && StandingLess(standings[j], standings[j-1]); j-- {
			standings[j], standings[j-1] = standings[j-1], standings[j]
		}
	}
}

// applyMatchToStandings actualiza los standings de ambos equipos dado un resultado.
// Retorna los standings actualizados (no modifica el slice original).
func applyMatchToStandings(standings []GroupStanding, result MatchResult) []GroupStanding {
	updated := make([]GroupStanding, len(standings))
	copy(updated, standings)

	for i := range updated {
		switch updated[i].TeamID {
		case result.HomeTeamID:
			updated[i] = applyResultToStanding(updated[i], result.HomeGoals, result.AwayGoals)
		case result.AwayTeamID:
			updated[i] = applyResultToStanding(updated[i], result.AwayGoals, result.HomeGoals)
		}
	}

	SortStandings(updated)
	for i := range updated {
		updated[i].Position = i + 1
	}

	return updated
}

// applyResultToStanding aplica un resultado (goalsFor / goalsAgainst) al standing
// de un equipo y retorna el standing actualizado.
func applyResultToStanding(s GroupStanding, goalsFor, goalsAgainst int) GroupStanding {
	s.Played++
	s.GoalsFor += goalsFor
	s.GoalsAgainst += goalsAgainst

	switch {
	case goalsFor > goalsAgainst:
		s.Won++
	case goalsFor == goalsAgainst:
		s.Drawn++
	default:
		s.Lost++
	}

	return s
}
