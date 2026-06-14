package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// RegisterMatchResultCmd transporta los datos del resultado a registrar.
// Proviene de result-ingestion vía evento NATS o llamada HTTP interna.
type RegisterMatchResultCmd struct {
	TournamentID uuid.UUID
	MatchID      uuid.UUID

	HomeGoals int
	AwayGoals int

	// Extra time — solo eliminatorias, nil en fase de grupos
	HomeGoalsET *int
	AwayGoalsET *int

	// Penales — solo cuando el partido termina en empate en ET
	HomeGoalsPen *int
	AwayGoalsPen *int

	// Metadatos del resultado
	HomeTeamID  uuid.UUID
	AwayTeamID  uuid.UUID
	CompletedAt time.Time
	RegisteredBy string // fuente: "fifa_api", "manual", "result-ingestion"
}

func (c RegisterMatchResultCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if c.MatchID == uuid.Nil {
		return apperrors.Validation("match_id es requerido")
	}
	if c.HomeTeamID == uuid.Nil {
		return apperrors.Validation("home_team_id es requerido")
	}
	if c.AwayTeamID == uuid.Nil {
		return apperrors.Validation("away_team_id es requerido")
	}
	if c.HomeTeamID == c.AwayTeamID {
		return apperrors.Validation("home_team_id y away_team_id no pueden ser iguales")
	}
	if c.HomeGoals < 0 || c.AwayGoals < 0 {
		return apperrors.Validation("los goles no pueden ser negativos")
	}
	if c.HomeGoalsET != nil && *c.HomeGoalsET < 0 {
		return apperrors.Validation("los goles en tiempo extra no pueden ser negativos")
	}
	if c.AwayGoalsET != nil && *c.AwayGoalsET < 0 {
		return apperrors.Validation("los goles en tiempo extra no pueden ser negativos")
	}
	// Si hay penales, debe haber ET y el ET debe estar empatado
	if c.HomeGoalsPen != nil || c.AwayGoalsPen != nil {
		if c.HomeGoalsET == nil || c.AwayGoalsET == nil {
			return apperrors.Validation("los penales requieren tiempo extra informado")
		}
		if *c.HomeGoalsET != *c.AwayGoalsET {
			return apperrors.Validation("los penales solo aplican cuando el tiempo extra termina empatado")
		}
		if c.HomeGoalsPen == nil || c.AwayGoalsPen == nil {
			return apperrors.Validation("deben informarse los penales de ambos equipos")
		}
		if *c.HomeGoalsPen == *c.AwayGoalsPen {
			return apperrors.Validation("los penales no pueden terminar empatados")
		}
	}
	if c.CompletedAt.IsZero() {
		return apperrors.Validation("completed_at es requerido")
	}
	return nil
}

// toMatchResult convierte el comando al value object de dominio.
func (c RegisterMatchResultCmd) toMatchResult() fixture.MatchResult {
	return fixture.MatchResult{
		HomeTeamID:   c.HomeTeamID,
		AwayTeamID:   c.AwayTeamID,
		HomeGoals:    c.HomeGoals,
		AwayGoals:    c.AwayGoals,
		HomeGoalsET:  c.HomeGoalsET,
		AwayGoalsET:  c.AwayGoalsET,
		HomeGoalsPen: c.HomeGoalsPen,
		AwayGoalsPen: c.AwayGoalsPen,
		CompletedAt:  c.CompletedAt,
	}
}

// RegisterMatchResultHandler es el command handler más ejecutado del sistema.
// Se invoca 104 veces por torneo (48 grupos + 32 + 8 + 4 + 1 + 1 + ...).
// Puede desencadenar la generación del bracket al completar la fase de grupos.
type RegisterMatchResultHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewRegisterMatchResultHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *RegisterMatchResultHandler {
	return &RegisterMatchResultHandler{repo: repo, publisher: pub}
}

func (h *RegisterMatchResultHandler) Handle(ctx context.Context, cmd RegisterMatchResultCmd) error {
	log := logger.WithFields(ctx,
		"handler", "RegisterMatchResult",
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

	result := cmd.toMatchResult()

	if err := f.RegisterMatchResult(cmd.MatchID, result); err != nil {
		// Convertir DomainError a AppError
		return mapDomainError(err)
	}

	evts := f.PendingEvents()

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el resultado", err)
	}

	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando eventos de resultado", "error", err, "evento_count", len(evts))
		// No retornamos error: estado persistido, eventos pueden reintentarse.
	}

	log.Info("resultado registrado",
		"home_goals", cmd.HomeGoals,
		"away_goals", cmd.AwayGoals,
		"registered_by", cmd.RegisteredBy,
		"eventos_publicados", len(evts),
	)
	return nil
}
