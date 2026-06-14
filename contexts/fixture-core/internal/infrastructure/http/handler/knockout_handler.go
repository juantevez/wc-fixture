package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// KnockoutHandler maneja los endpoints del bracket eliminatorio.
type KnockoutHandler struct {
	getKnockout      *queries.GetKnockoutHandler
	getKnockoutRound *queries.GetKnockoutRoundHandler
}

func NewKnockoutHandler(
	getKnockout *queries.GetKnockoutHandler,
	getKnockoutRound *queries.GetKnockoutRoundHandler,
) *KnockoutHandler {
	return &KnockoutHandler{
		getKnockout:      getKnockout,
		getKnockoutRound: getKnockoutRound,
	}
}

// GetKnockout retorna el bracket eliminatorio completo.
//
//	GET /api/v1/tournaments/{tournamentID}/knockout
func (h *KnockoutHandler) GetKnockout(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getKnockout.Handle(r.Context(), queries.GetKnockoutQuery{
		TournamentID: tournamentID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, dto)
}

// GetKnockoutRound retorna los partidos de una ronda eliminatoria específica.
//
//	GET /api/v1/tournaments/{tournamentID}/knockout/{phase}
//	Valores válidos de {phase}: round-of-32, quarterfinals, semifinals, third-place, final
func (h *KnockoutHandler) GetKnockoutRound(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	phase, err := parseKnockoutPhase(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	matches, err := h.getKnockoutRound.Handle(r.Context(), queries.GetKnockoutRoundQuery{
		TournamentID: tournamentID,
		Phase:        phase,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, matches)
}

// parseKnockoutPhase convierte el path param {phase} (kebab-case) al valor
// de dominio (UPPER_SNAKE_CASE) que espera el query handler.
//
//	round-of-32  → ROUND_OF_32
//	quarterfinals → QUARTERFINAL
//	semifinals    → SEMIFINAL
//	third-place   → THIRD_PLACE
//	final         → FINAL
func parseKnockoutPhase(r *http.Request) (string, error) {
	raw := chi.URLParam(r, "phase")

	mapping := map[string]string{
		"round-of-32":  "ROUND_OF_32",
		"quarterfinals": "QUARTERFINAL",
		"semifinals":   "SEMIFINAL",
		"third-place":  "THIRD_PLACE",
		"final":        "FINAL",
	}

	phase, ok := mapping[raw]
	if !ok {
		return "", apperrors.ValidationF(
			"phase %q inválida: use round-of-32, quarterfinals, semifinals, third-place o final", raw,
		)
	}
	return phase, nil
}
