package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/wc-fixture/shared/pkg/apperrors"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// eventPublisher implementa ports.EventPublisher publicando el evento
// ResultIngested en el subject fixture.result.ingested del stream FIXTURE_EVENTS.
type eventPublisher struct {
	js jetstream.JetStream
}

var _ interface {
	Publish(ctx context.Context, evt sharedevents.DomainEvent) error
	PublishAll(ctx context.Context, evts []sharedevents.DomainEvent) error
} = (*eventPublisher)(nil)

func NewEventPublisher(js jetstream.JetStream) *eventPublisher {
	return &eventPublisher{js: js}
}

// Publish serializa el evento y lo publica en el subject correspondiente.
func (p *eventPublisher) Publish(ctx context.Context, evt sharedevents.DomainEvent) error {
	subject := subjectFor(evt.EventType)

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("nats_publisher: error serializando evento %q: %w", evt.EventType, err)
	}

	pubAck, err := p.js.Publish(ctx, subject, data)
	if err != nil {
		return apperrors.Unavailable(
			fmt.Sprintf("nats_publisher: error publicando %q en %s: %v", evt.EventType, subject, err),
		)
	}

	logger.FromContext(ctx).Debug("evento publicado",
		"event_type", evt.EventType,
		"subject", subject,
		"stream", pubAck.Stream,
		"seq", pubAck.Sequence,
	)
	return nil
}

// PublishAll publica múltiples eventos en orden.
func (p *eventPublisher) PublishAll(ctx context.Context, evts []sharedevents.DomainEvent) error {
	for _, evt := range evts {
		if err := p.Publish(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

// subjectFor mapea el EventType al subject NATS.
// result-ingestion solo publica un tipo de evento.
func subjectFor(eventType string) string {
	if eventType == "ResultIngested" {
		return string(sharedevents.SubjectResultIngested)
	}
	// Fallback — no debería ocurrir
	return string(sharedevents.SubjectResultIngested)
}
