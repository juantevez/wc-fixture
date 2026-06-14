// Package ports define las interfaces de salida del bounded context result-ingestion.
package ports

import (
	"context"

	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// EventPublisher es el puerto de salida para publicar el evento ResultIngested
// hacia fixture-core vía NATS JetStream.
type EventPublisher interface {
	sharedevents.Publisher
}

// IdempotencyStore es el puerto de salida para verificar y registrar
// claves de idempotencia, evitando procesar el mismo resultado dos veces.
//
// Implementación sugerida: tabla ingestion_log en PostgreSQL o Redis con TTL.
type IdempotencyStore interface {
	// Exists reporta si la clave de idempotencia ya fue procesada.
	Exists(ctx context.Context, key string) (bool, error)

	// Register registra la clave como procesada.
	// Retorna ErrDuplicateIngestion si ya existía (race condition).
	Register(ctx context.Context, key string) error
}
