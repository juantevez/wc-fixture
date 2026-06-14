// Package ports define las interfaces de salida del bounded context venue-geo.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
)

// VenueRepository es el puerto de salida para persistir y recuperar venues.
type VenueRepository interface {
	// GetByID retorna un venue por su ID.
	// Retorna apperrors.NotFound si no existe.
	GetByID(ctx context.Context, id uuid.UUID) (*venue.Venue, error)

	// List retorna todos los venues, opcionalmente filtrados por país.
	List(ctx context.Context, country venue.Country) ([]venue.Venue, error)

	// Save persiste un venue nuevo o actualiza uno existente.
	Save(ctx context.Context, v venue.Venue) error

	// GetDistanceMatrix retorna la matriz completa de distancias precalculadas.
	GetDistanceMatrix(ctx context.Context) (venue.DistanceMatrix, error)

	// SaveDistanceMatrix persiste la matriz completa de distancias.
	SaveDistanceMatrix(ctx context.Context, matrix venue.DistanceMatrix) error

	// GetDistance retorna la distancia precalculada entre dos venues.
	// Retorna error si alguno de los IDs no existe.
	GetDistance(ctx context.Context, fromID, toID uuid.UUID) (float64, error)

	// FindNearby retorna los venues dentro de un radio en km de una coordenada,
	// ordenados por distancia ascendente.
	FindNearby(ctx context.Context, center venue.GeoPoint, radiusKm float64) ([]venue.NearbyVenue, error)
}
