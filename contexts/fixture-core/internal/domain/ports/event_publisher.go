package ports

import (
	"context"

	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// EventPublisher es el puerto de salida para publicar domain events
// al broker de mensajería (NATS JetStream).
// Es un alias semántico de shared/pkg/events.Publisher para que el dominio
// no dependa directamente del paquete de infraestructura.
type EventPublisher interface {
	sharedevents.Publisher
}

// FixtureEventHandler es la firma del handler de eventos entrantes
// que fixture-core consume desde result-ingestion.
type FixtureEventHandler func(ctx context.Context, evt sharedevents.DomainEvent) error
