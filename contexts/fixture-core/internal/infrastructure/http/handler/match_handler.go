package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/application/commands"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// MatchHandler maneja los endpoints de partidos.
type MatchHandler struct {
	listMatches    *queries.ListMatchesHandler
	getMatch       *queries.GetMatchHandler
	registerResult *commands.RegisterMatchResultHandler
	updateSchedule *commands.UpdateMatchScheduleHandler
}

func NewMatchHandler(
	listMatches *queries.ListMatchesHandler,
	getMatch *queries.GetMatchHandler,
	registerResult *commands.RegisterMatchResultHandler,
	updateSchedule *commands.UpdateMatchScheduleHandler,
) *MatchHandler {
	return &MatchHandler{
		listMatches:    listMatches,
		getMatch:       getMatch,
		registerResult: registerResult,
		updateSchedule: updateSchedule,
	}
}

// ListMatches retorna partidos paginados con filtros opcionales.
//
//	GET /api/v1/tournaments/{tournamentID}/matches
//	?phase=GROUP&status=COMPLETED&group=A&date_from=2026-06-01&date_to=2026-06-30
//	&page=1&per_page=20
func (h *MatchHandler) ListMatches(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	page, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	filters, err := parseMatchFilters(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	result, err := h.listMatches.Handle(r.Context(), queries.ListMatchesQuery{
		TournamentID: tournamentID,
		Filters:      filters,
		Page:         page,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WritePagedJSON(w, result.Matches, result.Meta)
}

// GetMatch retorna el detalle de un partido.
//
//	GET /api/v1/tournaments/{tournamentID}/matches/{matchID}
func (h *MatchHandler) GetMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseMatchID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	dto, err := h.getMatch.Handle(r.Context(), queries.GetMatchQuery{
		MatchID: matchID,
	})
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteOK(w, dto)
}

// RegisterResult registra el resultado de un partido.
// Endpoint interno — llamado desde result-ingestion o admin.
//
//	POST /api/v1/tournaments/{tournamentID}/matches/{matchID}/result
func (h *MatchHandler) RegisterResult(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	matchID, err := parseMatchID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	var body registerResultBody
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	cmd, err := body.toCommand(tournamentID, matchID)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	if err := h.registerResult.Handle(r.Context(), cmd); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteNoContent(w)
}

// UpdateSchedule modifica el horario y/o venue de un partido.
//
//	PUT /api/v1/tournaments/{tournamentID}/matches/{matchID}/schedule
func (h *MatchHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := parseTournamentID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	matchID, err := parseMatchID(r)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	var body updateScheduleBody
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	cmd, err := body.toCommand(tournamentID, matchID)
	if err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	if err := h.updateSchedule.Handle(r.Context(), cmd); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteNoContent(w)
}

// ── Request bodies ────────────────────────────────────────────────────────────

type registerResultBody struct {
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeGoals    int    `json:"home_goals"`
	AwayGoals    int    `json:"away_goals"`
	HomeGoalsET  *int   `json:"home_goals_et,omitempty"`
	AwayGoalsET  *int   `json:"away_goals_et,omitempty"`
	HomeGoalsPen *int   `json:"home_goals_pen,omitempty"`
	AwayGoalsPen *int   `json:"away_goals_pen,omitempty"`
	CompletedAt  string `json:"completed_at"` // RFC3339
	RegisteredBy string `json:"registered_by"`
}

func (b registerResultBody) toCommand(tournamentID, matchID uuid.UUID) (commands.RegisterMatchResultCmd, error) {
	homeTeamID, err := uuid.Parse(b.HomeTeamID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, apperrors.ValidationF("home_team_id inválido: %v", err)
	}
	awayTeamID, err := uuid.Parse(b.AwayTeamID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, apperrors.ValidationF("away_team_id inválido: %v", err)
	}
	completedAt, err := time.Parse(time.RFC3339, b.CompletedAt)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, apperrors.ValidationF("completed_at inválido (use RFC3339): %v", err)
	}

	return commands.RegisterMatchResultCmd{
		TournamentID: tournamentID,
		MatchID:      matchID,
		HomeTeamID:   homeTeamID,
		AwayTeamID:   awayTeamID,
		HomeGoals:    b.HomeGoals,
		AwayGoals:    b.AwayGoals,
		HomeGoalsET:  b.HomeGoalsET,
		AwayGoalsET:  b.AwayGoalsET,
		HomeGoalsPen: b.HomeGoalsPen,
		AwayGoalsPen: b.AwayGoalsPen,
		CompletedAt:  completedAt,
		RegisteredBy: b.RegisteredBy,
	}, nil
}

type updateScheduleBody struct {
	NewScheduledAt string `json:"new_scheduled_at"` // RFC3339
	NewVenueID     string `json:"new_venue_id"`
	Reason         string `json:"reason"`
}

func (b updateScheduleBody) toCommand(tournamentID, matchID uuid.UUID) (commands.UpdateMatchScheduleCmd, error) {
	newVenueID, err := uuid.Parse(b.NewVenueID)
	if err != nil {
		return commands.UpdateMatchScheduleCmd{}, apperrors.ValidationF("new_venue_id inválido: %v", err)
	}
	newScheduledAt, err := time.Parse(time.RFC3339, b.NewScheduledAt)
	if err != nil {
		return commands.UpdateMatchScheduleCmd{}, apperrors.ValidationF("new_scheduled_at inválido (use RFC3339): %v", err)
	}

	return commands.UpdateMatchScheduleCmd{
		TournamentID:   tournamentID,
		MatchID:        matchID,
		NewScheduledAt: newScheduledAt,
		NewVenueID:     newVenueID,
		Reason:         b.Reason,
	}, nil
}

// ── Helpers de filtros ────────────────────────────────────────────────────────

// parseMatchFilters extrae los query params de filtrado del request.
func parseMatchFilters(r *http.Request) (queries.MatchFilters, error) {
	q := r.URL.Query()
	filters := queries.MatchFilters{
		Phase:     q.Get("phase"),
		Status:    q.Get("status"),
		GroupName: q.Get("group"),
	}

	if v := q.Get("venue_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filters, apperrors.ValidationF("venue_id %q no es un UUID válido", v)
		}
		filters.VenueID = id
	}

	if v := q.Get("team_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filters, apperrors.ValidationF("team_id %q no es un UUID válido", v)
		}
		filters.TeamID = id
	}

	if v := q.Get("date_from"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return filters, apperrors.ValidationF("date_from %q inválido (use YYYY-MM-DD)", v)
		}
		filters.DateFrom = t
	}

	if v := q.Get("date_to"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return filters, apperrors.ValidationF("date_to %q inválido (use YYYY-MM-DD)", v)
		}
		filters.DateTo = t.Add(24*time.Hour - time.Second) // fin del día
	}

	return filters, nil
}
