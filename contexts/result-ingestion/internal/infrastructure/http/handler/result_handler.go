// Package handler contiene los handlers HTTP de result-ingestion.
package handler

import (
	"net/http"

	"github.com/wc-fixture/result-ingestion/internal/application/commands"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// ResultHandler maneja el endpoint de ingesta de resultados.
type ResultHandler struct {
	ingestResult *commands.IngestResultHandler
}

func NewResultHandler(ingestResult *commands.IngestResultHandler) *ResultHandler {
	return &ResultHandler{ingestResult: ingestResult}
}

// IngestResult recibe un resultado de partido y lo publica en NATS.
//
//	POST /api/v1/results
//
// Body:
//
//	{
//	  "tournament_id": "uuid",
//	  "match_id":      "uuid",
//	  "home_team_id":  "uuid",
//	  "away_team_id":  "uuid",
//	  "home_goals":    3,
//	  "away_goals":    1,
//	  "home_goals_et": null,
//	  "away_goals_et": null,
//	  "home_goals_pen": null,
//	  "away_goals_pen": null,
//	  "completed_at": "2026-06-15T22:15:00Z",
//	  "source": "fifa_api"
//	}
func (h *ResultHandler) IngestResult(w http.ResponseWriter, r *http.Request) {
	var body ingestResultBody
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	cmd := commands.IngestResultCmd{
		TournamentID: body.TournamentID,
		MatchID:      body.MatchID,
		HomeTeamID:   body.HomeTeamID,
		AwayTeamID:   body.AwayTeamID,
		HomeGoals:    body.HomeGoals,
		AwayGoals:    body.AwayGoals,
		HomeGoalsET:  body.HomeGoalsET,
		AwayGoalsET:  body.AwayGoalsET,
		HomeGoalsPen: body.HomeGoalsPen,
		AwayGoalsPen: body.AwayGoalsPen,
		CompletedAt:  body.CompletedAt,
		Source:       body.Source,
	}

	if err := h.ingestResult.Handle(r.Context(), cmd); err != nil {
		httputil.WriteError(r.Context(), w, r, err)
		return
	}

	httputil.WriteNoContent(w)
}

// ingestResultBody es el schema del request body.
type ingestResultBody struct {
	TournamentID string `json:"tournament_id"`
	MatchID      string `json:"match_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeGoals    int    `json:"home_goals"`
	AwayGoals    int    `json:"away_goals"`
	HomeGoalsET  *int   `json:"home_goals_et"`
	AwayGoalsET  *int   `json:"away_goals_et"`
	HomeGoalsPen *int   `json:"home_goals_pen"`
	AwayGoalsPen *int   `json:"away_goals_pen"`
	CompletedAt  string `json:"completed_at"`
	Source       string `json:"source"`
}
