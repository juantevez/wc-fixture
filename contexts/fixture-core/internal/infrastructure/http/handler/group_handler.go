package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// GroupHandler maneja los endpoints de grupos, standings y mejores terceros.
type GroupHandler struct {
	listGroups    *queries.ListGroupsHandler
	getGroup      *queries.GetGroupHandler
	getStandings  *queries.GetStandingsHandler
	getBestThirds *queries.GetBestThirdsHandler
}

func NewGroupHandler(
	listGroups *queries.ListGroupsHandler,
	getGroup *queries.GetGroupHandler,
	getStandings *queries.GetStandingsHandler,
	getBestThirds *queries.GetBestThirdsHandler,
) *GroupHandler {
	return &GroupHandler{
		listGroups:    listGroups,
		getGroup:      getGroup,
		getStandings:  getStandings,
		getBestThirds: getBestThirds,
	}
}

// ListGroups retorna los 12 grupos con equipos, partidos y standings.
//
//	GET /api/v1/tournaments/{tournamentID}/groups
func (h *GroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	groups, err := h.listGroups.Handle(r.Context(), queries.ListGroupsQuery{
		TournamentID: tournamentID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, groups)
}

// GetGroup retorna el detalle de un grupo específico (A–L).
//
//	GET /api/v1/tournaments/{tournamentID}/groups/{groupName}
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	groupName, err := parseGroupName(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getGroup.Handle(r.Context(), queries.GetGroupQuery{
		TournamentID: tournamentID,
		GroupName:    groupName,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, dto)
}

// GetStandings retorna la tabla de posiciones de un grupo.
//
//	GET /api/v1/tournaments/{tournamentID}/groups/{groupName}/standings
func (h *GroupHandler) GetStandings(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	groupName, err := parseGroupName(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	standings, err := h.getStandings.Handle(r.Context(), queries.GetStandingsQuery{
		TournamentID: tournamentID,
		GroupName:    groupName,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, standings)
}

// GetBestThirds retorna el ranking de mejores terceros clasificados.
//
//	GET /api/v1/tournaments/{tournamentID}/best-thirds
func (h *GroupHandler) GetBestThirds(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	thirds, err := h.getBestThirds.Handle(r.Context(), queries.GetBestThirdsQuery{
		TournamentID: tournamentID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, thirds)
}

// parseGroupName extrae y valida el path param {groupName}.
func parseGroupName(r *http.Request) (string, error) {
	name := chi.URLParam(r, "groupName")
	if !queries.ValidGroupName(name) {
		return "", apperrors.ValidationF("groupName %q inválido: debe ser A–L", name)
	}
	return name, nil
}
