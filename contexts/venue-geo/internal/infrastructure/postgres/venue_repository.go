package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/venue-geo/internal/domain/ports"
	"github.com/wc-fixture/venue-geo/internal/domain/venue"
)

// venueRepo implementa ports.VenueRepository.
// Usa PostGIS para queries geoespaciales (ST_DWithin, ST_DistanceSphere).
type venueRepo struct {
	pool *pgxpool.Pool
}

var _ ports.VenueRepository = (*venueRepo)(nil)

func NewVenueRepository(pool *pgxpool.Pool) ports.VenueRepository {
	return &venueRepo{pool: pool}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *venueRepo) GetByID(ctx context.Context, id uuid.UUID) (*venue.Venue, error) {
	const q = `
		SELECT id, name, city, country, country_code,
			   capacity, surface, timezone, altitude_m,
			   ST_Y(location::geometry) AS lat,
			   ST_X(location::geometry) AS lon
		FROM venues
		WHERE id = $1`

	v, err := r.scanVenue(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("venue", id.String())
	}
	if err != nil {
		return nil, fmt.Errorf("venue_repo: error consultando venue: %w", err)
	}
	return v, nil
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *venueRepo) List(ctx context.Context, country venue.Country) ([]venue.Venue, error) {
	q := `
		SELECT id, name, city, country, country_code,
			   capacity, surface, timezone, altitude_m,
			   ST_Y(location::geometry) AS lat,
			   ST_X(location::geometry) AS lon
		FROM venues`

	args := []any{}
	if country != "" {
		q += " WHERE country = $1"
		args = append(args, string(country))
	}
	q += " ORDER BY country, city, name"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("venue_repo: error listando venues: %w", err)
	}
	defer rows.Close()

	var venues []venue.Venue
	for rows.Next() {
		v, err := r.scanVenue(rows)
		if err != nil {
			return nil, err
		}
		venues = append(venues, *v)
	}
	return venues, rows.Err()
}

// ── Save ──────────────────────────────────────────────────────────────────────

func (r *venueRepo) Save(ctx context.Context, v venue.Venue) error {
	const q = `
		INSERT INTO venues
			(id, name, city, country, country_code,
			 capacity, surface, timezone, altitude_m, location)
		VALUES
			($1, $2, $3, $4, $5,
			 $6, $7, $8, $9,
			 ST_SetSRID(ST_MakePoint($10, $11), 4326))
		ON CONFLICT (id) DO UPDATE SET
			name         = EXCLUDED.name,
			city         = EXCLUDED.city,
			country      = EXCLUDED.country,
			country_code = EXCLUDED.country_code,
			capacity     = EXCLUDED.capacity,
			surface      = EXCLUDED.surface,
			timezone     = EXCLUDED.timezone,
			altitude_m   = EXCLUDED.altitude_m,
			location     = EXCLUDED.location,
			updated_at   = NOW()`

	if _, err := r.pool.Exec(ctx, q,
		v.ID, v.Name, v.City, string(v.Country), v.CountryCode,
		v.Capacity, string(v.Surface), v.Timezone, v.AltitudeM,
		v.Location.Lon, v.Location.Lat, // ST_MakePoint(lon, lat)
	); err != nil {
		return fmt.Errorf("venue_repo: error guardando venue %s: %w", v.ID, err)
	}
	return nil
}

// ── FindNearby ────────────────────────────────────────────────────────────────
// Usa ST_DWithin sobre geography para filtrar por radio esférico exacto,
// y ST_DistanceSphere para calcular la distancia de retorno.

func (r *venueRepo) FindNearby(ctx context.Context, center venue.GeoPoint, radiusKm float64) ([]venue.NearbyVenue, error) {
	const q = `
		SELECT id, name, city, country, country_code,
			   capacity, surface, timezone, altitude_m,
			   ST_Y(location::geometry)                                  AS lat,
			   ST_X(location::geometry)                                  AS lon,
			   ST_DistanceSphere(
			       location::geometry,
			       ST_SetSRID(ST_MakePoint($1, $2), 4326)
			   ) / 1000.0                                                AS distance_km
		FROM venues
		WHERE ST_DWithin(
			location::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3 * 1000  -- radio en metros
		)
		ORDER BY distance_km ASC`

	rows, err := r.pool.Query(ctx, q, center.Lon, center.Lat, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("venue_repo: error en FindNearby: %w", err)
	}
	defer rows.Close()

	var results []venue.NearbyVenue
	for rows.Next() {
		var nv venue.NearbyVenue
		var countryStr, surfaceStr string

		if err := rows.Scan(
			&nv.ID, &nv.Name, &nv.City, &countryStr, &nv.CountryCode,
			&nv.Capacity, &surfaceStr, &nv.Timezone, &nv.AltitudeM,
			&nv.Location.Lat, &nv.Location.Lon,
			&nv.DistanceKm,
		); err != nil {
			return nil, fmt.Errorf("venue_repo: error escaneando nearby venue: %w", err)
		}

		nv.Country = venue.Country(countryStr)
		nv.Surface = venue.Surface(surfaceStr)
		results = append(results, nv)
	}
	return results, rows.Err()
}

