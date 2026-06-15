// Package handlers contiene los handlers de eventos de dominio consumidos desde NATS.
// Cada handler procesa un tipo de evento y construye la notificación correspondiente.
//
// Diferencia de naming vs fixture-core:
// En fixture-core los "handlers" son command handlers.
// En notification los "handlers" son event handlers — procesan eventos entrantes.
package handlers

import (
	"context"
	"encoding/json"

	"github.com/wc-fixture/notification/internal/domain/notification"
	"github.com/wc-fixture/notification/internal/domain/ports"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// MatchResultHandler procesa el evento MatchResultRegistered
// y notifica a los suscriptores del torneo.
type MatchResultHandler struct {
	notifier ports.Notifier
}

func NewMatchResultHandler(notifier ports.Notifier) *MatchResultHandler {
	return &MatchResultHandler{notifier: notifier}
}

func (h *MatchResultHandler) Handle(ctx context.Context, evt sharedevents.DomainEvent) error {
	log := logger.WithFields(ctx,
		"handler", "MatchResultHandler",
		"event_type", evt.EventType,
		"aggregate_id", evt.AggregateID,
	)

	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	n := notification.New(
		notification.KindMatchResult,
		evt.AggregateID,
		payload,
	)

	if err := h.notifier.Notify(ctx, n); err != nil {
		log.Error("error notificando resultado de partido", "error", err)
		return err
	}

	log.Info("resultado de partido notificado", "notification_id", n.ID)
	return nil
}
