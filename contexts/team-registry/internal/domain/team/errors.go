// Package team contiene el modelo de dominio del bounded context team-registry.
// Gestiona equipos nacionales, confederaciones y su participación en torneos.
package team

import "fmt"

// DomainError es el error tipado del dominio de team-registry.
type DomainError struct {
	Code    ErrCode
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type ErrCode string

const (
	ErrCodeTeamNotFound          ErrCode = "TEAM_NOT_FOUND"
	ErrCodeConfederationNotFound ErrCode = "CONFEDERATION_NOT_FOUND"
	ErrCodeInvalidFIFARanking    ErrCode = "INVALID_FIFA_RANKING"
	ErrCodeDuplicateTeam         ErrCode = "DUPLICATE_TEAM"
)

func ErrTeamNotFound(id string) *DomainError {
	return &DomainError{Code: ErrCodeTeamNotFound, Message: fmt.Sprintf("equipo %q no encontrado", id)}
}

func ErrConfederationNotFound(code string) *DomainError {
	return &DomainError{Code: ErrCodeConfederationNotFound, Message: fmt.Sprintf("confederación %q no encontrada", code)}
}

func ErrInvalidFIFARanking(rank int) *DomainError {
	return &DomainError{Code: ErrCodeInvalidFIFARanking, Message: fmt.Sprintf("ranking FIFA inválido: %d (debe ser > 0)", rank)}
}
