package events

import "context"

// Publisher es la interfaz de salida para publicar domain events al broker de mensajería.
type Publisher interface {
	Publish(ctx context.Context, evt DomainEvent) error
	PublishAll(ctx context.Context, evts []DomainEvent) error
}
