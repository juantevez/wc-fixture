package handlers

import (
	"context"
	"encoding/json"

	"github.com/wc-fixture/notification/internal/domain/notification"
	"github.com/wc-fixture/notification/internal/domain/ports"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// BracketGeneratedHandler procesa el evento KnockoutBracketGenerated
// y notifica a los suscriptores — es el evento más esperado del torneo.
type BracketGeneratedHandler struct {
	notifier ports.Notifier
}

func NewBracketGeneratedHandler(notifier ports.Notifier) *BracketGeneratedHandler {
	return &BracketGeneratedHandler{notifier: notifier}
}

func (h *BracketGeneratedHandler) Handle(ctx context.Context, evt sharedevents.DomainEvent) error {
	log := logger.WithFields(ctx,
		"handler", "BracketGeneratedHandler",
		"tournament_id", evt.AggregateID,
	)

	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	n := notification.New(
		notification.KindBracketGenerated,
		evt.AggregateID,
		payload,
	)

	if err := h.notifier.Notify(ctx, n); err != nil {
		log.Error("error notificando generación de bracket", "error", err)
		return err
	}

	log.Info("bracket eliminatorio notificado", "notification_id", n.ID)
	return nil
}
