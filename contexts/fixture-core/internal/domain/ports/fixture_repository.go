// Package ports define las interfaces (puertos de salida) del bounded context
// fixture-core. Las implementaciones concretas viven en infrastructure/.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
)

// FixtureRepository es el puerto de salida para persistir y recuperar
// el aggregate Fixture. La implementación concreta usa PostgreSQL con
// event sourcing: persiste los domain events y reconstruye el aggregate
// reproduciéndolos desde el event store.
type FixtureRepository interface {
	// GetByTournamentID carga el aggregate Fixture del torneo dado.
	// Retorna apperrors.NotFound si el torneo no existe.
	GetByTournamentID(ctx context.Context, tournamentID uuid.UUID) (*fixture.Fixture, error)

	// Save persiste el estado del aggregate y los eventos pendientes.
	// Usa optimistic locking sobre fixture.Version para detectar conflictos.
	// Drena fixture.PendingEvents() y los persiste en fixture_events.
	Save(ctx context.Context, f *fixture.Fixture) error
}
