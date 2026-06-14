// Package fixture contiene el modelo de dominio del bounded context fixture-core.
// El aggregate root es Fixture, que gestiona el estado completo del torneo:
// grupos, partidos, resultados, clasificación y bracket eliminatorio.
package fixture

import "fmt"

// ── Errores de dominio ────────────────────────────────────────────────────────
// Son errores con semántica de negocio, distintos de los apperrors de aplicación.
// El command handler los convierte a apperrors antes de retornarlos al caller.

// DomainError es la base de todos los errores de dominio del fixture.
type DomainError struct {
	Code    DomainErrCode
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

type DomainErrCode string

const (
	ErrCodeMatchNotFound          DomainErrCode = "MATCH_NOT_FOUND"
	ErrCodeMatchAlreadyCompleted  DomainErrCode = "MATCH_ALREADY_COMPLETED"
	ErrCodeMatchNotInProgress     DomainErrCode = "MATCH_NOT_IN_PROGRESS"
	ErrCodeGroupNotFound          DomainErrCode = "GROUP_NOT_FOUND"
	ErrCodeGroupAlreadyCompleted  DomainErrCode = "GROUP_ALREADY_COMPLETED"
	ErrCodeInvalidPhaseTransition DomainErrCode = "INVALID_PHASE_TRANSITION"
	ErrCodeInvalidResult          DomainErrCode = "INVALID_RESULT"
	ErrCodeSlotNotResolved        DomainErrCode = "SLOT_NOT_RESOLVED"
	ErrCodeTournamentNotStarted   DomainErrCode = "TOURNAMENT_NOT_STARTED"
	ErrCodeTournamentAlreadyEnded DomainErrCode = "TOURNAMENT_ALREADY_ENDED"
	ErrCodeGroupStageNotComplete  DomainErrCode = "GROUP_STAGE_NOT_COMPLETE"
	ErrCodeDuplicateResult        DomainErrCode = "DUPLICATE_RESULT"
	ErrCodeInvalidTeamCount       DomainErrCode = "INVALID_TEAM_COUNT"
)

// ── Constructores de errores de dominio ───────────────────────────────────────

func errMatchNotFound(matchID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeMatchNotFound,
		Message: fmt.Sprintf("partido %q no encontrado en el fixture", matchID),
	}
}

func errMatchAlreadyCompleted(matchID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeMatchAlreadyCompleted,
		Message: fmt.Sprintf("el partido %q ya tiene un resultado registrado", matchID),
	}
}

func errGroupNotFound(name string) *DomainError {
	return &DomainError{
		Code:    ErrCodeGroupNotFound,
		Message: fmt.Sprintf("grupo %q no encontrado en el fixture", name),
	}
}

func errGroupAlreadyCompleted(name string) *DomainError {
	return &DomainError{
		Code:    ErrCodeGroupAlreadyCompleted,
		Message: fmt.Sprintf("el grupo %q ya está completado", name),
	}
}

func errInvalidPhaseTransition(from, to TournamentStatus) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidPhaseTransition,
		Message: fmt.Sprintf("transición de estado inválida: %s → %s", from, to),
	}
}

func errInvalidResult(reason string) *DomainError {
	return &DomainError{Code: ErrCodeInvalidResult, Message: reason}
}

func errSlotNotResolved(matchID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeSlotNotResolved,
		Message: fmt.Sprintf("el slot del partido %q aún no está resuelto", matchID),
	}
}

func errGroupStageNotComplete() *DomainError {
	return &DomainError{
		Code:    ErrCodeGroupStageNotComplete,
		Message: "no todos los grupos han completado sus partidos",
	}
}

func errDuplicateResult(matchID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeDuplicateResult,
		Message: fmt.Sprintf("ya existe un resultado para el partido %q", matchID),
	}
}
