// Package ports define las interfaces de salida del bounded context notification.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/notification/internal/domain/notification"
)

// Notifier es el puerto de salida para entregar notificaciones a suscriptores.
// Cada implementación concreta maneja un canal de entrega diferente.
type Notifier interface {
	// Notify entrega la notificación a todos los suscriptores activos del torneo.
	Notify(ctx context.Context, n *notification.Notification) error
}

// WebhookRepository es el puerto de salida para gestionar suscriptores webhook.
type WebhookRepository interface {
	// FindByTournament retorna todos los suscriptores activos de un torneo.
	FindByTournament(ctx context.Context, tournamentID uuid.UUID) ([]notification.WebhookSubscriber, error)

	// Save persiste un suscriptor nuevo o actualiza uno existente.
	Save(ctx context.Context, s notification.WebhookSubscriber) error

	// Deactivate desactiva un suscriptor (soft delete).
	Deactivate(ctx context.Context, id uuid.UUID) error
}