// ── GetDistance ───────────────────────────────────────────────────────────────

func (r *venueRepo) GetDistance(ctx context.Context, fromID, toID uuid.UUID) (float64, error) {
	if fromID == toID {
		return 0, apperrors.Validation("los venues deben ser distintos")
	}

	// Intentar primero desde la cache venue_distances
	const qCache = `
		SELECT distance_km
		FROM venue_distances
		WHERE (from_venue_id = $1 AND to_venue_id = $2)
		   OR (from_venue_id = $2 AND to_venue_id = $1)
		LIMIT 1`

	var distKm float64
	err := r.pool.QueryRow(ctx, qCache, fromID, toID).Scan(&distKm)
	if err == nil {
		return distKm, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("venue_repo: error consultando cache de distancia: %w", err)
	}

	// Cache miss — calcular con PostGIS ST_DistanceSphere
	const qCalc = `
		SELECT ST_DistanceSphere(v1.location::geometry, v2.location::geometry) / 1000.0
		FROM venues v1, venues v2
		WHERE v1.id = $1 AND v2.id = $2`

	err = r.pool.QueryRow(ctx, qCalc, fromID, toID).Scan(&distKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apperrors.NotFound("venue", fmt.Sprintf("%s o %s", fromID, toID))
	}
	if err != nil {
		return 0, fmt.Errorf("venue_repo: error calculando distancia con PostGIS: %w", err)
	}

	return distKm, nil
}

// ── GetDistanceMatrix ─────────────────────────────────────────────────────────

func (r *venueRepo) GetDistanceMatrix(ctx context.Context) (venue.DistanceMatrix, error) {
	const q = `
		SELECT from_venue_id, to_venue_id, distance_km
		FROM venue_distances
		ORDER BY from_venue_id, to_venue_id`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("venue_repo: error cargando matriz de distancias: %w", err)
	}
	defer rows.Close()

	matrix := make(venue.DistanceMatrix)
	for rows.Next() {
		var from, to uuid.UUID
		var distKm float64
		if err := rows.Scan(&from, &to, &distKm); err != nil {
			return nil, fmt.Errorf("venue_repo: error escaneando distancia: %w", err)
		}
		matrix.Set(from, to, distKm)
	}
	return matrix, rows.Err()
}

// ── SaveDistanceMatrix ────────────────────────────────────────────────────────

func (r *venueRepo) SaveDistanceMatrix(ctx context.Context, matrix venue.DistanceMatrix) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("venue_repo: error iniciando tx para matriz: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
		INSERT INTO venue_distances (from_venue_id, to_venue_id, distance_km)
		VALUES ($1, $2, $3)
		ON CONFLICT (from_venue_id, to_venue_id) DO UPDATE
			SET distance_km = EXCLUDED.distance_km`

	for key, distKm := range matrix {
		if _, err := tx.Exec(ctx, q, key[0], key[1], distKm); err != nil {
			return fmt.Errorf("venue_repo: error guardando distancia: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ── Helper de scan ────────────────────────────────────────────────────────────

// scanVenue escanea una fila con la estructura de venue desde cualquier
// scanner compatible (pgx.Row o pgx.Rows).
func (r *venueRepo) scanVenue(scanner interface {
	Scan(dest ...any) error
}) (*venue.Venue, error) {
	var v venue.Venue
	var countryStr, surfaceStr string

	if err := scanner.Scan(
		&v.ID, &v.Name, &v.City, &countryStr, &v.CountryCode,
		&v.Capacity, &surfaceStr, &v.Timezone, &v.AltitudeM,
		&v.Location.Lat, &v.Location.Lon,
	); err != nil {
		return nil, err
	}

	v.Country = venue.Country(countryStr)
	v.Surface = venue.Surface(surfaceStr)
	return &v, nil
}
