package fixture

import (
	"time"

	"github.com/google/uuid"
)

// ── Nombres canónicos de eventos de dominio ───────────────────────────────────
// Se usan como EventType en el envelope shared/pkg/events.DomainEvent.
const (
	EventTournamentInitialized    = "TournamentInitialized"
	EventMatchResultRegistered    = "MatchResultRegistered"
	EventGroupStageCompleted      = "GroupStageCompleted"
	EventKnockoutBracketGenerated = "KnockoutBracketGenerated"
	EventKnockoutMatchAdvanced    = "KnockoutMatchAdvanced"
	EventMatchScheduleUpdated     = "MatchScheduleUpdated"
	EventTournamentFinished       = "TournamentFinished"
)

// ── Payloads de eventos ───────────────────────────────────────────────────────
// Cada struct es el payload concreto que va dentro del envelope DomainEvent.
// Son parte del modelo de dominio de fixture-core — no se comparten con otros
// bounded contexts directamente; cada consumidor define su propio tipo receptor.

// TournamentInitializedPayload se emite al crear el torneo con su configuración inicial.
type TournamentInitializedPayload struct {
	TournamentID uuid.UUID `json:"tournament_id"`
	Edition      int       `json:"edition"`
	Name         string    `json:"name"`
	Groups       []string  `json:"groups"` // ["A","B",...,"L"]
	InitializedAt time.Time `json:"initialized_at"`
}

// MatchResultRegisteredPayload se emite cuando un resultado es registrado y validado.
type MatchResultRegisteredPayload struct {
	MatchID    uuid.UUID  `json:"match_id"`
	Phase      MatchPhase `json:"phase"`
	GroupName  string     `json:"group_name,omitempty"` // solo fase de grupos
	HomeTeamID uuid.UUID  `json:"home_team_id"`
	AwayTeamID uuid.UUID  `json:"away_team_id"`
	HomeGoals  int        `json:"home_goals"`
	AwayGoals  int        `json:"away_goals"`

	// Extra time y penales — omitempty para no incluir en partidos de grupos
	HomeGoalsET  *int `json:"home_goals_et,omitempty"`
	AwayGoalsET  *int `json:"away_goals_et,omitempty"`
	HomeGoalsPen *int `json:"home_goals_pen,omitempty"`
	AwayGoalsPen *int `json:"away_goals_pen,omitempty"`

	WinnerTeamID *uuid.UUID `json:"winner_team_id,omitempty"` // nil en empate de grupos
	CompletedAt  time.Time  `json:"completed_at"`
}

// GroupStageCompletedPayload se emite cuando todos los partidos de un grupo finalizan.
type GroupStageCompletedPayload struct {
	GroupName string          `json:"group_name"`
	Standings []StandingSnap  `json:"standings"`
	CompletedAt time.Time     `json:"completed_at"`
}

// StandingSnap es una snapshot serializable de un GroupStanding para eventos.
type StandingSnap struct {
	Position     int       `json:"position"`
	TeamID       uuid.UUID `json:"team_id"`
	Points       int       `json:"points"`
	Played       int       `json:"played"`
	Won          int       `json:"won"`
	Drawn        int       `json:"drawn"`
	Lost         int       `json:"lost"`
	GoalsFor     int       `json:"goals_for"`
	GoalsAgainst int       `json:"goals_against"`
	GoalDiff     int       `json:"goal_difference"`
}

// KnockoutBracketGeneratedPayload se emite cuando el bracket eliminatorio
// se genera tras completar la fase de grupos.
type KnockoutBracketGeneratedPayload struct {
	TournamentID       uuid.UUID     `json:"tournament_id"`
	BestThirdsSelected []uuid.UUID   `json:"best_thirds_selected"` // 8 equipos
	RoundOf32Matches   []MatchSnap   `json:"round_of_32_matches"`  // 16 partidos
	GeneratedAt        time.Time     `json:"generated_at"`
}

// MatchSnap es una snapshot serializable de un Match para eventos.
type MatchSnap struct {
	MatchID     uuid.UUID  `json:"match_id"`
	MatchNumber int        `json:"match_number"`
	Phase       MatchPhase `json:"phase"`
	HomeSlot    SlotSnap   `json:"home_slot"`
	AwaySlot    SlotSnap   `json:"away_slot"`
	VenueID     uuid.UUID  `json:"venue_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
}

// SlotSnap es una snapshot serializable de un MatchSlot para eventos.
type SlotSnap struct {
	Kind          SlotKind   `json:"kind"`
	TeamID        *uuid.UUID `json:"team_id,omitempty"`
	SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
}

// KnockoutMatchAdvancedPayload se emite cuando un partido eliminatorio finaliza
// y el ganador es asignado al siguiente partido del bracket.
type KnockoutMatchAdvancedPayload struct {
	CompletedMatchID uuid.UUID  `json:"completed_match_id"`
	WinnerTeamID     uuid.UUID  `json:"winner_team_id"`
	LoserTeamID      uuid.UUID  `json:"loser_team_id"`
	NextMatchID      *uuid.UUID `json:"next_match_id,omitempty"` // nil en la final
	Phase            MatchPhase `json:"phase"`
	AdvancedAt       time.Time  `json:"advanced_at"`
}

// MatchScheduleUpdatedPayload se emite cuando se modifica el horario o venue.
type MatchScheduleUpdatedPayload struct {
	MatchID        uuid.UUID  `json:"match_id"`
	OldScheduledAt time.Time  `json:"old_scheduled_at"`
	NewScheduledAt time.Time  `json:"new_scheduled_at"`
	OldVenueID     uuid.UUID  `json:"old_venue_id"`
	NewVenueID     uuid.UUID  `json:"new_venue_id"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TournamentFinishedPayload se emite cuando la final del torneo finaliza.
type TournamentFinishedPayload struct {
	TournamentID uuid.UUID `json:"tournament_id"`
	ChampionID   uuid.UUID `json:"champion_id"`
	RunnerUpID   uuid.UUID `json:"runner_up_id"`
	ThirdPlaceID uuid.UUID `json:"third_place_id"`
	FinalMatchID uuid.UUID `json:"final_match_id"`
	FinishedAt   time.Time `json:"finished_at"`
}

// ── Helper: snapshot de standing ──────────────────────────────────────────────

func toStandingSnap(s GroupStanding) StandingSnap {
	return StandingSnap{
		Position:     s.Position,
		TeamID:       s.TeamID,
		Points:       s.Points(),
		Played:       s.Played,
		Won:          s.Won,
		Drawn:        s.Drawn,
		Lost:         s.Lost,
		GoalsFor:     s.GoalsFor,
		GoalsAgainst: s.GoalsAgainst,
		GoalDiff:     s.GoalDifference(),
	}
}

// toMatchSnap convierte un Match a su snapshot para eventos.
func toMatchSnap(m Match) MatchSnap {
	homeSnap := SlotSnap{Kind: m.HomeSlot.Kind, TeamID: m.HomeSlot.TeamID, SourceMatchID: m.HomeSlot.SourceMatchID}
	awaySnap := SlotSnap{Kind: m.AwaySlot.Kind, TeamID: m.AwaySlot.TeamID, SourceMatchID: m.AwaySlot.SourceMatchID}
	return MatchSnap{
		MatchID:     m.ID,
		MatchNumber: m.MatchNumber,
		Phase:       m.Phase,
		HomeSlot:    homeSnap,
		AwaySlot:    awaySnap,
		VenueID:     m.VenueID,
		ScheduledAt: m.ScheduledAt,
	}
}
