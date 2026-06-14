package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/shared/pkg/apperrors"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// eventPublisher implementa ports.EventPublisher publicando eventos
// al stream FIXTURE_EVENTS en NATS JetStream.
//
// Garantías:
//   - PublishAsync con AckWait de 5s — si no hay ack, retorna error
//   - El subject se deriva del EventType del envelope
//   - El mensaje incluye el envelope completo serializado como JSON
type eventPublisher struct {
	js      jetstream.JetStream
	ackWait time.Duration
}

func NewEventPublisher(js jetstream.JetStream) *eventPublisher {
	return &eventPublisher{
		js:      js,
		ackWait: 5 * time.Second,
	}
}

// Publish serializa el evento y lo publica en el subject correspondiente.
// Bloquea hasta recibir el ack de JetStream o que expire ackWait.
func (p *eventPublisher) Publish(ctx context.Context, evt sharedevents.DomainEvent) error {
	subject, err := subjectForEventType(evt.EventType)
	if err != nil {
		return err
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("nats_publisher: error serializando evento %q: %w", evt.EventType, err)
	}

	pubAck, err := p.js.Publish(ctx, string(subject), data)
	if err != nil {
		return apperrors.Unavailable(fmt.Sprintf("nats_publisher: error publicando %q: %v", evt.EventType, err))
	}

	logger.FromContext(ctx).Debug("evento publicado",
		"event_type", evt.EventType,
		"subject", subject,
		"stream", pubAck.Stream,
		"seq", pubAck.Sequence,
	)
	return nil
}

// PublishAll publica una secuencia de eventos en orden.
// Si alguno falla, retorna el error sin continuar con los siguientes.
func (p *eventPublisher) PublishAll(ctx context.Context, evts []sharedevents.DomainEvent) error {
	for _, evt := range evts {
		if err := p.Publish(ctx, evt); err != nil {
			return fmt.Errorf("nats_publisher: fallo al publicar evento %d/%d (%s): %w",
				1, len(evts), evt.EventType, err)
		}
	}
	return nil
}

// subjectForEventType mapea el EventType del envelope al subject NATS correspondiente.
// Retorna error si el EventType no está registrado — evita publicar en subjects incorrectos.
func subjectForEventType(eventType string) (sharedevents.Subject, error) {
	subjects := map[string]sharedevents.Subject{
		fixture.EventTournamentInitialized:    sharedevents.SubjectMatchResultRegistered,
		fixture.EventMatchResultRegistered:    sharedevents.SubjectMatchResultRegistered,
		fixture.EventGroupStageCompleted:      sharedevents.SubjectGroupStageCompleted,
		fixture.EventKnockoutBracketGenerated: sharedevents.SubjectKnockoutBracketGenerated,
		fixture.EventKnockoutMatchAdvanced:    sharedevents.SubjectKnockoutMatchAdvanced,
		fixture.EventMatchScheduleUpdated:     sharedevents.SubjectMatchScheduleUpdated,
		fixture.EventTournamentFinished:       sharedevents.SubjectTournamentFinished,
	}

	subject, ok := subjects[eventType]
	if !ok {
		return "", fmt.Errorf("nats_publisher: EventType %q sin subject NATS registrado", eventType)
	}
	return subject, nil
}
