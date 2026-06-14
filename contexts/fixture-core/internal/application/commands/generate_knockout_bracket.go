package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// GenerateKnockoutBracketCmd solicita la (re)generación del bracket eliminatorio.
// En condiciones normales el bracket se genera automáticamente dentro de
// RegisterMatchResult al completarse la fase de grupos.
//
// Este comando es útil para:
//   - Regenerar el bracket tras una corrección de resultados
//   - Forzar la generación en entornos de testing/staging
//   - Recuperación ante fallos de persistencia del bracket
type GenerateKnockoutBracketCmd struct {
	TournamentID uuid.UUID
}

func (c GenerateKnockoutBracketCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	return nil
}

// GenerateKnockoutBracketHandler genera el bracket eliminatorio.
// Precondición: todos los grupos deben estar completados.
type GenerateKnockoutBracketHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewGenerateKnockoutBracketHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *GenerateKnockoutBracketHandler {
	return &GenerateKnockoutBracketHandler{repo: repo, publisher: pub}
}

func (h *GenerateKnockoutBracketHandler) Handle(ctx context.Context, cmd GenerateKnockoutBracketCmd) error {
	log := logger.WithFields(ctx,
		"handler", "GenerateKnockoutBracket",
		"tournament_id", cmd.TournamentID,
	)

	if err := cmd.validate(); err != nil {
		return err
	}

	f, err := h.repo.GetByTournamentID(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}

	if !f.AllGroupsCompleted() {
		return apperrors.Conflict("la fase de grupos no está completa; no se puede generar el bracket")
	}

	if f.Status == fixture.StatusKnockout || f.Status == fixture.StatusFinished {
		log.Info("bracket ya generado, comando idempotente", "status", f.Status)
		return nil
	}

	// El aggregate gestiona la generación internamente.
	// Este handler solo orquesta: carga, persiste y publica.
	evts := f.PendingEvents()

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el bracket eliminatorio", err)
	}

	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando eventos de generación de bracket", "error", err)
	}

	log.Info("bracket eliminatorio generado", "rondas", len(f.KnockoutRounds))
	return nil
}
