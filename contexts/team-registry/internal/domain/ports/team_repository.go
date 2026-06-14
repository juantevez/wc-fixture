// Package ports define las interfaces de salida del bounded context team-registry.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/team-registry/internal/domain/team"
)

// TeamRepository es el puerto de salida para persistir y recuperar equipos.
type TeamRepository interface {
	// GetByID retorna un equipo por su UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*team.Team, error)

	// GetByShortName retorna un equipo por su código de 3 letras (ej: "ARG").
	GetByShortName(ctx context.Context, shortName string) (*team.Team, error)

	// List retorna todos los equipos con filtros opcionales.
	List(ctx context.Context, filters TeamFilters) ([]team.Team, error)

	// Save persiste un equipo nuevo o actualiza uno existente.
	Save(ctx context.Context, t team.Team) error

	// ListConfederations retorna todas las confederaciones FIFA.
	ListConfederations(ctx context.Context) ([]team.Confederation, error)
}

// TeamFilters agrupa los filtros opcionales para la consulta de equipos.
type TeamFilters struct {
	Confederation team.ConfederationCode // "" = todas
	QualifiedOnly bool                   // true = solo clasificados al WC2026
}
