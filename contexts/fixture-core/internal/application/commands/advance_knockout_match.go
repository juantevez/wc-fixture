package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// AdvanceKnockoutMatchCmd registra el resultado de un partido eliminatorio
// y propaga automáticamente al ganador al siguiente partido del bracket.
//
// Nota: este comando es conceptualmente idéntico a RegisterMatchResultCmd
// para partidos eliminatorios. Se mantiene separado para mayor claridad
// semántica y para facilitar permisos diferenciados en la API
// (avanzar un partido eliminatorio requiere validaciones adicionales).
type AdvanceKnockoutMatchCmd struct {
	TournamentID uuid.UUID
	MatchID      uuid.UUID

	HomeTeamID uuid.UUID
	AwayTeamID uuid.UUID

	HomeGoals int
	AwayGoals int

	// Extra time — obligatorio si hay empate en 90 minutos
	HomeGoalsET *int
	AwayGoalsET *int

	// Penales — obligatorio si hay empate en ET
	HomeGoalsPen *int
	AwayGoalsPen *int

	CompletedAt  time.Time
	RegisteredBy string
}

func (c AdvanceKnockoutMatchCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if c.MatchID == uuid.Nil {
		return apperrors.Validation("match_id es requerido")
	}
	if c.HomeTeamID == uuid.Nil || c.AwayTeamID == uuid.Nil {
		return apperrors.Validation("home_team_id y away_team_id son requeridos")
	}
	if c.HomeTeamID == c.AwayTeamID {
		return apperrors.Validation("home_team_id y away_team_id no pueden ser iguales")
	}
	if c.HomeGoals < 0 || c.AwayGoals < 0 {
		return apperrors.Validation("los goles no pueden ser negativos")
	}
	// En eliminatoria el empate en 90' requiere ET
	if c.HomeGoals == c.AwayGoals && c.HomeGoalsET == nil {
		return apperrors.Validation("empate en tiempo regular: se requiere informar el tiempo extra")
	}
	if c.CompletedAt.IsZero() {
		return apperrors.Validation("completed_at es requerido")
	}
	return nil
}

// AdvanceKnockoutMatchHandler registra el resultado de un partido eliminatorio.
// Internamente delega a RegisterMatchResult del aggregate, que maneja
// la propagación del ganador al siguiente partido.
type AdvanceKnockoutMatchHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewAdvanceKnockoutMatchHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *AdvanceKnockoutMatchHandler {
	return &AdvanceKnockoutMatchHandler{repo: repo, publisher: pub}
}

func (h *AdvanceKnockoutMatchHandler) Handle(ctx context.Context, cmd AdvanceKnockoutMatchCmd) error {
	log := logger.WithFields(ctx,
		"handler", "AdvanceKnockoutMatch",
		"tournament_id", cmd.TournamentID,
		"match_id", cmd.MatchID,
	)

	if err := cmd.validate(); err != nil {
		return err
	}

	f, err := h.repo.GetByTournamentID(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}

	// Reutilizamos RegisterMatchResultCmd internamente — evita duplicar
	// la lógica de construcción de MatchResult.
	innerCmd := RegisterMatchResultCmd{
		TournamentID: cmd.TournamentID,
		MatchID:      cmd.MatchID,
		HomeTeamID:   cmd.HomeTeamID,
		AwayTeamID:   cmd.AwayTeamID,
		HomeGoals:    cmd.HomeGoals,
		AwayGoals:    cmd.AwayGoals,
		HomeGoalsET:  cmd.HomeGoalsET,
		AwayGoalsET:  cmd.AwayGoalsET,
		HomeGoalsPen: cmd.HomeGoalsPen,
		AwayGoalsPen: cmd.AwayGoalsPen,
		CompletedAt:  cmd.CompletedAt,
		RegisteredBy: cmd.RegisteredBy,
	}

	result := innerCmd.toMatchResult()

	if err := f.RegisterMatchResult(cmd.MatchID, result); err != nil {
		return mapDomainError(err)
	}

	evts := f.PendingEvents()

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el resultado eliminatorio", err)
	}

	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando eventos de avance en bracket", "error", err)
	}

	log.Info("partido eliminatorio avanzado",
		"home_goals", cmd.HomeGoals,
		"away_goals", cmd.AwayGoals,
		"status", f.Status,
		"eventos", len(evts),
	)
	return nil
}
