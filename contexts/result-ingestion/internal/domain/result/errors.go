// Package result contiene el modelo de dominio del bounded context result-ingestion.
// Su responsabilidad es recibir, validar y publicar resultados de partidos
// hacia fixture-core vía NATS JetStream.
//
// result-ingestion es un Supporting Domain: no gestiona el estado del torneo
// (eso es fixture-core), solo garantiza que los resultados lleguen validados
// y con at-least-once delivery.
package result

import "fmt"

// DomainError es el error tipado del dominio de result-ingestion.
type DomainError struct {
	Code    ErrCode
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type ErrCode string

const (
	ErrCodeInvalidGoals       ErrCode = "INVALID_GOALS"
	ErrCodeInvalidTeams       ErrCode = "INVALID_TEAMS"
	ErrCodeInvalidMatchID     ErrCode = "INVALID_MATCH_ID"
	ErrCodeInvalidTournament  ErrCode = "INVALID_TOURNAMENT"
	ErrCodeInvalidExtraTime   ErrCode = "INVALID_EXTRA_TIME"
	ErrCodeInvalidPenalties   ErrCode = "INVALID_PENALTIES"
	ErrCodeDuplicateIngestion ErrCode = "DUPLICATE_INGESTION"
	ErrCodeFutureCompletedAt  ErrCode = "FUTURE_COMPLETED_AT"
)

func ErrInvalidGoals(msg string) *DomainError {
	return &DomainError{Code: ErrCodeInvalidGoals, Message: msg}
}

func ErrInvalidTeams(msg string) *DomainError {
	return &DomainError{Code: ErrCodeInvalidTeams, Message: msg}
}

func ErrInvalidExtraTime(msg string) *DomainError {
	return &DomainError{Code: ErrCodeInvalidExtraTime, Message: msg}
}

func ErrInvalidPenalties(msg string) *DomainError {
	return &DomainError{Code: ErrCodeInvalidPenalties, Message: msg}
}

func ErrDuplicateIngestion(matchID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeDuplicateIngestion,
		Message: fmt.Sprintf("ya existe una ingesta para el partido %q — ignorando duplicado", matchID),
	}
}
