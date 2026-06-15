package notification

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

// NotificationKind clasifica el tipo de notificación.
type NotificationKind string

const (
	KindMatchResult       NotificationKind = "match_result"
	KindGroupCompleted    NotificationKind = "group_completed"
	KindBracketGenerated  NotificationKind = "bracket_generated"
	KindKnockoutAdvanced  NotificationKind = "knockout_advanced"
	KindScheduleUpdated   NotificationKind = "schedule_updated"
	KindTournamentFinished NotificationKind = "tournament_finished"
)

// DeliveryStatus refleja el estado de entrega de una notificación.
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "pending"
	StatusDelivered DeliveryStatus = "delivered"
	StatusFailed    DeliveryStatus = "failed"
	StatusSkipped   DeliveryStatus = "skipped" // sin suscriptores para ese evento
)

// Notification representa una notificación derivada de un evento de dominio,
// lista para ser entregada a los suscriptores registrados.
type Notification struct {
	ID           uuid.UUID
	Kind         NotificationKind
	TournamentID uuid.UUID
	Payload      []byte           // JSON del evento original
	CreatedAt    time.Time
	Status       DeliveryStatus
	DeliveredAt  *time.Time
	Attempts     int
}

// New construye una Notification desde un evento de dominio.
func New(kind NotificationKind, tournamentID uuid.UUID, payload []byte) *Notification {
	return &Notification{
		ID:           uuid.New(),
		Kind:         kind,
		TournamentID: tournamentID,
		Payload:      payload,
		CreatedAt:    time.Now().UTC(),
		Status:       StatusPending,
	}
}

// MarkDelivered marca la notificación como entregada exitosamente.
func (n *Notification) MarkDelivered() {
	now := time.Now().UTC()
	n.Status = StatusDelivered
	n.DeliveredAt = &now
}

// MarkFailed registra un intento fallido.
func (n *Notification) MarkFailed() {
	n.Attempts++
	n.Status = StatusFailed
}

// ── Webhook ───────────────────────────────────────────────────────────────────

// WebhookSubscriber representa un cliente registrado para recibir notificaciones
// de un torneo específico vía HTTP POST.
type WebhookSubscriber struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	URL          string
	Secret       string           // HMAC-SHA256 secret para firmar el payload
	EventTypes   []NotificationKind // nil = todos los eventos
	Active       bool
	CreatedAt    time.Time
}

// NewWebhookSubscriber construye y valida un WebhookSubscriber.
func NewWebhookSubscriber(tournamentID uuid.UUID, rawURL, secret string, eventTypes []NotificationKind) (*WebhookSubscriber, error) {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return nil, ErrInvalidWebhookURL(rawURL)
	}

	return &WebhookSubscriber{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		URL:          rawURL,
		Secret:       secret,
		EventTypes:   eventTypes,
		Active:       true,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// IsSubscribedTo reporta si el suscriptor está interesado en este tipo de evento.
func (w *WebhookSubscriber) IsSubscribedTo(kind NotificationKind) bool {
	if len(w.EventTypes) == 0 {
		return true // nil = todos los eventos
	}
	for _, et := range w.EventTypes {
		if et == kind {
			return true
		}
	}
	return false
}
