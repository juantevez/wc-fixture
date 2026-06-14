package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// CloseGroupStageCmd fuerza el cierre de la fase de grupos.
// En condiciones normales este comando NO es necesario — el aggregate
// detecta automáticamente cuando todos los grupos están completos y
// genera el bracket en RegisterMatchResult.
//
// Se expone como comando explícito para casos de administración:
// corrección de datos, cierre manual por decisión operativa, o testing.
type CloseGroupStageCmd struct {
	TournamentID uuid.UUID
	Force        bool // true = cerrar aunque algún grupo no esté completo (admin only)
}

func (c CloseGroupStageCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	return nil
}

// CloseGroupStageHandler cierra la fase de grupos y genera el bracket eliminatorio.
// Requiere que todos los grupos estén completos (salvo Force == true).
type CloseGroupStageHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewCloseGroupStageHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *CloseGroupStageHandler {
	return &CloseGroupStageHandler{repo: repo, publisher: pub}
}

func (h *CloseGroupStageHandler) Handle(ctx context.Context, cmd CloseGroupStageCmd) error {
	log := logger.WithFields(ctx,
		"handler", "CloseGroupStage",
		"tournament_id", cmd.TournamentID,
		"force", cmd.Force,
	)

	if err := cmd.validate(); err != nil {
		return err
	}

	f, err := h.repo.GetByTournamentID(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}

	if !f.AllGroupsCompleted() && !cmd.Force {
		return apperrors.Conflict("no todos los grupos han completado sus partidos; use force=true para forzar el cierre")
	}

	// El aggregate genera el bracket internamente si aún no lo hizo.
	// Si ya está en StatusKnockout, este comando es idempotente.
	evts := f.PendingEvents()

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el cierre de fase de grupos", err)
	}

	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando eventos de cierre de fase de grupos", "error", err)
	}

	log.Info("fase de grupos cerrada", "status", f.Status)
	return nil
}
