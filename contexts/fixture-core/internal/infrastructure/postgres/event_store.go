package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// eventRow es la fila tal como se lee de la tabla fixture_events.
type eventRow struct {
	ID               int64
	TournamentID     uuid.UUID
	EventType        string
	EventVersion     int
	Payload          []byte
	OccurredAt       time.Time
	AggregateVersion int64
}

// EventStore gestiona la persistencia y lectura de domain events
// en la tabla fixture_events. Es la fuente de verdad del aggregate Fixture.
type EventStore struct {
	pool *pgxpool.Pool
}

func NewEventStore(pool *pgxpool.Pool) *EventStore {
	return &EventStore{pool: pool}
}

// AppendEvents persiste los eventos pendientes del aggregate dentro de una
// transacción existente. Usa NEXTVAL implícito del BIGSERIAL para el orden.
func (s *EventStore) AppendEvents(ctx context.Context, tx pgx.Tx, tournamentID uuid.UUID, events []sharedevents.DomainEvent, fromVersion int64) error {
	if len(events) == 0 {
		return nil
	}

	const q = `
		INSERT INTO fixture_events
			(tournament_id, event_type, event_version, payload, occurred_at, aggregate_version)
		VALUES
			($1, $2, $3, $4, $5, $6)`

	for i, evt := range events {
		aggregateVersion := fromVersion + int64(i) + 1
		if _, err := tx.Exec(ctx, q,
			tournamentID,
			evt.EventType,
			evt.Version,
			[]byte(evt.Payload),
			evt.OccurredAt,
			aggregateVersion,
		); err != nil {
			return fmt.Errorf("event_store: error insertando evento %q: %w", evt.EventType, err)
		}
	}
	return nil
}

// LoadEvents carga todos los eventos de un torneo desde el event store,
// ordenados por aggregate_version ASC. Se usa para reconstruir el aggregate.
func (s *EventStore) LoadEvents(ctx context.Context, tournamentID uuid.UUID) ([]sharedevents.DomainEvent, error) {
	const q = `
		SELECT id, tournament_id, event_type, event_version, payload, occurred_at, aggregate_version
		FROM fixture_events
		WHERE tournament_id = $1
		ORDER BY aggregate_version ASC`

	rows, err := s.pool.Query(ctx, q, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("event_store: error cargando eventos: %w", err)
	}
	defer rows.Close()

	var events []sharedevents.DomainEvent
	for rows.Next() {
		var row eventRow
		if err := rows.Scan(
			&row.ID, &row.TournamentID, &row.EventType, &row.EventVersion,
			&row.Payload, &row.OccurredAt, &row.AggregateVersion,
		); err != nil {
			return nil, fmt.Errorf("event_store: error escaneando evento: %w", err)
		}
		events = append(events, sharedevents.DomainEvent{
			EventID:       uuid.New(),
			EventType:     row.EventType,
			OccurredAt:    row.OccurredAt,
			Version:       row.EventVersion,
			AggregateID:   row.TournamentID,
			AggregateType: "Fixture",
			Payload:       json.RawMessage(row.Payload),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("event_store: error iterando eventos: %w", err)
	}

	return events, nil
}

// LoadEventsFrom carga eventos desde una versión específica del aggregate.
// Útil para reconstrucción parcial desde un snapshot.
func (s *EventStore) LoadEventsFrom(ctx context.Context, tournamentID uuid.UUID, fromVersion int64) ([]sharedevents.DomainEvent, error) {
	const q = `
		SELECT id, tournament_id, event_type, event_version, payload, occurred_at, aggregate_version
		FROM fixture_events
		WHERE tournament_id = $1
		  AND aggregate_version > $2
		ORDER BY aggregate_version ASC`

	rows, err := s.pool.Query(ctx, q, tournamentID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("event_store: error cargando eventos desde v%d: %w", fromVersion, err)
	}
	defer rows.Close()

	var events []sharedevents.DomainEvent
	for rows.Next() {
		var row eventRow
		if err := rows.Scan(
			&row.ID, &row.TournamentID, &row.EventType, &row.EventVersion,
			&row.Payload, &row.OccurredAt, &row.AggregateVersion,
		); err != nil {
			return nil, fmt.Errorf("event_store: error escaneando evento: %w", err)
		}
		events = append(events, sharedevents.DomainEvent{
			EventType:     row.EventType,
			OccurredAt:    row.OccurredAt,
			Version:       row.EventVersion,
			AggregateID:   row.TournamentID,
			AggregateType: "Fixture",
			Payload:       json.RawMessage(row.Payload),
		})
	}

	return events, rows.Err()
}
