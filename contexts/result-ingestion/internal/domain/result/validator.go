package result

import (
	"time"

	"github.com/google/uuid"
)

// ValidationResult agrupa los errores de validación de un resultado.
// Permite retornar todos los errores de una vez en lugar de uno por uno.
type ValidationResult struct {
	Errors []string
}

func (v *ValidationResult) Add(msg string) {
	v.Errors = append(v.Errors, msg)
}

func (v *ValidationResult) IsValid() bool {
	return len(v.Errors) == 0
}

// Validator aplica reglas de validación adicionales sobre IngestedResult
// que requieren contexto externo (ej: verificar que el match existe,
// que los equipos son correctos para ese partido, etc.).
//
// Las validaciones de dominio puro (goles negativos, penales sin ET, etc.)
// se aplican en result.New(). Este validador se usa en el command handler
// para validaciones que requieren datos externos.
type Validator struct{}

// ValidateIDs verifica que los IDs principales tengan formato UUID válido.
// Se usa como primera validación en el HTTP handler antes de construir el dominio.
func (v Validator) ValidateIDs(
	tournamentID, matchID, homeTeamID, awayTeamID string,
) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
			&DomainError{Code: ErrCodeInvalidTournament, Message: "tournament_id no es un UUID válido"}
	}
	mID, err := uuid.Parse(matchID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
			&DomainError{Code: ErrCodeInvalidMatchID, Message: "match_id no es un UUID válido"}
	}
	hID, err := uuid.Parse(homeTeamID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
			ErrInvalidTeams("home_team_id no es un UUID válido")
	}
	aID, err := uuid.Parse(awayTeamID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
			ErrInvalidTeams("away_team_id no es un UUID válido")
	}
	return tID, mID, hID, aID, nil
}

// ValidateCompletedAt parsea y valida la fecha de completado.
func (v Validator) ValidateCompletedAt(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, &DomainError{
			Code:    ErrCodeFutureCompletedAt,
			Message: "completed_at inválido: use formato RFC3339 (ej: 2026-06-15T20:30:00Z)",
		}
	}
	return t.UTC(), nil
}
