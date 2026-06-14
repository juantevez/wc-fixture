package fixture

import "github.com/google/uuid"

// SlotKind indica cómo se resuelve el equipo que ocupa un slot en un partido.
type SlotKind string

const (
	// SlotKindTeam — el equipo ya está determinado (fase de grupos o slot resuelto).
	SlotKindTeam SlotKind = "TEAM"

	// SlotKindWinnerOf — el equipo es el ganador del partido referenciado.
	SlotKindWinnerOf SlotKind = "WINNER_OF"

	// SlotKindLoserOf — el equipo es el perdedor del partido referenciado.
	// Solo se usa para el partido por el tercer puesto.
	SlotKindLoserOf SlotKind = "LOSER_OF"

	// SlotKindBestThird — el equipo es uno de los mejores terceros clasificados.
	// Se resuelve al cerrar la fase de grupos según la política BestThirdsPolicy.
	SlotKindBestThird SlotKind = "BEST_THIRD"
)

// MatchSlot representa una posición (local o visitante) en un partido.
// Puede estar ya resuelta (SlotKindTeam) o ser una referencia dinámica
// que se resuelve cuando el partido o la fase anterior finaliza.
//
// Invariante: solo el campo correspondiente al Kind debe estar seteado.
type MatchSlot struct {
	Kind SlotKind

	// TeamID está presente cuando Kind == SlotKindTeam.
	TeamID *uuid.UUID

	// SourceMatchID está presente cuando Kind == SlotKindWinnerOf o SlotKindLoserOf.
	// Referencia el partido del que proviene el clasificado.
	SourceMatchID *uuid.UUID

	// GroupRef está presente cuando Kind == SlotKindBestThird.
	// Identifica de qué grupo y posición proviene el equipo.
	GroupRef *GroupRef
}

// GroupRef identifica la posición de un equipo dentro de un grupo.
// Se usa para referencias dinámicas en el bracket (ej: "1° del grupo A").
type GroupRef struct {
	GroupName string // "A" … "L"
	Position  int    // 1 = primero, 2 = segundo, 3 = tercero
}

// IsResolved reporta si el slot ya tiene un equipo concreto asignado.
func (s MatchSlot) IsResolved() bool {
	return s.Kind == SlotKindTeam && s.TeamID != nil
}

// Resolve retorna un nuevo MatchSlot con el equipo concreto asignado.
// Se llama cuando el partido o fase origen finaliza.
func (s MatchSlot) Resolve(teamID uuid.UUID) MatchSlot {
	return MatchSlot{Kind: SlotKindTeam, TeamID: &teamID}
}

// ── Constructores ─────────────────────────────────────────────────────────────

// SlotForTeam crea un slot ya resuelto para un equipo concreto.
func SlotForTeam(teamID uuid.UUID) MatchSlot {
	return MatchSlot{Kind: SlotKindTeam, TeamID: &teamID}
}

// SlotWinnerOf crea un slot dinámico que se resolverá con el ganador del partido dado.
func SlotWinnerOf(matchID uuid.UUID) MatchSlot {
	return MatchSlot{Kind: SlotKindWinnerOf, SourceMatchID: &matchID}
}

// SlotLoserOf crea un slot dinámico que se resolverá con el perdedor del partido dado.
func SlotLoserOf(matchID uuid.UUID) MatchSlot {
	return MatchSlot{Kind: SlotKindLoserOf, SourceMatchID: &matchID}
}

// SlotBestThird crea un slot para un mejor tercero de un grupo específico.
func SlotBestThird(groupName string) MatchSlot {
	return MatchSlot{
		Kind:     SlotKindBestThird,
		GroupRef: &GroupRef{GroupName: groupName, Position: 3},
	}
}

// SlotFromGroup crea un slot dinámico para el clasificado de una posición en un grupo.
// Ejemplo: SlotFromGroup("A", 1) → primer clasificado del grupo A.
func SlotFromGroup(groupName string, position int) MatchSlot {
	return MatchSlot{
		Kind:     SlotKindTeam, // se resuelve al cerrar la fase de grupos
		GroupRef: &GroupRef{GroupName: groupName, Position: position},
	}
}
