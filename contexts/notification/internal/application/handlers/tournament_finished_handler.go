package handlers

import (
	"context"
	"encoding/json"

	"github.com/wc-fixture/notification/internal/domain/notification"
	"github.com/wc-fixture/notification/internal/domain/ports"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// TournamentFinishedHandler procesa el evento TournamentFinished
// y envía la notificación de campeón a todos los suscriptores.
type TournamentFinishedHandler struct {
	notifier ports.Notifier
}

func NewTournamentFinishedHandler(notifier ports.Notifier) *TournamentFinishedHandler {
	return &TournamentFinishedHandler{notifier: notifier}
}

func (h *TournamentFinishedHandler) Handle(ctx context.Context, evt sharedevents.DomainEvent) error {
	log := logger.WithFields(ctx,
		"handler", "TournamentFinishedHandler",
		"tournament_id", evt.AggregateID,
	)

	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	n := notification.New(
		notification.KindTournamentFinished,
		evt.AggregateID,
		payload,
	)

	if err := h.notifier.Notify(ctx, n); err != nil {
		log.Error("error notificando fin del torneo", "error", err)
		return err
	}

	log.Info("fin de torneo notificado", "notification_id", n.ID)
	return nil
}

// ── Dispatcher — enruta eventos al handler correcto ──────────────────────────

// EventDispatcher enruta eventos de NATS al handler correspondiente
// según el EventType del envelope.
type EventDispatcher struct {
	handlers map[string]func(context.Context, sharedevents.DomainEvent) error
}

// NewEventDispatcher construye el dispatcher con todos los handlers registrados.
func NewEventDispatcher(
	matchResult     *MatchResultHandler,
	bracketGenerated *BracketGeneratedHandler,
	tournamentFinished *TournamentFinishedHandler,
) *EventDispatcher {
	d := &EventDispatcher{
		handlers: make(map[string]func(context.Context, sharedevents.DomainEvent) error),
	}

	// Registrar handlers por EventType
	d.handlers["MatchResultRegistered"]    = matchResult.Handle
	d.handlers["GroupStageCompleted"]      = matchResult.Handle      // reutiliza notifier
	d.handlers["KnockoutBracketGenerated"] = bracketGenerated.Handle
	d.handlers["KnockoutMatchAdvanced"]    = matchResult.Handle      // reutiliza notifier
	d.handlers["MatchScheduleUpdated"]     = matchResult.Handle      // reutiliza notifier
	d.handlers["TournamentFinished"]       = tournamentFinished.Handle

	return d
}

// Dispatch enruta el evento al handler correspondiente.
// Si no hay handler registrado para el EventType, el evento se ignora (skip).
func (d *EventDispatcher) Dispatch(ctx context.Context, evt sharedevents.DomainEvent) error {
	handler, ok := d.handlers[evt.EventType]
	if !ok {
		logger.FromContext(ctx).Debug("evento sin handler — ignorado",
			"event_type", evt.EventType,
		)
		return nil
	}
	return handler(ctx, evt)
}
