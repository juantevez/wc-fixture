package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
)

// DistanceCacheBuilder recalcula y persiste la matriz completa de distancias
// usando ST_DistanceSphere de PostGIS directamente en SQL.
// Se ejecuta una vez al inicializar el torneo o cuando se agrega un venue.
type DistanceCacheBuilder struct {
	pool *pgxpool.Pool
}

func NewDistanceCacheBuilder(pool *pgxpool.Pool) *DistanceCacheBuilder {
	return &DistanceCacheBuilder{pool: pool}
}

// Rebuild recalcula la matriz completa de distancias entre todos los venues
// usando un CROSS JOIN en PostgreSQL/PostGIS y la persiste en venue_distances.
//
// La query calcula ST_DistanceSphere directamente en la base de datos —
// más preciso que Haversine en Go y aprovecha el índice GIST de la columna location.
func (b *DistanceCacheBuilder) Rebuild(ctx context.Context) (int, error) {
	const q = `
		INSERT INTO venue_distances (from_venue_id, to_venue_id, distance_km)
		SELECT
			v1.id AS from_venue_id,
			v2.id AS to_venue_id,
			ROUND(
				(ST_DistanceSphere(v1.location::geometry, v2.location::geometry) / 1000.0)::numeric,
				2
			) AS distance_km
		FROM venues v1
		CROSS JOIN venues v2
		WHERE v1.id < v2.id   -- solo pares únicos, la matriz es simétrica
		ON CONFLICT (from_venue_id, to_venue_id)
			DO UPDATE SET distance_km = EXCLUDED.distance_km`

	tag, err := b.pool.Exec(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("distance_cache: error reconstruyendo matriz: %w", err)
	}

	return int(tag.RowsAffected()), nil
}

// GetFurthestPair retorna el par de venues más lejanos entre sí.
// Útil para auditoría y verificación de la matriz.
func (b *DistanceCacheBuilder) GetFurthestPair(ctx context.Context) (*venue.VenueDistance, error) {
	const q = `
		SELECT from_venue_id, to_venue_id, distance_km
		FROM venue_distances
		ORDER BY distance_km DESC
		LIMIT 1`

	var d venue.VenueDistance
	err := b.pool.QueryRow(ctx, q).Scan(&d.FromVenueID, &d.ToVenueID, &d.DistanceKm)
	if err != nil {
		return nil, fmt.Errorf("distance_cache: error obteniendo par más lejano: %w", err)
	}
	return &d, nil
}

// Stats retorna estadísticas de la matriz de distancias para monitoreo.
type MatrixStats struct {
	TotalPairs  int
	MinDistKm   float64
	MaxDistKm   float64
	AvgDistKm   float64
}

func (b *DistanceCacheBuilder) Stats(ctx context.Context) (*MatrixStats, error) {
	const q = `
		SELECT COUNT(*), MIN(distance_km), MAX(distance_km), AVG(distance_km)
		FROM venue_distances`

	var s MatrixStats
	if err := b.pool.QueryRow(ctx, q).Scan(
		&s.TotalPairs, &s.MinDistKm, &s.MaxDistKm, &s.AvgDistKm,
	); err != nil {
		return nil, fmt.Errorf("distance_cache: error obteniendo stats: %w", err)
	}
	return &s, nil
}
