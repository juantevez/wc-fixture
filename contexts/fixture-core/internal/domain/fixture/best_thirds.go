package fixture

import (
	"sort"

	"github.com/google/uuid"
)

// BestThirdsPolicy encapsula la lógica de selección de los 8 mejores terceros
// entre los 12 grupos del Mundial 2026.
//
// Problema central: los 12 terceros no jugaron todos contra los mismos rivales,
// por lo que no se pueden comparar directamente todos sus stats. La FIFA normaliza
// la comparación considerando solo los partidos contra el 1° y 2° de su grupo
// (descartando el partido contra el 4° de su grupo que es el más débil).
//
// Sin embargo, en el Mundial 2026 con grupos de 4 equipos y 3 partidos por equipo,
// FIFA usa la comparación directa de todos los partidos del grupo sin normalización,
// dado que todos los terceros juegan exactamente 3 partidos contra rivales del mismo
// formato. Esta política simplificada es la que implementamos aquí.
type BestThirdsPolicy struct{}

// BestThirdCandidate es el standing del tercer clasificado de un grupo,
// enriquecido con el nombre del grupo para trazabilidad.
type BestThirdCandidate struct {
	GroupStanding
	GroupName string
}

// Classify recibe los standings de los terceros de cada grupo (máx 12)
// y retorna los IDs de los 8 mejores, ordenados de mejor a peor.
//
// Criterios de desempate en cascada (mismos que dentro del grupo):
//  1. Mayor cantidad de puntos
//  2. Mayor diferencia de goles
//  3. Mayor cantidad de goles a favor
//  4. Menor puntos de fair play
//  5. Sorteo (no implementado — en producción lo gestiona FIFA manualmente)
func (p BestThirdsPolicy) Classify(thirds []BestThirdCandidate) []uuid.UUID {
	if len(thirds) < 8 {
		// Situación de error o torneo incompleto: retornar los que haya
		ids := make([]uuid.UUID, len(thirds))
		for i, t := range thirds {
			ids[i] = t.TeamID
		}
		return ids
	}

	sorted := make([]BestThirdCandidate, len(thirds))
	copy(sorted, thirds)

	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].GroupStanding, sorted[j].GroupStanding
		return StandingLess(a, b)
	})

	// Los 8 primeros son los clasificados
	result := make([]uuid.UUID, 8)
	for i := range 8 {
		result[i] = sorted[i].TeamID
	}
	return result
}

// BestThirdsAssignment mapea los 8 mejores terceros a los partidos de octavos
// según la tabla oficial FIFA 2026. La asignación depende de qué grupos
// aportaron los terceros clasificados.
//
// La lógica completa de asignación por combinación de grupos se implementa
// en el aggregate Fixture.generateKnockoutBracket() donde se tiene el contexto
// completo del torneo.
type BestThirdsAssignment struct {
	// ThirdTeamID es el ID del tercer clasificado asignado.
	ThirdTeamID uuid.UUID

	// TargetMatchID es el partido de octavos donde va como local o visitante.
	TargetMatchID uuid.UUID

	// AsHome indica si va como local (true) o visitante (false) en el partido.
	AsHome bool
}
