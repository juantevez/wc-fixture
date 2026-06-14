// Package queries contiene los query handlers del bounded context venue-geo.
package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
	"github.com/wc-fixture/venue-geo/internal/domain/ports"
)

// ── DTOs ──────────────────────────────────────────────────────────────────────

// VenueDTO es la representación serializable de un venue para la API REST.
type VenueDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	City        string    `json:"city"`
	Country     string    `json:"country"`
	CountryCode string    `json:"country_code"`
	Capacity    int       `json:"capacity"`
	Surface     string    `json:"surface"`
	Location    GeoPointDTO `json:"location"`
	Timezone    string    `json:"timezone"`
	AltitudeM   int       `json:"altitude_m"`
}

// GeoPointDTO es la representación serializable de una coordenada.
type GeoPointDTO struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// NearbyVenueDTO es un VenueDTO enriquecido con la distancia al punto de búsqueda.
type NearbyVenueDTO struct {
	VenueDTO
	DistanceKm float64 `json:"distance_km"`
}

// toVenueDTO convierte el entity de dominio al DTO de respuesta.
func toVenueDTO(v venue.Venue) VenueDTO {
	return VenueDTO{
		ID:          v.ID,
		Name:        v.Name,
		City:        v.City,
		Country:     string(v.Country),
		CountryCode: v.CountryCode,
		Capacity:    v.Capacity,
		Surface:     string(v.Surface),
		Location:    GeoPointDTO{Lat: v.Location.Lat, Lon: v.Location.Lon},
		Timezone:    v.Timezone,
		AltitudeM:   v.AltitudeM,
	}
}

// ── GetVenue ──────────────────────────────────────────────────────────────────

// GetVenueQuery solicita el detalle de un venue por su ID.
type GetVenueQuery struct {
	VenueID uuid.UUID
}

// GetVenueHandler retorna el detalle completo de un venue.
type GetVenueHandler struct {
	repo ports.VenueRepository
}

func NewGetVenueHandler(repo ports.VenueRepository) *GetVenueHandler {
	return &GetVenueHandler{repo: repo}
}

func (h *GetVenueHandler) Handle(ctx context.Context, q GetVenueQuery) (*VenueDTO, error) {
	if q.VenueID == uuid.Nil {
		return nil, apperrors.Validation("venue_id es requerido")
	}

	v, err := h.repo.GetByID(ctx, q.VenueID)
	if err != nil {
		return nil, err
	}

	dto := toVenueDTO(*v)
	logger.FromContext(ctx).Debug("venue consultado", "venue_id", q.VenueID, "name", v.Name)
	return &dto, nil
}

// ── ListVenues ────────────────────────────────────────────────────────────────

// ListVenuesQuery solicita todos los venues con filtro opcional por país.
type ListVenuesQuery struct {
	Country string // "USA" | "CAN" | "MEX" | "" (todos)
}

// ListVenuesHandler retorna los venues del torneo, opcionalmente filtrados por país.
type ListVenuesHandler struct {
	repo ports.VenueRepository
}

func NewListVenuesHandler(repo ports.VenueRepository) *ListVenuesHandler {
	return &ListVenuesHandler{repo: repo}
}

func (h *ListVenuesHandler) Handle(ctx context.Context, q ListVenuesQuery) ([]VenueDTO, error) {
	country := venue.Country(q.Country)

	// Validar el país si se especificó
	if q.Country != "" {
		switch country {
		case venue.CountryUSA, venue.CountryCanada, venue.CountryMexico:
		default:
			return nil, apperrors.ValidationF("country %q inválido: use USA, CAN o MEX", q.Country)
		}
	}

	venues, err := h.repo.List(ctx, country)
	if err != nil {
		return nil, err
	}

	dtos := make([]VenueDTO, len(venues))
	for i, v := range venues {
		dtos[i] = toVenueDTO(v)
	}

	logger.FromContext(ctx).Debug("venues listados", "count", len(dtos), "country_filter", q.Country)
	return dtos, nil
}
