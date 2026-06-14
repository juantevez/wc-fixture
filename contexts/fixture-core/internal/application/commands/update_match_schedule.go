package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// UpdateMatchScheduleCmd modifica el horario y/o venue de un partido programado.
// Solo aplica a partidos con status SCHEDULED o POSTPONED.
// No se permite modificar partidos ya completados.
type UpdateMatchScheduleCmd struct {
	TournamentID   uuid.UUID
	MatchID        uuid.UUID
	NewScheduledAt time.Time
	NewVenueID     uuid.UUID
	Reason         string // motivo del cambio, para auditoría
}

func (c UpdateMatchScheduleCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if c.MatchID == uuid.Nil {
		return apperrors.Validation("match_id es requerido")
	}
	if c.NewVenueID == uuid.Nil {
		return apperrors.Validation("new_venue_id es requerido")
	}
	if c.NewScheduledAt.IsZero() {
		return apperrors.Validation("new_scheduled_at es requerido")
	}
	if c.NewScheduledAt.Before(time.Now().UTC()) {
		return apperrors.Validation("new_scheduled_at debe ser una fecha futura")
	}
	return nil
}

// UpdateMatchScheduleHandler modifica el horario y venue de un partido.
type UpdateMatchScheduleHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewUpdateMatchScheduleHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *UpdateMatchScheduleHandler {
	return &UpdateMatchScheduleHandler{repo: repo, publisher: pub}
}

func (h *UpdateMatchScheduleHandler) Handle(ctx context.Context, cmd UpdateMatchScheduleCmd) error {
	log := logger.WithFields(ctx,
		"handler", "UpdateMatchSchedule",
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

	if err := f.UpdateMatchSchedule(cmd.MatchID, cmd.NewScheduledAt, cmd.NewVenueID); err != nil {
		return mapDomainError(err)
	}

	evts := f.PendingEvents()

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el cambio de horario", err)
	}

	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando evento de cambio de horario", "error", err)
	}

	log.Info("horario actualizado",
		"new_scheduled_at", cmd.NewScheduledAt,
		"new_venue_id", cmd.NewVenueID,
		"reason", cmd.Reason,
	)
	return nil
}
