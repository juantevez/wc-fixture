// Package handler contiene los handlers HTTP del bounded context venue-geo.
package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
	"github.com/wc-fixture/venue-geo/internal/application/queries"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
)

// VenueHandler maneja todos los endpoints de venue-geo.
type VenueHandler struct {
	listVenues      *queries.ListVenuesHandler
	getVenue        *queries.GetVenueHandler
	getDistance     *queries.GetDistanceHandler
	getMatrix       *queries.GetDistanceMatrixHandler
	getNearby       *queries.GetNearbyVenuesHandler
}

func NewVenueHandler(
	listVenues *queries.ListVenuesHandler,
	getVenue *queries.GetVenueHandler,
	getDistance *queries.GetDistanceHandler,
	getMatrix *queries.GetDistanceMatrixHandler,
	getNearby *queries.GetNearbyVenuesHandler,
) *VenueHandler {
	return &VenueHandler{
		listVenues:  listVenues,
		getVenue:    getVenue,
		getDistance: getDistance,
		getMatrix:   getMatrix,
		getNearby:   getNearby,
	}
}

// ListVenues retorna todos los venues, filtrable por país.
//
//	GET /api/v1/venues?country=USA
func (h *VenueHandler) ListVenues(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")

	dtos, err := h.listVenues.Handle(r.Context(), queries.ListVenuesQuery{Country: country})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dtos)
}

// GetVenue retorna el detalle de un venue por su ID.
//
//	GET /api/v1/venues/{venueID}
func (h *VenueHandler) GetVenue(w http.ResponseWriter, r *http.Request) {
	venueID, err := parseVenueID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getVenue.Handle(r.Context(), queries.GetVenueQuery{VenueID: venueID})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dto)
}

// GetDistance retorna la distancia en km entre dos venues.
//
//	GET /api/v1/venues/distance?from={venueID}&to={venueID}
func (h *VenueHandler) GetDistance(w http.ResponseWriter, r *http.Request) {
	fromID, err := parseUUIDParam(r, "from")
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	toID, err := parseUUIDParam(r, "to")
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getDistance.Handle(r.Context(), queries.GetDistanceQuery{
		FromVenueID: fromID,
		ToVenueID:   toID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dto)
}

// GetDistanceMatrix retorna la matriz completa de distancias entre venues.
//
//	GET /api/v1/venues/distance-matrix
func (h *VenueHandler) GetDistanceMatrix(w http.ResponseWriter, r *http.Request) {
	entries, err := h.getMatrix.Handle(r.Context(), queries.GetDistanceMatrixQuery{})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, entries)
}

// GetNearbyVenues retorna venues dentro de un radio dado de una coordenada.
//
//	GET /api/v1/venues/nearby?lat={lat}&lon={lon}&radius_km={km}
func (h *VenueHandler) GetNearbyVenues(w http.ResponseWriter, r *http.Request) {
	lat, err := parseFloat(r, "lat")
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	lon, err := parseFloat(r, "lon")
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	radiusKm, err := parseFloat(r, "radius_km")
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dtos, err := h.getNearby.Handle(r.Context(), queries.GetNearbyVenuesQuery{
		Center:   venue.GeoPoint{Lat: lat, Lon: lon},
		RadiusKm: radiusKm,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dtos)
}

// ── Helpers de parseo ─────────────────────────────────────────────────────────

func parseVenueID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "venueID")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.ValidationF("venueID %q no es un UUID válido", raw)
	}
	return id, nil
}

func parseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return uuid.Nil, apperrors.ValidationF("parámetro %q es requerido", param)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.ValidationF("parámetro %q %q no es un UUID válido", param, raw)
	}
	return id, nil
}

func parseFloat(r *http.Request, param string) (float64, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return 0, apperrors.ValidationF("parámetro %q es requerido", param)
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, apperrors.ValidationF("parámetro %q %q no es un número válido", param, raw)
	}
	return v, nil
}
