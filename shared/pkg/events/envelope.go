// Package events define la estructura base (envelope) para todos los domain
// events del sistema. Cada bounded context serializa su payload concreto
// dentro del campo Payload como json.RawMessage, manteniendo el desacoplamiento
// entre productor y consumidor.
//
// Estructura en NATS:
//
//	subject: fixture.result.registered
//	payload: { "event_id": "...", "event_type": "MatchResultRegistered", ... }
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DomainEvent es el envelope genérico que envuelve cualquier evento de dominio.
// El campo Payload contiene el JSON del evento concreto y es responsabilidad
// del consumidor deserializarlo al tipo correcto según EventType.
type DomainEvent struct {
	// EventID es el identificador único del evento (UUID v4).
	EventID uuid.UUID `json:"event_id"`

	// EventType es el nombre canónico del evento, usado por los consumidores
	// para dispatch. Ejemplos: "MatchResultRegistered", "KnockoutBracketGenerated".
	EventType string `json:"event_type"`

	// OccurredAt es el momento en que el evento ocurrió en el dominio.
	// Siempre en UTC.
	OccurredAt time.Time `json:"occurred_at"`

	// Version es la versión del schema del evento. Permite evolución compatible.
	Version int `json:"version"`

	// AggregateID es el ID del aggregate que generó el evento.
	// Permite al consumidor correlacionar eventos de un mismo aggregate.
	AggregateID uuid.UUID `json:"aggregate_id"`

	// AggregateType es el nombre del aggregate. Ej: "Fixture", "Venue".
	AggregateType string `json:"aggregate_type"`

	// Payload contiene el cuerpo concreto del evento serializado como JSON.
	// Cada consumidor lo deserializa al tipo apropiado según EventType.
	Payload json.RawMessage `json:"payload"`
}

// New construye un DomainEvent con un nuevo UUID y el timestamp actual (UTC).
func New(eventType string, aggregateID uuid.UUID, aggregateType string, payload any) (DomainEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return DomainEvent{}, err
	}
	return DomainEvent{
		EventID:       uuid.New(),
		EventType:     eventType,
		OccurredAt:    time.Now().UTC(),
		Version:       1,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Payload:       raw,
	}, nil
}

// DecodePayload deserializa el Payload del envelope al tipo T.
//
//	var result MatchResultRegistered
//	if err := evt.DecodePayload(&result); err != nil { ... }
func (e DomainEvent) DecodePayload(dst any) error {
	return json.Unmarshal(e.Payload, dst)
}
