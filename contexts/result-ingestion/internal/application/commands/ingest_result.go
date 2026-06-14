// Package commands contiene los command handlers de result-ingestion.
package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/result-ingestion/internal/domain/ports"
	"github.com/wc-fixture/result-ingestion/internal/domain/result"
)

// IngestResultCmd transporta los datos del resultado recibido desde la fuente externa.
type IngestResultCmd struct {
	TournamentID string
	MatchID      string
	HomeTeamID   string
	AwayTeamID   string

	HomeGoals int
	AwayGoals int

	HomeGoalsET  *int
	AwayGoalsET  *int
	HomeGoalsPen *int
	AwayGoalsPen *int

	CompletedAt  string // RFC3339
	Source       string // "fifa_api" | "manual" | "webhook"
}

// IngestResultHandler valida el resultado, verifica idempotencia
// y lo publica en NATS JetStream para que fixture-core lo procese.
//
// Flujo:
//  1. Parsear y validar IDs y fecha
//  2. Construir IngestedResult (valida reglas de dominio)
//  3. Verificar idempotencia — si ya existe, retornar sin error (operación idempotente)
//  4. Registrar la clave de idempotencia
//  5. Publicar evento ResultIngested en NATS
type IngestResultHandler struct {
	publisher  ports.EventPublisher
	idempotency ports.IdempotencyStore
	validator  result.Validator
}

func NewIngestResultHandler(
	publisher ports.EventPublisher,
	idempotency ports.IdempotencyStore,
) *IngestResultHandler {
	return &IngestResultHandler{
		publisher:   publisher,
		idempotency: idempotency,
		validator:   result.Validator{},
	}
}

func (h *IngestResultHandler) Handle(ctx context.Context, cmd IngestResultCmd) error {
	log := logger.WithFields(ctx,
		"handler", "IngestResult",
		"match_id", cmd.MatchID,
		"source", cmd.Source,
	)

	// ── 1. Parsear y validar IDs ───────────────────────────────────────────────
	tournamentID, matchID, homeTeamID, awayTeamID, err := h.validator.ValidateIDs(
		cmd.TournamentID, cmd.MatchID, cmd.HomeTeamID, cmd.AwayTeamID,
	)
	if err != nil {
		return mapDomainError(err)
	}

	completedAt, err := h.validator.ValidateCompletedAt(cmd.CompletedAt)
	if err != nil {
		return mapDomainError(err)
	}

	// ── 2. Construir el resultado de dominio (valida reglas) ───────────────────
	source := result.IngestionSource(cmd.Source)
	if source == "" {
		source = result.SourceManual
	}

	ingestedResult, err := result.New(
		tournamentID, matchID, homeTeamID, awayTeamID,
		cmd.HomeGoals, cmd.AwayGoals,
		cmd.HomeGoalsET, cmd.AwayGoalsET,
		cmd.HomeGoalsPen, cmd.AwayGoalsPen,
		completedAt, source,
	)
	if err != nil {
		return mapDomainError(err)
	}

	// ── 3. Verificar idempotencia ─────────────────────────────────────────────
	exists, err := h.idempotency.Exists(ctx, ingestedResult.IdempotencyKey)
	if err != nil {
		log.Error("error verificando idempotencia", "error", err)
		// No bloqueamos por error de idempotencia — mejor publicar duplicado
		// que perder un resultado. fixture-core maneja duplicados.
	}
	if exists {
		log.Info("resultado duplicado ignorado",
			"idempotency_key", ingestedResult.IdempotencyKey,
		)
		return nil // idempotente: no es un error
	}

	// ── 4. Registrar clave de idempotencia ────────────────────────────────────
	if err := h.idempotency.Register(ctx, ingestedResult.IdempotencyKey); err != nil {
		var de *result.DomainError
		if errors.As(err, &de) && de.Code == result.ErrCodeDuplicateIngestion {
			log.Info("race condition en idempotencia — resultado ya registrado")
			return nil
		}
		log.Error("error registrando idempotencia", "error", err)
		// Continuar — mejor publicar que perder el resultado
	}

	// ── 5. Construir y publicar el evento ─────────────────────────────────────
	evt, err := buildResultIngestedEvent(ingestedResult)
	if err != nil {
		return apperrors.Internal("error construyendo evento de resultado", err)
	}

	if err := h.publisher.Publish(ctx, evt); err != nil {
		return apperrors.Unavailable(
			fmt.Sprintf("error publicando resultado del partido %s en NATS", matchID),
		)
	}

	log.Info("resultado ingestado y publicado",
		"tournament_id", tournamentID,
		"home_goals", cmd.HomeGoals,
		"away_goals", cmd.AwayGoals,
		"has_et", ingestedResult.HasExtraTime(),
		"has_pen", ingestedResult.HasPenalties(),
	)
	return nil
}

// buildResultIngestedEvent construye el DomainEvent con el payload del resultado.
func buildResultIngestedEvent(r *result.IngestedResult) (sharedevents.DomainEvent, error) {
	payload := resultIngestedPayload{
		TournamentID: r.TournamentID.String(),
		MatchID:      r.MatchID.String(),
		HomeTeamID:   r.HomeTeamID.String(),
		AwayTeamID:   r.AwayTeamID.String(),
		HomeGoals:    r.HomeGoals,
		AwayGoals:    r.AwayGoals,
		HomeGoalsET:  r.HomeGoalsET,
		AwayGoalsET:  r.AwayGoalsET,
		HomeGoalsPen: r.HomeGoalsPen,
		AwayGoalsPen: r.AwayGoalsPen,
		CompletedAt:  r.CompletedAt.Format(time.RFC3339),
		RegisteredBy: string(r.Source),
	}

	return sharedevents.New(
		"ResultIngested",
		r.TournamentID,
		"IngestedResult",
		payload,
	)
}

// resultIngestedPayload es el schema del payload que fixture-core espera.
// Debe mantenerse sincronizado con result_consumer.go en fixture-core.
type resultIngestedPayload struct {
	TournamentID string `json:"tournament_id"`
	MatchID      string `json:"match_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeGoals    int    `json:"home_goals"`
	AwayGoals    int    `json:"away_goals"`
	HomeGoalsET  *int   `json:"home_goals_et,omitempty"`
	AwayGoalsET  *int   `json:"away_goals_et,omitempty"`
	HomeGoalsPen *int   `json:"home_goals_pen,omitempty"`
	AwayGoalsPen *int   `json:"away_goals_pen,omitempty"`
	CompletedAt  string `json:"completed_at"`
	RegisteredBy string `json:"registered_by"`
}

// mapDomainError convierte DomainError a AppError para el layer HTTP.
func mapDomainError(err error) error {
	var de *result.DomainError
	if !errors.As(err, &de) {
		return apperrors.Internal("error inesperado", err)
	}

	switch de.Code {
	case result.ErrCodeInvalidGoals,
		result.ErrCodeInvalidTeams,
		result.ErrCodeInvalidMatchID,
		result.ErrCodeInvalidTournament,
		result.ErrCodeInvalidExtraTime,
		result.ErrCodeInvalidPenalties,
		result.ErrCodeFutureCompletedAt:
		return apperrors.Validation(de.Message)

	case result.ErrCodeDuplicateIngestion:
		return apperrors.Conflict(de.Message)

	default:
		return apperrors.Internal("error de dominio no mapeado", err)
	}
}

// Tipos de ID para construcción segura
var _ = uuid.Nil
