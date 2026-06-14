// Package venue contiene el modelo de dominio del bounded context venue-geo.
// Gestiona estadios, coordenadas geoespaciales y distancias entre venues.
package venue

import "fmt"

// DomainError es el error tipado del dominio de venue-geo.
type DomainError struct {
	Code    ErrCode
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type ErrCode string

const (
	ErrCodeVenueNotFound      ErrCode = "VENUE_NOT_FOUND"
	ErrCodeInvalidCoordinates ErrCode = "INVALID_COORDINATES"
	ErrCodeInvalidRadius      ErrCode = "INVALID_RADIUS"
	ErrCodeSameVenue          ErrCode = "SAME_VENUE"
)

func ErrVenueNotFound(id string) *DomainError {
	return &DomainError{Code: ErrCodeVenueNotFound, Message: fmt.Sprintf("venue %q no encontrado", id)}
}

func ErrInvalidCoordinates(lat, lon float64) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidCoordinates,
		Message: fmt.Sprintf("coordenadas inválidas: lat=%.6f lon=%.6f (lat debe estar en [-90,90], lon en [-180,180])", lat, lon),
	}
}

func ErrInvalidRadius(radius float64) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidRadius,
		Message: fmt.Sprintf("radio inválido: %.2f km (debe ser > 0 y <= 20000 km)", radius),
	}
}

func ErrSameVenue() *DomainError {
	return &DomainError{
		Code:    ErrCodeSameVenue,
		Message: "los dos venues deben ser distintos para calcular distancia",
	}
}
