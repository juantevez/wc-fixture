// Package handler contiene los handlers HTTP de fixture-core.
// Cada handler es responsable de:
//  1. Extraer y validar parámetros del request (path params, query params, body)
//  2. Construir el query o comando correspondiente
//  3. Delegar al handler de aplicación
//  4. Escribir la respuesta con httputil
//
// Los handlers NO contienen lógica de negocio — solo traducción HTTP ↔ aplicación.
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// FixtureHandler maneja los endpoints del fixture completo del torneo.
type FixtureHandler struct {
	getFixture *queries.GetFixtureHandler
}

func NewFixtureHandler(getFixture *queries.GetFixtureHandler) *FixtureHandler {
	return &FixtureHandler{getFixture: getFixture}
}

// GetFixture godoc
//
//	@Summary     Fixture completo del torneo
//	@Description Retorna grupos, partidos, standings y bracket eliminatorio
//	@Tags        fixture
//	@Produce     json
//	@Param       tournamentID path string true "ID del torneo (UUID)"
//	@Success     200 {object} queries.FixtureDTO
//	@Failure     404 {object} httputil.errorBody
//	@Router      /api/v1/tournaments/{tournamentID}/fixture [get]
func (h *FixtureHandler) GetFixture(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getFixture.Handle(r.Context(), queries.GetFixtureQuery{
		TournamentID: tournamentID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, dto)
}

// ── Helpers compartidos entre handlers ────────────────────────────────────────

// parseTournamentID extrae y valida el path param {tournamentID}.
func parseTournamentID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "tournamentID")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.ValidationF("tournamentID %q no es un UUID válido", raw)
	}
	return id, nil
}

// parseMatchID extrae y valida el path param {matchID}.
func parseMatchID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "matchID")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.ValidationF("matchID %q no es un UUID válido", raw)
	}
	return id, nil
}
