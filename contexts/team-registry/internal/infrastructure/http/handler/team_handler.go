// Package handler contiene los handlers HTTP del bounded context team-registry.
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
	"github.com/wc-fixture/team-registry/internal/application/queries"
)

// TeamHandler maneja todos los endpoints de team-registry.
type TeamHandler struct {
	getTeam             *queries.GetTeamHandler
	listTeams           *queries.ListTeamsHandler
	getConfederation    *queries.GetConfederationHandler
	listConfederations  *queries.ListConfederationsHandler
}

func NewTeamHandler(
	getTeam *queries.GetTeamHandler,
	listTeams *queries.ListTeamsHandler,
	getConfederation *queries.GetConfederationHandler,
	listConfederations *queries.ListConfederationsHandler,
) *TeamHandler {
	return &TeamHandler{
		getTeam:            getTeam,
		listTeams:          listTeams,
		getConfederation:   getConfederation,
		listConfederations: listConfederations,
	}
}

// ListTeams retorna equipos con filtros opcionales.
//
//	GET /api/v1/teams?confederation=UEFA&qualified=true
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	qualifiedOnly := q.Get("qualified") == "true"

	dtos, err := h.listTeams.Handle(r.Context(), queries.ListTeamsQuery{
		Confederation: q.Get("confederation"),
		QualifiedOnly: qualifiedOnly,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dtos)
}

// GetTeam retorna el detalle de un equipo por su UUID.
//
//	GET /api/v1/teams/{teamID}
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "teamID")
	id, err := uuid.Parse(raw)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, apperrors.ValidationF("teamID %q no es un UUID válido", raw))
		return
	}

	dto, err := h.getTeam.Handle(r.Context(), queries.GetTeamQuery{TeamID: id})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dto)
}

// ListConfederations retorna las 6 confederaciones FIFA con sus cupos WC2026.
//
//	GET /api/v1/confederations
func (h *TeamHandler) ListConfederations(w http.ResponseWriter, r *http.Request) {
	dtos, err := h.listConfederations.Handle(r.Context(), queries.ListConfederationsQuery{})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dtos)
}

// GetConfederation retorna el detalle de una confederación por código.
//
//	GET /api/v1/confederations/{code}
func (h *TeamHandler) GetConfederation(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	dto, err := h.getConfederation.Handle(r.Context(), queries.GetConfederationQuery{Code: code})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}
	httputil.WriteOK(w, dto)
}
