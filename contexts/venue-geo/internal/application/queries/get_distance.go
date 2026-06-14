package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/venue-geo/internal/domain/ports"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
)

// ── DistanceDTO ───────────────────────────────────────────────────────────────

// DistanceDTO retorna la distancia entre dos venues.
type DistanceDTO struct {
	FromVenueID uuid.UUID `json:"from_venue_id"`
	ToVenueID   uuid.UUID `json:"to_venue_id"`
	DistanceKm  float64   `json:"distance_km"`
}

// DistanceMatrixEntryDTO es una entrada de la matriz completa de distancias.
type DistanceMatrixEntryDTO struct {
	FromVenueID uuid.UUID `json:"from_venue_id"`
	ToVenueID   uuid.UUID `json:"to_venue_id"`
	DistanceKm  float64   `json:"distance_km"`
}

// ── GetDistance ───────────────────────────────────────────────────────────────

// GetDistanceQuery solicita la distancia entre dos venues específicos.
type GetDistanceQuery struct {
	FromVenueID uuid.UUID
	ToVenueID   uuid.UUID
}

func (q GetDistanceQuery) validate() error {
	if q.FromVenueID == uuid.Nil {
		return apperrors.Validation("from_venue_id es requerido")
	}
	if q.ToVenueID == uuid.Nil {
		return apperrors.Validation("to_venue_id es requerido")
	}
	if q.FromVenueID == q.ToVenueID {
		return apperrors.Validation("from_venue_id y to_venue_id deben ser distintos")
	}
	return nil
}

// GetDistanceHandler retorna la distancia en km entre dos venues.
// Usa la tabla venue_distances (cache precalculado con PostGIS).
type GetDistanceHandler struct {
	repo ports.VenueRepository
}

func NewGetDistanceHandler(repo ports.VenueRepository) *GetDistanceHandler {
	return &GetDistanceHandler{repo: repo}
}

func (h *GetDistanceHandler) Handle(ctx context.Context, q GetDistanceQuery) (*DistanceDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	distKm, err := h.repo.GetDistance(ctx, q.FromVenueID, q.ToVenueID)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("distancia consultada",
		"from", q.FromVenueID, "to", q.ToVenueID, "km", distKm,
	)

	return &DistanceDTO{
		FromVenueID: q.FromVenueID,
		ToVenueID:   q.ToVenueID,
		DistanceKm:  distKm,
	}, nil
}

// ── GetDistanceMatrix ─────────────────────────────────────────────────────────

// GetDistanceMatrixQuery solicita la matriz completa de distancias entre venues.
type GetDistanceMatrixQuery struct{}

// GetDistanceMatrixHandler retorna la matriz completa de distancias.
// El resultado es una lista plana de pares (from, to, km) para facilitar
// serialización JSON. Solo incluye pares únicos (from < to).
type GetDistanceMatrixHandler struct {
	repo ports.VenueRepository
}

func NewGetDistanceMatrixHandler(repo ports.VenueRepository) *GetDistanceMatrixHandler {
	return &GetDistanceMatrixHandler{repo: repo}
}

func (h *GetDistanceMatrixHandler) Handle(ctx context.Context, _ GetDistanceMatrixQuery) ([]DistanceMatrixEntryDTO, error) {
	matrix, err := h.repo.GetDistanceMatrix(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]DistanceMatrixEntryDTO, 0, len(matrix))
	for key, distKm := range matrix {
		entries = append(entries, DistanceMatrixEntryDTO{
			FromVenueID: key[0],
			ToVenueID:   key[1],
			DistanceKm:  distKm,
		})
	}

	logger.FromContext(ctx).Debug("matriz de distancias consultada", "entradas", len(entries))
	return entries, nil
}

// ── GetNearbyVenues ───────────────────────────────────────────────────────────

// GetNearbyVenuesQuery solicita venues dentro de un radio dado.
type GetNearbyVenuesQuery struct {
	Center   venue.GeoPoint
	RadiusKm float64
}

func (q GetNearbyVenuesQuery) validate() error {
	if err := q.Center.Validate(); err != nil {
		return apperrors.ValidationF("coordenadas inválidas: %v", err)
	}
	if q.RadiusKm <= 0 || q.RadiusKm > 20000 {
		return apperrors.ValidationF("radius_km debe estar entre 0 y 20000, se recibió %.2f", q.RadiusKm)
	}
	return nil
}

// GetNearbyVenuesHandler retorna venues cercanos a una coordenada.
// Usa ST_DWithin de PostGIS para el filtro geoespacial eficiente.
type GetNearbyVenuesHandler struct {
	repo ports.VenueRepository
}

func NewGetNearbyVenuesHandler(repo ports.VenueRepository) *GetNearbyVenuesHandler {
	return &GetNearbyVenuesHandler{repo: repo}
}

func (h *GetNearbyVenuesHandler) Handle(ctx context.Context, q GetNearbyVenuesQuery) ([]NearbyVenueDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	nearby, err := h.repo.FindNearby(ctx, q.Center, q.RadiusKm)
	if err != nil {
		return nil, err
	}

	dtos := make([]NearbyVenueDTO, len(nearby))
	for i, n := range nearby {
		dtos[i] = NearbyVenueDTO{
			VenueDTO:   toVenueDTO(n.Venue),
			DistanceKm: n.DistanceKm,
		}
	}

	logger.FromContext(ctx).Debug("venues cercanos consultados",
		"lat", q.Center.Lat, "lon", q.Center.Lon,
		"radius_km", q.RadiusKm, "count", len(dtos),
	)
	return dtos, nil
}
